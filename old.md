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


func (l *ServiceImpl) ProcessUnifiedChatMessageStream(ctx context.Context, userID, profileID uuid.UUID, cityName, message string, userLocation *types.UserLocation, eventCh chan<- types.StreamEvent) error {
startTime := time.Now() // Track when processing starts
ctx, span := otel.Tracer("LlmInteractionService").Start(ctx, "ProcessUnifiedChatMessageStream", trace.WithAttributes(
attribute.String("message", message),
))
defer span.End()

	// Extract city and clean message
	extractedCity, cleanedMessage, err := l.extractCityFromMessage(ctx, message)
	if err != nil {
		span.RecordError(err)
		l.sendEvent(ctx, eventCh, types.StreamEvent{Type: types.EventTypeError, Error: err.Error()}, 3)
		return fmt.Errorf("failed to parse message: %w", err)
	}
	if extractedCity != "" {
		cityName = extractedCity
	}
	span.SetAttributes(attribute.String("extracted.city", cityName), attribute.String("cleaned.message", cleanedMessage))

	// Detect domain
	domainDetector := &types.DomainDetector{}
	domain := domainDetector.DetectDomain(ctx, cleanedMessage)
	span.SetAttributes(attribute.String("detected.domain", string(domain)))

	// Step 3: Fetch user data
	_, searchProfile, _, err := l.FetchUserData(ctx, userID, profileID)
	if err != nil {
		span.RecordError(err)
		l.sendEvent(ctx, eventCh, types.StreamEvent{Type: types.EventTypeError, Error: err.Error()}, 3)
		return fmt.Errorf("failed to fetch user data: %w", err)
	}
	basePreferences := getUserPreferencesPrompt(searchProfile)

	// Use default location if not provided
	var lat, lon float64
	if userLocation == nil && searchProfile.UserLatitude != nil && searchProfile.UserLongitude != nil {
		userLocation = &types.UserLocation{
			UserLat: *searchProfile.UserLatitude,
			UserLon: *searchProfile.UserLongitude,
		}
	}
	if userLocation != nil {
		lat, lon = userLocation.UserLat, userLocation.UserLon
	}

	// Step 4: Cache Integration - Generate cache key based on session parameters
	sessionID := uuid.New()

	// Initialize session
	session := types.ChatSession{
		ID:        sessionID,
		UserID:    userID,
		ProfileID: profileID,
		CityName:  cityName,
		ConversationHistory: []types.ConversationMessage{
			{Role: "user", Content: message, Timestamp: time.Now()},
		},
		SessionContext: types.SessionContext{
			CityName:            cityName,
			ConversationSummary: fmt.Sprintf("Trip plan for %s", cityName),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Status:    "active",
	}
	if err := l.llmInteractionRepo.CreateSession(ctx, session); err != nil {
		span.RecordError(err)
		l.sendEvent(ctx, eventCh, types.StreamEvent{Type: types.EventTypeError, Error: err.Error()}, 3)
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Generate cache key based on session parameters
	cacheKeyData := map[string]interface{}{
		"user_id":     userID.String(),
		"profile_id":  profileID.String(),
		"city":        cityName,
		"message":     cleanedMessage,
		"domain":      string(domain),
		"preferences": basePreferences,
	}
	cacheKeyBytes, err := json.Marshal(cacheKeyData)
	if err != nil {
		l.logger.ErrorContext(ctx, "Failed to marshal cache key data", slog.Any("error", err))
		// Use a fallback cache key
		cacheKeyBytes = []byte(fmt.Sprintf("fallback_%s_%s", cleanedMessage, cityName))
	}
	hash := md5.Sum(cacheKeyBytes)
	cacheKey := hex.EncodeToString(hash[:])

	// Step 5: Fan-in Fan-out Setup
	var wg sync.WaitGroup
	var closeOnce sync.Once

	l.sendEvent(ctx, eventCh, types.StreamEvent{
		Type: types.EventTypeStart,
		Data: map[string]interface{}{
			"domain":     string(domain),
			"city":       cityName,
			"session_id": sessionID.String(),
			"cache_key":  cacheKey,
		},
	}, 3)

	// Step 5: Collect responses for saving interaction
	responses := make(map[string]*strings.Builder)
	responsesMutex := sync.Mutex{}

	// Modified sendEventWithResponse to capture responses
	sendEventWithResponse := func(event types.StreamEvent) {
		if event.Type == types.EventTypeChunk {
			responsesMutex.Lock()
			if data, ok := event.Data.(map[string]interface{}); ok {
				if partType, exists := data["part"].(string); exists {
					if chunk, chunkExists := data["chunk"].(string); chunkExists {
						if responses[partType] == nil {
							responses[partType] = &strings.Builder{}
						}
						responses[partType].WriteString(chunk)
					}
				}
			}
			responsesMutex.Unlock()
		}
		l.sendEvent(ctx, eventCh, event, 3)
	}

	// Step 6: Spawn streaming workers based on domain with cache support
	switch domain {
	case types.DomainItinerary, types.DomainGeneral:
		wg.Add(3)

		// Worker 1: Stream City Data with cache
		go func() {
			defer wg.Done()
			prompt := getCityDataPrompt(cityName)
			partCacheKey := cacheKey + "_city_data"
			l.streamWorkerWithResponseAndCache(ctx, prompt, "city_data", sendEventWithResponse, domain, partCacheKey)
		}()

		// Worker 2: Stream General POIs with cache
		go func() {
			defer wg.Done()
			prompt := getGeneralPOIPrompt(cityName)
			partCacheKey := cacheKey + "_general_pois"
			l.streamWorkerWithResponseAndCache(ctx, prompt, "general_pois", sendEventWithResponse, domain, partCacheKey)
		}()

		// Worker 3: Stream Personalized Itinerary with cache
		go func() {
			defer wg.Done()
			prompt := getPersonalizedItineraryPrompt(cityName, basePreferences)
			partCacheKey := cacheKey + "_itinerary"
			l.streamWorkerWithResponseAndCache(ctx, prompt, "itinerary", sendEventWithResponse, domain, partCacheKey)
		}()

	case types.DomainAccommodation:
		wg.Add(1)
		go func() {
			defer wg.Done()
			prompt := getAccommodationPrompt(cityName, lat, lon, basePreferences)
			partCacheKey := cacheKey + "_hotels"
			l.streamWorkerWithResponseAndCache(ctx, prompt, "hotels", sendEventWithResponse, domain, partCacheKey)
		}()

	case types.DomainDining:
		wg.Add(1)
		go func() {
			defer wg.Done()
			prompt := getDiningPrompt(cityName, lat, lon, basePreferences)
			partCacheKey := cacheKey + "_restaurants"
			l.streamWorkerWithResponseAndCache(ctx, prompt, "restaurants", sendEventWithResponse, domain, partCacheKey)
		}()

	case types.DomainActivities:
		wg.Add(1)
		go func() {
			defer wg.Done()
			prompt := getActivitiesPrompt(cityName, lat, lon, basePreferences)
			partCacheKey := cacheKey + "_activities"
			l.streamWorkerWithResponseAndCache(ctx, prompt, "activities", sendEventWithResponse, domain, partCacheKey)
		}()

	default:
		sendEventWithResponse(types.StreamEvent{Type: types.EventTypeError, Error: fmt.Sprintf("unhandled domain: %s", domain)})
		return fmt.Errorf("unhandled domain type: %s", domain)
	}

	// Step 7: Completion goroutine with sync.Once for channel closure
	go func() {
		wg.Wait()             // Wait for all workers to complete
		if ctx.Err() == nil { // Only send completion event if context is still active
			// Determine route type based on domain
			var routeType string
			var baseURL string
			switch domain {
			case types.DomainAccommodation:
				routeType = "hotels"
				baseURL = "/hotels"
			case types.DomainDining:
				routeType = "restaurants"
				baseURL = "/restaurants"
			case types.DomainActivities:
				routeType = "activities"
				baseURL = "/activities"
			default:
				routeType = "itinerary"
				baseURL = "/itinerary"
			}

			l.sendEvent(ctx, eventCh, types.StreamEvent{
				Type: types.EventTypeComplete,
				Data: map[string]interface{}{"session_id": sessionID.String()},
				Navigation: &types.NavigationData{
					URL:       fmt.Sprintf("%s?sessionId=%s&cityName=%s&domain=%s", baseURL, sessionID.String(), url.QueryEscape(cityName), routeType),
					RouteType: routeType,
					QueryParams: map[string]string{
						"sessionId": sessionID.String(),
						"cityName":  cityName,
						"domain":    routeType,
					},
				},
			}, 3)
		}
		closeOnce.Do(func() {
			close(eventCh) // Close the channel only once
			l.logger.InfoContext(ctx, "Event channel closed by completion goroutine")
		})
	}()

	go func() {
		//wg.Wait() // Wait for all workers to complete
		asyncCtx := context.Background()

		var fullResponseBuilder strings.Builder
		responsesMutex.Lock()
		cityDataContent := ""
		if responses["city_data"] != nil {
			cityDataContent = responses["city_data"].String()
		}
		for partType, builder := range responses {
			if builder != nil && builder.Len() > 0 {
				fullResponseBuilder.WriteString(fmt.Sprintf("[%s]\n%s\n\n", partType, builder.String()))
			}
		}
		responsesMutex.Unlock()

		fullResponse := fullResponseBuilder.String()
		if fullResponse == "" {
			fullResponse = fmt.Sprintf("Processed %s request for %s", domain, cityName)
		}

		// Process and save city data if available
		var cityID uuid.UUID
		if cityDataContent != "" {
			// Parse city data from the response
			if parsedCityData, parseErr := l.parseCityDataFromResponse(asyncCtx, cityDataContent); parseErr == nil && parsedCityData != nil {
				// Save city data to the cities table
				if savedCityID, handleErr := l.HandleCityData(asyncCtx, *parsedCityData); handleErr != nil {
					l.logger.WarnContext(asyncCtx, "Failed to save city data during unified stream processing",
						slog.String("city", cityName), slog.Any("error", handleErr))
				} else {
					l.logger.InfoContext(asyncCtx, "Successfully saved city data during unified stream processing",
						slog.String("city", cityName))
					cityID = savedCityID
				}
			} else if parseErr != nil {
				l.logger.WarnContext(asyncCtx, "Failed to parse city data from unified stream response",
					slog.String("city", cityName), slog.Any("error", parseErr))
			}
		}

		// If we don't have a cityID from the response, try to get it from the database
		if cityID == uuid.Nil {
			if existingCity, err := l.cityRepo.FindCityByNameAndCountry(asyncCtx, cityName, ""); err == nil && existingCity != nil {
				cityID = existingCity.ID
			} else {
				l.logger.WarnContext(asyncCtx, "Could not find or save city data, skipping POI processing",
					slog.String("city", cityName))
				return
			}
		}

		// Create and save interaction first to get proper llmInteractionID
		interaction := types.LlmInteraction{
			ID:           uuid.New(),
			SessionID:    sessionID,
			UserID:       userID,
			ProfileID:    profileID,
			CityName:     cityName,
			Prompt:       fmt.Sprintf("Unified Chat Stream - Domain: %s, Message: %s", domain, cleanedMessage),
			ResponseText: fullResponse,
			ModelUsed:    model,
			LatencyMs:    int(time.Since(startTime).Milliseconds()),
			Timestamp:    startTime,
		}
		savedInteractionID, err := l.llmInteractionRepo.SaveInteraction(asyncCtx, interaction)
		if err != nil {
			l.logger.ErrorContext(asyncCtx, "Failed to save stream interaction", slog.Any("error", err))
			return
		}

		l.logger.InfoContext(asyncCtx, "Stream interaction saved successfully",
			slog.String("saved_interaction_id", savedInteractionID.String()),
			slog.String("original_session_id", sessionID.String()))

		// Always try to process and save POI data regardless of domain
		// since responses may contain POI data in different formats
		l.ProcessAndSaveUnifiedResponse(asyncCtx, responses, userID, profileID, cityID, savedInteractionID, userLocation)
	}()

	span.SetStatus(codes.Ok, "Unified chat stream processed successfully")
	return nil
}

package llmchat

import (
"context"
"encoding/json"
"fmt"
"sync"
"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/genai"

	"github.com/FACorreiaa/go-poi-au-suggestions/internal/types"
)

func (l *ServiceImpl) GenerateCityDataWorker(wg *sync.WaitGroup,
ctx context.Context,
cityName string,
resultCh chan<- types.GenAIResponse,
config *genai.GenerateContentConfig) {
go func() {
ctx, span := otel.Tracer("LlmInteractionService").Start(ctx, "GenerateCityDataWorker", trace.WithAttributes(
attribute.String("city.name", cityName),
))
defer span.End()
defer wg.Done()

		prompt := getCityDescriptionPrompt(cityName)
		span.SetAttributes(attribute.Int("prompt.length", len(prompt)))

		response, err := l.aiClient.GenerateResponse(ctx, prompt, config)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to generate city data")
			resultCh <- types.GenAIResponse{Err: fmt.Errorf("failed to generate city data: %w", err)}
			return
		}

		var txt string
		for _, candidate := range response.Candidates {
			if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
				txt = candidate.Content.Parts[0].Text
				break
			}
		}
		if txt == "" {
			err := fmt.Errorf("no valid city data content from AI")
			span.RecordError(err)
			span.SetStatus(codes.Error, "Empty response from AI")
			resultCh <- types.GenAIResponse{Err: err}
			return
		}
		span.SetAttributes(attribute.Int("response.length", len(txt)))

		cleanTxt := cleanJSONResponse(txt)
		var cityDataFromAI struct {
			CityName        string  `json:"city_name"`
			StateProvince   *string `json:"state_province"` // Use pointer for nullable string
			Country         string  `json:"country"`
			CenterLatitude  float64 `json:"center_latitude"`
			CenterLongitude float64 `json:"center_longitude"`
			Description     string  `json:"description"`
			// BoundingBox     string  `json:"bounding_box,omitempty"` // If trying to get BBox string
		}
		if err := json.Unmarshal([]byte(cleanTxt), &cityDataFromAI); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to parse city data JSON")
			resultCh <- types.GenAIResponse{Err: fmt.Errorf("failed to parse city data JSON: %w", err)}
			return
		}

		stateProvinceValue := ""
		if cityDataFromAI.StateProvince != nil {
			stateProvinceValue = *cityDataFromAI.StateProvince
		}

		span.SetAttributes(
			attribute.String("city.name", cityDataFromAI.CityName),
			attribute.String("city.country", cityDataFromAI.Country),
			attribute.Float64("city.latitude", cityDataFromAI.CenterLatitude),
			attribute.Float64("city.longitude", cityDataFromAI.CenterLongitude),
		)
		span.SetStatus(codes.Ok, "City data generated successfully")

		resultCh <- types.GenAIResponse{
			City:            cityDataFromAI.CityName,
			Country:         cityDataFromAI.Country,
			StateProvince:   stateProvinceValue,
			CityDescription: cityDataFromAI.Description,
			Latitude:        cityDataFromAI.CenterLatitude,
			Longitude:       cityDataFromAI.CenterLongitude,
			// BoundingBoxWKT: cityDataFromAI.BoundingBox, // TODO
		}
	}()
}

