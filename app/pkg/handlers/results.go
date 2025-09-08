package handlers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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

	fmt.Printf("User ID: %s\n", userID)
	favorites := []string{}
	isLoadingFavorites := false

	// Render results using the new restaurant results component
	c.HTML(http.StatusOK, "", results.RestaurantResults(
		restaurants,
		false,     // compact
		true,      // showToggle
		5,         // initialLimit
		favorites, // favorites
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
	favorites := []string{}

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
	favorites := []string{}

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
	favorites := []string{}

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

// SSE Response structures to match your backend
type SSEEvent struct {
	ID    string `json:"id"`
	Event string `json:"event"`
	Data  string `json:"data"`
}

type LLMStreamResponse struct {
	Content      string `json:"content"`
	Type         string `json:"type"`
	IsComplete   bool   `json:"is_complete"`
	Restaurants  []results.RestaurantDetailedInfo `json:"restaurants"`
	Activities   []results.POIDetailedInfo        `json:"activities"`
	Hotels       []results.HotelDetailedInfo      `json:"hotels"`
	Itinerary    *results.AIItineraryResponse     `json:"itinerary"`
}

func (h *ResultsHandlers) callLLMForRestaurants(endpoint string, payload map[string]interface{}) ([]results.RestaurantDetailedInfo, error) {
	restaurants, err := h.makeSSERequest(endpoint, payload)
	if err != nil {
		logger.Log.Error("Failed to get restaurants from LLM", zap.Error(err))
		return h.getMockRestaurants(), nil // Fallback to mock data
	}
	
	if len(restaurants.Restaurants) > 0 {
		return restaurants.Restaurants, nil
	}
	
	return h.getMockRestaurants(), nil
}

func (h *ResultsHandlers) getMockRestaurants() []results.RestaurantDetailedInfo {
	cuisineType := "Portuguese"
	address := "Rua Augusta, Lisbon"
	phone := "+351 21 123 4567"
	priceLevel := "€€"
	hours := "12:00 - 24:00"
	website := "https://sample-restaurant.pt"
	
	return []results.RestaurantDetailedInfo{
		{
			Name:         "Taberna do Bacalhau",
			Description:  "Traditional Portuguese cuisine with the finest codfish dishes in a cozy historic setting",
			Rating:       4.5,
			CuisineType:  &cuisineType,
			Address:      &address,
			PhoneNumber:  &phone,
			PriceLevel:   &priceLevel,
			OpeningHours: &hours,
			Website:      &website,
			Tags:         []string{"Traditional", "Seafood", "Local Favorite", "Historic"},
		},
		{
			Name:        "Ramiro",
			Description: "Famous for its percebes and prawns, this legendary seafood house has been serving locals since 1956",
			Rating:      4.7,
			CuisineType: &cuisineType,
			Address:     &address,
			PriceLevel:  &priceLevel,
			Tags:        []string{"Seafood", "Historic", "Must-Visit", "Local Institution"},
		},
	}
}

func (h *ResultsHandlers) callLLMForActivities(endpoint string, payload map[string]interface{}) ([]results.POIDetailedInfo, error) {
	response, err := h.makeSSERequest(endpoint, payload)
	if err != nil {
		logger.Log.Error("Failed to get activities from LLM", zap.Error(err))
		return h.getMockActivities(), nil // Fallback to mock data
	}
	
	if len(response.Activities) > 0 {
		return response.Activities, nil
	}
	
	return h.getMockActivities(), nil
}

func (h *ResultsHandlers) getMockActivities() []results.POIDetailedInfo {
	return []results.POIDetailedInfo{
		{
			Name:        "Belém Tower",
			Description: "A 16th-century fortified tower serving as both a fortress and ceremonial gateway to Lisbon",
			Rating:      4.4,
			Category:    "Historical Monument",
			Address:     "Av. Brasília, 1400-038 Lisboa",
			TimeToSpend: "1-2 hours",
			Budget:      "€€",
			Website:     "https://www.torrebelem.gov.pt",
			Tags:        []string{"UNESCO", "Historical", "Architecture", "Must-Visit"},
		},
		{
			Name:        "Jerónimos Monastery",
			Description: "Magnificent Manueline monastery showcasing Portugal's maritime discoveries and religious heritage",
			Rating:      4.6,
			Category:    "Religious Monument", 
			Address:     "Praça do Império 1400-206 Lisboa",
			TimeToSpend: "2-3 hours",
			Budget:      "€€",
			Tags:        []string{"UNESCO", "Architecture", "Religious", "Cultural"},
		},
	}
}

func (h *ResultsHandlers) callLLMForHotels(endpoint string, payload map[string]interface{}) ([]results.HotelDetailedInfo, error) {
	response, err := h.makeSSERequest(endpoint, payload)
	if err != nil {
		logger.Log.Error("Failed to get hotels from LLM", zap.Error(err))
		return h.getMockHotels(), nil // Fallback to mock data
	}
	
	if len(response.Hotels) > 0 {
		return response.Hotels, nil
	}
	
	return h.getMockHotels(), nil
}

func (h *ResultsHandlers) getMockHotels() []results.HotelDetailedInfo {
	return []results.HotelDetailedInfo{
		{
			Name:        "Pousada de Lisboa",
			Description: "Elegant boutique hotel in a restored 18th-century building in the heart of Lisbon's historic center",
			Rating:      4.5,
			Category:    "Boutique Hotel",
			Address:     "Rua das Flores 4, 1200-194 Lisboa",
			PriceRange:  "€€€",
			Amenities:   []string{"WiFi", "Restaurant", "Bar", "Concierge", "Historic Building"},
			Tags:        []string{"Historic", "Central Location", "Boutique", "Cultural"},
		},
		{
			Name:        "Hotel Tivoli Avenidas Liberdade",
			Description: "Modern luxury hotel on Lisbon's premier shopping avenue with contemporary amenities",
			Rating:      4.3,
			Category:    "Luxury Hotel",
			Address:     "Avenida da Liberdade 185, 1269-050 Lisboa",
			PriceRange:  "€€€€",
			Amenities:   []string{"WiFi", "Pool", "Spa", "Gym", "Restaurant", "Business Center"},
			Tags:        []string{"Luxury", "Shopping", "Modern", "Business"},
		},
	}
}

func (h *ResultsHandlers) callLLMForItinerary(endpoint string, payload map[string]interface{}) (results.AIItineraryResponse, error) {
	response, err := h.makeSSERequest(endpoint, payload)
	if err != nil {
		logger.Log.Error("Failed to get itinerary from LLM", zap.Error(err))
		return h.getMockItinerary(), nil // Fallback to mock data
	}
	
	if response.Itinerary != nil {
		return *response.Itinerary, nil
	}
	
	return h.getMockItinerary(), nil
}

func (h *ResultsHandlers) getMockItinerary() results.AIItineraryResponse {
	return results.AIItineraryResponse{
		ItineraryName:      "Lisbon Discovery: 3-Day Cultural Journey",
		OverallDescription: "Experience Lisbon's rich maritime history, stunning architecture, and vibrant culture in this carefully crafted 3-day itinerary",
		PointsOfInterest: []results.POIDetailedInfo{
			{
				Name:        "Belém Tower",
				Description: "16th-century fortified tower and UNESCO World Heritage site",
				Rating:      4.4,
				Category:    "Historical Monument",
				Address:     "Av. Brasília, 1400-038 Lisboa",
				TimeToSpend: "1-2 hours",
				Budget:      "€€",
				Tags:        []string{"UNESCO", "Historical", "Architecture"},
			},
			{
				Name:        "Jerónimos Monastery", 
				Description: "Magnificent monastery showcasing Manueline architecture",
				Rating:      4.6,
				Category:    "Religious Monument",
				Address:     "Praça do Império 1400-206 Lisboa",
				TimeToSpend: "2-3 hours",
				Budget:      "€€",
				Tags:        []string{"UNESCO", "Architecture", "Religious"},
			},
		},
		Restaurants: []results.POIDetailedInfo{
			{
				Name:        "Pastéis de Belém",
				Description: "Home of the original pastéis de nata since 1837",
				Rating:      4.2,
				Category:    "Pastry Shop",
				Address:     "R. de Belém 84-92, 1300-085 Lisboa",
				Budget:      "€",
				Tags:        []string{"Traditional", "Dessert", "Historic"},
			},
		},
		Bars: []results.POIDetailedInfo{
			{
				Name:        "Pensão Amor",
				Description: "Quirky bar in a former brothel with vintage decor",
				Rating:      4.0,
				Category:    "Bar",
				Address:     "R. do Alecrim 19, 1200-014 Lisboa",
				Budget:      "€€",
				Tags:        []string{"Unique", "Nightlife", "Historic"},
			},
		},
	}
}

// makeSSERequest handles the Server-Sent Events request to the LLM service
func (h *ResultsHandlers) makeSSERequest(endpoint string, payload map[string]interface{}) (*LLMStreamResponse, error) {
	// Convert payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	// Make request with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse SSE stream
	return h.parseSSEStream(resp)
}

// parseSSEStream parses the Server-Sent Events stream from the LLM service
func (h *ResultsHandlers) parseSSEStream(resp *http.Response) (*LLMStreamResponse, error) {
	scanner := bufio.NewScanner(resp.Body)
	result := &LLMStreamResponse{}
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if line == "" {
			continue
		}
		
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			
			// Skip keep-alive messages
			if data == "keep-alive" || data == "[DONE]" {
				continue
			}
			
			// Try to parse JSON data
			var streamData map[string]interface{}
			if err := json.Unmarshal([]byte(data), &streamData); err != nil {
				logger.Log.Warn("Failed to parse SSE data", zap.String("data", data), zap.Error(err))
				continue
			}
			
			// Extract structured data from the stream
			if restaurants, ok := streamData["restaurants"].([]interface{}); ok {
				h.parseRestaurants(restaurants, result)
			}
			
			if activities, ok := streamData["activities"].([]interface{}); ok {
				h.parseActivities(activities, result)
			}
			
			if hotels, ok := streamData["hotels"].([]interface{}); ok {
				h.parseHotels(hotels, result)
			}
			
			if itinerary, ok := streamData["itinerary"].(map[string]interface{}); ok {
				h.parseItinerary(itinerary, result)
			}
			
			if complete, ok := streamData["is_complete"].(bool); ok && complete {
				break
			}
		}
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading SSE stream: %w", err)
	}
	
	return result, nil
}

