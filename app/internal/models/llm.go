package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// LLMInteraction represents an interaction with the LLM (Gemini)
type LLMInteraction struct {
	ID               uuid.UUID       `json:"id" db:"id"`
	UserID           *uuid.UUID      `json:"user_id" db:"user_id"`
	SessionID        string          `json:"session_id" db:"session_id"`
	Prompt           string          `json:"prompt" db:"prompt"`
	RequestPayload   json.RawMessage `json:"request_payload" db:"request_payload"`
	ResponseText     string          `json:"response" db:"response"`
	ResponsePayload  json.RawMessage `json:"response_payload" db:"response_payload"`
	ModelUsed        string          `json:"model_name" db:"model_name"`
	PromptTokens     int             `json:"prompt_tokens" db:"prompt_tokens"`
	CompletionTokens int             `json:"completion_tokens" db:"completion_tokens"`
	TotalTokens      int             `json:"total_tokens" db:"total_tokens"`
	LatencyMs        int             `json:"latency_ms" db:"latency_ms"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
}

// NewLLMInteraction creates a new LLM interaction with default values
func NewLLMInteraction(userID *uuid.UUID, prompt, responseText, modelUsed string) *LLMInteraction {
	return &LLMInteraction{
		ID:           uuid.New(),
		UserID:       userID,
		Prompt:       prompt,
		ResponseText: responseText,
		ModelUsed:    modelUsed,
		CreatedAt:    time.Now(),
	}
}

// ItinerarySSEEvent represents an SSE event for itinerary updates
type ItinerarySSEEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// TrendingDiscovery represents a trending search/discovery
type TrendingDiscovery struct {
	CityName     string `json:"city_name" db:"city_name"`
	SearchCount  int    `json:"search_count" db:"search_count"`
	Emoji        string `json:"emoji" db:"emoji"`
	Category     string `json:"category" db:"category"`
	FirstMessage string `json:"first_message" db:"first_message"`
}

// FeaturedCollection represents a curated collection
type FeaturedCollection struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description" db:"description"`
	Emoji       string    `json:"emoji" db:"emoji"`
	ItemCount   int       `json:"item_count" db:"item_count"`
	Category    string    `json:"category" db:"category"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}
