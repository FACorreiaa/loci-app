package common

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/FACorreiaa/go-templui/internal/app/domain/activities"
	"github.com/FACorreiaa/go-templui/internal/app/domain/hotels"
	"github.com/FACorreiaa/go-templui/internal/app/domain/restaurants"
	"github.com/FACorreiaa/go-templui/internal/app/models"
	"github.com/FACorreiaa/go-templui/internal/pkg/middleware"
)

func TestMain(m *testing.M) {
	// Initialize logger for tests
	err := logger.Init(zapcore.InfoLevel, zap.String("service", "common-test"))
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}

	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestHandlers_BasicInstantiation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("should handle nil repository gracefully in page load", func(t *testing.T) {
		// This tests that common don't panic on basic operations
		// even with nil repository (for default page rendering)

		// Test Activities Handler
		activitiesHandler := &activities.ActivitiesHandlers{chatRepo: nil, itineraryService: nil}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		var err error
		c.Request, err = http.NewRequest("GET", "/activities", nil)
		assert.NoError(t, err)

		// Should not panic and return some component
		component := activitiesHandler.HandleActivitiesPage(c)
		assert.NotNil(t, component, "Activities handler should return a component")

		// Test Hotels Handler
		hotelsHandler := &hotels.HotelsHandlers{chatRepo: nil, itineraryService: nil}
		w = httptest.NewRecorder()
		c, _ = gin.CreateTestContext(w)
		c.Request, err = http.NewRequest("GET", "/hotels", nil)
		assert.NoError(t, err)

		component = hotelsHandler.HandleHotelsPage(c)
		assert.NotNil(t, component, "Hotels handler should return a component")

		// Test Restaurants Handler
		restaurantsHandler := &restaurants.RestaurantsHandlers{chatRepo: nil, itineraryService: nil}
		w = httptest.NewRecorder()
		c, _ = gin.CreateTestContext(w)
		c.Request, err = http.NewRequest("GET", "/restaurants", nil)
		assert.NoError(t, err)

		component = restaurantsHandler.HandleRestaurantsPage(c)
		assert.NotNil(t, component, "Restaurants handler should return a component")
	})

	t.Run("should handle query parameters without crashing", func(t *testing.T) {
		// Test with query parameters
		activitiesHandler := &activities.ActivitiesHandlers{chatRepo: nil, itineraryService: nil}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		var err error
		c.Request, err = http.NewRequest("GET", "/activities?q=museums", nil)
		assert.NoError(t, err)

		component := activitiesHandler.HandleActivitiesPage(c)
		assert.NotNil(t, component, "Activities handler should handle query parameters")
	})
}

func TestUnifiedDataSourceConcept(t *testing.T) {
	t.Run("should demonstrate unified data source filtering", func(t *testing.T) {
		// Create a unified dataset that represents what would come from SSE
		unifiedData := models.AiCityResponse{
			GeneralCityData: models.GeneralCityData{
				City:    "Amsterdam",
				Country: "Netherlands",
			},
			PointsOfInterest: []models.POIDetailedInfo{
				// Mixed POI types that would come from a single SSE stream
				{Name: "Van Gogh Museum", Category: "museum", Rating: 4.8},
				{Name: "Hotel V Nesplein", Category: "hotel", Rating: 4.6},
				{Name: "Café de Reiger", Category: "restaurant", Rating: 4.5},
				{Name: "Vondelpark", Category: "park", Rating: 4.7},
				{Name: "ClinkNOORD Hostel", Category: "hostel", Rating: 4.3},
				{Name: "Restaurant De Kas", Category: "restaurant", Rating: 4.9},
			},
		}

		// Demonstrate that each domain handler filters from the same data source
		allPOIs := unifiedData.PointsOfInterest

		// Activities filtering
		activities := activities.filterPOIsForActivities(allPOIs)
		activityNames := llmchat.extractPOINames(activities)
		assert.Contains(t, activityNames, "Van Gogh Museum")
		assert.Contains(t, activityNames, "Vondelpark")
		assert.Len(t, activities, 2)

		// Hotels filtering
		hotels := hotels.filterPOIsForHotels(allPOIs)
		hotelNames := llmchat.extractHotelNames(hotels)
		assert.Contains(t, hotelNames, "Hotel V Nesplein")
		assert.Contains(t, hotelNames, "ClinkNOORD Hostel")
		assert.Len(t, hotels, 2)

		// Restaurants filtering
		restaurants := filterPOIsForRestaurants(allPOIs)
		restaurantNames := llmchat.extractRestaurantNames(restaurants)
		assert.Contains(t, restaurantNames, "Café de Reiger")
		assert.Contains(t, restaurantNames, "Restaurant De Kas")
		assert.Len(t, restaurants, 2)

		// Verify total coverage (no POIs lost or duplicated across domains)
		totalFiltered := len(activities) + len(hotels) + len(restaurants)
		assert.Equal(t, len(allPOIs), totalFiltered,
			"All POIs should be captured by exactly one domain filter")
	})

	t.Run("should work with cache data structure", func(t *testing.T) {
		// Test that our filtering works with the actual cache data structure
		sessionID := "test-unified-data-session"

		testData := models.AiCityResponse{
			GeneralCityData: models.GeneralCityData{
				City:    "Barcelona",
				Country: "Spain",
			},
			PointsOfInterest: []models.POIDetailedInfo{
				{Name: "Sagrada Familia", Category: "attraction", Rating: 4.9},
				{Name: "Hotel Casa Fuster", Category: "hotel", Rating: 4.8},
				{Name: "Disfrutar", Category: "restaurant", Rating: 4.7},
			},
		}

		// Store in cache (this is what SSE would do)
		middleware.CompleteItineraryCache.Set(sessionID, testData)

		// Retrieve and verify (this is what common would do)
		cachedData, found := middleware.CompleteItineraryCache.Get(sessionID)
		assert.True(t, found, "Data should be found in cache")
		assert.Equal(t, "Barcelona", cachedData.GeneralCityData.City)
		assert.Len(t, cachedData.PointsOfInterest, 3)

		// Apply domain filtering on cached data
		activities := activities.filterPOIsForActivities(cachedData.PointsOfInterest)
		hotels := hotels.filterPOIsForHotels(cachedData.PointsOfInterest)
		restaurants := filterPOIsForRestaurants(cachedData.PointsOfInterest)

		assert.Len(t, activities, 1)
		assert.Len(t, hotels, 1)
		assert.Len(t, restaurants, 1)
	})
}
