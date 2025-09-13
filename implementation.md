func (h *HandlerImpl) StartChatMessageStream(w http.ResponseWriter, r *http.Request) {
ctx, span := otel.Tracer("HandlerImpl").Start(r.Context(), "ProcessUnifiedChatMessageStream", trace.WithAttributes(
semconv.HTTPRequestMethodKey.String(r.Method),
semconv.HTTPRouteKey.String("/prompt-response/unified-chat/stream"),
))
defer span.End()

	l := h.logger.With(slog.String("handler", "ProcessUnifiedChatMessageStream"))
	l.DebugContext(ctx, "Processing unified chat message with streaming")

	// Parse profile ID from URL
	profileIDStr := chi.URLParam(r, "profileID")
	profileID, err := uuid.Parse(profileIDStr)
	if err != nil {
		l.ErrorContext(ctx, "Invalid profile ID", slog.String("profileID", profileIDStr), slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid profile ID")
		api.ErrorResponse(w, r, http.StatusBadRequest, "Invalid profile ID")
		return
	}

	// Get user ID from auth context
	userIDStr, ok := auth.GetUserIDFromContext(ctx)
	if !ok || userIDStr == "" {
		l.ErrorContext(ctx, "User ID not found in context")
		api.ErrorResponse(w, r, http.StatusUnauthorized, "Authentication required")
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		l.ErrorContext(ctx, "Invalid user ID format", slog.Any("error", err))
		api.ErrorResponse(w, r, http.StatusBadRequest, "Invalid user ID format")
		return
	}

	// Parse request body
	var req struct {
		Message      string              `json:"message"`
		UserLocation *types.UserLocation `json:"user_location,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		l.ErrorContext(ctx, "Failed to decode request body", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid request body")
		api.ErrorResponse(w, r, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.Message == "" {
		l.ErrorContext(ctx, "Missing required fields", slog.String("message", req.Message))
		span.SetStatus(codes.Error, "Missing required fields")
		api.ErrorResponse(w, r, http.StatusBadRequest, "message is required")
		return
	}

	span.SetAttributes(
		attribute.String("user.id", userID.String()),
		attribute.String("profile.id", profileID.String()),
		attribute.String("message", req.Message),
	)

	// Set up SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Create event channel
	eventCh := make(chan types.StreamEvent, 100)

	go func() {
		l.InfoContext(ctx, "REST calling service with params",
			slog.String("userID", userID.String()),
			slog.String("profileID", profileID.String()),
			slog.String("cityName", ""),
			slog.String("message", req.Message))
		err := h.llmInteractionService.ProcessUnifiedChatMessageStream(
			ctx, userID, profileID, "", req.Message, req.UserLocation, eventCh,
		)
		if err != nil {
			l.ErrorContext(ctx, "Failed to process unified chat message stream", slog.Any("error", err))
			span.RecordError(err)

			// Safely send error event, check if context is still active
			select {
			case eventCh <- types.StreamEvent{
				Type:      types.EventTypeError,
				Error:     err.Error(),
				Timestamp: time.Now(),
				EventID:   uuid.New().String(),
			}:
				// Event sent successfully
			case <-ctx.Done():
				// Context cancelled, don't send event
				return
			}
		}
	}()

	// Set up flusher for real-time streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		l.ErrorContext(ctx, "Response writer does not support flushing")
		span.SetStatus(codes.Error, "Streaming not supported")
		api.ErrorResponse(w, r, http.StatusInternalServerError, "Streaming not supported")
		return
	}

	// Process events in real-time as they arrive
	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				l.InfoContext(ctx, "Event channel closed, ending stream")
				span.SetStatus(codes.Ok, "Stream completed")
				return
			}

			eventData, err := json.Marshal(event)
			if err != nil {
				l.ErrorContext(ctx, "Failed to marshal event", slog.Any("error", err))
				span.RecordError(err)
				continue
			}

			fmt.Fprintf(w, "data: %s\n\n", eventData)
			flusher.Flush() // Send immediately to client

			if event.Type == types.EventTypeComplete || event.Type == types.EventTypeError {
				l.InfoContext(ctx, "Stream completed", slog.String("eventType", event.Type))
				span.SetStatus(codes.Ok, "Stream completed")
				return
			}

		case <-r.Context().Done():
			l.InfoContext(ctx, "Client disconnected")
			span.SetStatus(codes.Ok, "Client disconnected")
			return
		}
	}
}


This is my old implementation of the REST API to access streaming data. 
My current implementation is using HTMX streaming sessions. I want to delete it. Delete it and delete 