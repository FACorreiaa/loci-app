package handlers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/FACorreiaa/go-templui/app/internal/models"
)

// TestFilteringFunctions tests the domain-specific filtering functions
func TestFilteringFunctions(t *testing.T) {
	// Create comprehensive test POIs with various categories
	testPOIs := []models.POIDetailedInfo{
		// Activities/Attractions
		{ID: uuid.New(), Name: "Louvre Museum", Category: "museum", Rating: 4.8},
		{ID: uuid.New(), Name: "Central Park", Category: "park", Rating: 4.7},
		{ID: uuid.New(), Name: "Broadway Theater", Category: "theater", Rating: 4.6},
		{ID: uuid.New(), Name: "Art Gallery", Category: "gallery", Rating: 4.5},
		{ID: uuid.New(), Name: "Sports Stadium", Category: "sports", Rating: 4.4},
		{ID: uuid.New(), Name: "Adventure Park", Category: "adventure", Rating: 4.3},
		{ID: uuid.New(), Name: "Cultural Center", Category: "cultural", Rating: 4.2},
		{ID: uuid.New(), Name: "Entertainment Complex", Category: "entertainment", Rating: 4.1},
		{ID: uuid.New(), Name: "Outdoor Trail", Category: "outdoor", Rating: 4.0},
		{ID: uuid.New(), Name: "Recreation Center", Category: "recreation", Rating: 3.9},

		// Hotels/Accommodation
		{ID: uuid.New(), Name: "Luxury Hotel", Category: "hotel", Rating: 4.9},
		{ID: uuid.New(), Name: "Budget Hostel", Category: "hostel", Rating: 4.2},
		{ID: uuid.New(), Name: "Beach Resort", Category: "resort", Rating: 4.8},
		{ID: uuid.New(), Name: "Cozy Guesthouse", Category: "guesthouse", Rating: 4.5},
		{ID: uuid.New(), Name: "City Apartment", Category: "apartment", Rating: 4.4},
		{ID: uuid.New(), Name: "Mountain Villa", Category: "villa", Rating: 4.7},
		{ID: uuid.New(), Name: "Roadside Motel", Category: "motel", Rating: 3.8},
		{ID: uuid.New(), Name: "Historic Inn", Category: "inn", Rating: 4.3},
		{ID: uuid.New(), Name: "B&B Cottage", Category: "b&b", Rating: 4.2},
		{ID: uuid.New(), Name: "Accommodation Center", Category: "accommodation", Rating: 4.0},
		{ID: uuid.New(), Name: "Lodging House", Category: "lodging", Rating: 3.9},
		{ID: uuid.New(), Name: "BnB Place", Category: "bnb", Rating: 4.1},

		// Restaurants/Dining
		{ID: uuid.New(), Name: "Fine Restaurant", Category: "restaurant", Rating: 4.8},
		{ID: uuid.New(), Name: "Local Cafe", Category: "cafe", Rating: 4.5},
		{ID: uuid.New(), Name: "Coffee Shop", Category: "coffee", Rating: 4.3},
		{ID: uuid.New(), Name: "Wine Bar", Category: "bar", Rating: 4.6},
		{ID: uuid.New(), Name: "Traditional Pub", Category: "pub", Rating: 4.4},
		{ID: uuid.New(), Name: "French Bistro", Category: "bistro", Rating: 4.7},
		{ID: uuid.New(), Name: "Elegant Brasserie", Category: "brasserie", Rating: 4.5},
		{ID: uuid.New(), Name: "Italian Pizzeria", Category: "pizzeria", Rating: 4.2},
		{ID: uuid.New(), Name: "Local Bakery", Category: "bakery", Rating: 4.1},
		{ID: uuid.New(), Name: "Farmers Market", Category: "market", Rating: 4.0},
		{ID: uuid.New(), Name: "Food Court", Category: "foodcourt", Rating: 3.8},
		{ID: uuid.New(), Name: "Fast Food", Category: "fastfood", Rating: 3.5},
		{ID: uuid.New(), Name: "Takeaway Place", Category: "takeaway", Rating: 3.7},
		{ID: uuid.New(), Name: "Dining Hall", Category: "dining", Rating: 4.0},
		{ID: uuid.New(), Name: "Food Truck", Category: "food", Rating: 3.9},

		// Non-relevant categories (should be excluded from all filters)
		{ID: uuid.New(), Name: "Transport Station", Category: "transport", Rating: 4.0},
		{ID: uuid.New(), Name: "Shopping Mall", Category: "shopping", Rating: 4.1},
		{ID: uuid.New(), Name: "Office Building", Category: "office", Rating: 3.5},
		{ID: uuid.New(), Name: "Hospital", Category: "healthcare", Rating: 4.2},
		{ID: uuid.New(), Name: "School", Category: "education", Rating: 4.0},
	}

	t.Run("filterPOIsForActivities should include only activity categories", func(t *testing.T) {
		filtered := filterPOIsForActivities(testPOIs)

		// Expected activity categories: museum, park, theater, gallery, sports, adventure, cultural, entertainment, outdoor, recreation
		expectedCount := 10
		assert.Len(t, filtered, expectedCount, "Should filter exactly %d activity POIs", expectedCount)

		// Verify all filtered POIs are activity-related
		activityCategories := map[string]bool{
			"museum": true, "park": true, "theater": true, "gallery": true, "sports": true,
			"adventure": true, "cultural": true, "entertainment": true, "outdoor": true, "recreation": true,
		}

		for _, poi := range filtered {
			assert.True(t, activityCategories[poi.Category],
				"POI '%s' with category '%s' should not be in activities filter", poi.Name, poi.Category)
		}

		// Verify specific important POIs are included
		names := extractPOINames(filtered)
		assert.Contains(t, names, "Louvre Museum")
		assert.Contains(t, names, "Central Park")
		assert.Contains(t, names, "Broadway Theater")

		// Verify non-activity POIs are excluded
		assert.NotContains(t, names, "Luxury Hotel")
		assert.NotContains(t, names, "Fine Restaurant")
		assert.NotContains(t, names, "Transport Station")
	})

	t.Run("filterPOIsForHotels should include only accommodation categories", func(t *testing.T) {
		filtered := filterPOIsForHotels(testPOIs)

		// Expected hotel categories: hotel, hostel, resort, guesthouse, apartment, villa, motel, inn, b&b, accommodation, lodging, bnb
		expectedCount := 12
		assert.Len(t, filtered, expectedCount, "Should filter exactly %d hotel POIs", expectedCount)

		// Verify all filtered POIs are hotel-related
		hotelCategories := map[string]bool{
			"hotel": true, "hostel": true, "resort": true, "guesthouse": true, "apartment": true,
			"villa": true, "motel": true, "inn": true, "b&b": true, "accommodation": true, "lodging": true, "bnb": true,
		}

		for _, hotel := range filtered {
			assert.True(t, hotelCategories[hotel.Category],
				"POI '%s' with category '%s' should not be in hotels filter", hotel.Name, hotel.Category)
		}

		// Verify specific important POIs are included
		names := extractHotelNames(filtered)
		assert.Contains(t, names, "Luxury Hotel")
		assert.Contains(t, names, "Budget Hostel")
		assert.Contains(t, names, "Beach Resort")

		// Verify non-hotel POIs are excluded
		assert.NotContains(t, names, "Louvre Museum")
		assert.NotContains(t, names, "Fine Restaurant")
		assert.NotContains(t, names, "Transport Station")
	})

	t.Run("filterPOIsForRestaurants should include only dining categories", func(t *testing.T) {
		filtered := filterPOIsForRestaurants(testPOIs)

		// Expected restaurant categories: restaurant, cafe, coffee, bar, pub, bistro, brasserie, pizzeria, bakery, market, foodcourt, fastfood, takeaway, dining, food
		expectedCount := 15
		assert.Len(t, filtered, expectedCount, "Should filter exactly %d restaurant POIs", expectedCount)

		// Verify all filtered POIs are restaurant-related
		restaurantCategories := map[string]bool{
			"restaurant": true, "cafe": true, "coffee": true, "bar": true, "pub": true,
			"bistro": true, "brasserie": true, "pizzeria": true, "bakery": true, "market": true,
			"foodcourt": true, "fastfood": true, "takeaway": true, "dining": true, "food": true,
		}

		for _, restaurant := range filtered {
			assert.True(t, restaurantCategories[restaurant.Category],
				"POI '%s' with category '%s' should not be in restaurants filter", restaurant.Name, restaurant.Category)
		}

		// Verify specific important POIs are included
		names := extractRestaurantNames(filtered)
		assert.Contains(t, names, "Fine Restaurant")
		assert.Contains(t, names, "Local Cafe")
		assert.Contains(t, names, "Wine Bar")
		assert.Contains(t, names, "French Bistro")

		// Verify non-restaurant POIs are excluded
		assert.NotContains(t, names, "Louvre Museum")
		assert.NotContains(t, names, "Luxury Hotel")
		assert.NotContains(t, names, "Transport Station")
	})

	t.Run("filters should be case insensitive", func(t *testing.T) {
		mixedCasePOIs := []models.POIDetailedInfo{
			{ID: uuid.New(), Name: "MUSEUM", Category: "MUSEUM", Rating: 4.8},
			{ID: uuid.New(), Name: "Hotel", Category: "HOTEL", Rating: 4.7},
			{ID: uuid.New(), Name: "restaurant", Category: "Restaurant", Rating: 4.6},
		}

		activities := filterPOIsForActivities(mixedCasePOIs)
		hotels := filterPOIsForHotels(mixedCasePOIs)
		restaurants := filterPOIsForRestaurants(mixedCasePOIs)

		assert.Len(t, activities, 1, "Should handle case insensitive museum category")
		assert.Len(t, hotels, 1, "Should handle case insensitive hotel category")
		assert.Len(t, restaurants, 1, "Should handle case insensitive restaurant category")
	})

	t.Run("filters should handle empty input", func(t *testing.T) {
		emptyPOIs := []models.POIDetailedInfo{}

		activities := filterPOIsForActivities(emptyPOIs)
		hotels := filterPOIsForHotels(emptyPOIs)
		restaurants := filterPOIsForRestaurants(emptyPOIs)

		assert.Len(t, activities, 0)
		assert.Len(t, hotels, 0)
		assert.Len(t, restaurants, 0)
	})

	t.Run("filters should not overlap - unified data source integrity", func(t *testing.T) {
		activities := filterPOIsForActivities(testPOIs)
		hotels := filterPOIsForHotels(testPOIs)
		restaurants := filterPOIsForRestaurants(testPOIs)

		// Create sets of POI names from each filter
		activityNames := extractPOINames(activities)
		hotelNames := extractHotelNames(hotels)
		restaurantNames := extractRestaurantNames(restaurants)

		// Verify no overlap between categories
		for _, name := range activityNames {
			assert.NotContains(t, hotelNames, name, "Activity POI '%s' should not appear in hotels", name)
			assert.NotContains(t, restaurantNames, name, "Activity POI '%s' should not appear in restaurants", name)
		}

		for _, name := range hotelNames {
			assert.NotContains(t, activityNames, name, "Hotel POI '%s' should not appear in activities", name)
			assert.NotContains(t, restaurantNames, name, "Hotel POI '%s' should not appear in restaurants", name)
		}

		for _, name := range restaurantNames {
			assert.NotContains(t, activityNames, name, "Restaurant POI '%s' should not appear in activities", name)
			assert.NotContains(t, hotelNames, name, "Restaurant POI '%s' should not appear in hotels", name)
		}
	})
}

