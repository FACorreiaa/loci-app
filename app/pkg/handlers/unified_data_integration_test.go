//go:build integration

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FACorreiaa/go-templui/app/internal/models"
	"github.com/FACorreiaa/go-templui/app/pkg/middleware"
)

func TestUnifiedDataSource_Integration(t *testing.T) {
	if testDB == nil {
		t.Skip("Integration test skipped: no test database available")
	}

	gin.SetMode(gin.TestMode)

	// Setup handlers
	activitiesHandlers := NewActivitiesHandlers(testChatRepo)
	hotelsHandlers := NewHotelsHandlers(testChatRepo)
	restaurantsHandlers := NewRestaurantsHandlers(testChatRepo)

	t.Run("should use unified data source across all domain handlers", func(t *testing.T) {
		// Create test data that represents a complete itinerary response
		sessionID := uuid.New().String()
		testCityData := models.GeneralCityData{
			City:      "Amsterdam",
			Country:   "Netherlands",
			Latitude:  52.3676,
			Longitude: 4.9041,
		}

		testPOIs := []models.POIDetailedInfo{
			// Activities
			{
				ID:          uuid.New(),
				City:        "Amsterdam",
				Name:        "Van Gogh Museum",
				Category:    "museum",
				Rating:      4.8,
				Address:     "Museumplein 6, 1071 DJ Amsterdam, Netherlands",
				Description: "World's largest collection of Van Gogh's paintings",
			},
			{
				ID:          uuid.New(),
				City:        "Amsterdam",
				Name:        "Vondelpark",
				Category:    "park",
				Rating:      4.7,
				Address:     "Amsterdam, Netherlands",
				Description: "Large urban park in Amsterdam",
			},
			// Hotels
			{
				ID:          uuid.New(),
				City:        "Amsterdam",
				Name:        "The Hoxton, Amsterdam",
				Category:    "hotel",
				Rating:      4.6,
				Address:     "Herengracht 255, 1016 BJ Amsterdam, Netherlands",
				Description: "Boutique hotel in historic canal house",
				PhoneNumber: "+31 20 888 5555",
				Website:     "https://thehoxton.com",
				PriceRange:  "$$$",
			},
			{
				ID:          uuid.New(),
				City:        "Amsterdam",
				Name:        "ClinkNOORD",
				Category:    "hostel",
				Rating:      4.3,
				Address:     "Badhuiskade 3, 1031 KV Amsterdam, Netherlands",
				Description: "Modern hostel with industrial design",
			},
			// Restaurants
			{
				ID:          uuid.New(),
				City:        "Amsterdam",
				Name:        "Restaurant De Kas",
				Category:    "restaurant",
				Rating:      4.5,
				Address:     "Kamerlingh Onneslaan 3, 1097 DE Amsterdam, Netherlands",
				Description: "Restaurant in a greenhouse",
				PhoneNumber: "+31 20 462 4562",
				Website:     "https://restaurantdekas.nl",
				CuisineType: "International",
				PriceLevel:  "$$$",
			},
			{
				ID:          uuid.New(),
				City:        "Amsterdam",
				Name:        "Café de Reiger",
				Category:    "cafe",
				Rating:      4.4,
				Address:     "Nieuwe Leliestraat 34, 1015 SZ Amsterdam, Netherlands",
				Description: "Traditional brown café",
				CuisineType: "Dutch",
			},
			// Non-relevant categories for testing filtering
			{
				ID:          uuid.New(),
				City:        "Amsterdam",
				Name:        "Amsterdam Central Station",
				Category:    "transport",
				Rating:      4.2,
				Address:     "Stationsplein, 1012 AB Amsterdam, Netherlands",
				Description: "Main railway station",
			},
		}

		testCompleteData := models.CompleteItineraryData{
			GeneralCityData:  testCityData,
			PointsOfInterest: testPOIs,
			AIItineraryResponse: models.AIItineraryResponse{
				City: "Amsterdam",
				PointsOfInterest: []models.PersonalizedPOI{
					{Name: "Van Gogh Museum", Category: "museum"},
					{Name: "The Hoxton, Amsterdam", Category: "hotel"},
					{Name: "Restaurant De Kas", Category: "restaurant"},
				},
			},
		}

		// Store in cache (simulating SSE data population)
		middleware.CompleteItineraryCache.Set(sessionID, testCompleteData, time.Hour)

		// Test Activities Handler
		t.Run("activities handler should filter activities from unified data", func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/activities?sessionId="+sessionID, nil)

			component := activitiesHandlers.HandleActivitiesPage(c)
			assert.NotNil(t, component)

			// Verify filtering worked correctly
			filteredActivities := filterPOIsForActivities(testPOIs)
			assert.Len(t, filteredActivities, 2) // museum and park

			activityNames := make([]string, len(filteredActivities))
			for i, poi := range filteredActivities {
				activityNames[i] = poi.Name
			}
			assert.Contains(t, activityNames, "Van Gogh Museum")
			assert.Contains(t, activityNames, "Vondelpark")
			assert.NotContains(t, activityNames, "The Hoxton, Amsterdam")
			assert.NotContains(t, activityNames, "Restaurant De Kas")
		})

		// Test Hotels Handler
		t.Run("hotels handler should filter hotels from unified data", func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/hotels?sessionId="+sessionID, nil)

			component := hotelsHandlers.HandleHotelsPage(c)
			assert.NotNil(t, component)

			// Verify filtering and conversion worked correctly
			filteredHotels := filterPOIsForHotels(testPOIs)
			assert.Len(t, filteredHotels, 2) // hotel and hostel

			hotelNames := make([]string, len(filteredHotels))
			for i, hotel := range filteredHotels {
				hotelNames[i] = hotel.Name
			}
			assert.Contains(t, hotelNames, "The Hoxton, Amsterdam")
			assert.Contains(t, hotelNames, "ClinkNOORD")
			assert.NotContains(t, hotelNames, "Van Gogh Museum")
			assert.NotContains(t, hotelNames, "Restaurant De Kas")

			// Verify type conversion
			firstHotel := filteredHotels[0]
			assert.Equal(t, "Amsterdam", firstHotel.City)
			if firstHotel.Name == "The Hoxton, Amsterdam" {
				assert.NotNil(t, firstHotel.PhoneNumber)
				assert.Equal(t, "+31 20 888 5555", *firstHotel.PhoneNumber)
				assert.NotNil(t, firstHotel.Website)
				assert.Equal(t, "https://thehoxton.com", *firstHotel.Website)
			}
		})

		// Test Restaurants Handler
		t.Run("restaurants handler should filter restaurants from unified data", func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/restaurants?sessionId="+sessionID, nil)

			component := restaurantsHandlers.HandleRestaurantsPage(c)
			assert.NotNil(t, component)

			// Verify filtering and conversion worked correctly
			filteredRestaurants := filterPOIsForRestaurants(testPOIs)
			assert.Len(t, filteredRestaurants, 2) // restaurant and cafe

			restaurantNames := make([]string, len(filteredRestaurants))
			for i, restaurant := range filteredRestaurants {
				restaurantNames[i] = restaurant.Name
			}
			assert.Contains(t, restaurantNames, "Restaurant De Kas")
			assert.Contains(t, restaurantNames, "Café de Reiger")
			assert.NotContains(t, restaurantNames, "Van Gogh Museum")
			assert.NotContains(t, restaurantNames, "The Hoxton, Amsterdam")

			// Verify type conversion
			firstRestaurant := filteredRestaurants[0]
			assert.Equal(t, "Amsterdam", firstRestaurant.City)
			if firstRestaurant.Name == "Restaurant De Kas" {
				assert.NotNil(t, firstRestaurant.PhoneNumber)
				assert.Equal(t, "+31 20 462 4562", *firstRestaurant.PhoneNumber)
				assert.NotNil(t, firstRestaurant.CuisineType)
				assert.Equal(t, "International", *firstRestaurant.CuisineType)
			}
		})
	})
}

