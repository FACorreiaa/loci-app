package handlers

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	streamingfeatures "github.com/FACorreiaa/go-templui/app/internal/features/streaming"
	"github.com/FACorreiaa/go-templui/app/internal/models"
	llmchat "github.com/FACorreiaa/go-templui/app/pkg/domain/chat_prompt"
	"github.com/FACorreiaa/go-templui/app/pkg/services"
	streamingpkg "github.com/FACorreiaa/go-templui/app/pkg/streaming"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// UnifiedStreamingHandlers handles all streaming requests
type UnifiedStreamingHandlers struct {
	llmService       llmchat.LlmInteractiontService
	itineraryService *services.ItineraryService
	logger           *slog.Logger
	streamManager    *streamingpkg.StreamManager
}

// NewUnifiedStreamingHandlers creates a new unified streaming handler
func NewUnifiedStreamingHandlers(
	llmService llmchat.LlmInteractiontService,
	itineraryService *services.ItineraryService,
	logger *slog.Logger,
) *UnifiedStreamingHandlers {
	return &UnifiedStreamingHandlers{
		llmService:       llmService,
		itineraryService: itineraryService,
		logger:           logger,
		streamManager:    streamingpkg.GlobalStreamManager,
	}
}

// HandleGenerateStream starts the streaming process for any request type
func (h *UnifiedStreamingHandlers) HandleGenerateStream(c *gin.Context) {
	ctx, span := otel.Tracer("UnifiedStreamingHandlers").Start(
		c.Request.Context(),
		"HandleGenerateStream",
		trace.WithAttributes(),
	)
	defer span.End()

	// Parse the streaming request
	var req streamingpkg.StreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.ErrorContext(ctx, "Failed to parse streaming request", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Generate unique session ID
	sessionID := uuid.New().String()
	req.SessionID = sessionID

	h.logger.InfoContext(ctx, "Starting unified streaming request",
		slog.String("sessionId", sessionID),
		slog.String("requestType", string(req.RequestType)),
		slog.String("query", req.Query))

	// Start the streaming process in background
	go h.processStreamingRequest(ctx, req)

	// Return the streaming placeholder component
	component := streamingfeatures.StreamingPlaceholder(sessionID, string(req.RequestType), req.Query)
	c.HTML(http.StatusOK, "", component)
}

// HandleStreamEvents serves the SSE endpoint for streaming events
func (h *UnifiedStreamingHandlers) HandleStreamEvents(c *gin.Context) {
	sessionID := c.Param("sessionId")
	
	ctx, span := otel.Tracer("UnifiedStreamingHandlers").Start(
		c.Request.Context(),
		"HandleStreamEvents",
		trace.WithAttributes(),
	)
	defer span.End()

	l := h.logger.With(slog.String("sessionId", sessionID))
	l.InfoContext(ctx, "Client connected to streaming endpoint")

	// Get the streaming channel
	eventCh, found := h.streamManager.GetStream(sessionID)
	if !found {
		l.WarnContext(ctx, "No streaming session found")
		c.Status(http.StatusNotFound)
		return
	}

	// Set SSE headers
	w := c.Writer
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	flusher, ok := w.(http.Flusher)
	if !ok {
		l.ErrorContext(ctx, "Streaming not supported")
		span.SetStatus(codes.Error, "Streaming not supported")
		return
	}

	// Stream events to client
	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				l.InfoContext(ctx, "Event channel closed, ending stream")
				return
			}

			// Render the appropriate component based on event type and request type
			htmlContent, err := h.renderEventAsHTML(ctx, event)
			if err != nil {
				l.ErrorContext(ctx, "Failed to render event as HTML", slog.Any("error", err))
				continue
			}

			// Send the HTML content as SSE
			if htmlContent != "" {
				sseEventType := h.getSSEEventType(event)
				fmt.Fprintf(w, "event: %s\n", sseEventType)
				fmt.Fprintf(w, "data: %s\n\n", htmlContent)
				flusher.Flush()
			}

			// End stream if this is a final event
			if event.IsFinal {
				l.InfoContext(ctx, "Stream completed", slog.String("eventType", event.Type))
				return
			}

		case <-c.Request.Context().Done():
			l.InfoContext(ctx, "Client disconnected")
			return
		}
	}
}

