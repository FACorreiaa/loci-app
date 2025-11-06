package llmchat

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/genai"

	"github.com/FACorreiaa/go-templui/internal/app/models"
)

// ParallelWorkerConfig holds configuration for parallel worker execution
type ParallelWorkerConfig struct {
	NumWorkers int
	ChunkSize  int
}

// poiResult holds the result of POI generation for a city
type poiResult struct {
	cityName string
	pois     []models.POIDetailedInfo
	err      error
}

// NewParallelWorkerConfig creates a default parallel worker configuration
func NewParallelWorkerConfig() *ParallelWorkerConfig {
	return &ParallelWorkerConfig{
		NumWorkers: runtime.NumCPU(),
		ChunkSize:  1, // Default for single-item processing
	}
}

// GenerateCityDataParallel processes city data generation across multiple workers
func (l *ServiceImpl) GenerateCityDataParallel(ctx context.Context,
	cityNames []string,
	config *genai.GenerateContentConfig) ([]models.GenAIResponse, error) {

	ctx, span := otel.Tracer("LlmInteractionService").Start(ctx, "GenerateCityDataParallel", trace.WithAttributes(
		attribute.Int("cities.count", len(cityNames)),
	))
	defer span.End()

	if len(cityNames) == 0 {
		return []models.GenAIResponse{}, nil
	}

	workerConfig := NewParallelWorkerConfig()
	numWorkers := workerConfig.NumWorkers
	if numWorkers > len(cityNames) {
		numWorkers = len(cityNames)
	}

	span.SetAttributes(attribute.Int("workers.count", numWorkers))

	// Create channels for work distribution and result collection
	workCh := make(chan string, len(cityNames))
	resultCh := make(chan models.GenAIResponse, len(cityNames))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for cityName := range workCh {
				l.processCityData(ctx, cityName, config, resultCh, workerID)
			}
		}(i)
	}

	// Distribute work
	go func() {
		for _, cityName := range cityNames {
			workCh <- cityName
		}
		close(workCh)
	}()

	// Wait for workers to complete and close result channel
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results
	results := make([]models.GenAIResponse, 0, len(cityNames))
	for result := range resultCh {
		results = append(results, result)
	}

	span.SetAttributes(attribute.Int("results.count", len(results)))
	span.SetStatus(codes.Ok, "City data generation completed")

	return results, nil
}

// processCityData is the worker function for parallel city data generation
func (l *ServiceImpl) processCityData(ctx context.Context, cityName string,
	config *genai.GenerateContentConfig, resultCh chan<- models.GenAIResponse, workerID int) {

	ctx, span := otel.Tracer("LlmInteractionService").Start(ctx, "processCityData", trace.WithAttributes(
		attribute.String("city.name", cityName),
		attribute.Int("worker.id", workerID),
	))
	defer span.End()

	prompt := getCityDescriptionPrompt(cityName)
	span.SetAttributes(attribute.Int("prompt.length", len(prompt)))

	response, err := l.aiClient.GenerateResponse(ctx, prompt, config)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to generate city data")
		resultCh <- models.GenAIResponse{Err: fmt.Errorf("failed to generate city data for %s: %w", cityName, err)}
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
		err := fmt.Errorf("no valid city data content from AI for %s", cityName)
		span.RecordError(err)
		span.SetStatus(codes.Error, "Empty response from AI")
		resultCh <- models.GenAIResponse{Err: err}
		return
	}
	span.SetAttributes(attribute.Int("response.length", len(txt)))

	cleanTxt := cleanJSONResponse(txt)
	var cityDataFromAI struct {
		CityName        string  `json:"city_name"`
		StateProvince   *string `json:"state_province"`
		Country         string  `json:"country"`
		CenterLatitude  float64 `json:"center_latitude"`
		CenterLongitude float64 `json:"center_longitude"`
		Description     string  `json:"description"`
	}
	if err := json.Unmarshal([]byte(cleanTxt), &cityDataFromAI); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to parse city data JSON")
		resultCh <- models.GenAIResponse{Err: fmt.Errorf("failed to parse city data JSON for %s: %w", cityName, err)}
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

	resultCh <- models.GenAIResponse{
		City:            cityDataFromAI.CityName,
		Country:         cityDataFromAI.Country,
		StateProvince:   stateProvinceValue,
		CityDescription: cityDataFromAI.Description,
		Latitude:        cityDataFromAI.CenterLatitude,
		Longitude:       cityDataFromAI.CenterLongitude,
	}
}

