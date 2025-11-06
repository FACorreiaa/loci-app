# Chat Service Implementation Summary

## Overview

The chat service implements a **streaming AI chat system** with real-time event propagation. It uses Go's `iter.Seq` iterators for clean streaming patterns and manages AI responses through buffered channels with error handling and retry logic.

---

## Core Components

### 1. Service Structure

**Location:** `/Users/fernando_idwell/Projects/Loci/go-templui/app/pkg/domain/llmChat/chat_service.go`

```go
type ServiceImpl struct {
    logger             *zap.Logger
    interestRepo       interests.Repository
    searchProfileRepo  profiles.Repository
    searchProfileSvc   profiles.Service
    tagsRepo           tags.Repository
    aiClient           *generativeAI.LLMChatClient      // AI API client
    embeddingService   *generativeAI.EmbeddingService
    llmInteractionRepo Repository
    cityRepo           city.Repository
    poiRepo            poi.Repository
    cache              *cache.Cache                      // Response caching
    deadLetterCh       chan models.StreamEvent            // Failed event queue
    intentClassifier   IntentClassifier
}
```

---

## Key Methods

### 1. `textPartIterator()` - Core Streaming Iterator

**Location:** Line 1902-1928

```go
func (l *ServiceImpl) textPartIterator(
    ctx context.Context,
    iter iter.Seq2[*genai.GenerateContentResponse, error],
    eventCh chan<- models.StreamEvent,
    poiName string,
) iter.Seq[string] {
    return func(yield func(string) bool) {
        for resp, err := range iter {
            if err != nil {
                // Send error event and stop
                l.sendEvent(ctx, eventCh, models.StreamEvent{
                    Type:      models.EventTypeError,
                    Error:     fmt.Sprintf("Streaming failed for POI '%s': %v", poiName, err),
                    Timestamp: time.Now(),
                    EventID:   uuid.New().String(),
                }, 3)
                return
            }
            
            // Extract text from response candidates
            for _, cand := range resp.Candidates {
                if cand.Content != nil {
                    for _, part := range cand.Content.Parts {
                        if part.Text != "" {
                            text := string(part.Text)
                            if !yield(text) {
                                return // Caller stopped iteration
                            }
                        }
                    }
                }
            }
        }
    }
}
```

**Responsibilities:**
- Wraps AI stream iterator (`iter.Seq2`)
- Handles streaming errors inline
- Extracts text from GenAI response candidates
- Uses Go's iterator protocol for clean consumption
- Respects caller's cancellation signals

---

### 2. `sendEvent()` - Reliable Event Sending

**Location:** Line 1416-1447

```go
func (l *ServiceImpl) sendEvent(
    ctx context.Context,
    ch chan<- models.StreamEvent,
    event models.StreamEvent,
    retries int,
) bool {
    for i := 0; i < retries; i++ {
        if event.EventID == "" {
            event.EventID = uuid.New().String()
        }
        if event.Timestamp.IsZero() {
            event.Timestamp = time.Now()
        }

        select {
        case <-ctx.Done():
            // Context cancelled - send to dead letter
            l.logger.Warn("Context cancelled, not sending stream event")
            l.deadLetterCh <- event
            return false
        default:
            select {
            case ch <- event:
                return true  // Successfully sent
            case <-ctx.Done():
                l.deadLetterCh <- event
                return false
            case <-time.After(2 * time.Second):  // Timeout on slow consumer
                l.logger.Warn("Dropped stream event due to slow consumer")
                l.deadLetterCh <- event
                // Continue retry after backoff
            }
        }
        time.Sleep(100 * time.Millisecond)  // Backoff between retries
    }
    return false
}
```

**Features:**
- **Retry Logic:** Up to 3 attempts with 100ms backoff
- **Timeout Protection:** 2-second timeout per send attempt
- **Context Awareness:** Respects cancellation signals
- **Dead Letter Queue:** Captures failed events
- **Auto-Enrichment:** Adds timestamp and ID if missing

---

### 3. `generatePOIDataStream()` - POI Detail Streaming

**Location:** Line 1931-2010

```go
func (l *ServiceImpl) generatePOIDataStream(
    ctx context.Context,
    poiName, cityName string,
    userLocation *models.UserLocation,
    userID, cityID uuid.UUID,
    eventCh chan<- models.StreamEvent,
) (models.POIDetailedInfo, error) {
    // ... setup and tracing ...
    
    iter, err := l.aiClient.GenerateContentStream(ctx, prompt, config)
    if err != nil {
        l.sendEvent(ctx, eventCh, models.StreamEvent{
            Type:  models.EventTypeError,
            Error: fmt.Sprintf("Failed to generate POI data for '%s': %v", poiName, err),
        }, 3)
        return models.POIDetailedInfo{}, err
    }

    l.sendEvent(ctx, eventCh, models.StreamEvent{
        Type: models.EventTypeProgress,
        Data: map[string]string{"status": fmt.Sprintf("Getting details for %s...", poiName)},
    }, 3)

    // Consume stream with textPartIterator
    var responseTextBuilder strings.Builder
    for chunk := range l.textPartIterator(ctx, iter, eventCh, poiName) {
        responseTextBuilder.WriteString(chunk)
        
        // Emit individual chunks
        l.sendEvent(ctx, eventCh, models.StreamEvent{
            Type: "poi_detail_chunk",
            Data: map[string]string{
                "poi_name": poiName,
                "chunk":    chunk,
            },
        }, 3)
    }

    // Process and persist full response
    fullText := responseTextBuilder.String()
    // ... parsing and saving ...
    
    return poiInfo, nil
}
```

