package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/app/internal/features/restaurants"
	"github.com/FACorreiaa/go-templui/app/internal/features/results"
	"github.com/FACorreiaa/go-templui/app/internal/models"
	llmchat "github.com/FACorreiaa/go-templui/app/pkg/domain/chat_prompt"
	"github.com/FACorreiaa/go-templui/app/pkg/logger"
	"github.com/FACorreiaa/go-templui/app/pkg/middleware"
	"github.com/FACorreiaa/go-templui/app/pkg/services"
)

type RestaurantsHandlers struct {
	chatRepo         llmchat.Repository
	itineraryService *services.ItineraryService
	logger           *slog.Logger
}

func NewRestaurantsHandlers(chatRepo llmchat.Repository, logger *slog.Logger) *RestaurantsHandlers {
	return &RestaurantsHandlers{
		chatRepo:         chatRepo,
		itineraryService: services.NewItineraryService(),
		logger:           logger,
	}
}

// HandleRestaurantsPage handles the main restaurants page logic
func (h *RestaurantsHandlers) HandleRestaurantsPage(c *gin.Context) templ.Component {
	query := c.Query("q")
	sessionIDParam := c.Query("sessionId")

	logger.Log.Info("Restaurants page accessed",
		zap.String("ip", c.ClientIP()),
		zap.String("query", query),
		zap.String("sessionId", sessionIDParam))

	// Handle session-based loading
	if sessionIDParam != "" {
		return h.loadRestaurantsBySession(sessionIDParam)
	}

	// Handle query-based loading
	if query != "" {
		return restaurants.RestaurantsPageWithQuery(query)
	}

	// Default restaurants page
	return restaurants.RestaurantsPage()
}

// loadRestaurantsBySession loads restaurants using IDENTICAL logic as itinerary
func (h *RestaurantsHandlers) loadRestaurantsBySession(sessionIDParam string) templ.Component {
	logger.Log.Info("Attempting to load restaurants from cache",
		zap.String("sessionID", sessionIDParam),
		zap.String("cacheKey", sessionIDParam))

	// Try complete cache first (IDENTICAL to itinerary logic)
	if completeData, found := middleware.CompleteItineraryCache.Get(sessionIDParam); found {
		logger.Log.Info("Complete restaurants found in cache. Rendering results.",
			zap.String("city", completeData.GeneralCityData.City),
			zap.Int("generalPOIs", len(completeData.PointsOfInterest)),
			zap.Int("personalizedPOIs", len(completeData.AIItineraryResponse.PointsOfInterest)))

		// Filter POIs for restaurants and render (IDENTICAL to itinerary results pattern)
		restaurantPOIs := filterPOIsForRestaurants(completeData.PointsOfInterest)
		return results.RestaurantsResults(
			completeData.GeneralCityData,
			restaurantPOIs,
			true, true, 5, []string{})
	}

	// Try legacy cache (IDENTICAL to itinerary logic)
	if itineraryData, found := middleware.ItineraryCache.Get(sessionIDParam); found {
		logger.Log.Info("Legacy restaurants found in cache. Rendering results.",
			zap.Int("personalizedPOIs", len(itineraryData.PointsOfInterest)))

		// Create empty city data for legacy cached data (IDENTICAL to itinerary)
		emptyCityData := models.GeneralCityData{}

		// Filter restaurants from legacy data and convert to restaurant results
		restaurantPOIs := filterPOIsForRestaurants(itineraryData.PointsOfInterest)
		return results.RestaurantsResults(emptyCityData, restaurantPOIs, true, true, 5, []string{})
	}

	// Load from database (IDENTICAL to itinerary logic)
	return h.loadRestaurantsFromDatabase(sessionIDParam)
}

// loadRestaurantsFromDatabase loads restaurants from database when not found in cache
func (h *RestaurantsHandlers) loadRestaurantsFromDatabase(sessionIDParam string) templ.Component {
	logger.Log.Info("Restaurants not found in cache, attempting to load from database", zap.String("sessionID", sessionIDParam))

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
		emptyRestaurants := []models.RestaurantDetailedInfo{}
		return results.RestaurantsResults(emptyCityData, emptyRestaurants, true, true, 5, []string{})
	}

	// Parse the stored response as complete data
	completeData, err := h.itineraryService.ParseCompleteItineraryResponse(interaction.ResponseText, slog.Default())
	if err != nil || completeData == nil {
		logger.Log.Warn("Could not parse complete data from stored response",
			zap.String("sessionID", sessionIDParam),
			zap.Error(err))
		// Return empty results instead of PageNotFound for parsing errors
		emptyCityData := models.GeneralCityData{}
		emptyRestaurants := []models.RestaurantDetailedInfo{}
		return results.RestaurantsResults(emptyCityData, emptyRestaurants, true, true, 5, []string{})
	}

	logger.Log.Info("Successfully loaded complete data from database for restaurants",
		zap.String("sessionID", sessionIDParam),
		zap.String("city", completeData.GeneralCityData.City),
		zap.Int("totalPOIs", len(completeData.PointsOfInterest)))

	// Filter POIs for restaurants and render (IDENTICAL to itinerary results pattern)
	restaurantPOIs := filterPOIsForRestaurants(completeData.PointsOfInterest)
	return results.RestaurantsResults(
		completeData.GeneralCityData,
		restaurantPOIs,
		true, true, 5, []string{})
}

