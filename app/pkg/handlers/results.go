package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/app/lib/features/results"
	"github.com/FACorreiaa/go-templui/app/pkg/config"
	"github.com/FACorreiaa/go-templui/app/pkg/logger"
	"github.com/FACorreiaa/go-templui/app/pkg/middleware"
)

type ResultsHandlers struct {
	config *config.Config
}

func NewResultsHandlers() *ResultsHandlers {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}
	return &ResultsHandlers{
		config: cfg,
	}
}

// HandleRestaurantSearch processes restaurant search requests and returns results
func (h *ResultsHandlers) HandleRestaurantSearch(c *gin.Context) {
	logger.Log.Info("Restaurant search request received",
		zap.String("ip", c.ClientIP()),
		zap.String("user", middleware.GetUserIDFromContext(c)),
	)

	query := c.PostForm("query")
	city := c.PostForm("city")
	intent := c.PostForm("intent")

	if query == "" && city == "" {
		logger.Log.Warn("Empty search query received")
		c.String(http.StatusBadRequest, `<div class="text-red-500">Please provide a search query.</div>`)
		return
	}

	// Call backend service to get restaurant data
	restaurants, err := h.fetchRestaurantResults(query, city, intent)
	if err != nil {
		logger.Log.Error("Failed to fetch restaurant results", zap.Error(err))
		c.String(http.StatusInternalServerError, `<div class="text-red-500">Failed to load restaurant results.</div>`)
		return
	}

	// Get user favorites (if authenticated)
	userID := middleware.GetUserIDFromContext(c)
	favorites := []string{} // TODO: fetch from service
	isLoadingFavorites := false

	// Render results using the new restaurant results component
	c.HTML(http.StatusOK, "", results.RestaurantResults(
		restaurants,
		false,        // compact
		true,         // showToggle
		5,            // initialLimit
		favorites,    // favorites
		isLoadingFavorites,
	))
}

// HandleActivitySearch processes activity search requests and returns results
func (h *ResultsHandlers) HandleActivitySearch(c *gin.Context) {
	logger.Log.Info("Activity search request received",
		zap.String("ip", c.ClientIP()),
		zap.String("user", middleware.GetUserIDFromContext(c)),
	)

	query := c.PostForm("query")
	city := c.PostForm("city")
	intent := c.PostForm("intent")

	if query == "" && city == "" {
		logger.Log.Warn("Empty search query received")
		c.String(http.StatusBadRequest, `<div class="text-red-500">Please provide a search query.</div>`)
		return
	}

	// Call backend service to get activity data
	activities, err := h.fetchActivityResults(query, city, intent)
	if err != nil {
		logger.Log.Error("Failed to fetch activity results", zap.Error(err))
		c.String(http.StatusInternalServerError, `<div class="text-red-500">Failed to load activity results.</div>`)
		return
	}

	// Get user favorites (if authenticated)
	favorites := []string{} // TODO: fetch from service

	// Render results using the new activity results component
	c.HTML(http.StatusOK, "", results.ActivityResults(
		activities,
		false, // compact
		true,  // showToggle
		6,     // initialLimit
		favorites,
	))
}

// HandleHotelSearch processes hotel search requests and returns results
func (h *ResultsHandlers) HandleHotelSearch(c *gin.Context) {
	logger.Log.Info("Hotel search request received",
		zap.String("ip", c.ClientIP()),
		zap.String("user", middleware.GetUserIDFromContext(c)),
	)

	query := c.PostForm("query")
	city := c.PostForm("city")
	intent := c.PostForm("intent")

	if query == "" && city == "" {
		logger.Log.Warn("Empty search query received")
		c.String(http.StatusBadRequest, `<div class="text-red-500">Please provide a search query.</div>`)
		return
	}

	// Call backend service to get hotel data
	hotels, err := h.fetchHotelResults(query, city, intent)
	if err != nil {
		logger.Log.Error("Failed to fetch hotel results", zap.Error(err))
		c.String(http.StatusInternalServerError, `<div class="text-red-500">Failed to load hotel results.</div>`)
		return
	}

	// Get user favorites (if authenticated)
	favorites := []string{} // TODO: fetch from service

	// Render results using the new hotel results component
	c.HTML(http.StatusOK, "", results.HotelResults(
		hotels,
		false, // compact
		true,  // showToggle
		4,     // initialLimit
		favorites,
	))
}

// HandleItinerarySearch processes itinerary requests and returns results
func (h *ResultsHandlers) HandleItinerarySearch(c *gin.Context) {
	logger.Log.Info("Itinerary search request received",
		zap.String("ip", c.ClientIP()),
		zap.String("user", middleware.GetUserIDFromContext(c)),
	)

	query := c.PostForm("query")
	city := c.PostForm("city")
	intent := c.PostForm("intent")

	if query == "" && city == "" {
		logger.Log.Warn("Empty search query received")
		c.String(http.StatusBadRequest, `<div class="text-red-500">Please provide a search query.</div>`)
		return
	}

	// Call backend service to get itinerary data
	itinerary, err := h.fetchItineraryResults(query, city, intent)
	if err != nil {
		logger.Log.Error("Failed to fetch itinerary results", zap.Error(err))
		c.String(http.StatusInternalServerError, `<div class="text-red-500">Failed to load itinerary results.</div>`)
		return
	}

	// Get user favorites (if authenticated)
	favorites := []string{} // TODO: fetch from service

	// Render results using the new itinerary results component
	c.HTML(http.StatusOK, "", results.ItineraryResults(
		itinerary,
		false, // compact
		true,  // showToggle
		5,     // initialLimit
		favorites,
	))
}