// GenerateGeneralPOIParallel processes general POI generation across multiple workers
func (l *ServiceImpl) GenerateGeneralPOIParallel(ctx context.Context,
	cityNames []string,
	config *genai.GenerateContentConfig) (map[string][]models.POIDetailedInfo, error) {

	ctx, span := otel.Tracer("LlmInteractionService").Start(ctx, "GenerateGeneralPOIParallel", trace.WithAttributes(
		attribute.Int("cities.count", len(cityNames)),
	))
	defer span.End()

	if len(cityNames) == 0 {
		return make(map[string][]models.POIDetailedInfo), nil
	}

	workerConfig := NewParallelWorkerConfig()
	numWorkers := workerConfig.NumWorkers
	if numWorkers > len(cityNames) {
		numWorkers = len(cityNames)
	}

	span.SetAttributes(attribute.Int("workers.count", numWorkers))

	// Create channels for work distribution and result collection
	workCh := make(chan string, len(cityNames))
	resultCh := make(chan poiResult, len(cityNames))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for cityName := range workCh {
				l.processGeneralPOI(ctx, cityName, config, resultCh, workerID)
			}
		}(i)
	}

	// Distribute work
	go func() {
		for _, cityName := range cityNames {
			workCh <- cityName
		}
		close(workCh)
	}()

	// Wait for workers to complete and close result channel
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results
	results := make(map[string][]models.POIDetailedInfo)
	for result := range resultCh {
		if result.err == nil {
			results[result.cityName] = result.pois
		}
	}

	span.SetAttributes(attribute.Int("cities_processed.count", len(results)))
	span.SetStatus(codes.Ok, "General POI generation completed")

	return results, nil
}

// processGeneralPOI is the worker function for parallel general POI generation
func (l *ServiceImpl) processGeneralPOI(ctx context.Context, cityName string,
	config *genai.GenerateContentConfig, resultCh chan<- poiResult, workerID int) {

	ctx, span := otel.Tracer("LlmInteractionService").Start(ctx, "processGeneralPOI", trace.WithAttributes(
		attribute.String("city.name", cityName),
		attribute.Int("worker.id", workerID),
	))
	defer span.End()

	prompt := getGeneralPOIPrompt(cityName)
	span.SetAttributes(attribute.Int("prompt.length", len(prompt)))

	startTime := time.Now()
	response, err := l.aiClient.GenerateResponse(ctx, prompt, config)
	latencyMs := int(time.Since(startTime).Milliseconds())
	span.SetAttributes(attribute.Int("response.latency_ms", latencyMs))

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to generate general POIs")
		resultCh <- struct {
			cityName string
			pois     []models.POIDetailedInfo
			err      error
		}{cityName: cityName, err: fmt.Errorf("failed to generate general POIs for %s: %w", cityName, err)}
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
		err := fmt.Errorf("no valid general POI content from AI for %s", cityName)
		span.RecordError(err)
		span.SetStatus(codes.Error, "Empty response from AI")
		resultCh <- struct {
			cityName string
			pois     []models.POIDetailedInfo
			err      error
		}{cityName: cityName, err: err}
		return
	}
	span.SetAttributes(attribute.Int("response.length", len(txt)))

	cleanTxt := cleanJSONResponse(txt)
	var poiData struct {
		PointsOfInterest []models.POIDetailedInfo `json:"points_of_interest"`
	}
	if err := json.Unmarshal([]byte(cleanTxt), &poiData); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to parse general POI JSON")
		resultCh <- struct {
			cityName string
			pois     []models.POIDetailedInfo
			err      error
		}{cityName: cityName, err: fmt.Errorf("failed to parse general POI JSON for %s: %w", cityName, err)}
		return
	}

	span.SetAttributes(attribute.Int("pois.count", len(poiData.PointsOfInterest)))
	span.SetStatus(codes.Ok, "General POIs generated successfully")
	resultCh <- struct {
		cityName string
		pois     []models.POIDetailedInfo
		err      error
	}{cityName: cityName, pois: poiData.PointsOfInterest}
}

// PersonalizedPOIRequest represents a request for personalized POI generation
type PersonalizedPOIRequest struct {
	CityName       string
	UserID         uuid.UUID
	ProfileID      uuid.UUID
	SessionID      uuid.UUID
	InterestNames  []string
	TagsPromptPart string
	UserPrefs      string
	SemanticPOIs   []models.POIDetailedInfo
}

