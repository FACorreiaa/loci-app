package mapcomponents

import "os"

// GetMapboxAccessToken returns the Mapbox access token from environment variables
// Users should set MAPBOX_ACCESS_TOKEN environment variable
func GetMapboxAccessToken() string {
	token := os.Getenv("MAPBOX_ACCESS_TOKEN")
	if token == "" {
		// Return a placeholder token for development
		// In production, this should be set via environment variables
		return "pk.eyJ1IjoieW91cm11c2VybmFtZSIsImEiOiJjbHl6eHl6eHl6eHl6In0.your_access_token_here"
	}
	return token
}

// MapboxConfig holds configuration for Mapbox integration
type MapboxConfig struct {
	AccessToken  string
	DefaultStyle string
	DefaultZoom  float64
}

// GetMapboxConfig returns the default Mapbox configuration
func GetMapboxConfig() MapboxConfig {
	return MapboxConfig{
		AccessToken:  GetMapboxAccessToken(),
		DefaultStyle: "mapbox://styles/mapbox/streets-v12",
		DefaultZoom:  12.0,
	}
}