func (l *ServiceImpl) GenerateGeneralPOIWorker(wg *sync.WaitGroup,
ctx context.Context,
cityName string,
resultCh chan<- types.GenAIResponse,
config *genai.GenerateContentConfig) {
ctx, span := otel.Tracer("LlmInteractionService").Start(ctx, "GenerateGeneralPOIWorker", trace.WithAttributes(
attribute.String("city.name", cityName),
))
defer span.End()
defer wg.Done()

	prompt := getGeneralPOIPrompt(cityName)
	span.SetAttributes(attribute.Int("prompt.length", len(prompt)))

	startTime := time.Now()
	response, err := l.aiClient.GenerateResponse(ctx, prompt, config)
	latencyMs := int(time.Since(startTime).Milliseconds())
	span.SetAttributes(attribute.Int("response.latency_ms", latencyMs))

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to generate general POIs")
		resultCh <- types.GenAIResponse{Err: fmt.Errorf("failed to generate general POIs: %w", err)}
		return
	}

	var txt string
	for _, candidate := range response.Candidates {
		if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
			txt = candidate.Content.Parts[0].Text
			break
		}
	}
	if txt == "" {
		err := fmt.Errorf("no valid general POI content from AI")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Empty response from AI")
		resultCh <- types.GenAIResponse{Err: err}
		return
	}
	span.SetAttributes(attribute.Int("response.length", len(txt)))

	cleanTxt := cleanJSONResponse(txt)
	var poiData struct {
		PointsOfInterest []types.POIDetailedInfo `json:"points_of_interest"`
	}
	if err := json.Unmarshal([]byte(cleanTxt), &poiData); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to parse general POI JSON")
		resultCh <- types.GenAIResponse{Err: fmt.Errorf("failed to parse general POI JSON: %w", err)}
		return
	}

	span.SetAttributes(attribute.Int("pois.count", len(poiData.PointsOfInterest)))
	span.SetStatus(codes.Ok, "General POIs generated successfully")
	resultCh <- types.GenAIResponse{GeneralPOI: poiData.PointsOfInterest}
}

