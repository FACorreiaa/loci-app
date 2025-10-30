package llmlogging

import (
	"context"

	"github.com/FACorreiaa/go-templui/app/internal/models"
	"github.com/google/uuid"
)

// Repository defines the minimal interface needed for LLM logging
type Repository interface {
	SaveInteraction(ctx context.Context, interaction models.LlmInteraction) (uuid.UUID, error)
}

// LoggingConfig contains configuration for LLM interaction logging
type LoggingConfig struct {
	// Required fields
	UserID    uuid.UUID
	SessionID uuid.UUID
	Intent    string // e.g., "itinerary", "restaurant", "hotel", "discover", "nearby"
	Prompt    string

	// Optional context fields
	CityID     *uuid.UUID
	CityName   string
	SearchType string // e.g., "general", "dining", "accommodation", "activities"
	ProfileID  *uuid.UUID

	// Model configuration
	ModelName   string
	Provider    string // Defaults to "google"
	Temperature *float32
	TopP        *float32
	TopK        *int
	MaxTokens   *int

	// Request metadata
	DeviceType string // e.g., "ios", "android", "web", "desktop"
	Platform   string // e.g., "mobile", "web", "api"
	UserAgent  string

	// Privacy settings
	RedactPII bool // Whether to redact PII from logged prompt/response

	// Streaming settings
	IsStreaming bool

	// Cache settings
	CacheKey string
}

// LLMResponse contains the response data from an LLM call
type LLMResponse struct {
	ResponseText     string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	StatusCode       int    // 200 for success, error codes for failures
	ErrorMessage     string
	CacheHit         bool

	// Streaming metadata (optional)
	StreamChunksCount *int
	StreamDurationMs  *int

	// Raw payloads (optional)
	RequestPayload  interface{}
	ResponsePayload interface{}
}
