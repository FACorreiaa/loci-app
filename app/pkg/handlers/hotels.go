package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

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
		return results.HotelsResults(
			completeData.GeneralCityData,
			hotelPOIs,
			true, true, 5, []string{})
	}

	// Try legacy cache (IDENTICAL to itinerary logic)
	if itineraryData, found := middleware.ItineraryCache.Get(sessionIDParam); found {
		logger.Log.Info("Legacy hotels found in cache. Rendering results.",
			zap.Int("personalizedPOIs", len(itineraryData.PointsOfInterest)))

		// Create empty city data for legacy cached data (IDENTICAL to itinerary)
		emptyCityData := models.GeneralCityData{}

		// Filter hotels from legacy data
		hotelPOIs := filterPOIsForHotels(itineraryData.PointsOfInterest)
		return results.HotelsResults(emptyCityData, hotelPOIs, true, true, 5, []string{})
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
		emptyCityData := models.GeneralCityData{}
		emptyHotels := []models.HotelDetailedInfo{}
		return results.HotelsResults(emptyCityData, emptyHotels, true, true, 5, []string{})
	}

	// Parse the stored response as complete data
	completeData, err := h.itineraryService.ParseCompleteItineraryResponse(interaction.ResponseText, slog.Default())
	if err != nil || completeData == nil {
		logger.Log.Warn("Could not parse complete data from stored response",
			zap.String("sessionID", sessionIDParam),
			zap.Error(err))
		// Return empty results instead of PageNotFound for parsing errors
		emptyCityData := models.GeneralCityData{}
		emptyHotels := []models.HotelDetailedInfo{}
		return results.HotelsResults(emptyCityData, emptyHotels, true, true, 5, []string{})
	}

	logger.Log.Info("Successfully loaded complete data from database for hotels",
		zap.String("sessionID", sessionIDParam),
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
	return h.loadHotelsBySessionSSE(sessionIDParam)
}

// loadHotelsBySessionSSE loads hotels with SSE support
func (h *HotelsHandlers) loadHotelsBySessionSSE(sessionIDParam string) templ.Component {
	logger.Log.Info("Attempting to load hotels from cache with SSE", zap.String("sessionID", sessionIDParam))

	// Try complete cache first
	if completeData, found := middleware.CompleteItineraryCache.Get(sessionIDParam); found {
		// Filter POIs for hotels
		hotelPOIs := filterPOIsForHotels(completeData.PointsOfInterest)

		// Print JSON data for debugging what hotels will be displayed
		jsonData, err := json.MarshalIndent(map[string]interface{}{
			"city_data":  completeData.GeneralCityData,
			"hotel_pois": hotelPOIs,
		}, "", "  ")
		if err != nil {
			logger.Log.Error("Failed to marshal hotels data to JSON", zap.Error(err))
		} else {
			logger.Log.Info("Hotels data being displayed in view", zap.String("json", string(jsonData)))
		}

		logger.Log.Info("Complete hotels found in cache. Rendering SSE results with data.",
			zap.String("city", completeData.GeneralCityData.City),
			zap.Int("totalPOIs", len(completeData.PointsOfInterest)),
			zap.Int("hotelPOIs", len(hotelPOIs)))

		return results.HotelsResultsSSE(
			sessionIDParam,
			completeData.GeneralCityData,
			hotelPOIs,
			true) // hasData = true
	}

	// Try legacy cache
	if itineraryData, found := middleware.ItineraryCache.Get(sessionIDParam); found {
		logger.Log.Info("Legacy hotels found in cache. Rendering SSE results with data.",
			zap.Int("personalizedPOIs", len(itineraryData.PointsOfInterest)))

		// Create empty city data for legacy cached data
		emptyCityData := models.GeneralCityData{}

		// Filter hotels from legacy data
		hotelPOIs := filterPOIsForHotels(itineraryData.PointsOfInterest)

		return results.HotelsResultsSSE(
			sessionIDParam,
			emptyCityData,
			hotelPOIs,
			true) // hasData = true
	}

	// No cached data found - show loading interface with SSE
	logger.Log.Info("No hotels found in cache. Rendering SSE loading interface.",
		zap.String("sessionID", sessionIDParam))

	emptyCityData := models.GeneralCityData{}
	emptyHotels := []models.HotelDetailedInfo{}

	return results.HotelsResultsSSE(
		sessionIDParam,
		emptyCityData,
		emptyHotels,
		false) // hasData = false, will show loading and connect to SSE
}

// HandleHotelsSSE handles Server-Sent Events for hotel updates
func (h *HotelsHandlers) HandleHotelsSSE(c *gin.Context) {
	sessionID := c.Query("sessionId")
	if sessionID == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	logger.Log.Info("SSE connection established for hotels",
		zap.String("sessionId", sessionID))

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Create a channel for updates
	updateChan := make(chan models.ItinerarySSEEvent)
	defer close(updateChan)

	// Start monitoring for updates in a separate goroutine
	go h.monitorHotelUpdates(sessionID, updateChan)

	// Stream updates to client
	flusher := c.Writer.(http.Flusher)
	for {
		select {
		case event := <-updateChan:
			if event.Type == "complete" {
				logger.Log.Info("Sending hotels completion event",
					zap.String("sessionId", sessionID))

				// Send final completion event
				c.SSEvent("hotels-complete", map[string]interface{}{
					"sessionId": sessionID,
					"message":   "Hotel search complete",
				})
				flusher.Flush()
				return
			}

			// Send HTML fragment updates based on event type
			switch event.Type {
			case "header-update":
				if data, ok := event.Data.(map[string]interface{}); ok {
					if completeData, exists := data["completeData"]; exists {
						if complete, valid := completeData.(models.AiCityResponse); valid {
							hotelPOIs := filterPOIsForHotels(complete.PointsOfInterest)
							headerHTML := h.renderHotelsHeaderHTML(complete.GeneralCityData, hotelPOIs)
							c.SSEvent("hotels-header", headerHTML)
						}
					}
				}
			case "content-update":
				if data, ok := event.Data.(map[string]interface{}); ok {
					if completeData, exists := data["completeData"]; exists {
						if complete, valid := completeData.(models.AiCityResponse); valid {
							hotelPOIs := filterPOIsForHotels(complete.PointsOfInterest)
							contentHTML := h.renderHotelsContentHTML(complete.GeneralCityData, hotelPOIs)
							c.SSEvent("hotels-content", contentHTML)
						}
					}
				}
			default:
				// Send progress update
				c.SSEvent(event.Type, event.Data)
			}
			flusher.Flush()

		case <-c.Request.Context().Done():
			logger.Log.Info("SSE connection closed",
				zap.String("sessionId", sessionID))
			return
		}
	}
}