// processStreamingRequest handles the actual streaming logic in background
func (h *UnifiedStreamingHandlers) processStreamingRequest(ctx context.Context, req streamingpkg.StreamRequest) {
	l := h.logger.With(slog.String("sessionId", req.SessionID))
	
	// Create the streaming channel
	eventCh := h.streamManager.CreateStream(req.SessionID, req.RequestType)
	defer h.streamManager.CloseStream(req.SessionID)

	// Send initial progress event
	progressEvent := streamingpkg.NewProgressEvent(req.SessionID, req.RequestType, "Starting your request...")
	streamingpkg.SendEventSafe(ctx, eventCh, progressEvent, time.Second*2)

	// Route to appropriate handler based on request type
	switch req.RequestType {
	case streamingpkg.RequestTypeItinerary:
		h.processItineraryRequest(ctx, req, eventCh)
	case streamingpkg.RequestTypeHotels:
		h.processHotelsRequest(ctx, req, eventCh)
	case streamingpkg.RequestTypeRestaurants:
		h.processRestaurantsRequest(ctx, req, eventCh)
	case streamingpkg.RequestTypeActivities:
		h.processActivitiesRequest(ctx, req, eventCh)
	default:
		errorEvent := streamingpkg.NewErrorEvent(req.SessionID, req.RequestType, 
			fmt.Errorf("unsupported request type: %s", req.RequestType))
		streamingpkg.SendEventSafe(ctx, eventCh, errorEvent, time.Second*2)
		return
	}

	l.InfoContext(ctx, "Streaming request completed")
}

// processItineraryRequest handles itinerary-specific streaming
func (h *UnifiedStreamingHandlers) processItineraryRequest(ctx context.Context, req streamingpkg.StreamRequest, eventCh chan streamingpkg.UnifiedStreamEvent) {
	// Create a traditional StreamEvent channel to interface with existing service
	legacyEventCh := make(chan models.StreamEvent, 100)
	
	// Start the LLM service in background
	go func() {
		defer close(legacyEventCh)
		err := h.llmService.ProcessUnifiedChatMessageStreamFree(
			ctx, req.CityName, req.Query, req.UserLocation, legacyEventCh,
		)
		if err != nil {
			h.logger.ErrorContext(ctx, "LLM service error", slog.Any("error", err))
		}
	}()

	// Convert legacy events to unified events
	for legacyEvent := range legacyEventCh {
		unifiedEvent := h.convertLegacyEvent(legacyEvent, req)
		streamingpkg.SendEventSafe(ctx, eventCh, unifiedEvent, time.Second*2)
		
		if legacyEvent.IsFinal {
			// Send completion event with navigation
			completeEvent := streamingpkg.NewCompleteEvent(req.SessionID, req.RequestType, 
				fmt.Sprintf("/itinerary?sessionId=%s", req.SessionID))
			streamingpkg.SendEventSafe(ctx, eventCh, completeEvent, time.Second*2)
			break
		}
	}
}

// processHotelsRequest handles hotel-specific streaming
func (h *UnifiedStreamingHandlers) processHotelsRequest(ctx context.Context, req streamingpkg.StreamRequest, eventCh chan streamingpkg.UnifiedStreamEvent) {
	// Simulate hotel processing (replace with actual hotel service call)
	progressEvent := streamingpkg.NewProgressEvent(req.SessionID, req.RequestType, "Searching for hotels...")
	streamingpkg.SendEventSafe(ctx, eventCh, progressEvent, time.Second*2)

	// Here you would call your hotel service
	// For now, we'll use the LLM service with hotel domain
	legacyEventCh := make(chan models.StreamEvent, 100)
	
	go func() {
		defer close(legacyEventCh)
		// Call with hotel domain
		err := h.llmService.ProcessUnifiedChatMessageStreamFree(
			ctx, req.CityName, req.Query+" hotels accommodation", req.UserLocation, legacyEventCh,
		)
		if err != nil {
			h.logger.ErrorContext(ctx, "Hotel service error", slog.Any("error", err))
		}
	}()

	for legacyEvent := range legacyEventCh {
		unifiedEvent := h.convertLegacyEvent(legacyEvent, req)
		streamingpkg.SendEventSafe(ctx, eventCh, unifiedEvent, time.Second*2)
		
		if legacyEvent.IsFinal {
			completeEvent := streamingpkg.NewCompleteEvent(req.SessionID, req.RequestType, 
				fmt.Sprintf("/hotels?sessionId=%s", req.SessionID))
			streamingpkg.SendEventSafe(ctx, eventCh, completeEvent, time.Second*2)
			break
		}
	}
}