// Helper functions to parse different data types from the SSE stream
func (h *ResultsHandlers) parseRestaurants(data []interface{}, result *LLMStreamResponse) {
	for _, item := range data {
		if restaurant, ok := item.(map[string]interface{}); ok {
			parsed := h.parseRestaurantData(restaurant)
			result.Restaurants = append(result.Restaurants, parsed)
		}
	}
}

func (h *ResultsHandlers) parseActivities(data []interface{}, result *LLMStreamResponse) {
	for _, item := range data {
		if activity, ok := item.(map[string]interface{}); ok {
			parsed := h.parsePOIData(activity)
			result.Activities = append(result.Activities, parsed)
		}
	}
}

func (h *ResultsHandlers) parseHotels(data []interface{}, result *LLMStreamResponse) {
	for _, item := range data {
		if hotel, ok := item.(map[string]interface{}); ok {
			parsed := h.parseHotelData(hotel)
			result.Hotels = append(result.Hotels, parsed)
		}
	}
}

func (h *ResultsHandlers) parseItinerary(data map[string]interface{}, result *LLMStreamResponse) {
	itinerary := &results.AIItineraryResponse{}
	
	if name, ok := data["name"].(string); ok {
		itinerary.ItineraryName = name
	}
	
	if desc, ok := data["description"].(string); ok {
		itinerary.OverallDescription = desc
	}
	
	if pois, ok := data["points_of_interest"].([]interface{}); ok {
		for _, poi := range pois {
			if poiData, ok := poi.(map[string]interface{}); ok {
				itinerary.PointsOfInterest = append(itinerary.PointsOfInterest, h.parsePOIData(poiData))
			}
		}
	}
	
	result.Itinerary = itinerary
}

