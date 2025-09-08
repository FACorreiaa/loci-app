package results

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// These types mirror the structs provided by the user but in a package-specific way

type RestaurantDetailedInfo struct {
	ID               uuid.UUID `json:"id"`
	City             string    `json:"city"`
	Name             string    `json:"name"`
	Latitude         float64   `json:"latitude"`
	Longitude        float64   `json:"longitude"`
	Category         string    `json:"category"`
	Description      string    `json:"description"`
	Address          *string   `json:"address"`
	Website          *string   `json:"website"`
	PhoneNumber      *string   `json:"phone_number"`
	OpeningHours     *string   `json:"opening_hours"`
	PriceLevel       *string   `json:"price_level"`
	CuisineType      *string   `json:"cuisine_type"`
	Tags             []string  `json:"tags"`
	Images           []string  `json:"images"`
	Rating           float64   `json:"rating"`
	LlmInteractionID uuid.UUID `json:"llm_interaction_id"`
	Err              error     `json:"-"`
}

type HotelDetailedInfo struct {
	ID               uuid.UUID `json:"id"`
	City             string    `json:"city"`
	Name             string    `json:"name"`
	Latitude         float64   `json:"latitude"`
	Longitude        float64   `json:"longitude"`
	Category         string    `json:"category"`
	Description      string    `json:"description"`
	Address          string    `json:"address"`
	PhoneNumber      *string   `json:"phone_number"`
	Website          *string   `json:"website"`
	OpeningHours     *string   `json:"opening_hours"`
	PriceRange       *string   `json:"price_range"`
	Rating           float64   `json:"rating"`
	Tags             []string  `json:"tags"`
	Images           []string  `json:"images"`
	LlmInteractionID uuid.UUID `json:"llm_interaction_id"`
	Err              error     `json:"-"`
}

type POIDetailedInfo struct {
	ID                uuid.UUID       `json:"id"`
	City              string          `json:"city"`
	Name              string          `json:"name"`
	Latitude          float64         `json:"latitude"`
	Longitude         float64         `json:"longitude"`
	Category          string          `json:"category"`
	Description       string          `json:"description"`
	Address           string          `json:"address"`
	Website           string          `json:"website"`
	PhoneNumber       *string         `json:"phone_number"`
	OpeningHours      string          `json:"opening_hours"`
	Rating            float64         `json:"rating"`
	Tags              []string        `json:"tags"`
	Images            []string        `json:"images"`
	LlmInteractionID  uuid.UUID       `json:"llm_interaction_id"`
	Priority          int             `json:"priority"`
	TimeToSpend       string          `json:"time_to_spend"`
	Budget            string          `json:"budget"`
	Distance          *float64        `json:"distance"`
	Err               error           `json:"-"`
}

type AIItineraryResponse struct {
	ItineraryName      string            `json:"itinerary_name"`
	OverallDescription string            `json:"overall_description"`
	PointsOfInterest   []POIDetailedInfo `json:"points_of_interest"`
	Restaurants        []POIDetailedInfo `json:"restaurants,omitempty"`
	Bars               []POIDetailedInfo `json:"bars,omitempty"`
}

// Context types for routing intents
type IntentType string

const (
	IntentModifyItinerary IntentType = "modify_itinerary"
	IntentAddPOI          IntentType = "add_poi"
	IntentRemovePOI       IntentType = "remove_poi"
	IntentAskQuestion     IntentType = "ask_question"
)

type DomainType string

const (
	DomainGeneral        DomainType = "general"
	DomainAccommodation  DomainType = "accommodation"
	DomainDining         DomainType = "dining"
	DomainActivities     DomainType = "activities"
	DomainItinerary      DomainType = "itinerary"
)

// Request types for search functionality
type SearchRequest struct {
	Query      string          `json:"query"`
	CityName   string          `json:"city_name"`
	Domain     DomainType      `json:"domain"`
	Intent     IntentType      `json:"intent"`
	UserID     uuid.UUID       `json:"user_id"`
	SessionID  uuid.UUID       `json:"session_id"`
	Latitude   *float64        `json:"latitude,omitempty"`
	Longitude  *float64        `json:"longitude,omitempty"`
	Filters    json.RawMessage `json:"filters,omitempty"`
}

type SearchResponse struct {
	Domain          DomainType            `json:"domain"`
	Intent          IntentType            `json:"intent"`
	Restaurants     []RestaurantDetailedInfo `json:"restaurants,omitempty"`
	Hotels          []HotelDetailedInfo      `json:"hotels,omitempty"`
	Activities      []POIDetailedInfo        `json:"activities,omitempty"`
	Itinerary       *AIItineraryResponse     `json:"itinerary,omitempty"`
	SessionID       uuid.UUID             `json:"session_id"`
	TotalResults    int                   `json:"total_results"`
	ProcessingTime  time.Duration         `json:"processing_time"`
	Suggestions     []string              `json:"suggestions,omitempty"`
}