// Backend service calls - these would call your actual LLM service

func (h *ResultsHandlers) fetchRestaurantResults(query, city, intent string) ([]results.RestaurantDetailedInfo, error) {
	// Build request payload
	requestPayload := map[string]interface{}{
		"message": query,
		"city":    city,
		"intent":  intent,
		"domain":  "dining",
	}

	// Make request to the correct LLM endpoint
	llmEndpoint := "http://localhost:8000/api/v1/llm/chat/stream/free"
	if h.config != nil && h.config.LLM.StreamEndpoint != "" {
		llmEndpoint = h.config.LLM.StreamEndpoint + "/api/v1/llm/chat/stream/free"
	}

	// Call LLM service and parse response
	// This would implement the SSE parsing logic to extract restaurant data
	restaurants, err := h.callLLMForRestaurants(llmEndpoint, requestPayload)
	if err != nil {
		return nil, err
	}

	return restaurants, nil
}

func (h *ResultsHandlers) fetchActivityResults(query, city, intent string) ([]results.POIDetailedInfo, error) {
	// Similar implementation for activities
	requestPayload := map[string]interface{}{
		"message": query,
		"city":    city,
		"intent":  intent,
		"domain":  "activities",
	}

	llmEndpoint := "http://localhost:8000/api/v1/llm/chat/stream/free"
	if h.config != nil && h.config.LLM.StreamEndpoint != "" {
		llmEndpoint = h.config.LLM.StreamEndpoint + "/api/v1/llm/chat/stream/free"
	}

	activities, err := h.callLLMForActivities(llmEndpoint, requestPayload)
	if err != nil {
		return nil, err
	}

	return activities, nil
}

func (h *ResultsHandlers) fetchHotelResults(query, city, intent string) ([]results.HotelDetailedInfo, error) {
	// Similar implementation for hotels
	requestPayload := map[string]interface{}{
		"message": query,
		"city":    city,
		"intent":  intent,
		"domain":  "accommodation",
	}

	llmEndpoint := "http://localhost:8000/api/v1/llm/chat/stream/free"
	if h.config != nil && h.config.LLM.StreamEndpoint != "" {
		llmEndpoint = h.config.LLM.StreamEndpoint + "/api/v1/llm/chat/stream/free"
	}

	hotels, err := h.callLLMForHotels(llmEndpoint, requestPayload)
	if err != nil {
		return nil, err
	}

	return hotels, nil
}

func (h *ResultsHandlers) fetchItineraryResults(query, city, intent string) (results.AIItineraryResponse, error) {
	// Similar implementation for itinerary
	requestPayload := map[string]interface{}{
		"message": query,
		"city":    city,
		"intent":  intent,
		"domain":  "itinerary",
	}

	llmEndpoint := "http://localhost:8000/api/v1/llm/chat/stream/free"
	if h.config != nil && h.config.LLM.StreamEndpoint != "" {
		llmEndpoint = h.config.LLM.StreamEndpoint + "/api/v1/llm/chat/stream/free"
	}

	itinerary, err := h.callLLMForItinerary(llmEndpoint, requestPayload)
	if err != nil {
		return results.AIItineraryResponse{}, err
	}

	return itinerary, nil
}

// Placeholder implementations - you would implement these to call your actual LLM service
// and parse the SSE stream to extract the relevant data structures

func (h *ResultsHandlers) callLLMForRestaurants(endpoint string, payload map[string]interface{}) ([]results.RestaurantDetailedInfo, error) {
	// TODO: Implement actual LLM service call and SSE parsing
	// This is where you'd make the HTTP request to your LLM service,
	// parse the SSE stream, and extract restaurant data
	
	// For now, return mock data to demonstrate the structure
	mockRestaurants := []results.RestaurantDetailedInfo{
		{
			// Initialize with sample data based on your models
			Name:        "Sample Restaurant",
			Description: "A great place to eat",
			Rating:      4.5,
			// ... other fields
		},
	}
	
	return mockRestaurants, nil
}

func (h *ResultsHandlers) callLLMForActivities(endpoint string, payload map[string]interface{}) ([]results.POIDetailedInfo, error) {
	// TODO: Implement actual LLM service call and SSE parsing
	mockActivities := []results.POIDetailedInfo{
		{
			Name:        "Sample Activity", 
			Description: "A fun activity to do",
			Rating:      4.2,
			Category:    "Entertainment",
		},
	}
	
	return mockActivities, nil
}

func (h *ResultsHandlers) callLLMForHotels(endpoint string, payload map[string]interface{}) ([]results.HotelDetailedInfo, error) {
	// TODO: Implement actual LLM service call and SSE parsing  
	mockHotels := []results.HotelDetailedInfo{
		{
			Name:        "Sample Hotel",
			Description: "A comfortable place to stay",
			Rating:      4.3,
			Category:    "Luxury Hotel",
		},
	}
	
	return mockHotels, nil
}

func (h *ResultsHandlers) callLLMForItinerary(endpoint string, payload map[string]interface{}) (results.AIItineraryResponse, error) {
	// TODO: Implement actual LLM service call and SSE parsing
	mockItinerary := results.AIItineraryResponse{
		ItineraryName:      "Sample Itinerary",
		OverallDescription: "A wonderful travel plan",
		PointsOfInterest: []results.POIDetailedInfo{
			{
				Name:        "Sample POI",
				Description: "An interesting place to visit",
				Rating:      4.4,
				Category:    "Attraction",
			},
		},
	}
	
	return mockItinerary, nil
}