package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/app/internal/features/hotels"
	"github.com/FACorreiaa/go-templui/app/internal/features/results"
	"github.com/FACorreiaa/go-templui/app/internal/models"
	llmchat "github.com/FACorreiaa/go-templui/app/pkg/domain/chat_prompt"
	"github.com/FACorreiaa/go-templui/app/pkg/logger"
	"github.com/FACorreiaa/go-templui/app/pkg/middleware"
	"github.com/FACorreiaa/go-templui/app/pkg/services"
)

type HotelsHandlers struct {
	chatRepo         llmchat.Repository
	itineraryService *services.ItineraryService
	logger           *slog.Logger
}

func NewHotelsHandlers(chatRepo llmchat.Repository, logger *slog.Logger) *HotelsHandlers {
	return &HotelsHandlers{
		chatRepo:         chatRepo,
		itineraryService: services.NewItineraryService(),
		logger:           logger,
	}
}

// HandleHotelsPage handles the main hotels page logic
func (h *HotelsHandlers) HandleHotelsPage(c *gin.Context) templ.Component {
	query := c.Query("q")
	sessionIDParam := c.Query("sessionId")

	logger.Log.Info("Hotels page accessed",
		zap.String("ip", c.ClientIP()),
		zap.String("query", query),
		zap.String("sessionId", sessionIDParam))

	// Handle session-based loading
	if sessionIDParam != "" {
		return h.loadHotelsBySession(sessionIDParam)
	}

	// Handle query-based loading
	if query != "" {
		return hotels.HotelsPageWithQuery(query)
	}

	// Default hotels page
	return hotels.HotelsPage()
}

// loadHotelsBySession loads hotels using IDENTICAL logic as itinerary
func (h *HotelsHandlers) loadHotelsBySession(sessionIDParam string) templ.Component {
	logger.Log.Info("Attempting to load hotels from cache", zap.String("sessionID", sessionIDParam))

	// Try complete cache first (IDENTICAL to itinerary logic)
	if completeData, found := middleware.CompleteItineraryCache.Get(sessionIDParam); found {
		jsonData, err := json.MarshalIndent(completeData, "", "  ")
		if err != nil {
			logger.Log.Error("Failed to marshal completeData to JSON", zap.Error(err))
		} else {
			logger.Log.Info("Complete itinerary JSON structure", zap.String("json", string(jsonData)))
		}

		if err := os.WriteFile("completeData.json", jsonData, 0644); err != nil {
			logger.Log.Error("Failed to write JSON to file", zap.Error(err))
		}
		logger.Log.Info("Complete hotels found in cache. Rendering results.",
			zap.String("city", completeData.GeneralCityData.City),
			zap.Int("generalPOIs", len(completeData.PointsOfInterest)),
			zap.Int("personalizedPOIs", len(completeData.AIItineraryResponse.PointsOfInterest)))

		// Filter POIs for hotels and render (IDENTICAL to itinerary results pattern)
		hotelPOIs := filterPOIsForHotels(completeData.PointsOfInterest)
		return results.HotelResults(
			hotelPOIs,
			true, true, 5, []string{})
	}

	// Try legacy cache (IDENTICAL to itinerary logic)
	if itineraryData, found := middleware.ItineraryCache.Get(sessionIDParam); found {
		logger.Log.Info("Legacy hotels found in cache. Rendering results.",
			zap.Int("personalizedPOIs", len(itineraryData.PointsOfInterest)))

		// Filter hotels from legacy data
		hotelPOIs := filterPOIsForHotels(itineraryData.PointsOfInterest)
		return results.HotelResults(hotelPOIs, true, true, 5, []string{})
	}

	// Load from database (IDENTICAL to itinerary logic)
	return h.loadHotelsFromDatabase(sessionIDParam)
}