// Helper functions to convert map[string]interface{} to typed structs
func (h *ResultsHandlers) parseRestaurantData(data map[string]interface{}) results.RestaurantDetailedInfo {
	restaurant := results.RestaurantDetailedInfo{}
	
	if name, ok := data["name"].(string); ok {
		restaurant.Name = name
	}
	if desc, ok := data["description"].(string); ok {
		restaurant.Description = desc
	}
	if rating, ok := data["rating"].(float64); ok {
		restaurant.Rating = rating
	}
	if cuisine, ok := data["cuisine_type"].(string); ok {
		restaurant.CuisineType = &cuisine
	}
	if addr, ok := data["address"].(string); ok {
		restaurant.Address = &addr
	}
	if phone, ok := data["phone"].(string); ok {
		restaurant.PhoneNumber = &phone
	}
	if price, ok := data["price_level"].(string); ok {
		restaurant.PriceLevel = &price
	}
	if hours, ok := data["opening_hours"].(string); ok {
		restaurant.OpeningHours = &hours
	}
	if website, ok := data["website"].(string); ok {
		restaurant.Website = &website
	}
	if tags, ok := data["tags"].([]interface{}); ok {
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				restaurant.Tags = append(restaurant.Tags, tagStr)
			}
		}
	}
	
	return restaurant
}