**Flow:**
1. Creates AI stream iterator
2. Sends progress event
3. Iterates chunks via `textPartIterator()`
4. Emits `poi_detail_chunk` for each text part
5. Validates and persists full response

---

### 4. `ContinueSessionStreamed()` - Session Management

**Location:** Line 1491-1700+ (complex orchestration)

**Purpose:** Handle subsequent messages in existing chat sessions with full streaming

**Key Steps:**
1. Fetch and validate session
2. Find city context
3. Add user message to history
4. Classify user intent
5. Generate semantic POI recommendations
6. Route to appropriate handler based on intent
7. Emit progress and result events

**Intent Handling:**
- `IntentAddPOI` → `handleSemanticAddPOIStreamed()`
- `IntentRemovePOI` → `handleSemanticRemovePOI()`
- `IntentAskQuestion` → Direct response
- `replace_poi` → Semantic replacement
- Default → Itinerary modification

---

### 5. `ProcessUnifiedChatMessageStream()` - Unified Chat Handler

**Location:** Line 2238+

**Purpose:** Main entry point for authenticated chat with unified domain detection

**Parameters:**
- `userID`, `profileID`: User context
- `cityName`: Target city (auto-detected if empty)
- `message`: User input
- `userLocation`: For distance-based sorting
- `eventCh`: Output channel for streaming events

---

## Data Structures

### StreamEvent Model

**Location:** `/Users/fernando_idwell/Projects/Loci/go-templui/app/internal/models/chat_session.go`

```go
type StreamEvent struct {
    Type       string          `json:"type"`           // Event classification
    Message    string          `json:"message"`
    Data       interface{}     `json:"data,omitempty"`
    Error      string          `json:"error,omitempty"`
    Timestamp  time.Time       `json:"timestamp"`
    EventID    string          `json:"event_id"`       // Unique identifier
    IsFinal    bool            `json:"is_final,omitempty"`
    Navigation *NavigationData `json:"navigation,omitempty"`
    HTML       string          `json:"html,omitempty"`          // Rendered fragments
    Domain     string          `json:"domain,omitempty"`
    ItemID     string          `json:"item_id,omitempty"`
    ItemData   interface{}     `json:"item_data,omitempty"`
}
```

### Event Types Constants

```go
EventTypeStart           = "start"
EventTypeProgress        = "progress"
EventTypeError           = "error"
EventTypeComplete        = "complete"
EventTypeCityData        = "city_data"
EventTypeGeneralPOI      = "general_poi"
EventTypePersonalizedPOI = "personalized_poi"
EventTypeItinerary       = "itinerary"
EventTypeMessage         = "message"
EventTypeChunk           = "chunk"
EventTypeItemAdded       = "item_added"
EventTypeItemRemoved     = "item_removed"
EventTypeItemUpdated     = "item_updated"
```

---

## HTTP Handler Integration

### Handler Pattern

**Location:** `/Users/fernando_idwell/Projects/Loci/go-templui/app/pkg/handlers/chat.go`

```go
// Create buffered channel for events
eventCh := make(chan models.StreamEvent, 200)

// Call service in goroutine
go func() {
    err := h.llmService.ProcessUnifiedChatMessageStream(
        middleware.CreateContextWithUser(c),
        userID,
        profileID,
        "",           // cityName - auto-detect
        message,
        nil,          // userLocation
        eventCh,
    )
    if err != nil {
        eventCh <- models.StreamEvent{
            Type:  models.EventTypeError,
            Error: err.Error(),
        }
    }
}()

// Stream events to client via SSE
for {
    select {
    case event, ok := <-eventCh:
        if !ok {
            return // Channel closed
        }
        
        eventData, _ := json.Marshal(event)
        fmt.Fprintf(c.Writer, "data: %s\n\n", eventData)
        flusher.Flush()
        
        // End stream on completion or error
        if event.Type == models.EventTypeComplete || 
           event.Type == models.EventTypeError {
            return
        }
    
    case <-c.Request.Context().Done():
        return // Client disconnected
    }
}
```

**Key Features:**
- Goroutine isolation of service logic
- Buffered channel prevents blocking
- Real-time SSE streaming to client
- Graceful shutdown on completion/error
- Context-based cancellation

---

## Streaming Flow Diagram