// GeneratePersonalizedPOIParallel processes personalized POI generation across multiple workers
func (l *ServiceImpl) GeneratePersonalizedPOIParallel(ctx context.Context,
	requests []PersonalizedPOIRequest,
	config *genai.GenerateContentConfig) ([]models.GenAIResponse, error) {

	ctx, span := otel.Tracer("LlmInteractionService").Start(ctx, "GeneratePersonalizedPOIParallel", trace.WithAttributes(
		attribute.Int("requests.count", len(requests)),
	))
	defer span.End()

	if len(requests) == 0 {
		return []models.GenAIResponse{}, nil
	}

	workerConfig := NewParallelWorkerConfig()
	numWorkers := workerConfig.NumWorkers
	if numWorkers > len(requests) {
		numWorkers = len(requests)
	}

	span.SetAttributes(attribute.Int("workers.count", numWorkers))

	// Create channels for work distribution and result collection
	workCh := make(chan PersonalizedPOIRequest, len(requests))
	resultCh := make(chan models.GenAIResponse, len(requests))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for req := range workCh {
				if len(req.SemanticPOIs) > 0 {
					l.processPersonalizedPOIWithSemantics(ctx, req, config, resultCh, workerID)
				} else {
					l.processPersonalizedPOI(ctx, req, config, resultCh, workerID)
				}
			}
		}(i)
	}

	// Distribute work
	go func() {
		for _, req := range requests {
			workCh <- req
		}
		close(workCh)
	}()

	// Wait for workers to complete and close result channel
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results
	results := make([]models.GenAIResponse, 0, len(requests))
	for result := range resultCh {
		results = append(results, result)
	}

	span.SetAttributes(attribute.Int("results.count", len(results)))
	span.SetStatus(codes.Ok, "Personalized POI generation completed")

	return results, nil
}

// processPersonalizedPOI is the worker function for parallel personalized POI generation
func (l *ServiceImpl) processPersonalizedPOI(ctx context.Context, req PersonalizedPOIRequest,
	config *genai.GenerateContentConfig, resultCh chan<- models.GenAIResponse, workerID int) {

	ctx, span := otel.Tracer("LlmInteractionService").Start(ctx, "processPersonalizedPOI", trace.WithAttributes(
		attribute.String("city.name", req.CityName),
		attribute.String("user.id", req.UserID.String()),
		attribute.String("profile.id", req.ProfileID.String()),
		attribute.Int("worker.id", workerID),
		attribute.Int("interests.count", len(req.InterestNames)),
	))
	defer span.End()

	startTime := time.Now()

	prompt := getPersonalizedPOI(req.InterestNames, req.CityName, req.TagsPromptPart, req.UserPrefs)
	span.SetAttributes(attribute.Int("prompt.length", len(prompt)))

	response, err := l.aiClient.GenerateResponse(ctx, prompt, config)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to generate personalized itinerary")
		resultCh <- models.GenAIResponse{Err: fmt.Errorf("failed to generate personalized itinerary: %w", err)}
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
		resultCh <- models.GenAIResponse{Err: err}
		return
	}
	span.SetAttributes(attribute.Int("response.length", len(txt)))

	cleanTxt := cleanJSONResponse(txt)
	var itineraryData struct {
		ItineraryName      string                   `json:"itinerary_name"`
		OverallDescription string                   `json:"overall_description"`
		PointsOfInterest   []models.POIDetailedInfo `json:"points_of_interest"`
	}

	if err := json.Unmarshal([]byte(cleanTxt), &itineraryData); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to parse personalized itinerary JSON")
		resultCh <- models.GenAIResponse{Err: fmt.Errorf("failed to parse personalized itinerary JSON: %w", err)}
		return
	}
	span.SetAttributes(
		attribute.String("itinerary.name", itineraryData.ItineraryName),
		attribute.Int("personalized_pois.count", len(itineraryData.PointsOfInterest)),
	)

	latencyMs := int(time.Since(startTime).Milliseconds())
	span.SetAttributes(attribute.Int("response.latency_ms", latencyMs))

	interaction := models.LlmInteraction{
		UserID:       req.UserID,
		SessionID:    req.SessionID,
		Prompt:       prompt,
		ResponseText: txt,
		ModelUsed:    model,
		LatencyMs:    latencyMs,
		CityName:     req.CityName,
	}
	savedInteractionID, err := l.llmInteractionRepo.SaveInteraction(ctx, interaction)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to save LLM interaction")
		resultCh <- models.GenAIResponse{Err: fmt.Errorf("failed to save LLM interaction: %w", err)}
		return
	}
	span.SetAttributes(attribute.String("llm_interaction.id", savedInteractionID.String()))
	span.SetStatus(codes.Ok, "Personalized POIs generated successfully")

	resultCh <- models.GenAIResponse{
		ItineraryName:        itineraryData.ItineraryName,
		ItineraryDescription: itineraryData.OverallDescription,
		PersonalisedPOI:      itineraryData.PointsOfInterest,
		LlmInteractionID:     savedInteractionID,
	}
}

