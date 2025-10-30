# LLM Logging Implementation Guide

## Overview

This document describes the comprehensive LLM logging system implemented for tracking and analyzing all LLM interactions across the application (discover, nearby, chat, itinerary, etc.).

## What Was Implemented

### 1. Database Schema Enhancement (Migration 0035)

Enhanced the `llm_interactions` table with comprehensive tracking fields:

**Core Tracking:**
- `request_id`: UUID for tracing requests across systems
- `status_code`: HTTP status codes (200 = success, 429 = rate limit, etc.)
- `error_message`: Detailed error information if call fails
- `provider`: LLM provider (google, openai, anthropic)

**Intent & Context:**
- `intent`: Query categorization (itinerary, restaurant, hotel, discover, nearby)
- `search_type`: Specific search type (general, dining, accommodation, activities)
- `device_type`: Device (ios, android, web, desktop)
- `platform`: Platform (mobile, web, api)
- `user_agent`: Full user agent string

**Model Parameters:**
- `temperature`, `top_p`, `top_k`, `max_tokens`: For quality correlation analysis

**Cost Tracking:**
- `cost_estimate_usd`: Calculated field (tokens × per-token price)

**User Feedback:**
- `user_feedback_rating`: 1-5 star rating
- `user_feedback_comment`: Text feedback
- `user_feedback_timestamp`: When feedback was given

**Cache Efficiency:**
- `cache_hit`: Whether response was cached
- `cache_key`: Cache key for response

**Privacy & Security:**
- `prompt_hash`: SHA256 hash for anonymized tracking
- `is_pii_redacted`: Flag for PII redaction

**Streaming Metadata:**
- `is_streaming`: Whether response was streamed
- `stream_chunks_count`: Number of chunks
- `stream_duration_ms`: Total streaming duration

### 2. TimescaleDB Integration (Migration 0036)

Enabled time-series optimization for efficient querying:

**Hypertable:**
- Automatic partitioning by `created_at` with 7-day chunks
- Compression of data older than 14 days
- Automatic retention (90 days by default)

**Continuous Aggregates:**
1. `llm_daily_stats_by_intent`: Daily statistics by intent and provider
2. `llm_hourly_performance`: Hourly performance metrics
3. `llm_city_usage_daily`: City-based usage patterns

**Benefits:**
- 10-100x faster queries for time-series data
- Automatic data management (compression, retention)
- Pre-computed aggregations for dashboards

### 3. Enhanced LlmInteraction Model

Updated `/app/internal/models/chat.go` with all new fields while maintaining backward compatibility with duplicate fields.

### 4. LLM Logger Utility

Created `/app/pkg/domain/chat_prompt/llm_logger.go` with:

**Features:**
- Async logging (non-blocking)
- Sync logging (when needed)
- Automatic cost calculation
- Device type detection from user agent
- Prompt hashing for privacy
- Streaming and non-streaming wrappers

**Cost Calculation:**
Automatic cost estimation based on current Gemini pricing:
- `gemini-1.5-pro`: $3.50/$10.50 per 1M tokens (input/output)
- `gemini-1.5-flash`: $0.075/$0.30 per 1M tokens
- `gemini-2.0-flash`: $0.10/$0.40 per 1M tokens

## How to Use

### Basic Usage

```go
// Initialize logger (typically done once in service initialization)
llmLogger := NewLLMLogger(logger, repository)

// Configure logging for your LLM call
config := LoggingConfig{
    UserID:    userID,
    SessionID: sessionID,
    Intent:    "discover",  // or "restaurant", "hotel", "nearby", etc.
    Prompt:    userPrompt,
    CityName:  "Tokyo",
    ModelName: "gemini-1.5-pro",
    SearchType: "general",
}

// Wrap your LLM call
response, err := llmLogger.continue(ctx, config, func() (*genai.GenerateContentResponse, error) {
    return aiClient.GenerateContent(ctx, prompt, config)
})
```

### Streaming Usage

```go
config := LoggingConfig{
    UserID:      userID,
    SessionID:   sessionID,
    Intent:      "itinerary",
    Prompt:      userPrompt,
    IsStreaming: true,
    // ... other fields
}

responseText, err := llmLogger.WrapStreamingCall(ctx, config, func() (string, int, int, int, error) {
    // Your streaming logic here
    // Return: (responseText, promptTokens, completionTokens, totalTokens, error)
    return processStream(ctx, iter)
})
```