// TestConversionFunctions tests the type conversion functions
func TestConversionFunctions(t *testing.T) {
	t.Run("convertPOIToHotel should correctly convert all fields", func(t *testing.T) {
		poi := models.POIDetailedInfo{
			ID:          uuid.New(),
			City:        "Paris",
			Name:        "Le Meurice",
			Latitude:    48.8656,
			Longitude:   2.3272,
			Category:    "hotel",
			Description: "Luxury palace hotel",
			Address:     "228 Rue de Rivoli, 75001 Paris",
			PhoneNumber: "+33 1 44 58 10 10",
			Website:     "https://www.lemeurice.com",
			OpeningHours: map[string]string{
				"Monday":    "24 hours",
				"Tuesday":   "24 hours",
				"Wednesday": "24 hours",
			},
			PriceRange:       "$$$$",
			Rating:           4.9,
			Tags:             []string{"luxury", "historic", "palace"},
			Images:           []string{"facade.jpg", "lobby.jpg", "suite.jpg"},
			LlmInteractionID: uuid.New(),
		}

		hotel := convertPOIToHotel(poi)

		// Test basic fields
		assert.Equal(t, poi.ID, hotel.ID)
		assert.Equal(t, poi.City, hotel.City)
		assert.Equal(t, poi.Name, hotel.Name)
		assert.Equal(t, poi.Latitude, hotel.Latitude)
		assert.Equal(t, poi.Longitude, hotel.Longitude)
		assert.Equal(t, poi.Category, hotel.Category)
		assert.Equal(t, poi.Description, hotel.Description)
		assert.Equal(t, poi.Address, hotel.Address)
		assert.Equal(t, poi.Rating, hotel.Rating)
		assert.Equal(t, poi.Tags, hotel.Tags)
		assert.Equal(t, poi.Images, hotel.Images)
		assert.Equal(t, poi.LlmInteractionID, hotel.LlmInteractionID)

		// Test pointer fields
		assert.NotNil(t, hotel.PhoneNumber)
		assert.Equal(t, poi.PhoneNumber, *hotel.PhoneNumber)
		assert.NotNil(t, hotel.Website)
		assert.Equal(t, poi.Website, *hotel.Website)
		assert.NotNil(t, hotel.PriceRange)
		assert.Equal(t, poi.PriceRange, *hotel.PriceRange)

		// Test opening hours conversion
		assert.NotNil(t, hotel.OpeningHours)
		hoursStr := *hotel.OpeningHours
		assert.Contains(t, hoursStr, "Monday: 24 hours")
		assert.Contains(t, hoursStr, "Tuesday: 24 hours")
		assert.Contains(t, hoursStr, "Wednesday: 24 hours")
	})

	t.Run("convertPOIToRestaurant should correctly convert all fields", func(t *testing.T) {
		poi := models.POIDetailedInfo{
			ID:          uuid.New(),
			City:        "Rome",
			Name:        "La Pergola",
			Latitude:    41.9109,
			Longitude:   12.4818,
			Category:    "restaurant",
			Description: "Three Michelin star restaurant",
			Address:     "Via Alberto Cadlolo, 101, 00136 Roma RM",
			PhoneNumber: "+39 06 3509 2152",
			Website:     "https://www.lapergolaroma.com",
			OpeningHours: map[string]string{
				"Tuesday":   "19:30-23:30",
				"Wednesday": "19:30-23:30",
				"Thursday":  "19:30-23:30",
			},
			PriceLevel:       "$$$$",
			CuisineType:      "Mediterranean",
			Rating:           4.9,
			Tags:             []string{"michelin", "fine-dining", "rooftop"},
			Images:           []string{"dining-room.jpg", "dish1.jpg", "terrace.jpg"},
			LlmInteractionID: uuid.New(),
		}

		restaurant := convertPOIToRestaurant(poi)

		// Test basic fields
		assert.Equal(t, poi.ID, restaurant.ID)
		assert.Equal(t, poi.City, restaurant.City)
		assert.Equal(t, poi.Name, restaurant.Name)
		assert.Equal(t, poi.Latitude, restaurant.Latitude)
		assert.Equal(t, poi.Longitude, restaurant.Longitude)
		assert.Equal(t, poi.Category, restaurant.Category)
		assert.Equal(t, poi.Description, restaurant.Description)
		assert.Equal(t, poi.Rating, restaurant.Rating)
		assert.Equal(t, poi.Tags, restaurant.Tags)
		assert.Equal(t, poi.Images, restaurant.Images)
		assert.Equal(t, poi.LlmInteractionID, restaurant.LlmInteractionID)

		// Test pointer fields
		assert.NotNil(t, restaurant.Address)
		assert.Equal(t, poi.Address, *restaurant.Address)
		assert.NotNil(t, restaurant.PhoneNumber)
		assert.Equal(t, poi.PhoneNumber, *restaurant.PhoneNumber)
		assert.NotNil(t, restaurant.Website)
		assert.Equal(t, poi.Website, *restaurant.Website)
		assert.NotNil(t, restaurant.PriceLevel)
		assert.Equal(t, poi.PriceLevel, *restaurant.PriceLevel)
		assert.NotNil(t, restaurant.CuisineType)
		assert.Equal(t, poi.CuisineType, *restaurant.CuisineType)

		// Test opening hours conversion
		assert.NotNil(t, restaurant.OpeningHours)
		hoursStr := *restaurant.OpeningHours
		assert.Contains(t, hoursStr, "Tuesday: 19:30-23:30")
		assert.Contains(t, hoursStr, "Wednesday: 19:30-23:30")
		assert.Contains(t, hoursStr, "Thursday: 19:30-23:30")
	})

	t.Run("conversion functions should handle nil/empty optional fields", func(t *testing.T) {
		poi := models.POIDetailedInfo{
			ID:       uuid.New(),
			City:     "TestCity",
			Name:     "Basic POI",
			Category: "basic",
			Rating:   4.0,
			// All optional fields are empty/nil
		}

		hotel := convertPOIToHotel(poi)
		restaurant := convertPOIToRestaurant(poi)

		// Hotel conversion - all pointer fields should be nil
		assert.Nil(t, hotel.PhoneNumber)
		assert.Nil(t, hotel.Website)
		assert.Nil(t, hotel.PriceRange)
		assert.Nil(t, hotel.OpeningHours)

		// Restaurant conversion - all pointer fields should be nil
		assert.Nil(t, restaurant.Address)
		assert.Nil(t, restaurant.PhoneNumber)
		assert.Nil(t, restaurant.Website)
		assert.Nil(t, restaurant.PriceLevel)
		assert.Nil(t, restaurant.CuisineType)
		assert.Nil(t, restaurant.OpeningHours)
	})

	t.Run("opening hours conversion should handle empty maps", func(t *testing.T) {
		poi := models.POIDetailedInfo{
			ID:           uuid.New(),
			Name:         "Test POI",
			OpeningHours: map[string]string{}, // Empty map
		}

		hotel := convertPOIToHotel(poi)
		restaurant := convertPOIToRestaurant(poi)

		assert.Nil(t, hotel.OpeningHours)
		assert.Nil(t, restaurant.OpeningHours)
	})
}

// Helper functions for cleaner test code
func extractPOINames(pois []models.POIDetailedInfo) []string {
	names := make([]string, len(pois))
	for i, poi := range pois {
		names[i] = poi.Name
	}
	return names
}

func extractHotelNames(hotels []models.HotelDetailedInfo) []string {
	names := make([]string, len(hotels))
	for i, hotel := range hotels {
		names[i] = hotel.Name
	}
	return names
}

func extractRestaurantNames(restaurants []models.RestaurantDetailedInfo) []string {
	names := make([]string, len(restaurants))
	for i, restaurant := range restaurants {
		names[i] = restaurant.Name
	}
	return names
}
