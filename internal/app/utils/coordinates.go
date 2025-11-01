package utils

import (
	"math"
)

// ValidateCoordinates checks if latitude and longitude are valid
// Latitude must be between -90 and 90
// Longitude must be between -180 and 180
func ValidateCoordinates(lat, lng float64) bool {
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180 && lat != 0 && lng != 0
}

// HasValidCoordinates checks if an object with lat/lng fields has valid coordinates
func HasValidCoordinates(lat, lng float64) bool {
	// Check for zero values (often indicates missing data)
	if lat == 0 && lng == 0 {
		return false
	}

	// Check for valid ranges
	return ValidateCoordinates(lat, lng)
}

// CalculateCenterPoint calculates the center point of multiple coordinates
// Returns the center point or a default fallback if no valid coordinates exist
func CalculateCenterPoint(coordinates [][2]float64, fallbackLat, fallbackLng float64) (float64, float64) {
	validCoords := make([][2]float64, 0)

	for _, coord := range coordinates {
		if ValidateCoordinates(coord[0], coord[1]) {
			validCoords = append(validCoords, coord)
		}
	}

	// If no valid coordinates, return fallback
	if len(validCoords) == 0 {
		return fallbackLat, fallbackLng
	}

	// Calculate average
	var latSum, lngSum float64
	for _, coord := range validCoords {
		latSum += coord[0]
		lngSum += coord[1]
	}

	return latSum / float64(len(validCoords)), lngSum / float64(len(validCoords))
}

// CalculateBounds returns the bounding box for a set of coordinates
// Returns minLat, maxLat, minLng, maxLng
func CalculateBounds(coordinates [][2]float64) (float64, float64, float64, float64) {
	if len(coordinates) == 0 {
		return 0, 0, 0, 0
	}

	minLat := math.MaxFloat64
	maxLat := -math.MaxFloat64
	minLng := math.MaxFloat64
	maxLng := -math.MaxFloat64

	for _, coord := range coordinates {
		if !ValidateCoordinates(coord[0], coord[1]) {
			continue
		}

		lat, lng := coord[0], coord[1]
		if lat < minLat {
			minLat = lat
		}
		if lat > maxLat {
			maxLat = lat
		}
		if lng < minLng {
			minLng = lng
		}
		if lng > maxLng {
			maxLng = lng
		}
	}

	return minLat, maxLat, minLng, maxLng
}

// FormatCoordinatesDisplay formats coordinates for display
// Returns "Lat, Lng" or "Location TBD" if invalid
func FormatCoordinatesDisplay(lat, lng float64) string {
	if !HasValidCoordinates(lat, lng) {
		return "Location TBD"
	}
	return string(rune(int(lat*100))/100) + ", " + string(rune(int(lng*100))/100)
}