// processRestaurantsRequest handles restaurant-specific streaming
func (h *UnifiedStreamingHandlers) processRestaurantsRequest(ctx context.Context, req streamingpkg.StreamRequest, eventCh chan streamingpkg.UnifiedStreamEvent) {
	progressEvent := streamingpkg.NewProgressEvent(req.SessionID, req.RequestType, "Finding great restaurants...")
	streamingpkg.SendEventSafe(ctx, eventCh, progressEvent, time.Second*2)

	legacyEventCh := make(chan models.StreamEvent, 100)
	
	go func() {
		defer close(legacyEventCh)
		err := h.llmService.ProcessUnifiedChatMessageStreamFree(
			ctx, req.CityName, req.Query+" restaurants dining food", req.UserLocation, legacyEventCh,
		)
		if err != nil {
			h.logger.ErrorContext(ctx, "Restaurant service error", slog.Any("error", err))
		}
	}()

	for legacyEvent := range legacyEventCh {
		unifiedEvent := h.convertLegacyEvent(legacyEvent, req)
		streamingpkg.SendEventSafe(ctx, eventCh, unifiedEvent, time.Second*2)
		
		if legacyEvent.IsFinal {
			completeEvent := streamingpkg.NewCompleteEvent(req.SessionID, req.RequestType, 
				fmt.Sprintf("/restaurants?sessionId=%s", req.SessionID))
			streamingpkg.SendEventSafe(ctx, eventCh, completeEvent, time.Second*2)
			break
		}
	}
}

// processActivitiesRequest handles activity-specific streaming  
func (h *UnifiedStreamingHandlers) processActivitiesRequest(ctx context.Context, req streamingpkg.StreamRequest, eventCh chan streamingpkg.UnifiedStreamEvent) {
	progressEvent := streamingpkg.NewProgressEvent(req.SessionID, req.RequestType, "Discovering exciting activities...")
	streamingpkg.SendEventSafe(ctx, eventCh, progressEvent, time.Second*2)

	legacyEventCh := make(chan models.StreamEvent, 100)
	
	go func() {
		defer close(legacyEventCh)
		err := h.llmService.ProcessUnifiedChatMessageStreamFree(
			ctx, req.CityName, req.Query+" activities attractions things to do", req.UserLocation, legacyEventCh,
		)
		if err != nil {
			h.logger.ErrorContext(ctx, "Activities service error", slog.Any("error", err))
		}
	}()

	for legacyEvent := range legacyEventCh {
		unifiedEvent := h.convertLegacyEvent(legacyEvent, req)
		streamingpkg.SendEventSafe(ctx, eventCh, unifiedEvent, time.Second*2)
		
		if legacyEvent.IsFinal {
			completeEvent := streamingpkg.NewCompleteEvent(req.SessionID, req.RequestType, 
				fmt.Sprintf("/activities?sessionId=%s", req.SessionID))
			streamingpkg.SendEventSafe(ctx, eventCh, completeEvent, time.Second*2)
			break
		}
	}
}

// convertLegacyEvent converts existing StreamEvent to UnifiedStreamEvent
func (h *UnifiedStreamingHandlers) convertLegacyEvent(legacyEvent models.StreamEvent, req streamingpkg.StreamRequest) streamingpkg.UnifiedStreamEvent {
	unifiedEvent := streamingpkg.UnifiedStreamEvent{
		Type:        legacyEvent.Type,
		RequestType: req.RequestType,
		SessionID:   req.SessionID,
		Message:     legacyEvent.Message,
		Timestamp:   legacyEvent.Timestamp,
		EventID:     legacyEvent.EventID,
		IsFinal:     legacyEvent.IsFinal,
		Error:       legacyEvent.Error,
		Data:        legacyEvent.Data,
	}

	// Map navigation data if present
	if legacyEvent.Navigation != nil {
		unifiedEvent.Navigation = legacyEvent.Navigation
	}

	return unifiedEvent
}