// HandleRestaurantsPageSSE handles the restaurants page with SSE support
func (h *RestaurantsHandlers) HandleRestaurantsPageSSE(c *gin.Context) templ.Component {
	query := c.Query("q")
	sessionIDParam := c.Query("sessionId")

	logger.Log.Info("Restaurants SSE page accessed",
		zap.String("ip", c.ClientIP()),
		zap.String("query", query),
		zap.String("sessionId", sessionIDParam))

	if sessionIDParam == "" {
		logger.Log.Info("Direct navigation to /restaurants SSE. Showing default page.")
		return restaurants.RestaurantsPage()
	}

	// Load restaurants data for session with SSE support
	return h.loadRestaurantsBySessionSSE(sessionIDParam)
}

// loadRestaurantsBySessionSSE loads restaurants with SSE support
func (h *RestaurantsHandlers) loadRestaurantsBySessionSSE(sessionIDParam string) templ.Component {
	logger.Log.Info("Attempting to load restaurants from cache with SSE", zap.String("sessionID", sessionIDParam))

	// Try complete cache first
	if completeData, found := middleware.CompleteItineraryCache.Get(sessionIDParam); found {
		// Filter POIs for restaurants
		restaurantPOIs := filterPOIsForRestaurants(completeData.PointsOfInterest)

		// Print JSON data for debugging what restaurants will be displayed
		jsonData, err := json.MarshalIndent(map[string]interface{}{
			"city_data":       completeData.GeneralCityData,
			"restaurant_pois": restaurantPOIs,
		}, "", "  ")
		if err != nil {
			logger.Log.Error("Failed to marshal restaurants data to JSON", zap.Error(err))
		} else {
			logger.Log.Info("Restaurants data being displayed in view", zap.String("json", string(jsonData)))
		}

		logger.Log.Info("Complete restaurants found in cache. Rendering SSE results with data.",
			zap.String("city", completeData.GeneralCityData.City),
			zap.Int("totalPOIs", len(completeData.PointsOfInterest)),
			zap.Int("restaurantPOIs", len(restaurantPOIs)))

		return results.RestaurantsResultsSSE(
			sessionIDParam,
			completeData.GeneralCityData,
			restaurantPOIs,
			true) // hasData = true
	}

	// Try legacy cache
	if itineraryData, found := middleware.ItineraryCache.Get(sessionIDParam); found {
		logger.Log.Info("Legacy restaurants found in cache. Rendering SSE results with data.",
			zap.Int("personalizedPOIs", len(itineraryData.PointsOfInterest)))

		// Create empty city data for legacy cached data
		emptyCityData := models.GeneralCityData{}

		// Filter restaurants from legacy data
		restaurantPOIs := filterPOIsForRestaurants(itineraryData.PointsOfInterest)

		return results.RestaurantsResultsSSE(
			sessionIDParam,
			emptyCityData,
			restaurantPOIs,
			true) // hasData = true
	}

	// No cached data found - show loading interface with SSE
	logger.Log.Info("No restaurants found in cache. Rendering SSE loading interface.",
		zap.String("sessionID", sessionIDParam))

	emptyCityData := models.GeneralCityData{}
	emptyRestaurants := []models.RestaurantDetailedInfo{}

	return results.RestaurantsResultsSSE(
		sessionIDParam,
		emptyCityData,
		emptyRestaurants,
		false) // hasData = false, will show loading and connect to SSE
}