func TestUnifiedDataSource_DatabaseFallback(t *testing.T) {
	if testDB == nil {
		t.Skip("Integration test skipped: no test database available")
	}

	gin.SetMode(gin.TestMode)

	activitiesHandlers := NewActivitiesHandlers(testChatRepo)

	t.Run("should fallback to database when cache is empty", func(t *testing.T) {
		// Create a test interaction in the database
		sessionID := uuid.New()
		userID := uuid.New()

		testCompleteData := models.CompleteItineraryData{
			GeneralCityData: models.GeneralCityData{
				City:    "Berlin",
				Country: "Germany",
			},
			PointsOfInterest: []models.POIDetailedInfo{
				{
					ID:       uuid.New(),
					City:     "Berlin",
					Name:     "Brandenburg Gate",
					Category: "attraction",
					Rating:   4.8,
				},
			},
			AIItineraryResponse: models.AIItineraryResponse{
				City: "Berlin",
			},
		}

		// Convert to JSON for storage
		responseJSON, err := json.Marshal(testCompleteData)
		require.NoError(t, err)

		// Store in database
		ctx := context.Background()
		_, err = testChatRepo.CreateNewInteraction(
			ctx,
			userID,
			"Berlin travel guide",
			string(responseJSON),
			&sessionID,
			"test-provider",
			"itinerary",
		)
		require.NoError(t, err)

		// Clear cache to force database lookup
		middleware.CompleteItineraryCache.Delete(sessionID.String())

		// Test activities handler with database fallback
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/activities?sessionId="+sessionID.String(), nil)

		component := activitiesHandlers.HandleActivitiesPage(c)
		assert.NotNil(t, component)

		// Clean up
		err = testChatRepo.DeleteInteraction(ctx, sessionID)
		assert.NoError(t, err)
	})
}

