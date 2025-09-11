package handlers

import (
	"context"
	"log/slog"
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
}

func NewHotelsHandlers(chatRepo llmchat.Repository) *HotelsHandlers {
	return &HotelsHandlers{
		chatRepo:         chatRepo,
		itineraryService: services.NewItineraryService(),
	}
}

// HandleHotelsPage handles the main hotels page logic
func (h *HotelsHandlers) HandleHotelsPage(c *gin.Context) templ.Component {
	query := c.Query("q")
	sessionIdParam := c.Query("sessionId")

	logger.Log.Info("Hotels page accessed",
		zap.String("ip", c.ClientIP()),
		zap.String("query", query),
		zap.String("sessionId", sessionIdParam))

	// Handle session-based loading
	if sessionIdParam != "" {
		return h.loadHotelsBySession(sessionIdParam)
	}

	// Handle query-based loading
	if query != "" {
		return hotels.HotelsPageWithQuery(query)
	}

	// Default hotels page
	return hotels.HotelsPage()
}

// loadHotelsBySession loads hotels using IDENTICAL logic as itinerary
func (h *HotelsHandlers) loadHotelsBySession(sessionIdParam string) templ.Component {
	logger.Log.Info("Attempting to load hotels from cache", zap.String("sessionID", sessionIdParam))

	// Try complete cache first (IDENTICAL to itinerary logic)
	if completeData, found := middleware.CompleteItineraryCache.Get(sessionIdParam); found {
		logger.Log.Info("Complete hotels found in cache. Rendering results.",
			zap.String("city", completeData.GeneralCityData.City),
			zap.Int("generalPOIs", len(completeData.PointsOfInterest)),
			zap.Int("personalizedPOIs", len(completeData.AIItineraryResponse.PointsOfInterest)))

		// Filter POIs for hotels and render (IDENTICAL to itinerary results pattern)
		hotelPOIs := filterPOIsForHotels(completeData.PointsOfInterest)
		return results.HotelsResults(
			completeData.GeneralCityData,
			hotelPOIs,
			true, true, 5, []string{})
	}

	// Try legacy cache (IDENTICAL to itinerary logic)
	if itineraryData, found := middleware.ItineraryCache.Get(sessionIdParam); found {
		logger.Log.Info("Legacy hotels found in cache. Rendering results.",
			zap.Int("personalizedPOIs", len(itineraryData.PointsOfInterest)))

		// Create empty city data for legacy cached data (IDENTICAL to itinerary)
		emptyCityData := models.GeneralCityData{}
		
		// Filter hotels from legacy data
		hotelPOIs := filterPOIsForHotels(itineraryData.PointsOfInterest)
		return results.HotelsResults(emptyCityData, hotelPOIs, true, true, 5, []string{})
	}

	// Load from database (IDENTICAL to itinerary logic)
	return h.loadHotelsFromDatabase(sessionIdParam)
}

// loadHotelsFromDatabase loads hotels from database when not found in cache
func (h *HotelsHandlers) loadHotelsFromDatabase(sessionIdParam string) templ.Component {
	logger.Log.Info("Hotels not found in cache, attempting to load from database", zap.String("sessionID", sessionIdParam))

	// Parse sessionID as UUID
	sessionID, err := uuid.Parse(sessionIdParam)
	if err != nil {
		logger.Log.Warn("Invalid session ID format", zap.String("sessionID", sessionIdParam), zap.Error(err))
		return results.PageNotFound("Invalid session ID")
	}

	// Get the latest interaction for this session from database
	ctx := context.Background()
	interaction, err := h.chatRepo.GetLatestInteractionBySessionID(ctx, sessionID)
	if err != nil || interaction == nil {
		logger.Log.Warn("No interaction found in database for session",
			zap.String("sessionID", sessionIdParam),
			zap.Error(err))
		// Return empty results instead of PageNotFound - data might still be processing
		emptyCityData := models.GeneralCityData{}
		emptyHotels := []models.HotelDetailedInfo{}
		return results.HotelsResults(emptyCityData, emptyHotels, true, true, 5, []string{})
	}

	// Parse the stored response as complete data
	completeData, err := h.itineraryService.ParseCompleteItineraryResponse(interaction.ResponseText, slog.Default())
	if err != nil || completeData == nil {
		logger.Log.Warn("Could not parse complete data from stored response",
			zap.String("sessionID", sessionIdParam),
			zap.Error(err))
		// Return empty results instead of PageNotFound for parsing errors
		emptyCityData := models.GeneralCityData{}
		emptyHotels := []models.HotelDetailedInfo{}
		return results.HotelsResults(emptyCityData, emptyHotels, true, true, 5, []string{})
	}

	logger.Log.Info("Successfully loaded complete data from database for hotels",
		zap.String("sessionID", sessionIdParam),
		zap.String("city", completeData.GeneralCityData.City),
		zap.Int("totalPOIs", len(completeData.PointsOfInterest)))

	// Filter POIs for hotels and render (IDENTICAL to itinerary results pattern)
	hotelPOIs := filterPOIsForHotels(completeData.PointsOfInterest)
	return results.HotelsResults(
		completeData.GeneralCityData,
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