func (l *ServiceImpl) GeneratePersonalisedPOIWorker(wg *sync.WaitGroup, ctx context.Context,
cityName string, userID, profileID, sessionID uuid.UUID, resultCh chan<- types.GenAIResponse,
interestNames []string, tagsPromptPart string, userPrefs string,
config *genai.GenerateContentConfig) {
ctx, span := otel.Tracer("LlmInteractionService").Start(ctx, "GeneratePersonalisedPOIWorker", trace.WithAttributes(
attribute.String("city.name", cityName),
attribute.String("user.id", userID.String()),
attribute.String("profile.id", profileID.String()),
attribute.Int("interests.count", len(interestNames)),
))
defer span.End()
defer wg.Done()

	startTime := time.Now()

	prompt := getPersonalizedPOI(interestNames, cityName, tagsPromptPart, userPrefs)
	span.SetAttributes(attribute.Int("prompt.length", len(prompt)))

	response, err := l.aiClient.GenerateResponse(ctx, prompt, config)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to generate personalized itinerary")
		resultCh <- types.GenAIResponse{Err: fmt.Errorf("failed to generate personalized itinerary: %w", err)}
		return
	}

	var txt string
	for _, candidate := range response.Candidates {
		if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
			txt = candidate.Content.Parts[0].Text
			break
		}
	}
	if txt == "" {
		err := fmt.Errorf("no valid personalized itinerary content from AI")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Empty response from AI")
		resultCh <- types.GenAIResponse{Err: err}
		return
	}
	span.SetAttributes(attribute.Int("response.length", len(txt)))

	cleanTxt := cleanJSONResponse(txt)
	var itineraryData struct {
		ItineraryName      string                  `json:"itinerary_name"`
		OverallDescription string                  `json:"overall_description"`
		PointsOfInterest   []types.POIDetailedInfo `json:"points_of_interest"`
	}

	if err := json.Unmarshal([]byte(cleanTxt), &itineraryData); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to parse personalized itinerary JSON")
		resultCh <- types.GenAIResponse{Err: fmt.Errorf("failed to parse personalized itinerary JSON: %w", err)}
		return
	}
	span.SetAttributes(
		attribute.String("itinerary.name", itineraryData.ItineraryName),
		attribute.Int("personalized_pois.count", len(itineraryData.PointsOfInterest)),
	)

	latencyMs := int(time.Since(startTime).Milliseconds())
	span.SetAttributes(attribute.Int("response.latency_ms", latencyMs))

	interaction := types.LlmInteraction{
		UserID:       userID,
		SessionID:    sessionID,
		Prompt:       prompt,
		ResponseText: txt,
		ModelUsed:    model, // Adjust based on your AI client
		LatencyMs:    latencyMs,
		CityName:     cityName,
		// request payload
		// response payload
		// Add token counts if available from response (depends on genai API)
		// PromptTokens, CompletionTokens, TotalTokens
		// RequestPayload, ResponsePayload if you serialize the full request/response
	}
	savedInteractionID, err := l.llmInteractionRepo.SaveInteraction(ctx, interaction)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to save LLM interaction")
		resultCh <- types.GenAIResponse{Err: fmt.Errorf("failed to save LLM interaction: %w", err)}
		return
	}
	span.SetAttributes(attribute.String("llm_interaction.id", savedInteractionID.String()))
	span.SetStatus(codes.Ok, "Personalized POIs generated successfully")

	resultCh <- types.GenAIResponse{
		ItineraryName:        itineraryData.ItineraryName,
		ItineraryDescription: itineraryData.OverallDescription,
		PersonalisedPOI:      itineraryData.PointsOfInterest,
		LlmInteractionID:     savedInteractionID,
	}
}