### HTTP Request Integration

```go
// Extract config from HTTP request
config := ExtractConfigFromRequest(r, userID, sessionID, "discover")
config.Prompt = userPrompt
config.CityName = cityName
config.ModelName = "gemini-1.5-flash"

// Make your LLM call with logging
response, err := llmLogger.WrapNonStreamingCall(ctx, config, ...)
```

### Manual Logging (for complex cases)

```go
startTime := time.Now()

// Your LLM call
response, err := aiClient.GenerateContent(ctx, prompt)

latencyMs := time.Since(startTime).Milliseconds()

// Build response object
llmResponse := LLMResponse{
    ResponseText:     extractText(response),
    PromptTokens:     int(response.UsageMetadata.PromptTokenCount),
    CompletionTokens: int(response.UsageMetadata.CandidatesTokenCount),
    TotalTokens:      int(response.UsageMetadata.TotalTokenCount),
    StatusCode:       200,
}

if err != nil {
    llmResponse.StatusCode = 500
    llmResponse.ErrorMessage = err.Error()
}

// Log asynchronously (recommended)
llmLogger.LogInteractionAsync(ctx, config, llmResponse, latencyMs)

// Or synchronously if you need to ensure it's saved
err = llmLogger.LogInteractionSync(ctx, config, llmResponse, latencyMs)
```

## Integration Points

### 1. Chat Service

Update `chat_service.go`:

```go
type ServiceImpl struct {
    // ... existing fields
    llmLogger *LLMLogger
}

func NewLlmInteractiontService(...) *ServiceImpl {
    return &ServiceImpl{
        // ... existing initialization
        llmLogger: NewLLMLogger(logger, llmInteractionRepo),
    }
}

// In your LLM call methods
func (l *ServiceImpl) StartChatMessageStream(ctx context.Context, ...) {
    config := LoggingConfig{
        UserID:    userID,
        SessionID: session.ID,
        Intent:    "chat",
        Prompt:    message.Content,
        ModelName: "gemini-1.5-pro",
        // ... other fields
    }

    // Wrap your streaming call
    response, err := l.llmLogger.WrapStreamingCall(ctx, config, func() (string, int, int, int, error) {
        // Your existing streaming logic
        return l.processStream(...)
    })
}
```

### 2. Discover Endpoint

```go
func (h *DiscoverHandlers) Search(c *gin.Context) {
    config := ExtractConfigFromRequest(c.Request, userID, sessionID, "discover")
    config.Prompt = query
    config.CityName = location
    config.SearchType = searchType
    config.ModelName = "gemini-1.5-flash"

    response, err := h.llmLogger.WrapNonStreamingCall(ctx, config, func() (*genai.GenerateContentResponse, error) {
        return h.aiClient.GenerateContent(ctx, prompt)
    })
}
```

### 3. Nearby/Restaurant/Hotel Endpoints

Similar pattern for all LLM-using endpoints - just change the `intent` field:
- `intent: "nearby"` for nearby searches
- `intent: "restaurant"` for restaurant searches
- `intent: "hotel"` for hotel searches
- `intent: "activity"` for activity searches

## Querying Analytics

### Daily Stats by Intent

```sql
SELECT
    day,
    intent,
    total_requests,
    successful_requests,
    failed_requests,
    avg_latency_ms,
    p95_latency_ms,
    total_cost_usd,
    cache_hits,
    cache_misses,
    (cache_hits::float / NULLIF(cache_hits + cache_misses, 0) * 100) as cache_hit_rate,
    avg_user_rating
FROM llm_daily_stats_by_intent
WHERE day >= NOW() - INTERVAL '30 days'
ORDER BY day DESC, total_requests DESC;
```

### Hourly Performance

```sql
SELECT
    hour,
    intent,
    search_type,
    device_type,
    request_count,
    avg_latency_ms,
    max_latency_ms,
    error_count,
    streaming_requests
FROM llm_hourly_performance
WHERE hour >= NOW() - INTERVAL '24 hours'
ORDER BY hour DESC;
```

### City Usage Patterns

```sql
SELECT
    day,
    city_name,
    intent,
    request_count,
    unique_users,
    avg_latency_ms,
    total_tokens_used,
    avg_rating
FROM llm_city_usage_daily
WHERE day >= NOW() - INTERVAL '7 days'
ORDER BY request_count DESC
LIMIT 20;
```