// processPersonalizedPOIWithSemantics is the worker function for parallel personalized POI generation with semantic enhancement
func (l *ServiceImpl) processPersonalizedPOIWithSemantics(ctx context.Context, req PersonalizedPOIRequest,
	config *genai.GenerateContentConfig, resultCh chan<- models.GenAIResponse, workerID int) {

	ctx, span := otel.Tracer("LlmInteractionService").Start(ctx, "processPersonalizedPOIWithSemantics", trace.WithAttributes(
		attribute.String("city.name", req.CityName),
		attribute.String("user.id", req.UserID.String()),
		attribute.String("profile.id", req.ProfileID.String()),
		attribute.Int("worker.id", workerID),
		attribute.Int("interests.count", len(req.InterestNames)),
		attribute.Int("semantic_pois.count", len(req.SemanticPOIs)),
	))
	defer span.End()

	startTime := time.Now()

	// Create enhanced prompt with semantic context
	prompt := l.getPersonalizedPOIWithSemanticContext(req.InterestNames, req.CityName, req.TagsPromptPart, req.UserPrefs, req.SemanticPOIs)
	span.SetAttributes(attribute.Int("prompt.length", len(prompt)))

	response, err := l.aiClient.GenerateResponse(ctx, prompt, config)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to generate semantic-enhanced personalized itinerary")
		resultCh <- models.GenAIResponse{Err: fmt.Errorf("failed to generate semantic-enhanced personalized itinerary: %w", err)}
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
		resultCh <- models.GenAIResponse{Err: err}
		return
	}
	span.SetAttributes(attribute.Int("response.length", len(txt)))

	cleanTxt := cleanJSONResponse(txt)
	var itineraryData struct {
		ItineraryName      string                   `json:"itinerary_name"`
		OverallDescription string                   `json:"overall_description"`
		PointsOfInterest   []models.POIDetailedInfo `json:"points_of_interest"`
	}

	if err := json.Unmarshal([]byte(cleanTxt), &itineraryData); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to parse semantic-enhanced personalized itinerary JSON")
		resultCh <- models.GenAIResponse{Err: fmt.Errorf("failed to parse semantic-enhanced personalized itinerary JSON: %w", err)}
		return
	}
	span.SetAttributes(
		attribute.String("itinerary.name", itineraryData.ItineraryName),
		attribute.Int("personalized_pois.count", len(itineraryData.PointsOfInterest)),
	)

	latencyMs := int(time.Since(startTime).Milliseconds())
	span.SetAttributes(attribute.Int("response.latency_ms", latencyMs))

	interaction := models.LlmInteraction{
		UserID:       req.UserID,
		SessionID:    req.SessionID,
		Prompt:       prompt,
		ResponseText: txt,
		ModelUsed:    model,
		LatencyMs:    latencyMs,
		CityName:     req.CityName,
	}
	savedInteractionID, err := l.llmInteractionRepo.SaveInteraction(ctx, interaction)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to save semantic-enhanced LLM interaction")
		resultCh <- models.GenAIResponse{Err: fmt.Errorf("failed to save semantic-enhanced LLM interaction: %w", err)}
		return
	}
	span.SetAttributes(attribute.String("llm_interaction.id", savedInteractionID.String()))
	span.SetStatus(codes.Ok, "Semantic-enhanced personalized POIs generated successfully")

	resultCh <- models.GenAIResponse{
		ItineraryName:        itineraryData.ItineraryName,
		ItineraryDescription: itineraryData.OverallDescription,
		PersonalisedPOI:      itineraryData.PointsOfInterest,
		LlmInteractionID:     savedInteractionID,
	}
}