// GeneratePersonalisedPOIWorkerWithSemantics generates personalized POIs with semantic search enhancement
func (l *ServiceImpl) GeneratePersonalisedPOIWorkerWithSemantics(wg *sync.WaitGroup, ctx context.Context,
cityName string, userID, profileID, sessionID uuid.UUID, resultCh chan<- types.GenAIResponse,
interestNames []string, tagsPromptPart string, userPrefs string, semanticPOIs []types.POIDetailedInfo,
config *genai.GenerateContentConfig) {
ctx, span := otel.Tracer("LlmInteractionService").Start(ctx, "GeneratePersonalisedPOIWorkerWithSemantics", trace.WithAttributes(
attribute.String("city.name", cityName),
attribute.String("user.id", userID.String()),
attribute.String("profile.id", profileID.String()),
attribute.Int("interests.count", len(interestNames)),
attribute.Int("semantic_pois.count", len(semanticPOIs)),
))
defer span.End()
defer wg.Done()

	startTime := time.Now()

	// Create enhanced prompt with semantic context
	prompt := l.getPersonalizedPOIWithSemanticContext(interestNames, cityName, tagsPromptPart, userPrefs, semanticPOIs)
	span.SetAttributes(attribute.Int("prompt.length", len(prompt)))

	response, err := l.aiClient.GenerateResponse(ctx, prompt, config)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to generate semantic-enhanced personalized itinerary")
		resultCh <- types.GenAIResponse{Err: fmt.Errorf("failed to generate semantic-enhanced personalized itinerary: %w", err)}
		return
	}

	var txt string
	for _, candidate := range response.Candidates {
		if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
			txt = candidate.Content.Parts[0].Text
			break
		}
	}
	if txt == "" {
		err := fmt.Errorf("no valid semantic-enhanced personalized itinerary content from AI")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Empty response from AI")
		resultCh <- types.GenAIResponse{Err: err}
		return
	}
	span.SetAttributes(attribute.Int("response.length", len(txt)))

	cleanTxt := cleanJSONResponse(txt)
	var itineraryData struct {
		ItineraryName      string                  `json:"itinerary_name"`
		OverallDescription string                  `json:"overall_description"`
		PointsOfInterest   []types.POIDetailedInfo `json:"points_of_interest"`
	}

	if err := json.Unmarshal([]byte(cleanTxt), &itineraryData); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to parse semantic-enhanced personalized itinerary JSON")
		resultCh <- types.GenAIResponse{Err: fmt.Errorf("failed to parse semantic-enhanced personalized itinerary JSON: %w", err)}
		return
	}
	span.SetAttributes(
		attribute.String("itinerary.name", itineraryData.ItineraryName),
		attribute.Int("personalized_pois.count", len(itineraryData.PointsOfInterest)),
	)

	latencyMs := int(time.Since(startTime).Milliseconds())
	span.SetAttributes(attribute.Int("response.latency_ms", latencyMs))

	interaction := types.LlmInteraction{
		UserID:       userID,
		SessionID:    sessionID,
		Prompt:       prompt,
		ResponseText: txt,
		ModelUsed:    model,
		LatencyMs:    latencyMs,
		CityName:     cityName,
	}
	savedInteractionID, err := l.llmInteractionRepo.SaveInteraction(ctx, interaction)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to save semantic-enhanced LLM interaction")
		resultCh <- types.GenAIResponse{Err: fmt.Errorf("failed to save semantic-enhanced LLM interaction: %w", err)}
		return
	}
	span.SetAttributes(attribute.String("llm_interaction.id", savedInteractionID.String()))
	span.SetStatus(codes.Ok, "Semantic-enhanced personalized POIs generated successfully")

	resultCh <- types.GenAIResponse{
		ItineraryName:        itineraryData.ItineraryName,
		ItineraryDescription: itineraryData.OverallDescription,
		PersonalisedPOI:      itineraryData.PointsOfInterest,
		LlmInteractionID:     savedInteractionID,
	}
}

