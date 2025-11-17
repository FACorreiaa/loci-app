package common

import (
	"strings"

	"github.com/FACorreiaa/go-templui/internal/app/models"
)

// FilterPOIsForActivities returns only activity-related POIs
func FilterPOIsForActivities(allPOIs []models.POIDetailedInfo) []models.POIDetailedInfo {
	var activityPOIs []models.POIDetailedInfo
	activityCategories := map[string]bool{
		"museum": true, "park": true, "theater": true, "gallery": true, "sports": true,
		"adventure": true, "cultural": true, "entertainment": true, "outdoor": true, "recreation": true,
		"attraction": true, "landmark": true, "experience": true,
	}

	for _, poi := range allPOIs {
		if activityCategories[strings.ToLower(poi.Category)] {
			activityPOIs = append(activityPOIs, poi)
		}
	}

	return activityPOIs
}

// FilterPOIsForHotels returns only accommodation-related POIs, leveraging shared conversion helpers.
func FilterPOIsForHotels(allPOIs []models.POIDetailedInfo) []models.HotelDetailedInfo {
	var hotelPOIs []models.HotelDetailedInfo
	hotelCategories := map[string]bool{
		"hotel": true, "hostel": true, "resort": true, "guesthouse": true, "apartment": true,
		"villa": true, "motel": true, "inn": true, "b&b": true, "accommodation": true,
		"lodging": true, "bnb": true,
	}

	for _, poi := range allPOIs {
		if hotelCategories[strings.ToLower(poi.Category)] {
			hotelPOIs = append(hotelPOIs, ConvertPOIToHotel(poi))
		}
	}

	return hotelPOIs
}

// FilterPOIsForRestaurants returns only dining-related POIs.
func FilterPOIsForRestaurants(allPOIs []models.POIDetailedInfo) []models.RestaurantDetailedInfo {
	var restaurantPOIs []models.RestaurantDetailedInfo
	restaurantCategories := map[string]bool{
		"restaurant": true, "cafe": true, "coffee": true, "bar": true, "pub": true,
		"bistro": true, "brasserie": true, "pizzeria": true, "bakery": true, "market": true,
		"foodcourt": true, "fastfood": true, "takeaway": true, "dining": true, "food": true,
		"eatery": true,
	}

	for _, poi := range allPOIs {
		if restaurantCategories[strings.ToLower(poi.Category)] {
			restaurantPOIs = append(restaurantPOIs, ConvertPOIToRestaurant(poi))
		}
	}

	return restaurantPOIs
}

// ConvertPOIToHotel converts POI metadata into a hotel structure.
func ConvertPOIToHotel(poi models.POIDetailedInfo) models.HotelDetailedInfo {
	var phoneNumber *string
	if poi.PhoneNumber != "" {
		phoneNumber = &poi.PhoneNumber
	}

	var website *string
	if poi.Website != "" {
		website = &poi.Website
	}

	var priceRange *string
	if poi.PriceRange != "" {
		priceRange = &poi.PriceRange
	}

	var openingHours *string
	if len(poi.OpeningHours) > 0 {
		var parts []string
		for day, hours := range poi.OpeningHours {
			parts = append(parts, day+": "+hours)
		}
		if len(parts) > 0 {
			joined := strings.Join(parts, ", ")
			openingHours = &joined
		}
	}

	return models.HotelDetailedInfo{
		ID:               poi.ID,
		City:             poi.City,
		Name:             poi.Name,
		Latitude:         poi.Latitude,
		Longitude:        poi.Longitude,
		Category:         poi.Category,
		Description:      poi.Description,
		Address:          poi.Address,
		Website:          website,
		PhoneNumber:      phoneNumber,
		OpeningHours:     openingHours,
		PriceRange:       priceRange,
		Tags:             poi.Tags,
		Images:           poi.Images,
		Rating:           poi.Rating,
		LlmInteractionID: poi.LlmInteractionID,
	}
}

// ConvertPOIToRestaurant converts POI metadata into a restaurant structure.
func ConvertPOIToRestaurant(poi models.POIDetailedInfo) models.RestaurantDetailedInfo {
	var address *string
	if poi.Address != "" {
		address = &poi.Address
	}

	var phoneNumber *string
	if poi.PhoneNumber != "" {
		phoneNumber = &poi.PhoneNumber
	}

	var website *string
	if poi.Website != "" {
		website = &poi.Website
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
		var parts []string
		for day, hours := range poi.OpeningHours {
			parts = append(parts, day+": "+hours)
		}
		if len(parts) > 0 {
			joined := strings.Join(parts, ", ")
			openingHours = &joined
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