// loadHotelsFromDatabase loads hotels from database when not found in cache
func (h *HotelsHandlers) loadHotelsFromDatabase(sessionIDParam string) templ.Component {
	logger.Log.Info("Hotels not found in cache, attempting to load from database", zap.String("sessionID", sessionIDParam))

	// Parse sessionID as UUID
	sessionID, err := uuid.Parse(sessionIDParam)
	if err != nil {
		logger.Log.Warn("Invalid session ID format", zap.String("sessionID", sessionIDParam), zap.Error(err))
		return results.PageNotFound("Invalid session ID")
	}

	// Get the latest interaction for this session from database
	ctx := context.Background()
	interaction, err := h.chatRepo.GetLatestInteractionBySessionID(ctx, sessionID)
	if err != nil || interaction == nil {
		logger.Log.Warn("No interaction found in database for session",
			zap.String("sessionID", sessionIDParam),
			zap.Error(err))
		// Return empty results instead of PageNotFound - data might still be processing
		emptyHotels := []models.HotelDetailedInfo{}
		return results.HotelResults(emptyHotels, true, true, 5, []string{})
	}

	// Parse the stored response as complete data
	completeData, err := h.itineraryService.ParseCompleteItineraryResponse(interaction.ResponseText, slog.Default())
	if err != nil || completeData == nil {
		logger.Log.Warn("Could not parse complete data from stored response",
			zap.String("sessionID", sessionIDParam),
			zap.Error(err))
		// Return empty results instead of PageNotFound for parsing errors
		emptyHotels := []models.HotelDetailedInfo{}
		return results.HotelResults(emptyHotels, true, true, 5, []string{})
	}

	logger.Log.Info("Successfully loaded complete data from database for hotels",
		zap.String("sessionID", sessionIDParam),
		zap.String("city", completeData.GeneralCityData.City),
		zap.Int("totalPOIs", len(completeData.PointsOfInterest)))

	// Filter POIs for hotels and render (IDENTICAL to itinerary results pattern)
	hotelPOIs := filterPOIsForHotels(completeData.PointsOfInterest)
	return results.HotelResults(
		hotelPOIs,
		true, true, 5, []string{})
}

// filterPOIsForHotels filters POIs to show only accommodation-related categories
func filterPOIsForHotels(allPOIs []models.POIDetailedInfo) []models.HotelDetailedInfo {
	var hotelPOIs []models.HotelDetailedInfo
	hotelCategories := map[string]bool{
		"hotel":         true,
		"accommodation": true,
		"lodging":       true,
		"hostel":        true,
		"resort":        true,
		"guesthouse":    true,
		"apartment":     true,
		"villa":         true,
		"motel":         true,
		"inn":           true,
		"b&b":           true,
		"bnb":           true,
	}

	for _, poi := range allPOIs {
		if hotelCategories[strings.ToLower(poi.Category)] {
			hotelPOIs = append(hotelPOIs, convertPOIToHotel(poi))
		}
	}

	return hotelPOIs
}

// convertPOIToHotel converts a POIDetailedInfo to HotelDetailedInfo
func convertPOIToHotel(poi models.POIDetailedInfo) models.HotelDetailedInfo {
	var phoneNumber *string
	if poi.PhoneNumber != "" {
		phoneNumber = &poi.PhoneNumber
	}

	var website *string
	if poi.Website != "" {
		website = &poi.Website
	}

	var priceRange *string
	if poi.PriceRange != "" {
		priceRange = &poi.PriceRange
	}

	var openingHours *string
	if len(poi.OpeningHours) > 0 {
		// Convert map to string representation
		hoursStr := ""
		for day, hours := range poi.OpeningHours {
			if hoursStr != "" {
				hoursStr += ", "
			}
			hoursStr += day + ": " + hours
		}
		if hoursStr != "" {
			openingHours = &hoursStr
		}
	}

	return models.HotelDetailedInfo{
		ID:               poi.ID,
		City:             poi.City,
		Name:             poi.Name,
		Latitude:         poi.Latitude,
		Longitude:        poi.Longitude,
		Category:         poi.Category,
		Description:      poi.Description,
		Address:          poi.Address,
		PhoneNumber:      phoneNumber,
		Website:          website,
		OpeningHours:     openingHours,
		PriceRange:       priceRange,
		Rating:           poi.Rating,
		Tags:             poi.Tags,
		Images:           poi.Images,
		LlmInteractionID: poi.LlmInteractionID,
	}
}

// HandleHotelsPageSSE handles the hotels page with SSE support
func (h *HotelsHandlers) HandleHotelsPageSSE(c *gin.Context) templ.Component {
	query := c.Query("q")
	sessionIDParam := c.Query("sessionId")

	logger.Log.Info("Hotels SSE page accessed",
		zap.String("ip", c.ClientIP()),
		zap.String("query", query),
		zap.String("sessionId", sessionIDParam))

	if sessionIDParam == "" {
		logger.Log.Info("Direct navigation to /hotels SSE. Showing default page.")
		return hotels.HotelsPage()
	}

	// Load hotels data for session with SSE support
	return h.loadHotelsBySession(sessionIDParam)
}
