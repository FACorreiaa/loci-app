package handlers

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/FACorreiaa/go-templui/app/lib/features/nearby"
	"github.com/FACorreiaa/go-templui/app/lib/models"
	"github.com/FACorreiaa/go-templui/app/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type NearbyHandlers struct{}

func NewNearbyHandlers() *NearbyHandlers {
	return &NearbyHandlers{}
}

func (h *NearbyHandlers) SearchPOIs(c *gin.Context) {
	query := c.PostForm("search-input")
	if query == "" {
		c.HTML(http.StatusOK, "", nearby.NearbySearchResults([]models.NearbyPOI{}))
		return
	}

	latitude, err := strconv.ParseFloat(c.PostForm("latitude"), 64)
	if err != nil {
		latitude = 0.0 // Default to 0 if parsing fails
	}
	longitude, err := strconv.ParseFloat(c.PostForm("longitude"), 64)
	if err != nil {
		longitude = 0.0 // Default to 0 if parsing fails
	}

	logger.Log.Info("POI search request",
		zap.String("query", query),
		zap.Float64("lat", latitude),
		zap.Float64("lng", longitude),
	)

	// Mock search results for now
	pois := h.getMockPOIs(latitude, longitude, query, "")

	c.HTML(http.StatusOK, "", nearby.NearbySearchResults(pois))
}

func (h *NearbyHandlers) GetPOIsByCategory(c *gin.Context) {
	category := c.Param("category")
	
	latitude, err := strconv.ParseFloat(c.PostForm("latitude"), 64)
	if err != nil {
		latitude = 0.0
	}
	longitude, err := strconv.ParseFloat(c.PostForm("longitude"), 64)
	if err != nil {
		longitude = 0.0
	}
	distance, err := strconv.ParseFloat(c.DefaultPostForm("distance", "5"), 64)
	if err != nil {
		distance = 5.0
	}
	minRating, err := strconv.ParseFloat(c.DefaultPostForm("rating", "0"), 64)
	if err != nil {
		minRating = 0.0
	}
	priceLevel := c.PostForm("price")

	logger.Log.Info("POI category request",
		zap.String("category", category),
		zap.Float64("lat", latitude),
		zap.Float64("lng", longitude),
		zap.Float64("distance", distance),
		zap.Float64("min_rating", minRating),
		zap.String("price", priceLevel),
	)

	if latitude == 0 || longitude == 0 {
		c.HTML(http.StatusOK, "", nearby.NearbyNoLocation())
		return
	}

	// Get POIs based on category and filters
	pois := h.getMockPOIs(latitude, longitude, "", category)
	
	// Apply filters
	filteredPOIs := h.applyFilters(pois, distance, minRating, priceLevel)

	c.HTML(http.StatusOK, "", nearby.NearbyPOIResults(filteredPOIs))
}

func (h *NearbyHandlers) FilterPOIs(c *gin.Context) {
	category := c.DefaultPostForm("category", "general")
	latitude, err := strconv.ParseFloat(c.PostForm("latitude"), 64)
	if err != nil {
		latitude = 0.0
	}
	longitude, err := strconv.ParseFloat(c.PostForm("longitude"), 64)
	if err != nil {
		longitude = 0.0
	}
	distance, err := strconv.ParseFloat(c.DefaultPostForm("distance", "5"), 64)
	if err != nil {
		distance = 5.0
	}
	minRating, err := strconv.ParseFloat(c.DefaultPostForm("rating", "0"), 64)
	if err != nil {
		minRating = 0.0
	}
	priceLevel := c.PostForm("price")

	logger.Log.Info("POI filter request",
		zap.String("category", category),
		zap.Float64("distance", distance),
		zap.Float64("min_rating", minRating),
		zap.String("price", priceLevel),
	)

	if latitude == 0 || longitude == 0 {
		c.HTML(http.StatusOK, "", nearby.NearbyNoLocation())
		return
	}

	// Get POIs and apply filters
	pois := h.getMockPOIs(latitude, longitude, "", category)
	filteredPOIs := h.applyFilters(pois, distance, minRating, priceLevel)

	c.HTML(http.StatusOK, "", nearby.NearbyPOIResults(filteredPOIs))
}

func (h *NearbyHandlers) GetMapData(c *gin.Context) {
	category := c.DefaultQuery("category", "general")
	latitude, _ := strconv.ParseFloat(c.Query("lat"), 64)
	longitude, _ := strconv.ParseFloat(c.Query("lng"), 64)
	distance, _ := strconv.ParseFloat(c.DefaultQuery("distance", "5"), 64)

	if latitude == 0 || longitude == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Location required"})
		return
	}

	pois := h.getMockPOIs(latitude, longitude, "", category)
	filteredPOIs := h.applyFilters(pois, distance, 0, "")

	// Convert to map markers
	markers := make([]map[string]interface{}, len(filteredPOIs))
	for i, poi := range filteredPOIs {
		markers[i] = map[string]interface{}{
			"id":        poi.ID,
			"name":      poi.Name,
			"category":  poi.Category,
			"lat":       poi.Latitude,
			"lng":       poi.Longitude,
			"rating":    poi.Rating,
			"address":   poi.Address,
			"imageUrl":  poi.ImageURL,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"markers": markers,
		"center": gin.H{
			"lat": latitude,
			"lng": longitude,
		},
	})
}

