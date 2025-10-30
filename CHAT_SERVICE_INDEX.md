# Chat Service Documentation Index

## Quick Start

Start here if you want to understand the streaming chat system quickly:
- **CHAT_SERVICE_QUICK_REFERENCE.md** (6.5 KB) - Focused guide on `textPartIterator` and key methods

Then dive deeper with:
- **CHAT_SERVICE_ANALYSIS.md** (15 KB) - Comprehensive implementation details

---

## What You'll Find

### Quick Reference (CHAT_SERVICE_QUICK_REFERENCE.md)
- `textPartIterator()` method signature and purpose
- Related streaming methods (`sendEvent`, `generatePOIDataStream`, etc.)
- Complete data flow diagram
- Error scenarios
- Integration points
- Quick debugging tips

**Best for:** Understanding how streaming works, quick lookups, debugging

### Comprehensive Analysis (CHAT_SERVICE_ANALYSIS.md)
- Full service structure
- Detailed method implementations with code
- StreamEvent data model
- HTTP handler integration patterns
- Production-grade design patterns
- Performance considerations
- Error handling strategies
- File locations and usage examples

**Best for:** Deep understanding, architecture reviews, implementation guidance

---

## Core Files

| File | Purpose | Size |
|------|---------|------|
| `/app/pkg/domain/chat_prompt/chat_service.go` | Main service implementation | ~4000 lines |
| `/app/pkg/handlers/chat.go` | HTTP handlers | ~1500 lines |
| `/app/internal/models/chat_session.go` | Data models | Includes StreamEvent |
| `/app/pkg/domain/chat_prompt/chat_helpers.go` | Helper functions | Cache keys, JSON cleaning |
| `/app/pkg/domain/chat_prompt/chat_repository.go` | Database layer | Session persistence |

---

## Key Methods Explained

### textPartIterator() - Line 1902
The core streaming iterator that extracts text from AI responses.

**What it does:**
1. Wraps `iter.Seq2[*genai.GenerateContentResponse, error]`
2. Iterates over AI stream responses
3. Extracts text parts from candidates
4. Yields text chunks to caller
5. Handles errors by sending to event channel

**Used by:** `generatePOIDataStream()`

### sendEvent() - Line 1416
Reliable event delivery with retry, timeout, and dead-letter queue.

**Features:**
- 3 retry attempts
- 100ms backoff between retries
- 2-second timeout per send
- Dead letter queue fallback
- Context cancellation support

**Used by:** All service methods for streaming events

### generatePOIDataStream() - Line 1931
Streams POI (Point of Interest) details from AI.

**Pattern:**
1. Create AI stream via `GenerateContentStream()`
2. Iterate with `textPartIterator()`
3. Emit `poi_detail_chunk` for each text chunk
4. Buffer full response
5. Persist to database

### ContinueSessionStreamed() - Line 1491
Main orchestrator for session-based chat.

**Responsibilities:**
1. Load and validate session
2. Add user message to history
3. Classify intent
4. Generate semantic POI recommendations
5. Route to appropriate handler
6. Stream events back to client

### ProcessUnifiedChatMessageStream() - Line 2238
Entry point for authenticated users with unified domain detection.

---

## Data Flow

```
Browser
  ↓
HTTP POST /stream/chat
  ↓
Handler (chat.go)
  ├─ Create eventCh := make(chan models.StreamEvent, 200)
  ├─ Launch service goroutine
  └─ Loop: receive events → JSON → SSE → flush
    
Service (chat_service.go)
  ├─ ProcessUnifiedChatMessageStream()
  ├─ ContinueSessionStreamed()
  ├─ generatePOIDataStream()
  │   ├─ AIClient.GenerateContentStream() → iter.Seq2
  │   ├─ textPartIterator(iter) → iter.Seq[string]
  │   ├─ for chunk := range textPartIterator()
  │   │   ├─ sendEvent(poi_detail_chunk)
  │   │   └─ Append to buffer
  │   └─ sendEvent(EventTypeComplete)
  └─ Repository.Save()

Database
  └─ Persist interaction
```