package types

import (
"context"
"database/sql"
"encoding/json"
"fmt"
"regexp"
"strings"
"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"google.golang.org/genai"
)

type LlmInteraction struct {
ID                 uuid.UUID       `json:"id"`
SessionID          uuid.UUID       `json:"session_id"`
UserID             uuid.UUID       `json:"user_id"`
ProfileID          uuid.UUID       `json:"profile_id"`
CityName           string          `json:"city_name,omitempty"` // The city context for this interaction
Prompt             string          `json:"prompt"`
RequestPayload     json.RawMessage `json:"request_payload"`
ResponseText       string          `json:"response"`
ResponsePayload    json.RawMessage `json:"response_payload"`
ModelUsed          string          `json:"model_name"`
PromptTokens       int             `json:"prompt_tokens"`
CompletionTokens   int             `json:"completion_tokens"`
TotalTokens        int             `json:"total_tokens"`
LatencyMs          int             `json:"latency_ms"`
Timestamp          time.Time       `json:"timestamp"`
ModelName          string          `json:"model"`
Response           string          `json:"response_content"`
Latitude           *float64        `json:"latitude"`
Longitude          *float64        `json:"longitude"`
Distance           *float64        `json:"distance"`
PromptTokenCount   int             `json:"prompt_token_count"`
ResponseTokenCount int             `json:"response_token_count"`
}