// HandleRestaurantsSSE handles Server-Sent Events for restaurant updates
func (h *RestaurantsHandlers) HandleRestaurantsSSE(c *gin.Context) {
	sessionID := c.Query("sessionId")
	if sessionID == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	logger.Log.Info("SSE connection established for restaurants",
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
	go h.monitorRestaurantUpdates(sessionID, updateChan)

	// Stream updates to client
	flusher := c.Writer.(http.Flusher)
	for {
		select {
		case event := <-updateChan:
			if event.Type == "complete" {
				logger.Log.Info("Sending restaurants completion event",
					zap.String("sessionId", sessionID))

				// Send final completion event
				c.SSEvent("restaurants-complete", map[string]interface{}{
					"sessionId": sessionID,
					"message":   "Restaurant search complete",
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
							restaurantPOIs := filterPOIsForRestaurants(complete.PointsOfInterest)
							headerHTML := h.renderRestaurantsHeaderHTML(complete.GeneralCityData, restaurantPOIs)
							c.SSEvent("restaurants-header", headerHTML)
						}
					}
				}
			case "content-update":
				if data, ok := event.Data.(map[string]interface{}); ok {
					if completeData, exists := data["completeData"]; exists {
						if complete, valid := completeData.(models.AiCityResponse); valid {
							restaurantPOIs := filterPOIsForRestaurants(complete.PointsOfInterest)
							contentHTML := h.renderRestaurantsContentHTML(complete.GeneralCityData, restaurantPOIs)
							c.SSEvent("restaurants-content", contentHTML)
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

// monitorRestaurantUpdates monitors for restaurant updates and sends SSE events
func (h *RestaurantsHandlers) monitorRestaurantUpdates(sessionID string, updateChan chan<- models.ItinerarySSEEvent) {
	// Check for cached data first
	if completeData, found := middleware.CompleteItineraryCache.Get(sessionID); found {
		logger.Log.Info("Complete data found in cache, sending restaurants completion immediately",
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
		logger.Log.Info("Legacy data found in cache, sending restaurants completion immediately",
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
				logger.Log.Info("Complete data appeared in cache for restaurants",
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
				logger.Log.Info("Legacy data appeared in cache for restaurants",
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
					"message":   "Finding restaurants...",
					"timestamp": time.Now().Unix(),
				},
			}

		case <-timeout:
			logger.Log.Warn("SSE monitoring timed out for restaurants", zap.String("sessionId", sessionID))
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

// renderRestaurantsHeaderHTML renders header HTML fragment for SSE
func (h *RestaurantsHandlers) renderRestaurantsHeaderHTML(cityData models.GeneralCityData, restaurants []models.RestaurantDetailedInfo) string {
	buf := &strings.Builder{}
	component := results.RestaurantsHeaderComplete(cityData, restaurants)
	err := component.Render(context.Background(), buf)
	if err != nil {
		h.logger.Error("failed to render restaurants header", slog.Any("error", err))
	}
	return buf.String()
}

// renderRestaurantsContentHTML renders content HTML fragment for SSE
func (h *RestaurantsHandlers) renderRestaurantsContentHTML(cityData models.GeneralCityData, restaurants []models.RestaurantDetailedInfo) string {
	buf := &strings.Builder{}
	component := results.RestaurantsContentComplete(cityData, restaurants)
	err := component.Render(context.Background(), buf)
	if err != nil {
		h.logger.Error("failed to render restaurants content", slog.Any("error", err))
	}
	return buf.String()
}

// filterPOIsForRestaurants filters POIs to show only dining-related categories
func filterPOIsForRestaurants(allPOIs []models.POIDetailedInfo) []models.RestaurantDetailedInfo {
	var restaurantPOIs []models.RestaurantDetailedInfo
	restaurantCategories := map[string]bool{
		"restaurant": true,
		"dining":     true,
		"food":       true,
		"cafe":       true,
		"coffee":     true,
		"bar":        true,
		"pub":        true,
		"bistro":     true,
		"brasserie":  true,
		"pizzeria":   true,
		"bakery":     true,
		"market":     true,
		"foodcourt":  true,
		"fastfood":   true,
		"takeaway":   true,
	}

	for _, poi := range allPOIs {
		if restaurantCategories[strings.ToLower(poi.Category)] {
			restaurantPOIs = append(restaurantPOIs, convertPOIToRestaurant(poi))
		}
	}

	return restaurantPOIs
}

// convertPOIToRestaurant converts a POIDetailedInfo to RestaurantDetailedInfo
func convertPOIToRestaurant(poi models.POIDetailedInfo) models.RestaurantDetailedInfo {
	var address *string
	if poi.Address != "" {
		address = &poi.Address
	}

	var website *string
	if poi.Website != "" {
		website = &poi.Website
	}

	var phoneNumber *string
	if poi.PhoneNumber != "" {
		phoneNumber = &poi.PhoneNumber
	}

	var priceLevel *string
	if poi.PriceLevel != "" {
		priceLevel = &poi.PriceLevel
	}

	var cuisineType *string
	if poi.CuisineType != "" {
		cuisineType = &poi.CuisineType
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

	return models.RestaurantDetailedInfo{
		ID:               poi.ID,
		City:             poi.City,
		Name:             poi.Name,
		Latitude:         poi.Latitude,
		Longitude:        poi.Longitude,
		Category:         poi.Category,
		Description:      poi.Description,
		Address:          address,
		Website:          website,
		PhoneNumber:      phoneNumber,
		OpeningHours:     openingHours,
		PriceLevel:       priceLevel,
		CuisineType:      cuisineType,
		Tags:             poi.Tags,
		Images:           poi.Images,
		Rating:           poi.Rating,
		LlmInteractionID: poi.LlmInteractionID,
	}
}