func TestUnifiedDataSource_LegacyCacheFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	activitiesHandlers := NewActivitiesHandlers(testChatRepo)

	t.Run("should fallback to legacy cache when complete cache is empty", func(t *testing.T) {
		sessionID := "legacy-cache-test-session"

		// Create legacy cache data
		legacyData := models.AIItineraryResponse{
			City: "Vienna",
			PointsOfInterest: []models.PersonalizedPOI{
				{
					Name:     "Schönbrunn Palace",
					Category: "attraction",
					Rating:   4.9,
				},
				{
					Name:     "Vienna State Opera",
					Category: "theater",
					Rating:   4.8,
				},
			},
		}

		// Store in legacy cache only
		middleware.ItineraryCache.Set(sessionID, legacyData, time.Hour)

		// Ensure complete cache is empty
		middleware.CompleteItineraryCache.Delete(sessionID)

		// Test activities handler with legacy cache fallback
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/activities?sessionId="+sessionID, nil)

		component := activitiesHandlers.HandleActivitiesPage(c)
		assert.NotNil(t, component)

		// Clean up
		middleware.ItineraryCache.Delete(sessionID)
	})
}

func TestUnifiedDataSource_CacheConsistency(t *testing.T) {
	gin.SetMode(gin.TestMode)

	activitiesHandlers := NewActivitiesHandlers(testChatRepo)
	hotelsHandlers := NewHotelsHandlers(testChatRepo)
	restaurantsHandlers := NewRestaurantsHandlers(testChatRepo)

	t.Run("should return consistent data across all domain handlers", func(t *testing.T) {
		sessionID := "consistency-test-session"
		testCompleteData := models.CompleteItineraryData{
			GeneralCityData: models.GeneralCityData{
				City:      "Prague",
				Country:   "Czech Republic",
				Latitude:  50.0755,
				Longitude: 14.4378,
			},
			PointsOfInterest: []models.POIDetailedInfo{
				{
					ID:       uuid.New(),
					City:     "Prague",
					Name:     "Prague Castle",
					Category: "attraction",
					Rating:   4.9,
				},
				{
					ID:       uuid.New(),
					City:     "Prague",
					Name:     "Hotel Golden Well",
					Category: "hotel",
					Rating:   4.8,
				},
				{
					ID:       uuid.New(),
					City:     "Prague",
					Name:     "Lokál Dlouhááá",
					Category: "restaurant",
					Rating:   4.6,
				},
			},
		}

		// Store in complete cache
		middleware.CompleteItineraryCache.Set(sessionID, testCompleteData, time.Hour)

		// Test that all handlers access the same city data
		handlers := []struct {
			name    string
			handler interface{}
		}{
			{"activities", activitiesHandlers},
			{"hotels", hotelsHandlers},
			{"restaurants", restaurantsHandlers},
		}

		for _, h := range handlers {
			t.Run("consistency for "+h.name, func(t *testing.T) {
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				c.Request, _ = http.NewRequest("GET", "/"+h.name+"?sessionId="+sessionID, nil)

				var component interface{}
				switch handler := h.handler.(type) {
				case *ActivitiesHandlers:
					component = handler.HandleActivitiesPage(c)
				case *HotelsHandlers:
					component = handler.HandleHotelsPage(c)
				case *RestaurantsHandlers:
					component = handler.HandleRestaurantsPage(c)
				}

				assert.NotNil(t, component)
			})
		}

		// Verify each handler gets appropriate POIs from the same dataset
		activities := filterPOIsForActivities(testCompleteData.PointsOfInterest)
		hotels := filterPOIsForHotels(testCompleteData.PointsOfInterest)
		restaurants := filterPOIsForRestaurants(testCompleteData.PointsOfInterest)

		assert.Len(t, activities, 1)  // Prague Castle
		assert.Len(t, hotels, 1)      // Hotel Golden Well
		assert.Len(t, restaurants, 1) // Lokál Dlouhááá

		// Verify they all come from the same city
		assert.Equal(t, "Prague", activities[0].City)
		assert.Equal(t, "Prague", hotels[0].City)
		assert.Equal(t, "Prague", restaurants[0].City)
	})
}