---

## Event Types in Streaming

| Type | Emitted By | Contains |
|------|-----------|----------|
| `EventTypeProgress` | Service setup | Status message |
| `EventTypeError` | textPartIterator, sendEvent | Error details |
| `poi_detail_chunk` | generatePOIDataStream | POI name + text chunk |
| `intent_classified` | ContinueSessionStreamed | Detected intent |
| `semantic_context_generated` | ContinueSessionStreamed | POI recommendations |
| `EventTypeComplete` | Service finish | Full response data |

---

## Error Handling Patterns

### Stream Error
AI client returns error → textPartIterator detects → sendEvent(ERROR) → stops iteration

### Channel Timeout
sendEvent() attempts send → 2s timeout → retry up to 3x → drop to dead letter queue

### Context Cancellation
User disconnects → Context.Done() → service stops → handler exits

### Slow Consumer
Handler loop blocked → sendEvent timeout → dead letter queue → logged

---

## Design Principles

1. **Iterator-based Streaming:** Go 1.22+ `iter.Seq` for clean consumption
2. **Buffered Channels:** Prevent goroutine blocking with 200-element buffers
3. **Graceful Degradation:** Dead letter queue captures failures
4. **Context Propagation:** Full cancellation and deadline support
5. **Observable:** OpenTelemetry tracing, comprehensive logging
6. **Type-Safe:** Generics throughout for compile-time safety

---

## Performance Tips

1. **Reduce Latency:**
   - Chunks sent immediately (not buffered)
   - Client can render progressively
   - Use `poi_detail_chunk` events for incremental UI updates

2. **Prevent Blocking:**
   - Use buffered channels (200 buffer)
   - Monitor "Dropped stream event" logs
   - Tune based on peak throughput

3. **Reliability:**
   - Dead letter queue monitors failures
   - Retry with backoff prevents thundering herd
   - Timeout protection prevents cascade failures

---

## Testing Strategy

### Unit Tests
Test `textPartIterator()` with mock `iter.Seq2`:
```go
mockIter := mockIterator([]string{"chunk1", "chunk2"})
results := collectResults(l.textPartIterator(ctx, mockIter, eventCh, "POI"))
```

### Integration Tests
Test `generatePOIDataStream()` with AI client mock

### E2E Tests
Test full `ProcessUnifiedChatMessageStream()` with real session

Location: `/app/pkg/domain/chat_prompt/chat_service_test.go`

---

## Related Documentation

### In Project
- `CLAUDE.md` - Project guidelines
- `Tasks.md` - Current task tracking
- Go templ docs - For template rendering (separate concern)

### External References
- [Go Iterator Pattern](https://go.dev/wiki/rangefunc) - iter.Seq details
- [Google GenAI SDK](https://github.com/FACorreiaa/go-genai-sdk) - AI client
- [Context Package](https://pkg.go.dev/context) - Deadline/cancellation

---

## Quick Navigation

**Trying to understand:**
- How streaming works? → Quick Reference → Data Flow diagram
- Why an event failed? → Quick Reference → Error Scenarios
- How to integrate new endpoint? → Analysis → HTTP Handler Integration
- How to debug? → Quick Reference → Quick Debugging section
- Complete picture? → Analysis (full 15KB document)

---

## File Checksums

For verification:
- CHAT_SERVICE_ANALYSIS.md: ~530 lines, 15 KB
- CHAT_SERVICE_QUICK_REFERENCE.md: ~300 lines, 6.5 KB
- CHAT_SERVICE_INDEX.md: This file, ~250 lines

---

## Version Info

- **Go Version:** 1.22+ (uses iter.Seq)
- **Last Updated:** October 30, 2024
- **Implementation Status:** Production-grade
- **Test Coverage:** See chat_service_test.go

---

## Need More Help?

1. Check the appropriate documentation above
2. Review code comments in chat_service.go
3. Look at chat_service_test.go for usage examples
4. Check git history for context on recent changes
5. Review CLAUDE.md for project guidelines

