package results

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/internal/app/models"
	"github.com/FACorreiaa/go-templui/internal/pkg/config"
	"github.com/FACorreiaa/go-templui/internal/pkg/middleware"
)

type ResultsHandlers struct {
	config *config.Config
	logger *zap.Logger
}

func NewResultsHandlers(logger *zap.Logger) *ResultsHandlers {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}
	return &ResultsHandlers{
		config: cfg,
		logger: logger,
	}
}

// HandleRestaurantSearch processes restaurant search requests and returns results
func (h *ResultsHandlers) HandleRestaurantSearch(c *gin.Context) {
	h.logger.Info("Restaurant search request received",
		zap.String("ip", c.ClientIP()),
		zap.String("user", middleware.GetUserIDFromContext(c)),
	)

	query := c.PostForm("query")
	city := c.PostForm("city")
	intent := c.PostForm("intent")

	if query == "" && city == "" {
		h.logger.Warn("Empty search query received")
		c.String(http.StatusBadRequest, `<div class="text-red-500">Please provide a search query.</div>`)
		return
	}

	// Call backend service to get restaurant data
	restaurants, err := h.fetchRestaurantResults(query, city, intent)
	if err != nil {
		h.logger.Error("Failed to fetch restaurant results", zap.Error(err))
		c.String(http.StatusInternalServerError, `<div class="text-red-500">Failed to load restaurant models.</div>`)
		return
	}

	// Generate session ID for caching results
	sessionID := uuid.New().String()

	// Cache the results for later access via direct route
	middleware.RestaurantsCache.Set(sessionID, restaurants)

	h.logger.Info("Cached restaurant results",
		zap.String("sessionID", sessionID),
		zap.Int("count", len(restaurants)))

	// Get user favorites (if authenticated)
	userID := middleware.GetUserIDFromContext(c)

	fmt.Printf("User ID: %s\n", userID)
	favorites := []string{}
	isLoadingFavorites := false

	// Set the sessionID in response header so frontend can use it for direct links
	c.Header("X-Session-ID", sessionID)

	// Render results using the new restaurant results component
	c.HTML(http.StatusOK, "", RestaurantResults(
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
	h.logger.Info("Activity search request received",
		zap.String("ip", c.ClientIP()),
		zap.String("user", middleware.GetUserIDFromContext(c)),
	)

	query := c.PostForm("query")
	city := c.PostForm("city")
	intent := c.PostForm("intent")

	if query == "" && city == "" {
		h.logger.Warn("Empty search query received")
		c.String(http.StatusBadRequest, `<div class="text-red-500">Please provide a search query.</div>`)
		return
	}

	// Call backend service to get activity data
	activities, err := h.fetchActivityResults(query, city, intent)
	if err != nil {
		h.logger.Error("Failed to fetch activity results", zap.Error(err))
		c.String(http.StatusInternalServerError, `<div class="text-red-500">Failed to load activity models.</div>`)
		return
	}

	// Generate session ID for caching results
	sessionID := uuid.New().String()

	// Cache the results for later access via direct route
	middleware.ActivitiesCache.Set(sessionID, activities)

	h.logger.Info("Cached activity results",
		zap.String("sessionID", sessionID),
		zap.Int("count", len(activities)))

	// Get user favorites (if authenticated)
	favorites := []string{}

	// Set the sessionID in response header so frontend can use it for direct links
	c.Header("X-Session-ID", sessionID)

	// Render results using the new activity results component
	c.HTML(http.StatusOK, "", ActivityResults(
		activities,
		false, // compact
		true,  // showToggle
		6,     // initialLimit
		favorites,
	))
}

// HandleHotelSearch processes hotel search requests and returns results
func (h *ResultsHandlers) HandleHotelSearch(c *gin.Context) {
	h.logger.Info("Hotel search request received",
		zap.String("ip", c.ClientIP()),
		zap.String("user", middleware.GetUserIDFromContext(c)),
	)

	query := c.PostForm("query")
	city := c.PostForm("city")
	intent := c.PostForm("intent")

	if query == "" && city == "" {
		h.logger.Warn("Empty search query received")
		c.String(http.StatusBadRequest, `<div class="text-red-500">Please provide a search query.</div>`)
		return
	}

	// Call backend service to get hotel data
	hotels, err := h.fetchHotelResults(query, city, intent)
	if err != nil {
		h.logger.Error("Failed to fetch hotel results", zap.Error(err))
		c.String(http.StatusInternalServerError, `<div class="text-red-500">Failed to load hotel models.</div>`)
		return
	}

	// Generate session ID for caching results
	sessionID := uuid.New().String()

	// Cache the results for later access via direct route
	middleware.HotelsCache.Set(sessionID, hotels)

	h.logger.Info("Cached hotel results",
		zap.String("sessionID", sessionID),
		zap.Int("count", len(hotels)))

	// Get user favorites (if authenticated)
	favorites := []string{}

	// Set the sessionID in response header so frontend can use it for direct links
	c.Header("X-Session-ID", sessionID)

	// Create empty city data for now
	emptyCityData := models.GeneralCityData{}

	// Render results using the new hotel results component
	c.HTML(http.StatusOK, "", HotelsResults(
		emptyCityData,
		hotels,
		false, // compact
		true,  // showToggle
		4,     // initialLimit
		favorites,
		sessionID, // sessionID for chat panel
	))
}

// HandleItinerarySearch processes itinerary requests and returns results
func (h *ResultsHandlers) HandleItinerarySearch(c *gin.Context) {
	h.logger.Info("Itinerary search request received",
		zap.String("ip", c.ClientIP()),
		zap.String("user", middleware.GetUserIDFromContext(c)),
	)

	query := c.PostForm("query")
	city := c.PostForm("city")
	intent := c.PostForm("intent")

	if query == "" && city == "" {
		h.logger.Warn("Empty search query received")
		c.String(http.StatusBadRequest, `<div class="text-red-500">Please provide a search query.</div>`)
		return
	}

	// Call backend service to get itinerary data
	itinerary, err := h.fetchItineraryResults(query, city, intent)
	if err != nil {
		h.logger.Error("Failed to fetch itinerary results", zap.Error(err))
		c.String(http.StatusInternalServerError, `<div class="text-red-500">Failed to load itinerary models.</div>`)
		return
	}

	// Get user favorites (if authenticated)
	favorites := []string{}

	// Generate session ID for chat continuity
	sessionID := uuid.New().String()

	// Render results using the new itinerary results component
	// For legacy common, create empty city data and general POIs
	emptyCityData := models.GeneralCityData{}
	emptyGeneralPOIs := []models.POIDetailedInfo{}

	c.HTML(http.StatusOK, "", ItineraryResults(
		emptyCityData,    // cityData
		emptyGeneralPOIs, // generalPOIs
		itinerary,        // itinerary
		false,            // compact
		true,             // showToggle
		5,                // initialLimit
		favorites,        // favorites
		sessionID,        // sessionID for chat panel
	))
}

// Backend service calls - these would call your actual LLM service

func (h *ResultsHandlers) fetchRestaurantResults(query, city, intent string) ([]models.RestaurantDetailedInfo, error) {
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

func (h *ResultsHandlers) fetchActivityResults(query, city, intent string) ([]models.POIDetailedInfo, error) {
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

func (h *ResultsHandlers) fetchHotelResults(query, city, intent string) ([]models.HotelDetailedInfo, error) {
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

func (h *ResultsHandlers) fetchItineraryResults(query, city, intent string) (models.AIItineraryResponse, error) {
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
		return models.AIItineraryResponse{}, err
	}

	return itinerary, nil
}

// HandleItineraryStreamSearch serves the streaming itinerary results page
func (h *ResultsHandlers) HandleItineraryStreamSearch(c *gin.Context) {
	h.logger.Info("Itinerary stream search request received",
		zap.String("ip", c.ClientIP()),
		zap.String("user", middleware.GetUserIDFromContext(c)),
	)

	sessionID := c.Query("session_id")

	if sessionID == "" {
		c.String(http.StatusBadRequest, `<div class="text-red-500">Session ID is required for streaming.</div>`)
		return
	}

	// Render the SSE itinerary results page with empty data initially
	emptyCityData := models.GeneralCityData{}
	emptyGeneralPOIs := []models.POIDetailedInfo{}
	emptyItinerary := models.AIItineraryResponse{}

	c.HTML(http.StatusOK, "", ItineraryResults(emptyCityData, emptyGeneralPOIs, emptyItinerary, true, false, 5, []string{}, sessionID))
}

// SSE Response structures to match your backend
type SSEEvent struct {
	ID    string `json:"id"`
	Event string `json:"event"`
	Data  string `json:"data"`
}

type LLMStreamResponse struct {
	Content     string                          `json:"content"`
	Type        string                          `json:"type"`
	IsComplete  bool                            `json:"is_complete"`
	Restaurants []models.RestaurantDetailedInfo `json:"restaurants"`
	Activities  []models.POIDetailedInfo        `json:"activities"`
	Hotels      []models.HotelDetailedInfo      `json:"hotels"`
	Itinerary   *models.AIItineraryResponse     `json:"itinerary"`
}

func (h *ResultsHandlers) callLLMForRestaurants(_ string, _ map[string]interface{}) ([]models.RestaurantDetailedInfo, error) {
	// Use mock data directly - no external calls
	h.logger.Info("Using mock restaurant data (no external LLM calls)")
	return h.getMockRestaurants(), nil
}

func (h *ResultsHandlers) getMockRestaurants() []models.RestaurantDetailedInfo {
	cuisineType := "Portuguese"
	address := "Rua Augusta, Lisbon"
	phone := "+351 21 123 4567"
	priceLevel := "€€"
	hours := "12:00 - 24:00"
	website := "https://sample-restaurant.pt"

	return []models.RestaurantDetailedInfo{
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

func (h *ResultsHandlers) callLLMForActivities(_ string, _ map[string]interface{}) ([]models.POIDetailedInfo, error) {
	// Use mock data directly - no external calls
	h.logger.Info("Using mock activities data (no external LLM calls)")
	return h.getMockActivities(), nil
}

func (h *ResultsHandlers) getMockActivities() []models.POIDetailedInfo {
	return []models.POIDetailedInfo{
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

func (h *ResultsHandlers) callLLMForHotels(_ string, _ map[string]interface{}) ([]models.HotelDetailedInfo, error) {
	// Use mock data directly - no external calls
	h.logger.Info("Using mock hotels data (no external LLM calls)")
	return h.getMockHotels(), nil
}

func (h *ResultsHandlers) getMockHotels() []models.HotelDetailedInfo {
	priceRange1 := "€€€"
	priceRange2 := "€€€€"

	return []models.HotelDetailedInfo{
		{
			Name:        "Pousada de Lisboa",
			Description: "Elegant boutique hotel in a restored 18th-century building in the heart of Lisbon's historic center",
			Rating:      4.5,
			Category:    "Boutique Hotel",
			Address:     "Rua das Flores 4, 1200-194 Lisboa",
			PriceRange:  &priceRange1,
			Tags:        []string{"Historic", "Central Location", "Boutique", "Cultural"},
		},
		{
			Name:        "Hotel Tivoli Avenidas Liberdade",
			Description: "Modern luxury hotel on Lisbon's premier shopping avenue with contemporary amenities",
			Rating:      4.3,
			Category:    "Luxury Hotel",
			Address:     "Avenida da Liberdade 185, 1269-050 Lisboa",
			PriceRange:  &priceRange2,
			Tags:        []string{"Luxury", "Shopping", "Modern", "Business"},
		},
	}
}

func (h *ResultsHandlers) callLLMForItinerary(_ string, _ map[string]interface{}) (models.AIItineraryResponse, error) {
	// Use mock data directly - no external calls
	h.logger.Info("Using mock itinerary data (no external LLM calls)")
	return h.getMockItinerary(), nil
}

func (h *ResultsHandlers) getMockItinerary() models.AIItineraryResponse {
	return models.AIItineraryResponse{
		ItineraryName:      "Lisbon Discovery: 3-Day Cultural Journey",
		OverallDescription: "Experience Lisbon's rich maritime history, stunning architecture, and vibrant culture in this carefully crafted 3-day itinerary",
		PointsOfInterest: []models.POIDetailedInfo{
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
		Restaurants: []models.POIDetailedInfo{
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
		Bars: []models.POIDetailedInfo{
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
