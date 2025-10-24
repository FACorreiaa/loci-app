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
	cacheKey := c.Query("cacheKey")

	logger.Log.Info("Restaurants page accessed",
		zap.String("ip", c.ClientIP()),
		zap.String("query", query),
		zap.String("sessionId", sessionIDParam),
		zap.String("cacheKey", cacheKey))

	// Handle session-based loading
	if sessionIDParam != "" {
		return h.loadRestaurantsBySession(sessionIDParam, cacheKey)
	}

	// Handle query-based loading
	if query != "" {
		return restaurants.RestaurantsPageWithQuery(query)
	}

	// Default restaurants page
	return restaurants.RestaurantsPage()
}

// loadRestaurantsBySession loads restaurants using IDENTICAL logic as itinerary

func (h *RestaurantsHandlers) loadRestaurantsBySession(sessionIDParam string, cacheKey string) templ.Component {

	logger.Log.Info("Attempting to load restaurants from cache",

		zap.String("sessionID", sessionIDParam),

		zap.String("cacheKey", cacheKey))

	// Try restaurants cache first with cacheKey (for reusable cache hits)
	if cacheKey != "" {
		if restaurantsData, found := middleware.RestaurantsCache.Get(cacheKey); found {

			logger.Log.Info("Restaurants found in cache. Rendering results.",

				zap.Int("restaurants", len(restaurantsData)))

			// Try to get city data from complete cache
			var cityData models.GeneralCityData
			if completeData, found := middleware.CompleteItineraryCache.Get(cacheKey); found {
				jsonData, err := json.MarshalIndent(completeData, "", "  ")
				if err != nil {
					logger.Log.Error("Failed to marshal completeData to JSON", zap.Error(err))
				} else {
					filename := "complete_restaurant.json" // Or fmt.Sprintf("complete_itinerary_%s.json", sessionID)
					if writeErr := os.WriteFile(filename, jsonData, 0644); writeErr != nil {
						logger.Log.Error("Failed to write completeData to file", zap.String("file", filename), zap.Error(writeErr))
					} else {
						logger.Log.Info("Complete itinerary data written to file", zap.String("file", filename))
					}
					logger.Log.Info("Complete itinerary data being displayed in view", zap.String("json", string(jsonData)))
				}

				cityData = completeData.GeneralCityData
				logger.Log.Info("City data loaded from complete cache",
					zap.String("city", cityData.City))
			} else {
				// Fallback: load from database using sessionID
				cityData = h.loadCityDataFromDatabase(sessionIDParam)
			}

			// Return static template when data is available

			return results.RestaurantsResults(

				cityData,

				restaurantsData,

				true, true, 15, []string{})
		}
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

	// Print JSON data for debugging
	jsonData, err := json.MarshalIndent(completeData, "", "  ")
	if err != nil {
		logger.Log.Error("Failed to marshal completeData to JSON", zap.Error(err))
	} else {
		if err := os.WriteFile("complete_restaurant.json", jsonData, 0644); err != nil {
			logger.Log.Error("Failed to write JSON to file", zap.Error(err))
		}
	}

	logger.Log.Info("Successfully loaded complete data from database for restaurants",
		zap.String("sessionID", sessionIDParam),
		zap.String("city", completeData.GeneralCityData.City),
		zap.Int("totalPOIs", len(completeData.PointsOfInterest)))

	// Filter POIs for restaurants and render (IDENTICAL to itinerary results pattern)
	restaurantPOIs := filterRestaurants(completeData.PointsOfInterest)
	return results.RestaurantsResults(
		completeData.GeneralCityData,
		restaurantPOIs,
		true, true, 5, []string{})
}

// HandleRestaurantsPageSSE handles the restaurants page with SSE support
func (h *RestaurantsHandlers) HandleRestaurantsPageSSE(c *gin.Context) templ.Component {
	query := c.Query("q")
	sessionIDParam := c.Query("sessionId")
	cacheKey := c.Query("cacheKey")

	logger.Log.Info("Restaurants SSE page accessed",
		zap.String("ip", c.ClientIP()),
		zap.String("query", query),
		zap.String("sessionId", sessionIDParam),
		zap.String("cacheKey", cacheKey))

	if sessionIDParam == "" {
		logger.Log.Info("Direct navigation to /restaurants SSE. Showing default page.")
		return restaurants.RestaurantsPage()
	}

	// Load restaurants data for session with SSE support
	return h.loadRestaurantsBySession(sessionIDParam, cacheKey)
}

// filterRestaurants filters POIs to show only dining-related categories
func filterRestaurants(allPOIs []models.POIDetailedInfo) []models.RestaurantDetailedInfo {
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
		var hoursBuilder strings.Builder
		for day, hours := range poi.OpeningHours {
			if hoursBuilder.Len() > 0 {
				hoursBuilder.WriteString(", ")
			}
			hoursBuilder.WriteString(day)
			hoursBuilder.WriteString(": ")
			hoursBuilder.WriteString(hours)
		}
		if hoursBuilder.Len() > 0 {
			hoursStr := hoursBuilder.String()
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

// loadCityDataFromDatabase loads city data from database by sessionID
func (h *RestaurantsHandlers) loadCityDataFromDatabase(sessionIDParam string) models.GeneralCityData {
	sessionID, err := uuid.Parse(sessionIDParam)
	if err != nil {
		logger.Log.Warn("Invalid session ID format when loading city data", zap.String("sessionID", sessionIDParam), zap.Error(err))
		return models.GeneralCityData{}
	}

	ctx := context.Background()
	interaction, err := h.chatRepo.GetLatestInteractionBySessionID(ctx, sessionID)
	if err != nil || interaction == nil {
		logger.Log.Warn("No interaction found in database for city data",
			zap.String("sessionID", sessionIDParam),
			zap.Error(err))
		return models.GeneralCityData{}
	}

	completeData, err := h.itineraryService.ParseCompleteItineraryResponse(interaction.ResponseText, slog.Default())
	if err != nil || completeData == nil {
		logger.Log.Warn("Could not parse complete data from stored response for city data",
			zap.String("sessionID", sessionIDParam),
			zap.Error(err))
		return models.GeneralCityData{}
	}

	logger.Log.Info("City data loaded from database",
		zap.String("sessionID", sessionIDParam),
		zap.String("city", completeData.GeneralCityData.City),
		zap.String("country", completeData.GeneralCityData.Country),
		zap.String("description", completeData.GeneralCityData.Description))

	return completeData.GeneralCityData
}
