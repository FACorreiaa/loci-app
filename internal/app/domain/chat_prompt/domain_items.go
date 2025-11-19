package llmchat

import (
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/FACorreiaa/go-templui/internal/app/models"
)

func prepareDomainItem(poi models.POIDetailedInfo, domain string, index int) (interface{}, map[string]interface{}) {
	switch domain {
	case "hotels":
		hotel := convertPOIToHotel(poi)
		return hotel, buildItemDataFromHotel(hotel, index)
	case "restaurants":
		restaurant := convertPOIToRestaurant(poi)
		return restaurant, buildItemDataFromRestaurant(restaurant, index)
	case "activities":
		return poi, buildItemDataFromPOI(poi, index)
	default:
		return poi, buildItemDataFromPOI(poi, index)
	}
}

func convertPOIToHotel(poi models.POIDetailedInfo) models.HotelDetailedInfo {
	if poi.ID == uuid.Nil {
		poi.ID = uuid.New()
	}
	return models.HotelDetailedInfo{
		ID:               poi.ID,
		City:             poi.City,
		Name:             poi.Name,
		Latitude:         poi.Latitude,
		Longitude:        poi.Longitude,
		Category:         fallbackString(poi.Category, "Hotel"),
		Description:      fallbackString(poi.Description, poi.DescriptionPOI),
		Address:          poi.Address,
		PhoneNumber:      optionalString(poi.PhoneNumber),
		Website:          optionalString(poi.Website),
		OpeningHours:     stringifyOpeningHours(poi.OpeningHours),
		PriceRange:       optionalString(poi.PriceRange),
		Rating:           poi.Rating,
		Tags:             poi.Tags,
		Images:           poi.Images,
		LlmInteractionID: poi.LlmInteractionID,
	}
}

func convertPOIToRestaurant(poi models.POIDetailedInfo) models.RestaurantDetailedInfo {
	if poi.ID == uuid.Nil {
		poi.ID = uuid.New()
	}
	return models.RestaurantDetailedInfo{
		ID:               poi.ID,
		City:             poi.City,
		Name:             poi.Name,
		Latitude:         poi.Latitude,
		Longitude:        poi.Longitude,
		Category:         poi.Category,
		Description:      fallbackString(poi.Description, poi.DescriptionPOI),
		Address:          optionalString(poi.Address),
		Website:          optionalString(poi.Website),
		PhoneNumber:      optionalString(poi.PhoneNumber),
		OpeningHours:     stringifyOpeningHours(poi.OpeningHours),
		PriceLevel:       optionalString(poi.PriceLevel),
		CuisineType:      optionalString(poi.CuisineType),
		Tags:             poi.Tags,
		Images:           poi.Images,
		Rating:           poi.Rating,
		LlmInteractionID: poi.LlmInteractionID,
	}
}

func buildItemDataFromHotel(hotel models.HotelDetailedInfo, index int) map[string]interface{} {
	return map[string]interface{}{
		"name":        hotel.Name,
		"category":    hotel.Category,
		"latitude":    hotel.Latitude,
		"longitude":   hotel.Longitude,
		"description": hotel.Description,
		"index":       index,
	}
}

func buildItemDataFromRestaurant(restaurant models.RestaurantDetailedInfo, index int) map[string]interface{} {
	return map[string]interface{}{
		"name":        restaurant.Name,
		"category":    restaurant.Category,
		"latitude":    restaurant.Latitude,
		"longitude":   restaurant.Longitude,
		"description": restaurant.Description,
		"index":       index,
	}
}

func buildItemDataFromPOI(poi models.POIDetailedInfo, index int) map[string]interface{} {
	return map[string]interface{}{
		"name":        poi.Name,
		"category":    poi.Category,
		"latitude":    poi.Latitude,
		"longitude":   poi.Longitude,
		"description": fallbackString(poi.Description, poi.DescriptionPOI),
		"index":       index,
	}
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	copy := trimmed
	return &copy
}

func stringifyOpeningHours(hours map[string]string) *string {
	if len(hours) == 0 {
		return nil
	}
	keys := make([]string, 0, len(hours))
	for key := range hours {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var builder strings.Builder
	for i, key := range keys {
		if i > 0 {
			builder.WriteString(" · ")
		}
		builder.WriteString(key)
		builder.WriteString(": ")
		builder.WriteString(hours[key])
	}
	text := strings.TrimSpace(builder.String())
	if text == "" {
		return nil
	}
	return &text
}

func fallbackString(primary, secondary string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return secondary
}