type AIItineraryResponse struct {
ItineraryName      string            `json:"itinerary_name"`
OverallDescription string            `json:"overall_description"`
PointsOfInterest   []POIDetailedInfo `json:"points_of_interest"`
Restaurants        []POIDetailedInfo `json:"restaurants,omitempty"`
Bars               []POIDetailedInfo `json:"bars,omitempty"`
}

type GeneralCityData struct {
City            string  `json:"city"`
Country         string  `json:"country"`
StateProvince   string  `json:"state_province,omitempty"`
Description     string  `json:"description"`
CenterLatitude  float64 `json:"center_latitude,omitempty"`
CenterLongitude float64 `json:"center_longitude,omitempty"`
Population      string  `json:"population"`
Area            string  `json:"area"`
Timezone        string  `json:"timezone"`
Language        string  `json:"language"`
Weather         string  `json:"weather"`
Attractions     string  `json:"attractions"`
History         string  `json:"history"`
}

type AiCityResponse struct {
GeneralCityData     GeneralCityData     `json:"general_city_data"`
PointsOfInterest    []POIDetailedInfo   `json:"points_of_interest"`
AIItineraryResponse AIItineraryResponse `json:"itinerary_response"`
SessionID           uuid.UUID           `json:"session_id"`
}

type GenAIResponse struct {
SessionID            string            `json:"session_id"`
LlmInteractionID     uuid.UUID         `json:"llm_interaction_id"`
City                 string            `json:"city,omitempty"`
Country              string            `json:"country,omitempty"`
StateProvince        string            `json:"state_province,omitempty"` // New
CityDescription      string            `json:"city_description,omitempty"`
Latitude             float64           `json:"latitude,omitempty"`  // New: for city center
Longitude            float64           `json:"longitude,omitempty"` // New: for city center
ItineraryName        string            `json:"itinerary_name,omitempty"`
ItineraryDescription string            `json:"itinerary_description,omitempty"`
GeneralPOI           []POIDetailedInfo `json:"general_poi,omitempty"`
PersonalisedPOI      []POIDetailedInfo `json:"personalised_poi,omitempty"` // Consider changing to []PersonalizedPOIDetail
POIDetailedInfo      []POIDetailedInfo `json:"poi_detailed_info,omitempty"`
Err                  error             `json:"-"`
ModelName            string            `json:"model_name"`
Prompt               string            `json:"prompt"`
Response             string            `json:"response"`
}

type AIRequestPayloadForLog struct {
ModelName        string                       `json:"model_name"`
GenerationConfig *genai.GenerateContentConfig `json:"generation_config,omitempty"`
Content          *genai.Content               `json:"content"` // The actual content sent (prompt)
// You could add other things like "tools" if you use function calling
}

type ChatTurn struct { // You might not need this explicit struct if directly using []*genai.Content
Role  string       `json:"role"` // "user" or "model"
Parts []genai.Part `json:"parts"`
}

type UserLocation struct {
UserLat        float64 `json:"user_lat"`
UserLon        float64 `json:"user_lon"`
SearchRadiusKm float64 // Radius in kilometers for searching nearby POIs
}

type UserSavedItinerary struct {
ID                     uuid.UUID      `json:"id"`
UserID                 uuid.UUID      `json:"user_id"`
SourceLlmInteractionID pgtype.UUID    `json:"source_llm_interaction_id,omitempty"` // Nullable UUID for the source LLM interaction
SessionID              pgtype.UUID    `json:"session_id,omitempty"`                // Nullable UUID for the chat session
PrimaryCityID          pgtype.UUID    `json:"primary_city_id,omitempty"`           // Nullable UUID for the primary city
Title                  string         `json:"title"`
Description            sql.NullString `json:"description"`             // Use sql.NullString for nullable text fields
MarkdownContent        string         `json:"markdown_content"`        // Markdown content for the itinerary
Tags                   []string       `json:"tags"`                    // Tags for the itinerary
EstimatedDurationDays  sql.NullInt32  `json:"estimated_duration_days"` // Nullable int32 for estimated duration in days
EstimatedCostLevel     sql.NullInt32  `json:"estimated_cost_level"`    // Nullable int32 for estimated cost level
IsPublic               bool           `json:"is_public"`               // Indicates if the itinerary is public
CreatedAt              time.Time      `json:"created_at"`
UpdatedAt              time.Time      `json:"updated_at"`
}

type UpdateItineraryRequest struct {
Title                 *string  `json:"title,omitempty"`
Description           *string  `json:"description,omitempty"` // If nil, means no change. If empty string, means clear description.
Tags                  []string `json:"tags,omitempty"`        // If nil, no change. If empty slice, clear tags.
EstimatedDurationDays *int32   `json:"estimated_duration_days,omitempty"`
EstimatedCostLevel    *int32   `json:"estimated_cost_level,omitempty"`
IsPublic              *bool    `json:"is_public,omitempty"`
MarkdownContent       *string  `json:"markdown_content,omitempty"`
}

type PaginatedUserItinerariesResponse struct {
Itineraries  []UserSavedItinerary `json:"itineraries"`
TotalRecords int                  `json:"total_records"`
Page         int                  `json:"page"`
PageSize     int                  `json:"page_size"`
}

