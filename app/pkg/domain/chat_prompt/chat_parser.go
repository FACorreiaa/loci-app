package llmchat

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/FACorreiaa/go-templui/app/internal/models"
	"github.com/google/uuid"
)

// parseItineraryFromResponse parses an AIItineraryResponse from a stored LLM response
func parseItineraryFromResponse(responseText string, logger *slog.Logger) (*models.AIItineraryResponse, error) {
	if responseText == "" {
		return nil, fmt.Errorf("empty response text")
	}

	// Clean the JSON response
	cleanedResponse := cleanJSONResponse(responseText)

	// Try to parse as unified chat response format with "data" wrapper first
	var unifiedResponse struct {
		Data models.AiCityResponse `json:"data"`
	}
	err := json.Unmarshal([]byte(cleanedResponse), &unifiedResponse)
	if err == nil && (unifiedResponse.Data.AIItineraryResponse.ItineraryName != "" || len(unifiedResponse.Data.AIItineraryResponse.PointsOfInterest) > 0) {
		logger.Debug("parseItineraryFromResponse: Parsed as unified chat response")
		return &unifiedResponse.Data.AIItineraryResponse, nil
	}

	// Try to parse as direct AiCityResponse
	var cityResponse models.AiCityResponse
	err = json.Unmarshal([]byte(cleanedResponse), &cityResponse)
	if err == nil && (cityResponse.AIItineraryResponse.ItineraryName != "" || len(cityResponse.AIItineraryResponse.PointsOfInterest) > 0) {
		logger.Debug("parseItineraryFromResponse: Parsed as AiCityResponse")
		return &cityResponse.AIItineraryResponse, nil
	}

	// Try to parse directly as AIItineraryResponse
	var itineraryResponse models.AIItineraryResponse
	err = json.Unmarshal([]byte(cleanedResponse), &itineraryResponse)
	if err == nil && (itineraryResponse.ItineraryName != "" || len(itineraryResponse.PointsOfInterest) > 0) {
		logger.Debug("parseItineraryFromResponse: Parsed as direct AIItineraryResponse")
		return &itineraryResponse, nil
	}

	logger.Debug("parseItineraryFromResponse: Could not parse response as itinerary", "error", err)
	return nil, fmt.Errorf("failed to parse itinerary: %w", err)
}

// parseCompleteResponseFromParts parses a complete AiCityResponse from individual SSE response parts
func (l *ServiceImpl) parseCompleteResponseFromParts(responses map[string]*strings.Builder, sessionID uuid.UUID) (*models.AiCityResponse, error) {
	completeResponse := &models.AiCityResponse{
		SessionID: sessionID,
	}

	// Parse city_data part
	if cityDataBuilder, exists := responses["city_data"]; exists && cityDataBuilder != nil {
		cityDataStr := cityDataBuilder.String()
		if cityDataStr != "" {
			cleanedCityData := cleanJSONResponse(cityDataStr)
			var cityData models.GeneralCityData
			if err := json.Unmarshal([]byte(cleanedCityData), &cityData); err != nil {
				l.logger.Warn("Failed to parse city_data part", slog.Any("error", err))
			} else {
				completeResponse.GeneralCityData = cityData
			}
		}
	}

	// Parse general_pois part
	if poisBuilder, exists := responses["general_pois"]; exists && poisBuilder != nil {
		poisStr := poisBuilder.String()
		if poisStr != "" {
			cleanedPOIs := cleanJSONResponse(poisStr)

			// Try parsing as wrapper with points_of_interest field first (this is what the LLM returns)
			var poisWrapper struct {
				PointsOfInterest []models.POIDetailedInfo `json:"points_of_interest"`
			}
			if err := json.Unmarshal([]byte(cleanedPOIs), &poisWrapper); err == nil && len(poisWrapper.PointsOfInterest) > 0 {
				completeResponse.PointsOfInterest = poisWrapper.PointsOfInterest
				l.logger.Debug("Parsed general_pois as wrapped object", slog.Int("count", len(poisWrapper.PointsOfInterest)))
			} else {
				// Fallback: try parsing as direct array
				var pois []models.POIDetailedInfo
				if err := json.Unmarshal([]byte(cleanedPOIs), &pois); err != nil {
					l.logger.Warn("Failed to parse general_pois part", slog.Any("error", err))
				} else {
					completeResponse.PointsOfInterest = pois
					l.logger.Debug("Parsed general_pois as direct array", slog.Int("count", len(pois)))
				}
			}
		}
	}

	// Parse itinerary part
	if itineraryBuilder, exists := responses["itinerary"]; exists && itineraryBuilder != nil {
		itineraryStr := itineraryBuilder.String()
		if parsedItinerary, err := parseItineraryFromResponse(itineraryStr, l.logger); err == nil && parsedItinerary != nil {
			completeResponse.AIItineraryResponse = *parsedItinerary
		} else {
			l.logger.Warn("Failed to parse itinerary part", slog.Any("error", err))
		}
	}

	// Parse restaurants part
	if restaurantBuilder, exists := responses["restaurants"]; exists && restaurantBuilder != nil {
		restaurantStr := restaurantBuilder.String()
		if restaurants, err := parseRestaurantsFromResponse(restaurantStr, l.logger); err == nil && len(restaurants) > 0 {
			var pois []models.POIDetailedInfo
			for _, r := range restaurants {
				pois = append(pois, models.POIDetailedInfo{
					Name:        r.Name,
					Category:    "Restaurant",
					Description: r.Description,
					Latitude:    r.Latitude,
					Longitude:   r.Longitude,
					Address:     *r.Address,
					Website:     *r.Website,
					PhoneNumber: *r.PhoneNumber,
					PriceLevel:  *r.PriceLevel,
					Rating:      r.Rating,
					Tags:        r.Tags,
					Images:      r.Images,
				})
			}
			completeResponse.PointsOfInterest = pois
		} else {
			l.logger.Warn("Failed to parse restaurants part", slog.Any("error", err))
		}
	}

	// Populate city name on all POIs from parsed city data
	cityName := completeResponse.GeneralCityData.City
	if cityName != "" {
		// Update general POIs with city name
		for i := range completeResponse.PointsOfInterest {
			if completeResponse.PointsOfInterest[i].City == "" {
				completeResponse.PointsOfInterest[i].City = cityName
			}
		}
		// Update itinerary POIs with city name
		for i := range completeResponse.AIItineraryResponse.PointsOfInterest {
			if completeResponse.AIItineraryResponse.PointsOfInterest[i].City == "" {
				completeResponse.AIItineraryResponse.PointsOfInterest[i].City = cityName
			}
		}
	}

	// Validate that we have at least some data
	if completeResponse.GeneralCityData.City == "" &&
		len(completeResponse.PointsOfInterest) == 0 &&
		completeResponse.AIItineraryResponse.ItineraryName == "" {
		return nil, fmt.Errorf("no valid data found in any response parts")
	}

	return completeResponse, nil
}

