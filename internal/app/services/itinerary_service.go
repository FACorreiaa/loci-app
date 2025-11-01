package services

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/FACorreiaa/go-templui/internal/app/models"
)

// ItineraryService provides business logic for itinerary operations
type ItineraryService struct{}

// NewItineraryService creates a new itinerary service
func NewItineraryService() *ItineraryService {
	return &ItineraryService{}
}

// ParseCompleteItineraryResponse parses multi-part SSE response format like [city_data]...[general_pois]...[itinerary]...
func (s *ItineraryService) ParseCompleteItineraryResponse(responseText string, logger *slog.Logger) (*models.AiCityResponse, error) {
	return parseSSEFormatResponse(responseText, logger)
}

// parseSSEFormatResponse parses multi-part SSE response format like [city_data]...[general_pois]...[itinerary]...
func parseSSEFormatResponse(responseText string, logger *slog.Logger) (*models.AiCityResponse, error) {
	result := &models.AiCityResponse{}

	// Parse city_data section
	if cityMatch := regexp.MustCompile(`\[city_data\]\s*(.*?)(?:\n\n|\[|$)`).FindStringSubmatch(responseText); len(cityMatch) > 1 {
		var cityData models.GeneralCityData
		if err := json.Unmarshal([]byte(strings.TrimSpace(cityMatch[1])), &cityData); err == nil {
			result.GeneralCityData = cityData
			logger.Debug("parseSSEFormatResponse: Parsed city_data section")
		}
	}

	// Parse general_pois section
	if poisMatch := regexp.MustCompile(`\[general_pois\]\s*(.*?)(?:\n\n|\[|$)`).FindStringSubmatch(responseText); len(poisMatch) > 1 {
		// Try parsing as wrapper with points_of_interest field first
		var poisWrapper struct {
			PointsOfInterest []models.POIDetailedInfo `json:"points_of_interest"`
		}
		cleanedPOIsJSON := strings.TrimSpace(poisMatch[1])
		if err := json.Unmarshal([]byte(cleanedPOIsJSON), &poisWrapper); err == nil && len(poisWrapper.PointsOfInterest) > 0 {
			result.PointsOfInterest = poisWrapper.PointsOfInterest
			logger.Debug("parseSSEFormatResponse: Parsed general_pois section with wrapper", "count", len(poisWrapper.PointsOfInterest))
		} else {
			// Fallback: try parsing as direct array
			var generalPOIs []models.POIDetailedInfo
			if err := json.Unmarshal([]byte(cleanedPOIsJSON), &generalPOIs); err == nil {
				result.PointsOfInterest = generalPOIs
				logger.Debug("parseSSEFormatResponse: Parsed general_pois section as direct array", "count", len(generalPOIs))
			} else {
				logger.Warn("parseSSEFormatResponse: Failed to parse general_pois section", "error", err)
			}
		}
	}

	// Parse itinerary section
	if itineraryMatch := regexp.MustCompile(`\[itinerary\]\s*(.*?)(?:\n\n|\[|$)`).FindStringSubmatch(responseText); len(itineraryMatch) > 1 {
		var itineraryData models.AIItineraryResponse
		if err := json.Unmarshal([]byte(strings.TrimSpace(itineraryMatch[1])), &itineraryData); err == nil {
			result.AIItineraryResponse = itineraryData
			logger.Debug("parseSSEFormatResponse: Parsed itinerary section", "poisCount", len(itineraryData.PointsOfInterest))
		}
	}

	// Return result if we have at least some data
	if result.GeneralCityData.City != "" || len(result.PointsOfInterest) > 0 || result.AIItineraryResponse.ItineraryName != "" {
		return result, nil
	}

	// Fallback: try to parse as legacy format for backwards compatibility
	return parseCompleteItineraryResponseLegacy(responseText, logger)
}

// parseCompleteItineraryResponseLegacy handles legacy format (backwards compatibility)
func parseCompleteItineraryResponseLegacy(responseText string, logger *slog.Logger) (*models.AiCityResponse, error) {
	// Try legacy parsing methods
	if legacyItinerary, err := parseItineraryFromResponse(responseText, logger); err == nil && legacyItinerary != nil {
		result := &models.AiCityResponse{
			AIItineraryResponse: *legacyItinerary,
		}
		logger.Debug("parseCompleteItineraryResponseLegacy: Parsed as legacy format")
		return result, nil
	}

	logger.Debug("parseCompleteItineraryResponse: Could not parse response in any format")
	return nil, fmt.Errorf("failed to parse complete itinerary response")
}

// parseItineraryFromResponse parses an AIItineraryResponse from a stored LLM response (legacy function for backwards compatibility)
func parseItineraryFromResponse(responseText string, logger *slog.Logger) (*models.AIItineraryResponse, error) {
	if responseText == "" {
		return nil, nil
	}

	// Clean the JSON response (similar to existing parsePOIsFromResponse)
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
	return nil, err
}

// cleanJSONResponse removes markdown code blocks and cleans up the JSON response
func cleanJSONResponse(response string) string {
	// Remove markdown code blocks
	cleaned := strings.ReplaceAll(response, "```json", "")
	cleaned = strings.ReplaceAll(cleaned, "```", "")

	// Remove any leading/trailing whitespace
	cleaned = strings.TrimSpace(cleaned)

	return cleaned
}