type BookmarkRequest struct {
LlmInteractionID *uuid.UUID `json:"llm_interaction_id,omitempty"` // Optional - if provided, use this specific interaction
SessionID        *uuid.UUID `json:"session_id,omitempty"`         // Optional - if provided, use latest interaction from this session
PrimaryCityID    *uuid.UUID `json:"primary_city_id,omitempty"`    // Optional - if provided, use this
PrimaryCityName  string     `json:"primary_city_name"`            // City name to look up if PrimaryCityID not provided
Title            string     `json:"title"`
Description      *string    `json:"description"` // Optional
Tags             []string   `json:"tags"`        // Optional
IsPublic         *bool      `json:"is_public"`   // Optional
}

type ChatMessage struct {
ID        uuid.UUID
UserID    uuid.UUID
Timestamp time.Time
Role      string
Content   string
}

type POIDetailrequest struct {
CityName  string  `json:"city_name"` // e.g., "New York"
Latitude  float64 `json:"latitude"`  // e.g., 40.7128
Longitude float64 `json:"longitude"` // e.g., -74.0060
}

type POIFilter struct {
Location GeoPoint `json:"location"` // e.g., "restaurant", "hotel", "bar"
Radius   float64  `json:"radius"`   // Radius in kilometers for filtering POIs
Category string   `json:"category"` // e.g., "restaurant", "hotel", "bar"
}

type GeoPoint struct {
Latitude  float64 `json:"latitude"`  // Latitude of the point
Longitude float64 `json:"longitude"` // Longitude of the point
}
type SearchPOIRequest struct {
Query      string   `json:"query"` // The search query text
CityName   string   `json:"city"`
Latitude   float64  `json:"lat"`
Longitude  float64  `json:"lon"`
RadiusKm   float64  `json:"radius_km"`   // Optional, for filtering POIs within a certain radius
SearchText string   `json:"search_text"` // Optional, for searching by name or description
SearchTags []string `json:"search_tags"` // Optional, for filtering by tags
SearchType string   `json:"search_type"` // Optional, e.g., "restaurant", "hotel", "bar"
SortBy     string   `json:"sort_by"`     // Optional, e.g., "rating", "distance"
SortOrder  string   `json:"sort_order"`  // Optional, e.g., "asc", "desc"
MinRating  float64  `json:"min_rating"`  // Optional, for filtering by minimum rating
MinPrice   string   `json:"min_price"`   // Optional, for filtering by minimum price range
MinGuests  int32    `json:"min_guests"`  // Optional, for filtering by minimum number of guests (for restaurants)
}

type HotelUserPreferences struct {
NumberOfGuests      int32     `json:"number_of_guests"`
PreferredCategories string    `json:"preferred_category"`    // e.g., "budget", "luxury"
PreferredTags       []string  `json:"preferredTags"`         // e.g., ["pet-friendly", "free wifi"]
MaxPriceRange       string    `json:"preferred_price_range"` // e.g., "$", "$$"
MinRating           float64   `json:"preferred_rating"`      // e.g., 4.0
NumberOfNights      int64     `json:"number_of_nights"`
NumberOfRooms       int32     `json:"number_of_rooms"`
PreferredCheckIn    time.Time `json:"preferred_check_in"`
PreferredCheckOut   time.Time `json:"preferred_check_out"`
SearchRadiusKm      float64   `json:"search_radius_km"` // Optional, for filtering hotels within a certain radius
}

type HotelDetailedInfo struct {
ID               uuid.UUID `json:"id"`
City             string    `json:"city"`
Name             string    `json:"name"`
Latitude         float64   `json:"latitude"`
Longitude        float64   `json:"longitude"`
Category         string    `json:"category"` // e.g., "Hotel", "Hostel"
Description      string    `json:"description"`
Address          string    `json:"address"`
PhoneNumber      *string   `json:"phone_number"`
Website          *string   `json:"website"`
OpeningHours     *string   `json:"opening_hours"`
PriceRange       *string   `json:"price_range"`
Rating           float64   `json:"rating"`
Tags             []string  `json:"tags"`
Images           []string  `json:"images"`
LlmInteractionID uuid.UUID `json:"llm_interaction_id"`
Err              error     `json:"-"` // Not serialized
}

type HotelPreferenceRequest struct {
City        string               `json:"city"`
Lat         float64              `json:"lat"`
Lon         float64              `json:"lon"`
Preferences HotelUserPreferences `json:"preferences"`
Distance    float64              `json:"distance"` // Optional, for filtering hotels within a certain radius
}

type RestaurantUserPreferences struct {
PreferredCuisine    string
PreferredPriceRange string
DietaryRestrictions string
Ambiance            string
SpecialFeatures     string
}

type RestaurantDetailedInfo struct {
ID               uuid.UUID `json:"id"`
City             string    `json:"city"`
Name             string    `json:"name"`
Latitude         float64   `json:"latitude"`
Longitude        float64   `json:"longitude"`
Category         string    `json:"category"`
Description      string    `json:"description"`
Address          *string   `json:"address"`
Website          *string   `json:"website"`
PhoneNumber      *string   `json:"phone_number"`
OpeningHours     *string   `json:"opening_hours"`
PriceLevel       *string   `json:"price_level"`  // Changed to *string
CuisineType      *string   `json:"cuisine_type"` // Changed to *string
Tags             []string  `json:"tags"`
Images           []string  `json:"images"`
Rating           float64   `json:"rating"`
LlmInteractionID uuid.UUID `json:"llm_interaction_id"`
Err              error     `json:"-"`
}