// parseRestaurantsFromResponse parses restaurant data from SSE response
func parseRestaurantsFromResponse(responseText string, _ *slog.Logger) ([]models.RestaurantDetailedInfo, error) {
	if responseText == "" {
		return nil, fmt.Errorf("empty restaurant response text")
	}

	cleanedResponse := cleanJSONResponse(responseText)

	// Try to parse as array of restaurants
	var restaurants []models.RestaurantDetailedInfo
	if err := json.Unmarshal([]byte(cleanedResponse), &restaurants); err == nil {
		return restaurants, nil
	}

	// Try to parse as wrapper with data field
	var wrapper struct {
		Data []models.RestaurantDetailedInfo `json:"data"`
	}
	if err := json.Unmarshal([]byte(cleanedResponse), &wrapper); err == nil && len(wrapper.Data) > 0 {
		return wrapper.Data, nil
	}

	// Try to parse as wrapper with restaurants field
	var restaurantsWrapper struct {
		Restaurants []models.RestaurantDetailedInfo `json:"restaurants"`
	}
	if err := json.Unmarshal([]byte(cleanedResponse), &restaurantsWrapper); err == nil && len(restaurantsWrapper.Restaurants) > 0 {
		return restaurantsWrapper.Restaurants, nil
	}

	return nil, fmt.Errorf("failed to parse restaurant response")
}

// parseActivitiesFromResponse parses activity data from SSE response
func parseActivitiesFromResponse(responseText string, _ *slog.Logger) ([]models.POIDetailedInfo, error) {
	if responseText == "" {
		return nil, fmt.Errorf("empty activities response text")
	}

	cleanedResponse := cleanJSONResponse(responseText)

	// Try to parse as array of activities
	var activities []models.POIDetailedInfo
	if err := json.Unmarshal([]byte(cleanedResponse), &activities); err == nil {
		return activities, nil
	}

	// Try to parse as wrapper with data field
	var wrapper struct {
		Data []models.POIDetailedInfo `json:"data"`
	}
	if err := json.Unmarshal([]byte(cleanedResponse), &wrapper); err == nil && len(wrapper.Data) > 0 {
		return wrapper.Data, nil
	}

	// Try to parse as wrapper with activities field
	var activitiesWrapper struct {
		Activities []models.POIDetailedInfo `json:"activities"`
	}
	if err := json.Unmarshal([]byte(cleanedResponse), &activitiesWrapper); err == nil && len(activitiesWrapper.Activities) > 0 {
		return activitiesWrapper.Activities, nil
	}

	return nil, fmt.Errorf("failed to parse activities response")
}

// parseHotelsFromResponse parses hotel data from SSE response
func parseHotelsFromResponse(responseText string, _ *slog.Logger) ([]models.HotelDetailedInfo, error) {
	if responseText == "" {
		return nil, fmt.Errorf("empty hotels response text")
	}

	cleanedResponse := cleanJSONResponse(responseText)

	// Try to parse as array of hotels
	var hotels []models.HotelDetailedInfo
	if err := json.Unmarshal([]byte(cleanedResponse), &hotels); err == nil {
		return hotels, nil
	}

	// Try to parse as wrapper with data field
	var wrapper struct {
		Data []models.HotelDetailedInfo `json:"data"`
	}
	if err := json.Unmarshal([]byte(cleanedResponse), &wrapper); err == nil && len(wrapper.Data) > 0 {
		return wrapper.Data, nil
	}

	// Try to parse as wrapper with hotels field
	var hotelsWrapper struct {
		Hotels []models.HotelDetailedInfo `json:"hotels"`
	}
	if err := json.Unmarshal([]byte(cleanedResponse), &hotelsWrapper); err == nil && len(hotelsWrapper.Hotels) > 0 {
		return hotelsWrapper.Hotels, nil
	}

	return nil, fmt.Errorf("failed to parse hotels response")
}
