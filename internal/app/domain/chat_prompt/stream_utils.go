package llmchat

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/genai"

	"github.com/FACorreiaa/go-templui/internal/app/models"
)

// StreamProcessor handles common streaming operations for AI responses
type StreamProcessor struct {
	logger *slog.Logger
}

// NewStreamProcessor creates a new stream processor
func NewStreamProcessor(logger *slog.Logger) *StreamProcessor {
	return &StreamProcessor{logger: logger}
}

// TextPartIterator yields text parts from the AI stream iterator, handling errors inline.
// It uses Go's iter.Seq2 for clean iteration and can be reused across different streaming contexts.
//
// Parameters:
//   - ctx: Context for cancellation and tracing
//   - iter: The AI response stream iterator
//   - eventCh: Optional channel to send error events (can be nil)
//   - contextName: Name for error messages (e.g., "POI generation", "chat response")
//
// Returns:
//   - An iterator that yields string chunks from the AI response
func (sp *StreamProcessor) TextPartIterator(
	ctx context.Context,
	iter iter.Seq2[*genai.GenerateContentResponse, error],
	eventCh chan<- models.StreamEvent,
	contextName string,
) iter.Seq[string] {
	return func(yield func(string) bool) {
		for resp, err := range iter {
			// Handle errors in the stream
			if err != nil {
				sp.logger.ErrorContext(ctx, "Stream error occurred",
					slog.String("context", contextName),
					slog.Any("error", err))

				// Send error event if channel is provided
				if eventCh != nil {
					sp.sendErrorEvent(ctx, eventCh, contextName, err)
				}
				return // Stop iteration on error
			}

			// Extract text from all candidates and parts
			for _, cand := range resp.Candidates {
				if cand.Content != nil {
					for _, part := range cand.Content.Parts {
						if part.Text != "" {
							text := string(part.Text)
							// Yield text to caller, stop if they don't want more
							if !yield(text) {
								return
							}
						}
					}
				}
			}
		}
	}
}

// StreamToString collects all chunks from an iterator into a single string
// This is useful when you need the complete response before processing
func (sp *StreamProcessor) StreamToString(ctx context.Context, iter iter.Seq[string]) string {
	var builder strings.Builder
	for chunk := range iter {
		builder.WriteString(chunk)
	}
	return builder.String()
}

// StreamWithCallback processes each chunk with a callback function
// This allows for custom processing of each chunk (e.g., progress updates, validation)
func (sp *StreamProcessor) StreamWithCallback(
	ctx context.Context,
	iter iter.Seq[string],
	callback func(chunk string) error,
) error {
	for chunk := range iter {
		if err := callback(chunk); err != nil {
			sp.logger.ErrorContext(ctx, "Callback error in stream processing", slog.Any("error", err))
			return err
		}
	}
	return nil
}

// StreamToChannel sends chunks to a channel, useful for concurrent processing
// The channel is NOT closed by this function - caller is responsible for closing it
func (sp *StreamProcessor) StreamToChannel(
	ctx context.Context,
	iter iter.Seq[string],
	outCh chan<- string,
) {
	for chunk := range iter {
		select {
		case <-ctx.Done():
			sp.logger.WarnContext(ctx, "Stream cancelled by context")
			return
		case outCh <- chunk:
			// Chunk sent successfully
		}
	}
}

// StreamWithProgress processes chunks and sends progress events
// Useful for long-running operations where progress feedback is needed
func (sp *StreamProcessor) StreamWithProgress(
	ctx context.Context,
	iter iter.Seq[string],
	eventCh chan<- models.StreamEvent,
	contextName string,
	eventType string,
) string {
	var builder strings.Builder
	chunkCount := 0

	for chunk := range iter {
		builder.WriteString(chunk)
		chunkCount++

		// Send progress event
		if eventCh != nil {
			select {
			case <-ctx.Done():
				return builder.String()
			case eventCh <- models.StreamEvent{
				Type:      eventType,
				Data:      map[string]interface{}{"chunk": chunk, "count": chunkCount},
				Timestamp: time.Now(),
				EventID:   uuid.New().String(),
			}:
			default:
				// Channel full, skip this progress update
			}
		}
	}

	return builder.String()
}

// sendErrorEvent sends an error event to the event channel with retry logic
func (sp *StreamProcessor) sendErrorEvent(
	ctx context.Context,
	eventCh chan<- models.StreamEvent,
	contextName string,
	err error,
) {
	event := models.StreamEvent{
		Type:      models.EventTypeError,
		Error:     fmt.Sprintf("Streaming failed for %s: %v", contextName, err),
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
	}

	// Try to send with timeout to avoid blocking
	select {
	case <-ctx.Done():
		sp.logger.WarnContext(ctx, "Context cancelled before sending error event")
		return
	case eventCh <- event:
		// Event sent successfully
	case <-time.After(5 * time.Second):
		sp.logger.WarnContext(ctx, "Timeout sending error event", slog.String("context", contextName))
	}
}

// BufferedStreamProcessor processes stream chunks in batches for better performance
type BufferedStreamProcessor struct {
	sp         *StreamProcessor
	bufferSize int
}

// NewBufferedStreamProcessor creates a buffered stream processor
func NewBufferedStreamProcessor(logger *slog.Logger, bufferSize int) *BufferedStreamProcessor {
	return &BufferedStreamProcessor{
		sp:         NewStreamProcessor(logger),
		bufferSize: bufferSize,
	}
}

// ProcessInBatches processes stream chunks in batches, calling the callback for each batch
// This can improve performance when processing many small chunks
func (bsp *BufferedStreamProcessor) ProcessInBatches(
	ctx context.Context,
	iter iter.Seq[string],
	callback func(batch []string) error,
) error {
	batch := make([]string, 0, bsp.bufferSize)

	for chunk := range iter {
		batch = append(batch, chunk)

		// Process when batch is full
		if len(batch) >= bsp.bufferSize {
			if err := callback(batch); err != nil {
				bsp.sp.logger.ErrorContext(ctx, "Batch processing error", slog.Any("error", err))
				return err
			}
			batch = batch[:0] // Clear batch, reuse underlying array
		}
	}

	// Process remaining items
	if len(batch) > 0 {
		if err := callback(batch); err != nil {
			bsp.sp.logger.ErrorContext(ctx, "Final batch processing error", slog.Any("error", err))
			return err
		}
	}

	return nil
}