// monitorHotelUpdates monitors for hotel updates and sends SSE events
func (h *HotelsHandlers) monitorHotelUpdates(sessionID string, updateChan chan<- models.ItinerarySSEEvent) {
	// Check for cached data first
	if completeData, found := middleware.CompleteItineraryCache.Get(sessionID); found {
		logger.Log.Info("Complete data found in cache, sending hotels completion immediately",
			zap.String("sessionId", sessionID))

		// Send header update
		updateChan <- models.ItinerarySSEEvent{
			Type: "header-update",
			Data: map[string]interface{}{
				"sessionId":    sessionID,
				"completeData": completeData,
			},
		}

		// Send content update
		updateChan <- models.ItinerarySSEEvent{
			Type: "content-update",
			Data: map[string]interface{}{
				"sessionId":    sessionID,
				"completeData": completeData,
			},
		}

		// Send completion
		updateChan <- models.ItinerarySSEEvent{
			Type: "complete",
			Data: map[string]interface{}{
				"sessionId": sessionID,
				"totalPOIs": len(completeData.PointsOfInterest),
			},
		}
		return
	}

	// Legacy cache check
	if itineraryData, found := middleware.ItineraryCache.Get(sessionID); found {
		logger.Log.Info("Legacy data found in cache, sending hotels completion immediately",
			zap.String("sessionId", sessionID))

		updateChan <- models.ItinerarySSEEvent{
			Type: "complete",
			Data: map[string]interface{}{
				"sessionId":        sessionID,
				"personalizedPOIs": len(itineraryData.PointsOfInterest),
			},
		}
		return
	}

	// If no cached data, poll for updates
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute) // 5 minute timeout

	for {
		select {
		case <-ticker.C:
			// Check cache again
			if completeData, found := middleware.CompleteItineraryCache.Get(sessionID); found {
				logger.Log.Info("Complete data appeared in cache for hotels",
					zap.String("sessionId", sessionID))

				// Send header update
				updateChan <- models.ItinerarySSEEvent{
					Type: "header-update",
					Data: map[string]interface{}{
						"sessionId":    sessionID,
						"completeData": completeData,
					},
				}

				// Send content update
				updateChan <- models.ItinerarySSEEvent{
					Type: "content-update",
					Data: map[string]interface{}{
						"sessionId":    sessionID,
						"completeData": completeData,
					},
				}

				// Send completion
				updateChan <- models.ItinerarySSEEvent{
					Type: "complete",
					Data: map[string]interface{}{
						"sessionId": sessionID,
						"totalPOIs": len(completeData.PointsOfInterest),
					},
				}
				return
			}

			if itineraryData, found := middleware.ItineraryCache.Get(sessionID); found {
				logger.Log.Info("Legacy data appeared in cache for hotels",
					zap.String("sessionId", sessionID))

				updateChan <- models.ItinerarySSEEvent{
					Type: "complete",
					Data: map[string]interface{}{
						"sessionId":        sessionID,
						"personalizedPOIs": len(itineraryData.PointsOfInterest),
					},
				}
				return
			}

			// Send progress update
			updateChan <- models.ItinerarySSEEvent{
				Type: "progress",
				Data: map[string]interface{}{
					"sessionId": sessionID,
					"message":   "Finding hotels...",
					"timestamp": time.Now().Unix(),
				},
			}

		case <-timeout:
			logger.Log.Warn("SSE monitoring timed out for hotels", zap.String("sessionId", sessionID))
			updateChan <- models.ItinerarySSEEvent{
				Type: "error",
				Data: map[string]interface{}{
					"sessionId": sessionID,
					"message":   "Request timed out. Please try again.",
				},
			}
			return
		}
	}
}

// renderHotelsHeaderHTML renders header HTML fragment for SSE
func (h *HotelsHandlers) renderHotelsHeaderHTML(cityData models.GeneralCityData, hotels []models.HotelDetailedInfo) string {
	buf := &strings.Builder{}
	component := results.HotelsHeaderComplete(cityData, hotels)
	err := component.Render(context.Background(), buf)
	if err != nil {
		h.logger.Error("failed to render hotels header", slog.Any("error", err))
	}
	return buf.String()
}

// renderHotelsContentHTML renders content HTML fragment for SSE
func (h *HotelsHandlers) renderHotelsContentHTML(cityData models.GeneralCityData, hotels []models.HotelDetailedInfo) string {
	buf := &strings.Builder{}
	component := results.HotelsContentComplete(cityData, hotels)
	err := component.Render(context.Background(), buf)
	if err != nil {
		h.logger.Error("failed to render hotels content", slog.Any("error", err))
	}
	return buf.String()
}