```
User Input (HTTP)
    ↓
Handler receives request
    ↓
Create buffered eventCh (200 buffer)
    ↓
Launch service in goroutine:
    ├─ ContinueSessionStreamed() / ProcessUnifiedChatMessageStream()
    │   ├─ Fetch session/context
    │   ├─ Classify intent
    │   ├─ sendEvent(EventTypeProgress)
    │   └─ Call intent handler (e.g., handleSemanticAddPOIStreamed)
    │       └─ generatePOIDataStream()
    │           ├─ AIClient.GenerateContentStream()
    │           │   ↓ iter.Seq2[Response, error]
    │           ├─ textPartIterator() loop:
    │           │   ├─ Extract text from resp.Candidates[*].Content.Parts[*].Text
    │           │   ├─ sendEvent(poi_detail_chunk) for each chunk
    │           │   └─ Build full response in buffer
    │           ├─ sendEvent(poi_detail_received)
    │           └─ Persist to database
    │   ├─ sendEvent(intent_result)
    │   └─ sendEvent(EventTypeComplete)
    ↓
Handler receives events:
    ├─ JSON marshal
    ├─ SSE format: "data: {json}\n\n"
    ├─ Flush to client
    └─ Close on terminal event
    ↓
Browser receives SSE events
    └─ Update DOM in real-time
```

---

## Error Handling

### 1. **Stream Iteration Errors**
```go
for resp, err := range iter {
    if err != nil {
        l.sendEvent(ctx, eventCh, models.StreamEvent{
            Type:  models.EventTypeError,
            Error: fmt.Sprintf("Streaming failed for POI '%s': %v", poiName, err),
        }, 3)
        return
    }
}
```

### 2. **Event Sending Failures**
- **Timeout (2s):** Event dropped to dead letter queue
- **Context Cancelled:** Event sent to dead letter queue
- **Retries (3x):** With 100ms backoff between attempts

### 3. **Dead Letter Queue**
```go
func (l *ServiceImpl) processDeadLetterQueue() {
    for event := range l.deadLetterCh {
        l.logger.ErrorContext(context.Background(), 
            "Unprocessed event sent to dead letter queue",
            zap.Any("event", event))
    }
}
```

---

## Key Design Patterns

### 1. **Iterator Protocol**
Uses Go's `iter.Seq` and `iter.Seq2` for composable streaming:
- Clean for/range consumption
- Respects caller cancellation
- Type-safe generics

### 2. **Buffered Channels**
```go
eventCh := make(chan models.StreamEvent, 200)
```
- Prevents goroutine blocking
- Handles bursty event emission
- Configurable per use case

### 3. **Retry with Backoff**
- Deterministic retry count (3)
- Fixed backoff (100ms)
- Timeout protection (2s)

### 4. **Context Propagation**
- Respects deadlines
- Cancellation on client disconnect
- Tracing attributes

### 5. **Telemetry Integration**
Uses OpenTelemetry for:
- Distributed tracing
- Span attributes
- Error recording

---

## Performance Considerations

1. **Streaming Over Buffering:**
   - Chunks sent immediately (poi_detail_chunk)
   - Not waiting for full response
   - Reduces perceived latency

2. **Channel Sizing:**
   - 200-buffer for high-frequency events
   - Prevents goroutine blocking
   - Memory vs. responsiveness tradeoff

3. **Timeout Protection:**
   - 2s per send attempt
   - Prevents cascade failures
   - Dead letter queue captures orphans

4. **Caching:**
   - Itinerary responses cached
   - POI details cached
   - Reduces redundant AI calls

---

## File Locations

| Component | File Path |
|-----------|-----------|
| Service Implementation | `/app/pkg/domain/llmChat/chat_service.go` |
| HTTP Handlers | `/app/pkg/handlers/chat.go` |
| Data Models | `/app/internal/models/chat_session.go` |
| Helpers | `/app/pkg/domain/llmChat/chat_helpers.go` |
| Repository | `/app/pkg/domain/llmChat/chat_repository.go` |
| Tests | `/app/pkg/domain/llmChat/chat_service_test.go` |

---

## Usage Example

```go
// In HTTP handler
eventCh := make(chan models.StreamEvent, 200)

go func() {
    err := llmService.ContinueSessionStreamed(
        ctx,
        sessionID,
        "Add the Eiffel Tower to my itinerary",
        userLocation,
        eventCh,
    )
    if err != nil {
        eventCh <- models.StreamEvent{
            Type:    models.EventTypeError,
            Error:   err.Error(),
            IsFinal: true,
        }
    }
}()

for event := range eventCh {
    fmt.Printf("Event: %+v\n", event)
    // Stream to client...
}
```

---

## Summary

The chat service implements a **production-grade streaming system** for AI-powered chat interactions with:

1. **Clean Iterator Pattern:** `textPartIterator` for composable stream processing
2. **Reliable Event Delivery:** `sendEvent` with retry, timeout, and dead-letter handling
3. **Real-time Streaming:** Chunks emitted immediately via `poi_detail_chunk` events
4. **Error Resilience:** Comprehensive error handling at each layer
5. **Context Awareness:** Full cancellation and deadline support
6. **Observability:** OpenTelemetry tracing throughout

This architecture enables responsive real-time chat while maintaining reliability and error recovery.