### Cost Analysis

```sql
SELECT
    DATE(created_at) as date,
    intent,
    COUNT(*) as requests,
    SUM(total_tokens) as total_tokens,
    SUM(cost_estimate_usd) as total_cost,
    AVG(cost_estimate_usd) as avg_cost_per_request,
    SUM(cost_estimate_usd) FILTER (WHERE cache_hit = false) as cost_without_cache
FROM llm_interactions
WHERE created_at >= NOW() - INTERVAL '30 days'
GROUP BY date, intent
ORDER BY date DESC, total_cost DESC;
```

### Error Rate by Provider

```sql
SELECT
    provider,
    intent,
    COUNT(*) as total_requests,
    COUNT(*) FILTER (WHERE status_code = 200) as successful,
    COUNT(*) FILTER (WHERE status_code != 200) as failed,
    ROUND(COUNT(*) FILTER (WHERE status_code != 200)::numeric / COUNT(*) * 100, 2) as error_rate_pct,
    AVG(latency_ms) FILTER (WHERE status_code = 200) as avg_success_latency_ms
FROM llm_interactions
WHERE created_at >= NOW() - INTERVAL '7 days'
GROUP BY provider, intent
ORDER BY total_requests DESC;
```

## Prometheus Metrics (Future)

The next step is to add Prometheus metrics for real-time monitoring:

```go
// Metrics to implement
var (
    llmRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "llm_requests_total",
            Help: "Total number of LLM requests",
        },
        []string{"intent", "provider", "status"},
    )

    llmLatencyHistogram = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "llm_latency_seconds",
            Help: "LLM request latency in seconds",
            Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
        },
        []string{"intent", "provider"},
    )

    llmTokensTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "llm_tokens_total",
            Help: "Total number of tokens used",
        },
        []string{"intent", "provider", "type"}, // type = "prompt" or "completion"
    )

    llmCostTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "llm_cost_usd_total",
            Help: "Total cost in USD",
        },
        []string{"intent", "provider"},
    )
)
```

## Performance Considerations

### Async Logging (Recommended)
- Non-blocking: doesn't slow down user requests
- Uses goroutines with context.WithoutCancel()
- Logs errors but doesn't fail the request

### Storage Efficiency
- TimescaleDB automatic compression (14+ days old)
- Automatic retention policy (90 days by default)
- Materialized views reduce query load
- Efficient indexing for common queries

### Privacy
- `prompt_hash` for anonymized tracking
- `is_pii_redacted` flag
- Optional prompt/response redaction

## Next Steps

1. ✅ Database migrations created
2. ✅ Enhanced LlmInteraction model
3. ✅ LLM logger utility implemented
4. ✅ TimescaleDB integration
5. ⏳ Integrate into chat service
6. ⏳ Integrate into discover endpoint
7. ⏳ Integrate into nearby/restaurant/hotel endpoints
8. ⏳ Add Prometheus metrics
9. ⏳ Create Grafana dashboards
10. ⏳ Set up alerts for high error rates/latency

## Migration Instructions

1. **Run migrations:**
   ```bash
   # Ensure PostgreSQL and TimescaleDB are installed
   make migrate-up
   ```

2. **Verify TimescaleDB:**
   ```sql
   SELECT * FROM timescaledb_information.hypertables
   WHERE hypertable_name = 'llm_interactions';
   ```

3. **Check continuous aggregates:**
   ```sql
   SELECT * FROM timescaledb_information.continuous_aggregates;
   ```

4. **Monitor compression:**
   ```sql
   SELECT * FROM timescaledb_information.compression_settings
   WHERE hypertable_name = 'llm_interactions';
   ```

## Troubleshooting

### TimescaleDB Not Found
```bash
# Install TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;
```

### Migration Fails
- Check PostgreSQL version (requires 12+)
- Ensure TimescaleDB extension is installed
- Verify no existing constraints conflict with new columns

### High Async Write Load
- Consider using a write-ahead buffer (Redis, Kafka)
- Increase PostgreSQL max_connections
- Tune PostgreSQL shared_buffers and effective_cache_size

## References

- [Gemini Pricing](https://ai.google.dev/pricing)
- [TimescaleDB Documentation](https://docs.timescale.com/)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/)