// Context-aware chat types
type ChatContextType string

const (
ContextHotels      ChatContextType = "hotels"
ContextRestaurants ChatContextType = "restaurants"
ContextItineraries ChatContextType = "itineraries"
ContextGeneral     ChatContextType = "general"
)

type StartChatRequest struct {
CityName       string          `json:"city_name"`
ContextType    ChatContextType `json:"context_type"`
InitialMessage string          `json:"initial_message,omitempty"`
}

type ContinueChatRequest struct {
Message     string          `json:"message"`
CityName    string          `json:"city_name,omitempty"`
ContextType ChatContextType `json:"context_type"`
}

//

type SimpleIntentClassifier struct{}

func (c *SimpleIntentClassifier) Classify(_ context.Context, message string) (IntentType, error) {
message = strings.ToLower(message)
matched, err := regexp.MatchString(`add|include|visit`, message)
if err != nil {
return IntentModifyItinerary, fmt.Errorf("failed to match add pattern: %w", err)
}
if matched {
return IntentAddPOI, nil
}
matched, err = regexp.MatchString(`remove|delete|skip`, message)
if err != nil {
return IntentModifyItinerary, fmt.Errorf("failed to match remove pattern: %w", err)
}
if matched {
return IntentRemovePOI, nil
}
matched, err = regexp.MatchString(`what|where|how|why|when`, message)
if err != nil {
return IntentModifyItinerary, fmt.Errorf("failed to match question pattern: %w", err)
}
if matched {
return IntentAskQuestion, nil
}
return IntentModifyItinerary, nil // Default intent
}

// DomainDetector detects the primary domain from user queries
type DomainDetector struct{}

func (d *DomainDetector) DetectDomain(_ context.Context, message string) DomainType {
message = strings.ToLower(message)

	// Accommodation domain keywords
	matched, err := regexp.MatchString(`hotel|hostel|accommodation|stay|sleep|room|booking|airbnb|lodge|resort|guesthouse`, message)
	if err == nil && matched {
		return DomainAccommodation
	}

	// Dining domain keywords
	matched, err = regexp.MatchString(`restaurant|food|eat|dine|meal|cuisine|drink|cafe|bar|lunch|dinner|breakfast|brunch`, message)
	if err == nil && matched {
		return DomainDining
	}

	// Activity domain keywords
	matched, err = regexp.MatchString(`activity|museum|park|attraction|tour|visit|see|do|experience|adventure|shopping|nightlife`, message)
	if err == nil && matched {
		return DomainActivities
	}

	// Itinerary domain keywords
	matched, err = regexp.MatchString(`itinerary|plan|schedule|trip|day|week|journey|route|organize|arrange`, message)
	if err == nil && matched {
		return DomainItinerary
	}

	// Default to general domain
	return DomainGeneral
}

// RecentInteraction represents a recent user interaction with cities and POIs
type RecentInteraction struct {
ID           uuid.UUID                `json:"id"`
UserID       uuid.UUID                `json:"user_id"`
CityName     string                   `json:"city_name"`
CityID       *uuid.UUID               `json:"city_id,omitempty"`
Prompt       string                   `json:"prompt"`
ResponseText string                   `json:"response,omitempty"`
ModelUsed    string                   `json:"model_name"`
LatencyMs    int                      `json:"latency_ms"`
CreatedAt    time.Time                `json:"created_at"`
POIs         []POIDetailedInfo        `json:"pois,omitempty"`
Hotels       []HotelDetailedInfo      `json:"hotels,omitempty"`
Restaurants  []RestaurantDetailedInfo `json:"restaurants,omitempty"`
}

// RecentInteractionsResponse groups interactions by city
type RecentInteractionsResponse struct {
Cities  []CityInteractions `json:"cities"`
Total   int                `json:"total"`
Page    int                `json:"page"`
Limit   int                `json:"limit"`
HasMore bool               `json:"has_more"`
}

// CityInteractions groups interactions for a specific city
type CityInteractions struct {
CityName         string              `json:"city_name"`
SessionID        uuid.UUID           `json:"session_id"`
Interactions     []RecentInteraction `json:"interactions"`
POICount         int                 `json:"poi_count"`
LastActivity     time.Time           `json:"last_activity"`
SessionIDs       []uuid.UUID         `json:"session_ids"` // Changed from SessionID
Title            string              `json:"title"`
TotalFavorites   *int                `json:"total_favorites,omitempty"`
TotalItineraries *int                `json:"total_itineraries,omitempty"`
}

// RecentInteractionsFilter defines filters for recent interactions
type RecentInteractionsFilter struct {
SortBy          string `json:"sort_by"`          // last_activity, city_name, interaction_count, poi_count
SortOrder       string `json:"sort_order"`       // asc, desc
Search          string `json:"search"`           // Search term for city name
MinInteractions int    `json:"min_interactions"` // Minimum number of interactions
MaxInteractions int    `json:"max_interactions"` // Maximum number of interactions
}
