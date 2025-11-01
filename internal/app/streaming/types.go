package streaming

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/FACorreiaa/go-templui/internal/app/models"
)

// RequestType represents the type of content being requested
type RequestType string

const (
	RequestTypeItinerary   RequestType = "itinerary"
	RequestTypeHotels      RequestType = "hotels"
	RequestTypeRestaurants RequestType = "restaurants"
	RequestTypeActivities  RequestType = "activities"
)

// StreamRequest contains the unified request data for streaming
type StreamRequest struct {
	Query        string               `json:"query"`
	RequestType  RequestType          `json:"request_type"`
	SessionID    string               `json:"session_id"`
	UserLocation *models.UserLocation `json:"user_location,omitempty"`
	Domain       models.DomainType    `json:"domain"`
	CityName     string               `json:"city_name,omitempty"`
	// Add any additional filters or parameters
	Filters map[string]interface{} `json:"filters,omitempty"`
}

// StreamEvent represents a unified streaming event that can handle any content type
type UnifiedStreamEvent struct {
	Type        string      `json:"type"`
	RequestType RequestType `json:"request_type"`
	Message     string      `json:"message,omitempty"`
	SessionID   string      `json:"session_id"`
	Timestamp   time.Time   `json:"timestamp"`
	EventID     string      `json:"event_id"`
	IsFinal     bool        `json:"is_final,omitempty"`

	// Content-specific data
	CityData    *models.GeneralCityData     `json:"city_data,omitempty"`
	POIs        []models.POIDetailedInfo    `json:"pois,omitempty"`
	Hotels      []models.POIDetailedInfo    `json:"hotels,omitempty"`
	Restaurants []models.POIDetailedInfo    `json:"restaurants,omitempty"`
	Activities  []models.POIDetailedInfo    `json:"activities,omitempty"`
	Itinerary   *models.AIItineraryResponse `json:"itinerary,omitempty"`

	// Generic data for extensibility
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`

	// Navigation data for redirects
	Navigation *models.NavigationData `json:"navigation,omitempty"`
}

// StreamChannel holds the channel and metadata for a streaming session
type StreamChannel struct {
	Channel     chan UnifiedStreamEvent
	CreatedAt   time.Time
	RequestType RequestType
	SessionID   string
}

// StreamManager manages all active streaming sessions
type StreamManager struct {
	channels map[string]*StreamChannel
	mutex    sync.RWMutex
}

// NewStreamManager creates a new stream manager
func NewStreamManager() *StreamManager {
	return &StreamManager{
		channels: make(map[string]*StreamChannel),
	}
}

// CreateStream creates a new streaming channel for a session
func (sm *StreamManager) CreateStream(sessionID string, requestType RequestType) chan UnifiedStreamEvent {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	ch := make(chan UnifiedStreamEvent, 100)
	sm.channels[sessionID] = &StreamChannel{
		Channel:     ch,
		CreatedAt:   time.Now(),
		RequestType: requestType,
		SessionID:   sessionID,
	}

	return ch
}

// GetStream retrieves a streaming channel for a session
func (sm *StreamManager) GetStream(sessionID string) (chan UnifiedStreamEvent, bool) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	streamChan, exists := sm.channels[sessionID]
	if !exists {
		return nil, false
	}

	return streamChan.Channel, true
}

// CloseStream closes and removes a streaming channel
func (sm *StreamManager) CloseStream(sessionID string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if streamChan, exists := sm.channels[sessionID]; exists {
		close(streamChan.Channel)
		delete(sm.channels, sessionID)
	}
}

// CleanupExpiredStreams removes streams older than the specified duration
func (sm *StreamManager) CleanupExpiredStreams(maxAge time.Duration) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for sessionID, streamChan := range sm.channels {
		if streamChan.CreatedAt.Before(cutoff) {
			close(streamChan.Channel)
			delete(sm.channels, sessionID)
		}
	}
}

// Global stream manager instance
var GlobalStreamManager = NewStreamManager()

// Utility functions for creating events
func NewProgressEvent(sessionID string, requestType RequestType, message string) UnifiedStreamEvent {
	return UnifiedStreamEvent{
		Type:        models.EventTypeProgress,
		RequestType: requestType,
		SessionID:   sessionID,
		Message:     message,
		Timestamp:   time.Now(),
		EventID:     uuid.New().String(),
	}
}

func NewDataEvent(sessionID string, requestType RequestType, data interface{}) UnifiedStreamEvent {
	event := UnifiedStreamEvent{
		Type:        getDataEventType(requestType),
		RequestType: requestType,
		SessionID:   sessionID,
		Timestamp:   time.Now(),
		EventID:     uuid.New().String(),
		Data:        data,
	}

	// Type-specific data mapping
	switch requestType {
	case RequestTypeItinerary:
		if itinerary, ok := data.(*models.AIItineraryResponse); ok {
			event.Itinerary = itinerary
		} else if pois, ok := data.([]models.POIDetailedInfo); ok {
			event.POIs = pois
		}
	case RequestTypeHotels:
		if hotels, ok := data.([]models.POIDetailedInfo); ok {
			event.Hotels = hotels
		}
	case RequestTypeRestaurants:
		if restaurants, ok := data.([]models.POIDetailedInfo); ok {
			event.Restaurants = restaurants
		}
	case RequestTypeActivities:
		if activities, ok := data.([]models.POIDetailedInfo); ok {
			event.Activities = activities
		}
	}

	return event
}

func NewErrorEvent(sessionID string, requestType RequestType, err error) UnifiedStreamEvent {
	return UnifiedStreamEvent{
		Type:        models.EventTypeError,
		RequestType: requestType,
		SessionID:   sessionID,
		Error:       err.Error(),
		Timestamp:   time.Now(),
		EventID:     uuid.New().String(),
		IsFinal:     true,
	}
}

func NewCompleteEvent(sessionID string, requestType RequestType, navigationURL string) UnifiedStreamEvent {
	event := UnifiedStreamEvent{
		Type:        models.EventTypeComplete,
		RequestType: requestType,
		SessionID:   sessionID,
		Timestamp:   time.Now(),
		EventID:     uuid.New().String(),
		IsFinal:     true,
	}

	if navigationURL != "" {
		event.Navigation = &models.NavigationData{
			URL:       navigationURL,
			RouteType: string(requestType),
			QueryParams: map[string]string{
				"sessionId": sessionID,
			},
		}
	}

	return event
}

func getDataEventType(requestType RequestType) string {
	switch requestType {
	case RequestTypeItinerary:
		return models.EventTypeItinerary
	case RequestTypeHotels:
		return models.EventTypeHotels
	case RequestTypeRestaurants:
		return models.EventTypeRestaurants
	case RequestTypeActivities:
		return models.EventTypePersonalizedPOI // Reuse existing constant
	default:
		return models.EventTypeMessage
	}
}

// Helper to send events safely with timeout
func SendEventSafe(ctx context.Context, ch chan UnifiedStreamEvent, event UnifiedStreamEvent, timeout time.Duration) bool {
	select {
	case ch <- event:
		return true
	case <-time.After(timeout):
		return false
	case <-ctx.Done():
		return false
	}
}