// Mock POI data generator for development
func (h *NearbyHandlers) getMockPOIs(lat, lng float64, query, category string) []models.NearbyPOI {
	// Generate mock POIs based on location and category
	pois := []models.NearbyPOI{}
	
	categories := []string{}
	switch category {
	case "restaurants":
		categories = []string{"restaurant", "cafe", "bar", "bakery"}
	case "activities":
		categories = []string{"entertainment", "recreation", "sports", "museum"}
	case "hotels":
		categories = []string{"lodging", "hotel", "motel", "resort"}
	case "attractions":
		categories = []string{"tourist_attraction", "park", "monument", "landmark"}
	default:
		categories = []string{"restaurant", "cafe", "entertainment", "shopping", "park"}
	}

	names := map[string][]string{
		"restaurant": {"The Gourmet Spot", "Bella Vista Restaurant", "Local Flavors", "Downtown Diner", "Corner Bistro"},
		"cafe": {"Coffee Central", "The Daily Grind", "Espresso Corner", "Morning Brew", "Café Delight"},
		"bar": {"The Rooftop", "Local Pub", "Wine & Dine", "Sports Bar", "The Nightspot"},
		"entertainment": {"Cinema Plaza", "The Gaming Lounge", "Live Music Venue", "Comedy Club", "Dance Studio"},
		"shopping": {"Fashion District", "Local Market", "Tech Store", "Bookshop", "Artisan Crafts"},
		"park": {"Central Park", "Riverside Walk", "Community Gardens", "Nature Reserve", "City Plaza"},
		"hotel": {"Grand Hotel", "Business Inn", "Boutique Lodge", "City Center Hotel", "Luxury Resort"},
		"museum": {"History Museum", "Art Gallery", "Science Center", "Cultural Center", "Heritage Site"},
	}

	// Generate 6-12 POIs
	numPOIs := 6 + int(math.Abs(lat+lng)) % 6
	
	for i := 0; i < numPOIs; i++ {
		cat := categories[i%len(categories)]
		nameList := names[cat]
		if nameList == nil {
			nameList = []string{"Local Business", "Popular Spot", "Neighborhood Favorite"}
		}
		
		name := nameList[i%len(nameList)]
		
		// Add query filtering
		if query != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(query)) && 
		   !strings.Contains(strings.ToLower(cat), strings.ToLower(query)) {
			continue
		}

		// Generate nearby coordinates (within ~5km)
		latOffset := (float64(i%100) - 50) * 0.001 // ~100m variations
		lngOffset := (float64((i*7)%100) - 50) * 0.001
		
		poiLat := lat + latOffset
		poiLng := lng + lngOffset
		
		// Calculate distance
		distance := h.calculateDistance(lat, lng, poiLat, poiLng)
		
		poi := models.NearbyPOI{
			ID:         fmt.Sprintf("poi_%d", i+1),
			Name:       name,
			Category:   strings.Title(strings.ReplaceAll(cat, "_", " ")),
			Address:    fmt.Sprintf("%d Main St, Local City", 100+i*10),
			Rating:     3.0 + float64(i%3) + (float64(i%10) / 10.0), // 3.0-5.9 range
			PriceLevel: 1 + (i % 4), // 1-4 price levels
			Distance:   distance,
			Latitude:   poiLat,
			Longitude:  poiLng,
			ImageURL:   fmt.Sprintf("https://picsum.photos/400/300?random=%d", i+1),
			Website:    fmt.Sprintf("https://example.com/poi_%d", i+1),
			PhoneNumber: fmt.Sprintf("+1-555-%03d-%04d", i+100, i*73%10000),
			IsOpen:     i%4 != 0, // 75% are open
		}
		
		pois = append(pois, poi)
	}
	
	return pois
}

func (h *NearbyHandlers) applyFilters(pois []models.NearbyPOI, maxDistance, minRating float64, priceLevel string) []models.NearbyPOI {
	filtered := []models.NearbyPOI{}
	
	for _, poi := range pois {
		// Distance filter
		if maxDistance > 0 && poi.Distance > maxDistance {
			continue
		}
		
		// Rating filter
		if minRating > 0 && poi.Rating < minRating {
			continue
		}
		
		// Price filter
		if priceLevel != "" {
			targetPrice, err := strconv.Atoi(priceLevel)
			if err == nil && targetPrice > 0 && poi.PriceLevel != targetPrice {
				continue
			}
		}
		
		filtered = append(filtered, poi)
	}
	
	return filtered
}

// Haversine formula to calculate distance between two points
func (h *NearbyHandlers) calculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371 // Earth's radius in kilometers
	
	dLat := (lat2 - lat1) * (math.Pi / 180)
	dLng := (lng2 - lng1) * (math.Pi / 180)
	
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*(math.Pi/180))*math.Cos(lat2*(math.Pi/180))*
		math.Sin(dLng/2)*math.Sin(dLng/2)
	
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	
	return R * c
}