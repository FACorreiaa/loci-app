# Chat Service Quick Reference

## textPartIterator Method

**Location:** `/app/pkg/domain/chat_prompt/chat_service.go` (Line 1902)

### Purpose
Core streaming iterator that extracts text from AI-generated responses and yields them for consumption.

### Signature
```go
func (l *ServiceImpl) textPartIterator(
    ctx context.Context,
    iter iter.Seq2[*genai.GenerateContentResponse, error],
    eventCh chan<- models.StreamEvent,
    poiName string,
) iter.Seq[string]
```

### What It Does
1. **Iterates** over AI stream responses using Go's `iter.Seq2` protocol
2. **Handles Errors** by emitting `EventTypeError` to event channel
3. **Extracts Text** from `resp.Candidates[*].Content.Parts[*].Text`
4. **Yields** text chunks one at a time for caller consumption
5. **Respects Cancellation** by checking yield() return value

### Key Code
```go
return func(yield func(string) bool) {
    for resp, err := range iter {
        if err != nil {
            // Error handling
            l.sendEvent(ctx, eventCh, models.StreamEvent{
                Type:  models.EventTypeError,
                Error: fmt.Sprintf("Streaming failed for POI '%s': %v", poiName, err),
            }, 3)
            return
        }
        
        // Extract text from candidates
        for _, cand := range resp.Candidates {
            if cand.Content != nil {
                for _, part := range cand.Content.Parts {
                    if part.Text != "" {
                        if !yield(string(part.Text)) {
                            return  // Caller stopped
                        }
                    }
                }
            }
        }
    }
}
```

---

## Related Streaming Methods

### sendEvent() - Line 1416
**Reliable event delivery with retry, timeout, and dead-letter queue**

- Retries: 3 attempts
- Backoff: 100ms between attempts
- Timeout: 2 seconds per send
- Fallback: Dead letter queue on failure

### generatePOIDataStream() - Line 1931
**Uses textPartIterator to stream POI details**

```go
for chunk := range l.textPartIterator(ctx, iter, eventCh, poiName) {
    responseTextBuilder.WriteString(chunk)
    l.sendEvent(ctx, eventCh, models.StreamEvent{
        Type: "poi_detail_chunk",
        Data: map[string]string{"poi_name": poiName, "chunk": chunk},
    }, 3)
}
```

### ContinueSessionStreamed() - Line 1491
**Main session handler that orchestrates intent-based responses**

Calls → `generatePOIDataStream()` → uses `textPartIterator()`

### ProcessUnifiedChatMessageStream() - Line 2238
**Entry point for authenticated chat with unified domain detection**

---

## Data Flow

```
HTTP Request
    ↓
Handler Creates: eventCh := make(chan models.StreamEvent, 200)
    ↓
Service Goroutine:
    ProcessUnifiedChatMessageStream()
        ↓
    generatePOIDataStream()
        ↓
    aiClient.GenerateContentStream() → iter.Seq2[Response, error]
        ↓
    textPartIterator(iter) → iter.Seq[string]
        ↓
    for chunk := range textPartIterator():
        - sendEvent(poi_detail_chunk)
        - Append to buffer
    ↓
    Persist full response to DB
    ↓
    sendEvent(EventTypeComplete)
    ↓
Handler Loop:
    for event := range eventCh:
        - Marshal to JSON
        - Send via SSE
        - Flush to client
    ↓
Browser receives events → Update DOM
```

---

## StreamEvent Types Used in textPartIterator Path

| Event Type | When | Data |
|-----------|------|------|
| `EventTypeError` | Stream fails | Error message |
| `EventTypeProgress` | Starting POI detail | Status string |
| `poi_detail_chunk` | Each text chunk | `{poi_name, chunk}` |
| `EventTypeComplete` | Final response ready | Full POI info |

---

## Error Scenarios

### Stream Error
```
Generator (AI) → iter error → textPartIterator detects → sendEvent(ERROR) → yield stops
```

### Timeout on Event Send
```
textPartIterator yields chunk → sendEvent timeout (2s) → retry 3x → dead letter queue
```

### Context Cancelled
```
User disconnects → Handler context cancelled → textPartIterator sees ctx.Done() → stops iteration
```

---

## Performance Notes

1. **Streaming Pattern:**
   - Chunks sent to client immediately (no buffering full response)
   - Reduces perceived latency
   - Client can render progressively

2. **Channel Buffers:**
   - 200-buffer accommodates bursty event emission
   - Prevents goroutine blocking
   - Configurable per use case

3. **Iterator Protocol:**
   - Go 1.22+ feature for clean streaming
   - Composable with `for/range`
   - Type-safe compared to channels

---

## Integration Points

### Handler → Service
**File:** `/app/pkg/handlers/chat.go` (Line 939+)

```go
eventCh := make(chan models.StreamEvent, 200)
go func() {
    h.llmService.ProcessUnifiedChatMessageStream(ctx, userID, profileID, "", message, nil, eventCh)
}()
for event := range eventCh {
    // Stream to client...
}
```

### Service → AI Client
**Type:** `generativeAI.LLMChatClient`

```go
iter, err := l.aiClient.GenerateContentStream(ctx, prompt, config)
for chunk := range l.textPartIterator(ctx, iter, eventCh, poiName) {
    // Process chunk...
}
```

---

## Key Dependencies

- `iter.Seq2[*genai.GenerateContentResponse, error]` - AI stream
- `chan<- models.StreamEvent` - Event channel
- `context.Context` - Cancellation/deadline
- `*generativeAI.LLMChatClient` - AI integration
- `Repository` - Database persistence

---

## Testing Entry Points

1. **Unit:** Test `textPartIterator()` directly with mock `iter.Seq2`
2. **Integration:** Test `generatePOIDataStream()` with AI client
3. **E2E:** Test `ProcessUnifiedChatMessageStream()` with full session

See: `/app/pkg/domain/chat_prompt/chat_service_test.go`

---

## Quick Debugging

### Check Event Channel Blocking
- Look for "Dropped stream event due to slow consumer" logs
- Increase buffer size if needed: `make(chan models.StreamEvent, 500)`
- Check handler loop isn't stuck

### Trace Streaming Data
- `sendEvent()` logs all failures
- Check `deadLetterCh` for dropped events
- Handler prints: `fmt.Printf("SSE >> %s\n", eventData)`

### Verify Iterator Consumption
- Add logging in `textPartIterator()` yield loop
- Verify `for chunk := range` loop completes
- Check `ctx.Err()` after loop for cancellation

---

## Related Files

| Purpose | Path |
|---------|------|
| Full implementation | `/app/pkg/domain/chat_prompt/chat_service.go` |
| Handler integration | `/app/pkg/handlers/chat.go` |
| Models | `/app/internal/models/chat_session.go` |
| Helpers | `/app/pkg/domain/chat_prompt/chat_helpers.go` |
| Comprehensive docs | `CHAT_SERVICE_ANALYSIS.md` |