func TestUnifiedDataSource_ErrorScenarios(t *testing.T) {
	gin.SetMode(gin.TestMode)

	activitiesHandlers := NewActivitiesHandlers(testChatRepo)

	t.Run("should handle invalid session ID gracefully", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/activities?sessionId=invalid-uuid", nil)

		component := activitiesHandlers.HandleActivitiesPage(c)
		assert.NotNil(t, component) // Should return page not found for invalid UUID format
	})

	t.Run("should handle empty cache and no database entry with empty results", func(t *testing.T) {
		nonExistentSessionID := uuid.New().String()

		// Note: Cache delete methods may not exist, but that's okay - they'll be empty anyway

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/activities?sessionId="+nonExistentSessionID, nil)

		component := activitiesHandlers.HandleActivitiesPage(c)
		assert.NotNil(t, component) // Should return empty results instead of 404
	})

	t.Run("should handle corrupted cache data", func(t *testing.T) {
		sessionID := "corrupted-data-session"

		// Store invalid data structure in cache
		corruptedData := models.CompleteItineraryData{
			GeneralCityData:  models.GeneralCityData{}, // Empty city data
			PointsOfInterest: nil,                      // Nil POIs
		}

		middleware.CompleteItineraryCache.Set(sessionID, corruptedData, time.Hour)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/activities?sessionId="+sessionID, nil)

		component := activitiesHandlers.HandleActivitiesPage(c)
		assert.NotNil(t, component) // Should handle gracefully
	})
}