// renderEventAsHTML renders streaming events as HTML components
func (h *UnifiedStreamingHandlers) renderEventAsHTML(ctx context.Context, event streamingpkg.UnifiedStreamEvent) (string, error) {
	var component interface{}
	var buffer bytes.Buffer

	switch event.Type {
	case models.EventTypeProgress:
		component = streamingfeatures.ProgressMessage(event.Message, event.Timestamp.Format("15:04:05"))
		
	case models.EventTypeError:
		component = streamingfeatures.ErrorMessage(event.Error, event.SessionID)
		
	case models.EventTypeComplete:
		navURL := ""
		if event.Navigation != nil {
			navURL = event.Navigation.URL
		}
		component = streamingfeatures.CompletionActions(string(event.RequestType), event.SessionID, navURL)
		
	case models.EventTypeCityData:
		if cityData, ok := event.Data.(*models.GeneralCityData); ok && cityData != nil {
			component = streamingfeatures.StreamingCityInfo(*cityData, event.SessionID)
		}
		
	case models.EventTypePersonalizedPOI, models.EventTypeItinerary:
		// Handle POI data based on request type
		if pois, ok := event.Data.([]models.POIDetailedInfo); ok {
			return h.renderPOIsByType(ctx, pois, event.RequestType, event.SessionID)
		}
		
	default:
		// Skip unknown event types
		return "", nil
	}

	if component != nil {
		// Use reflection or type assertion to render the component
		// This is a simplified approach - you may need to adjust based on your templ setup
		if renderable, ok := component.(interface{ Render(context.Context, *bytes.Buffer) error }); ok {
			if err := renderable.Render(ctx, &buffer); err != nil {
				return "", fmt.Errorf("failed to render component: %w", err)
			}
			return buffer.String(), nil
		}
	}

	return "", nil
}

// renderPOIsByType renders POIs based on the request type
func (h *UnifiedStreamingHandlers) renderPOIsByType(ctx context.Context, pois []models.POIDetailedInfo, requestType streamingpkg.RequestType, sessionID string) (string, error) {
	var htmlParts []string
	
	for i, poi := range pois {
		var component interface{}
		var buffer bytes.Buffer
		
		switch requestType {
		case streamingpkg.RequestTypeItinerary:
			component = streamingfeatures.StreamingItineraryCard(poi, i+1, sessionID)
		case streamingpkg.RequestTypeHotels:
			component = streamingfeatures.StreamingHotelCard(poi, sessionID)
		case streamingpkg.RequestTypeRestaurants:
			component = streamingfeatures.StreamingRestaurantCard(poi, sessionID)
		case streamingpkg.RequestTypeActivities:
			component = streamingfeatures.StreamingActivityCard(poi, sessionID)
		}
		
		if component != nil {
			if renderable, ok := component.(interface{ Render(context.Context, *bytes.Buffer) error }); ok {
				if err := renderable.Render(ctx, &buffer); err != nil {
					h.logger.ErrorContext(ctx, "Failed to render POI component", slog.Any("error", err))
					continue
				}
				htmlParts = append(htmlParts, buffer.String())
			}
		}
	}
	
	return fmt.Sprintf("<div>%s</div>", fmt.Sprintf("%v", htmlParts)), nil
}

// getSSEEventType maps event types to SSE event names
func (h *UnifiedStreamingHandlers) getSSEEventType(event streamingpkg.UnifiedStreamEvent) string {
	switch event.Type {
	case models.EventTypeProgress:
		return "progress"
	case models.EventTypeError:
		return "error"
	case models.EventTypeComplete:
		return "complete"
	case models.EventTypeCityData:
		return "city_data"
	default:
		return "message"
	}
}

// HandleRetryStream handles retry requests
func (h *UnifiedStreamingHandlers) HandleRetryStream(c *gin.Context) {
	sessionID := c.Param("sessionId")
	
	// Close existing stream if any
	h.streamManager.CloseStream(sessionID)
	
	// Return an error message or redirect to restart
	c.JSON(http.StatusOK, gin.H{
		"message": "Please refresh the page to try again",
		"action":  "reload",
	})
}