func (h *ResultsHandlers) parsePOIData(data map[string]interface{}) results.POIDetailedInfo {
	poi := results.POIDetailedInfo{}
	
	if name, ok := data["name"].(string); ok {
		poi.Name = name
	}
	if desc, ok := data["description"].(string); ok {
		poi.Description = desc
	}
	if rating, ok := data["rating"].(float64); ok {
		poi.Rating = rating
	}
	if category, ok := data["category"].(string); ok {
		poi.Category = category
	}
	if addr, ok := data["address"].(string); ok {
		poi.Address = addr
	}
	if timeToSpend, ok := data["time_to_spend"].(string); ok {
		poi.TimeToSpend = timeToSpend
	}
	if budget, ok := data["budget"].(string); ok {
		poi.Budget = budget
	}
	if website, ok := data["website"].(string); ok {
		poi.Website = website
	}
	if tags, ok := data["tags"].([]interface{}); ok {
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				poi.Tags = append(poi.Tags, tagStr)
			}
		}
	}
	
	return poi
}

func (h *ResultsHandlers) parseHotelData(data map[string]interface{}) results.HotelDetailedInfo {
	hotel := results.HotelDetailedInfo{}
	
	if name, ok := data["name"].(string); ok {
		hotel.Name = name
	}
	if desc, ok := data["description"].(string); ok {
		hotel.Description = desc
	}
	if rating, ok := data["rating"].(float64); ok {
		hotel.Rating = rating
	}
	if category, ok := data["category"].(string); ok {
		hotel.Category = category
	}
	if addr, ok := data["address"].(string); ok {
		hotel.Address = addr
	}
	if priceRange, ok := data["price_range"].(string); ok {
		hotel.PriceRange = priceRange
	}
	if amenities, ok := data["amenities"].([]interface{}); ok {
		for _, amenity := range amenities {
			if amenityStr, ok := amenity.(string); ok {
				hotel.Amenities = append(hotel.Amenities, amenityStr)
			}
		}
	}
	if tags, ok := data["tags"].([]interface{}); ok {
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				hotel.Tags = append(hotel.Tags, tagStr)
			}
		}
	}
	
	return hotel
}
