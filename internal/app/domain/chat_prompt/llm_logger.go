package llmchat

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/genai"

	"github.com/FACorreiaa/go-templui/internal/app/models"
	"github.com/FACorreiaa/go-templui/internal/pkg/llmlogging"
)

// LLMLogger handles comprehensive logging of LLM interactions with async support
type LLMLogger struct {
	logger *zap.Logger
	repo   llmlogging.Repository
}

// NewLLMLogger creates a new LLM logger instance
func NewLLMLogger(logger *zap.Logger, repo llmlogging.Repository) *LLMLogger {
	return &LLMLogger{
		logger: logger,
		repo:   repo,
	}
}

// Type aliases for backward compatibility
type LoggingConfig = llmlogging.LoggingConfig
type LLMResponse = llmlogging.LLMResponse

// Pricing for Gemini models (as of 2024)
// Source: https://ai.google.dev/pricing
var geminiPricing = map[string]struct {
	InputPer1M  float64
	OutputPer1M float64
}{
	"gemini-1.5-pro":   {InputPer1M: 3.50, OutputPer1M: 10.50}, // $3.50 per 1M input, $10.50 per 1M output
	"gemini-1.5-flash": {InputPer1M: 0.075, OutputPer1M: 0.30}, // $0.075 per 1M input, $0.30 per 1M output
	"gemini-2.0-flash": {InputPer1M: 0.10, OutputPer1M: 0.40},  // $0.10 per 1M input, $0.40 per 1M output
	"gemini-pro":       {InputPer1M: 0.50, OutputPer1M: 1.50},  // Legacy model
}

// CalculateCost estimates the cost in USD for an LLM interaction
func CalculateCost(modelName string, promptTokens, completionTokens int) float64 {
	// Normalize model name (remove version suffixes, etc.)
	normalizedModel := strings.ToLower(modelName)
	for key := range geminiPricing {
		if strings.Contains(normalizedModel, strings.ToLower(key)) {
			pricing := geminiPricing[key]
			inputCost := (float64(promptTokens) / 1_000_000) * pricing.InputPer1M
			outputCost := (float64(completionTokens) / 1_000_000) * pricing.OutputPer1M
			return inputCost + outputCost
		}
	}
	// Return 0 if model not found in pricing table
	return 0
}

// HashPrompt creates a SHA256 hash of the prompt for anonymized tracking
func HashPrompt(prompt string) string {
	hash := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(hash[:])
}

// DetermineDeviceType extracts device type from user agent
func DetermineDeviceType(userAgent string) string {
	ua := strings.ToLower(userAgent)
	if strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") {
		return "ios"
	}
	if strings.Contains(ua, "android") {
		return "android"
	}
	if strings.Contains(ua, "mobile") {
		return "mobile"
	}
	if strings.Contains(ua, "electron") {
		return "desktop"
	}
	return "web"
}

// LogInteractionAsync logs an LLM interaction asynchronously to avoid blocking the main request
// This is the recommended method for production use
func (l *LLMLogger) LogInteractionAsync(
	ctx context.Context,
	config LoggingConfig,
	response LLMResponse,
	latencyMs int64,
) {
	// Create a new context for async operation to avoid cancellation when request ends
	asyncCtx := context.WithoutCancel(ctx)

	go func() {
		if err := l.logInteraction(asyncCtx, config, response, latencyMs); err != nil {
			l.logger.Error("Failed to log LLM interaction asynchronously",
				zap.String("intent", config.Intent),
				zap.String("session_id", config.SessionID.String()),
				zap.Any("error", err))
		}
	}()
}

// LogInteractionSync logs an LLM interaction synchronously
// Use this only when you need to ensure logging completes before proceeding
func (l *LLMLogger) LogInteractionSync(
	ctx context.Context,
	config LoggingConfig,
	response LLMResponse,
	latencyMs int64,
) error {
	return l.logInteraction(ctx, config, response, latencyMs)
}

