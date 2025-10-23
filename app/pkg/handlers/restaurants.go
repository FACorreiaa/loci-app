package handlers

import (
	"context"
	"log/slog"
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
	return h.loadRestaurantsBySession(sessionIDParam)
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