// logInteraction is the app implementation of logging
func (l *LLMLogger) logInteraction(
	ctx context.Context,
	config LoggingConfig,
	response LLMResponse,
	latencyMs int64,
) error {
	ctx, span := otel.Tracer("LLMLogger").Start(ctx, "logInteraction",
		trace.WithAttributes(
			attribute.String("intent", config.Intent),
			attribute.String("model", config.ModelName),
			attribute.Int64("latency_ms", latencyMs),
			attribute.Int("status_code", response.StatusCode),
		))
	defer span.End()

	// Calculate cost
	cost := CalculateCost(config.ModelName, response.PromptTokens, response.CompletionTokens)

	// Determine device type if not provided
	deviceType := config.DeviceType
	if deviceType == "" && config.UserAgent != "" {
		deviceType = DetermineDeviceType(config.UserAgent)
	}

	// Set default provider
	provider := config.Provider
	if provider == "" {
		provider = "google"
	}

	// Create interaction record
	interaction := models.LlmInteraction{
		RequestID:         uuid.New(),
		SessionID:         config.SessionID,
		UserID:            config.UserID,
		CityName:          config.CityName,
		CityID:            config.CityID,
		Prompt:            config.Prompt,
		ResponseText:      response.ResponseText,
		ModelUsed:         config.ModelName,
		Provider:          provider,
		PromptTokens:      response.PromptTokens,
		CompletionTokens:  response.CompletionTokens,
		TotalTokens:       response.TotalTokens,
		LatencyMs:         int(latencyMs),
		StatusCode:        response.StatusCode,
		ErrorMessage:      response.ErrorMessage,
		Intent:            config.Intent,
		SearchType:        config.SearchType,
		Temperature:       config.Temperature,
		TopP:              config.TopP,
		TopK:              config.TopK,
		MaxTokens:         config.MaxTokens,
		CostEstimateUSD:   &cost,
		CacheHit:          response.CacheHit,
		CacheKey:          config.CacheKey,
		DeviceType:        deviceType,
		Platform:          config.Platform,
		UserAgent:         config.UserAgent,
		IsPIIRedacted:     config.RedactPII,
		IsStreaming:       config.IsStreaming,
		StreamChunksCount: response.StreamChunksCount,
		StreamDurationMs:  response.StreamDurationMs,
		Timestamp:         time.Now(),
	}

	// Add profile ID if provided
	if config.ProfileID != nil {
		interaction.ProfileID = *config.ProfileID
	}

	// Hash prompt for privacy if requested
	if config.RedactPII {
		interaction.PromptHash = HashPrompt(config.Prompt)
	}

	// Save to database
	savedID, err := l.repo.SaveInteraction(ctx, interaction)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to save LLM interaction: %w", err)
	}

	span.SetAttributes(attribute.String("interaction_id", savedID.String()))
	l.logger.Info("LLM interaction logged successfully",
		zap.String("interaction_id", savedID.String()),
		zap.String("intent", config.Intent),
		zap.Int("prompt_tokens", response.PromptTokens),
		zap.Int("completion_tokens", response.CompletionTokens),
		zap.Float64("cost_usd", cost),
		zap.Int64("latency_ms", latencyMs))

	return nil
}

// ExtractConfigFromRequest extracts logging configuration from an HTTP request
func ExtractConfigFromRequest(r *http.Request, userID, sessionID uuid.UUID, intent string) LoggingConfig {
	// Extract device info from headers
	userAgent := r.UserAgent()
	deviceType := DetermineDeviceType(userAgent)

	// Determine platform
	platform := "web"
	if strings.Contains(strings.ToLower(userAgent), "mobile") {
		platform = "mobile"
	}

	return LoggingConfig{
		UserID:     userID,
		SessionID:  sessionID,
		Intent:     intent,
		DeviceType: deviceType,
		Platform:   platform,
		UserAgent:  userAgent,
		Provider:   "google", // Default to Google/Gemini
	}
}

// WrapStreamingCall wraps a streaming LLM call with comprehensive logging
// Returns the response text and any error
func (l *LLMLogger) WrapStreamingCall(
	ctx context.Context,
	config LoggingConfig,
	streamFunc func() (string, int, int, int, error),
) (string, error) {
	startTime := time.Now()
	config.IsStreaming = true

	responseText, promptTokens, completionTokens, totalTokens, err := streamFunc()
	latencyMs := time.Since(startTime).Milliseconds()

	response := LLMResponse{
		ResponseText:     responseText,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		StatusCode:       200,
	}

	if err != nil {
		response.StatusCode = 500
		response.ErrorMessage = err.Error()
	}

	// Log asynchronously
	l.LogInteractionAsync(ctx, config, response, latencyMs)

	return responseText, err
}

// WrapNonStreamingCall wraps a non-streaming LLM call with comprehensive logging
func (l *LLMLogger) WrapNonStreamingCall(
	ctx context.Context,
	config LoggingConfig,
	callFunc func() (*genai.GenerateContentResponse, error),
) (*genai.GenerateContentResponse, error) {
	startTime := time.Now()
	config.IsStreaming = false

	resp, err := callFunc()
	latencyMs := time.Since(startTime).Milliseconds()

	response := LLMResponse{
		StatusCode: 200,
	}

	if err != nil {
		response.StatusCode = 500
		response.ErrorMessage = err.Error()
	} else if resp != nil && resp.UsageMetadata != nil {
		response.PromptTokens = int(resp.UsageMetadata.PromptTokenCount)
		response.CompletionTokens = int(resp.UsageMetadata.CandidatesTokenCount)
		response.TotalTokens = int(resp.UsageMetadata.TotalTokenCount)

		// Extract response text
		if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
			for _, part := range resp.Candidates[0].Content.Parts {
				if part.Text != "" {
					response.ResponseText += string(part.Text)
				}
			}
		}
	}

	// Log asynchronously
	l.LogInteractionAsync(ctx, config, response, latencyMs)

	return resp, err
}
