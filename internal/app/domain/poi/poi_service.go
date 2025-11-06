package poi

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"google.golang.org/genai"

	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	generativeAI "github.com/FACorreiaa/go-genai-sdk/lib"

	"github.com/FACorreiaa/go-templui/internal/app/domain/city"
	"github.com/FACorreiaa/go-templui/internal/app/models"
	"github.com/FACorreiaa/go-templui/internal/pkg/llmlogging"
	"github.com/FACorreiaa/go-templui/internal/pkg/middleware"
)

var _ Service = (*ServiceImpl)(nil)

// Service defines the business logic contract for POI operations.
type Service interface {
	// Favourites methods - POIs
	AddPoiToFavourites(ctx context.Context, userID, poiID uuid.UUID, isLLMGenerated bool) (uuid.UUID, error)
	RemovePoiFromFavourites(ctx context.Context, userID, poiID uuid.UUID, isLLMGenerated bool) error
	GetFavouritePOIsByUserID(ctx context.Context, userID uuid.UUID) ([]models.POIDetailedInfo, error)
	GetFavouritePOIsByUserIDPaginated(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.POIDetailedInfo, int, error)
	GetFavouritesFiltered(ctx context.Context, filter models.FavouritesFilter) ([]models.POIDetailedInfo, int, error)
	GetPOIsByCityID(ctx context.Context, cityID uuid.UUID) ([]models.POIDetailedInfo, error)
	CheckIsFavorited(ctx context.Context, userID, poiID uuid.UUID, isLLMGenerated bool) (bool, error)

	// Favourites methods - Hotels
	AddHotelToFavourites(ctx context.Context, userID, hotelID uuid.UUID) (uuid.UUID, error)
	RemoveHotelFromFavourites(ctx context.Context, userID, hotelID uuid.UUID) error
	CheckIsHotelFavorited(ctx context.Context, userID, hotelID uuid.UUID) (bool, error)

	// Favourites methods - Restaurants
	AddRestaurantToFavourites(ctx context.Context, userID, restaurantID uuid.UUID) (uuid.UUID, error)
	RemoveRestaurantFromFavourites(ctx context.Context, userID, restaurantID uuid.UUID) error
	CheckIsRestaurantFavorited(ctx context.Context, userID, restaurantID uuid.UUID) (bool, error)

	// Bookmarks methods
	AddItineraryToBookmarks(ctx context.Context, userID, itineraryID uuid.UUID) (uuid.UUID, error)
	RemoveItineraryFromBookmarks(ctx context.Context, userID, itineraryID uuid.UUID) error
	GetBookmarksFiltered(ctx context.Context, filter models.BookmarksFilter) ([]models.SavedItinerary, int, error)
	CheckIsBookmarked(ctx context.Context, userID, itineraryID uuid.UUID) (bool, error)

	// SearchPOIs Traditional search
	SearchPOIs(ctx context.Context, filter models.POIFilter) ([]models.POIDetailedInfo, error)

	// SearchPOIsSemantic Semantic search methods
	SearchPOIsSemantic(ctx context.Context, query string, limit int) ([]models.POIDetailedInfo, error)
	SearchPOIsSemanticByCity(ctx context.Context, query string, cityID uuid.UUID, limit int) ([]models.POIDetailedInfo, error)
	SearchPOIsHybrid(ctx context.Context, filter models.POIFilter, query string, semanticWeight float64) ([]models.POIDetailedInfo, error)
	GenerateEmbeddingForPOI(ctx context.Context, poiID uuid.UUID) error
	GenerateEmbeddingsForAllPOIs(ctx context.Context, batchSize int) error

	// GetItinerary Itinerary management
	GetItinerary(ctx context.Context, userID, itineraryID uuid.UUID) (*models.UserSavedItinerary, error)
	GetItineraries(ctx context.Context, userID uuid.UUID, page, pageSize int) (*models.PaginatedUserItinerariesResponse, error)
	UpdateItinerary(ctx context.Context, userID, itineraryID uuid.UUID, updates models.UpdateItineraryRequest) (*models.UserSavedItinerary, error)

	// GetGeneralPOIByDistance Discover Service
	GetGeneralPOIByDistance(ctx context.Context, userID uuid.UUID, lat, lon, distance float64) ([]models.POIDetailedInfo, error) //, categoryFilter string

	// GetNearbyRestaurants Domain-specific discover services
	GetNearbyRestaurants(ctx context.Context, userID uuid.UUID, lat, lon, distance float64, cuisineType, priceRange string) ([]models.POIDetailedInfo, error)
	GetNearbyActivities(ctx context.Context, userID uuid.UUID, lat, lon, distance float64, activityType, duration string) ([]models.POIDetailedInfo, error)
	GetNearbyHotels(ctx context.Context, userID uuid.UUID, lat, lon, distance float64, starRating, amenities string) ([]models.POIDetailedInfo, error)
	GetNearbyAttractions(ctx context.Context, userID uuid.UUID, lat, lon, distance float64, attractionType, isOutdoor string) ([]models.POIDetailedInfo, error)

	// FindOrCreateLLMPOI LLM POI management
	FindOrCreateLLMPOI(ctx context.Context, poiData *models.POIDetailedInfo) (uuid.UUID, error)
}

type ServiceImpl struct {
	logger             *zap.Logger
	poiRepository      Repository
	embeddingService   *generativeAI.EmbeddingService
	aiClient           *generativeAI.LLMChatClient
	cityRepo           city.Repository
	cache              *cache.Cache
	llmInteractionRepo llmlogging.Repository
}

func NewServiceImpl(poiRepository Repository,
	embeddingService *generativeAI.EmbeddingService,
	cityRepo city.Repository,
	llmInteractionRepo llmlogging.Repository,
	logger *zap.Logger) *ServiceImpl {
	apiKey := os.Getenv("GEMINI_API_KEY")
	aiClient, err := generativeAI.NewLLMChatClient(context.Background(), apiKey)
	if err != nil {
		logger.Error("Failed to initialize AI client", zap.Any("error", err))
		// For now, set to nil and handle gracefully in methods
		aiClient = nil
	}

	return &ServiceImpl{
		logger:             logger,
		poiRepository:      poiRepository,
		aiClient:           aiClient,
		cityRepo:           cityRepo,
		cache:              cache.New(5*time.Minute, 10*time.Minute),
		embeddingService:   embeddingService,
		llmInteractionRepo: llmInteractionRepo,
	}
}

// Helper functions for LLM logging without circular dependency

// calculateCost estimates the cost in USD for an LLM interaction
func calculateCost(modelName string, promptTokens, completionTokens int) float64 {
	// Gemini pricing (as of 2024)
	pricing := map[string]struct {
		InputPer1M  float64
		OutputPer1M float64
	}{
		"gemini-1.5-pro":   {InputPer1M: 3.50, OutputPer1M: 10.50},
		"gemini-1.5-flash": {InputPer1M: 0.075, OutputPer1M: 0.30},
		"gemini-2.0-flash": {InputPer1M: 0.10, OutputPer1M: 0.40},
	}

	normalizedModel := strings.ToLower(modelName)
	for key, p := range pricing {
		if strings.Contains(normalizedModel, strings.ToLower(key)) {
			inputCost := (float64(promptTokens) / 1_000_000) * p.InputPer1M
			outputCost := (float64(completionTokens) / 1_000_000) * p.OutputPer1M
			return inputCost + outputCost
		}
	}
	return 0
}

// logLLMInteractionAsync logs an LLM interaction asynchronously
func (s *ServiceImpl) logLLMInteractionAsync(ctx context.Context, userID, sessionID uuid.UUID, intent, searchType, prompt, modelName, provider, responseText, errorMessage string, temperature *float32, promptTokens, completionTokens, totalTokens, statusCode int, latencyMs int64) {
	// Create a new context for async operation to avoid cancellation when request ends
	asyncCtx := context.WithoutCancel(ctx)

	go func() {
		cost := calculateCost(modelName, promptTokens, completionTokens)

		interaction := models.LlmInteraction{
			RequestID:        uuid.New(),
			SessionID:        sessionID,
			UserID:           userID,
			Prompt:           prompt,
			ResponseText:     responseText,
			ModelUsed:        modelName,
			Provider:         provider,
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      totalTokens,
			LatencyMs:        int(latencyMs),
			StatusCode:       statusCode,
			ErrorMessage:     errorMessage,
			Intent:           intent,
			SearchType:       searchType,
			Temperature:      temperature,
			CostEstimateUSD:  &cost,
			IsStreaming:      false,
			Timestamp:        time.Now(),
		}

		if _, err := s.llmInteractionRepo.SaveInteraction(asyncCtx, interaction); err != nil {
			s.logger.Error("Failed to log LLM interaction asynchronously",
				zap.String("intent", intent),
				zap.String("session_id", sessionID.String()),
				zap.Any("error", err))
		}
	}()
}

func (s *ServiceImpl) AddPoiToFavourites(ctx context.Context, userID, poiID uuid.UUID, isLLMGenerated bool) (uuid.UUID, error) {
	var id uuid.UUID
	if !isLLMGenerated {

		id, err := s.poiRepository.AddPoiToFavourites(ctx, userID, poiID)
		if err != nil {
			s.logger.Error("failed to add POI to favourites", zap.Error(err))
			return uuid.Nil, err
		}
		return id, nil
	}

	id, err := s.poiRepository.AddLLMPoiToFavourite(ctx, userID, poiID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert favorite LLM POI: %w", err)
	}

	return id, nil
}

func (s *ServiceImpl) RemovePoiFromFavourites(ctx context.Context, userID, poiID uuid.UUID, isLLMGenerated bool) error {
	if isLLMGenerated {
		err := s.poiRepository.RemoveLLMPoiFromFavourite(ctx, userID, poiID)
		if err != nil {
			s.logger.Error("failed to remove LLM POI from favourites", zap.Error(err))
			return err
		}
	} else {
		err := s.poiRepository.RemovePoiFromFavourites(ctx, userID, poiID)
		if err != nil {
			s.logger.Error("failed to remove POI from favourites", zap.Error(err))
			return err
		}
	}
	return nil
}

func (s *ServiceImpl) CheckIsFavorited(ctx context.Context, userID, poiID uuid.UUID, isLLMGenerated bool) (bool, error) {
	isFavorited, err := s.poiRepository.CheckIsFavorited(ctx, userID, poiID, isLLMGenerated)
	if err != nil {
		s.logger.Error("failed to check if POI is favorited", zap.Error(err))
		return false, err
	}
	return isFavorited, nil
}

// Hotel Favorites Methods
func (s *ServiceImpl) AddHotelToFavourites(ctx context.Context, userID, hotelID uuid.UUID) (uuid.UUID, error) {
	id, err := s.poiRepository.AddHotelToFavourites(ctx, userID, hotelID)
	if err != nil {
		s.logger.Error("failed to add hotel to favourites", zap.Error(err))
		return uuid.Nil, err
	}
	return id, nil
}

func (s *ServiceImpl) RemoveHotelFromFavourites(ctx context.Context, userID, hotelID uuid.UUID) error {
	err := s.poiRepository.RemoveHotelFromFavourites(ctx, userID, hotelID)
	if err != nil {
		s.logger.Error("failed to remove hotel from favourites", zap.Error(err))
		return err
	}
	return nil
}

func (s *ServiceImpl) CheckIsHotelFavorited(ctx context.Context, userID, hotelID uuid.UUID) (bool, error) {
	isFavorited, err := s.poiRepository.CheckIsHotelFavorited(ctx, userID, hotelID)
	if err != nil {
		s.logger.Error("failed to check if hotel is favorited", zap.Error(err))
		return false, err
	}
	return isFavorited, nil
}

// Restaurant Favorites Methods
func (s *ServiceImpl) AddRestaurantToFavourites(ctx context.Context, userID, restaurantID uuid.UUID) (uuid.UUID, error) {
	id, err := s.poiRepository.AddRestaurantToFavourites(ctx, userID, restaurantID)
	if err != nil {
		s.logger.Error("failed to add restaurant to favourites", zap.Error(err))
		return uuid.Nil, err
	}
	return id, nil
}

func (s *ServiceImpl) RemoveRestaurantFromFavourites(ctx context.Context, userID, restaurantID uuid.UUID) error {
	err := s.poiRepository.RemoveRestaurantFromFavourites(ctx, userID, restaurantID)
	if err != nil {
		s.logger.Error("failed to remove restaurant from favourites", zap.Error(err))
		return err
	}
	return nil
}

func (s *ServiceImpl) CheckIsRestaurantFavorited(ctx context.Context, userID, restaurantID uuid.UUID) (bool, error) {
	isFavorited, err := s.poiRepository.CheckIsRestaurantFavorited(ctx, userID, restaurantID)
	if err != nil {
		s.logger.Error("failed to check if restaurant is favorited", zap.Error(err))
		return false, err
	}
	return isFavorited, nil
}

func (s *ServiceImpl) GetFavouritePOIsByUserID(ctx context.Context, userID uuid.UUID) ([]models.POIDetailedInfo, error) {
	pois, err := s.poiRepository.GetFavouritePOIsByUserID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get favourite POIs by user ID", zap.Error(err))
		return nil, err
	}
	return pois, nil
}

func (s *ServiceImpl) GetFavouritePOIsByUserIDPaginated(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.POIDetailedInfo, int, error) {
	pois, total, err := s.poiRepository.GetFavouritePOIsByUserIDPaginated(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("failed to get paginated favourite POIs by user ID", zap.Error(err))
		return nil, 0, err
	}
	return pois, total, nil
}

// GetFavouritesFiltered retrieves favourites with search and filter support
func (s *ServiceImpl) GetFavouritesFiltered(ctx context.Context, filter models.FavouritesFilter) ([]models.POIDetailedInfo, int, error) {
	// Set default pagination if not provided
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	pois, total, err := s.poiRepository.GetFavouritesFiltered(ctx, filter)
	if err != nil {
		s.logger.Error("failed to get filtered favourites",
			zap.Int("limit", filter.Limit),
			zap.Any("userID", filter.UserID),
			zap.String("search", filter.SearchText),
			zap.String("category", filter.Category),
			zap.Error(err),
		)
		return nil, 0, err
	}

	s.logger.Info("Retrieved filtered favourites",
		zap.Any("userID", filter.UserID),
		zap.Int("count", len(pois)),
		zap.Int("total", total),
	)

	return pois, total, nil
}

func (s *ServiceImpl) GetPOIsByCityID(ctx context.Context, cityID uuid.UUID) ([]models.POIDetailedInfo, error) {
	pois, err := s.poiRepository.GetPOIsByCityID(ctx, cityID)
	if err != nil {
		s.logger.Error("failed to get POIs by city ID", zap.Error(err))
		return nil, err
	}
	return pois, nil
}

func (s *ServiceImpl) SearchPOIs(ctx context.Context, filter models.POIFilter) ([]models.POIDetailedInfo, error) {
	pois, err := s.poiRepository.SearchPOIs(ctx, filter)
	if err != nil {
		s.logger.Error("failed to search POIs", zap.Error(err))
		return nil, err
	}
	return pois, nil
}

func (s *ServiceImpl) GetItinerary(ctx context.Context, userID, itineraryID uuid.UUID) (*models.UserSavedItinerary, error) {
	ctx, span := otel.Tracer("LlmInteractionService").Start(ctx, "GetItinerary")
	defer span.End()

	itinerary, err := s.poiRepository.GetItinerary(ctx, userID, itineraryID)
	if err != nil {
		s.logger.Error("Repository failed to get itinerary", zap.Any("error", err))
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get itinerary: %w", err)
	}
	if itinerary == nil {
		return nil, fmt.Errorf("itinerary not found")
	}

	span.SetStatus(codes.Ok, "Itinerary retrieved successfully")
	return itinerary, nil
}

func (s *ServiceImpl) GetItineraries(ctx context.Context, userID uuid.UUID, page, pageSize int) (*models.PaginatedUserItinerariesResponse, error) {
	_, span := otel.Tracer("LlmInteractionService").Start(ctx, "GetItineraries", trace.WithAttributes(
		attribute.String("user.id", userID.String()),
		attribute.Int("page", page),
		attribute.Int("page_size", pageSize),
	))
	defer span.End()

	s.logger.Debug("Service: Getting itineraries for user", zap.String("userID", userID.String()))

	if page <= 0 {
		page = 1 // Default to page 1
	}
	if pageSize <= 0 {
		pageSize = 10 // Default page size
	}

	itineraries, totalRecords, err := s.poiRepository.GetItineraries(ctx, userID, page, pageSize)
	if err != nil {
		s.logger.Error("Repository failed to get itineraries", zap.Any("error", err))
		span.RecordError(err)
		return nil, fmt.Errorf("failed to retrieve itineraries: %w", err)
	}

	span.SetAttributes(attribute.Int("itineraries.count", len(itineraries)), attribute.Int("total_records", totalRecords))
	span.SetStatus(codes.Ok, "Itineraries retrieved")

	return &models.PaginatedUserItinerariesResponse{
		Itineraries:  itineraries,
		TotalRecords: totalRecords,
		Page:         page,
		PageSize:     pageSize,
	}, nil
}

func (s *ServiceImpl) UpdateItinerary(ctx context.Context, userID, itineraryID uuid.UUID, updates models.UpdateItineraryRequest) (*models.UserSavedItinerary, error) {
	_, span := otel.Tracer("LlmInteractionService").Start(ctx, "UpdateItinerary", trace.WithAttributes(
		attribute.String("user.id", userID.String()),
		attribute.String("itinerary.id", itineraryID.String()),
	))
	defer span.End()

	s.logger.Debug("Service: Updating itinerary", zap.String("userID", userID.String()), zap.String("itineraryID", itineraryID.String()), zap.Any("updates", updates))

	if updates.Title == nil && updates.Description == nil && updates.Tags == nil &&
		updates.EstimatedDurationDays == nil && updates.EstimatedCostLevel == nil &&
		updates.IsPublic == nil && updates.MarkdownContent == nil {
		span.AddEvent("No update fields provided.")
		s.logger.Info("No fields provided for itinerary update, fetching current.", zap.String("itineraryID", itineraryID.String()))
		return s.poiRepository.GetItinerary(ctx, userID, itineraryID) // Assumes GetItinerary checks ownership
	}

	updatedItinerary, err := s.poiRepository.UpdateItinerary(ctx, userID, itineraryID, updates)
	if err != nil {
		s.logger.Error("Repository failed to update itinerary", zap.Any("error", err))
		span.RecordError(err)
		return nil, err // Propagate error (could be not found, or DB error)
	}

	span.SetStatus(codes.Ok, "Itinerary updated")
	return updatedItinerary, nil
}

func (s *ServiceImpl) AddItineraryToBookmarks(ctx context.Context, userID, itineraryID uuid.UUID) (uuid.UUID, error) {
	_, span := otel.Tracer("LlmInteractionService").Start(ctx, "AddItineraryToBookmarks", trace.WithAttributes(
		attribute.String("user.id", userID.String()),
		attribute.String("itinerary.id", itineraryID.String()),
	))
	defer span.End()

	s.logger.Debug("Service: Adding itinerary to bookmarks", zap.String("userID", userID.String()), zap.String("itineraryID", itineraryID.String()))

	bookmarkID, err := s.poiRepository.AddItineraryToBookmarks(ctx, userID, itineraryID)
	if err != nil {
		s.logger.Error("Repository failed to add itinerary to bookmarks", zap.Any("error", err))
		span.RecordError(err)
		return uuid.Nil, err
	}

	span.SetStatus(codes.Ok, "Itinerary added to bookmarks")
	return bookmarkID, nil
}

// SearchPOIsSemantic performs semantic search for POIs using natural language queries
func (s *ServiceImpl) SearchPOIsSemantic(ctx context.Context, query string, limit int) ([]models.POIDetailedInfo, error) {
	ctx, span := otel.Tracer("POIService").Start(ctx, "SearchPOIsSemantic", trace.WithAttributes(
		attribute.String("query", query),
		attribute.Int("limit", limit),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "SearchPOIsSemantic"))

	if s.embeddingService == nil {
		err := fmt.Errorf("embedding service not available")
		l.Error("Embedding service not initialized", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Embedding service not available")
		return nil, err
	}

	// Build cache key for exact match
	cacheKey := middleware.BuildVectorCacheKey(query, "", nil)

	// Check for exact cache hit
	if cachedEntry, found := middleware.Cache.VectorSearch.Get(cacheKey); found {
		l.Info("Vector cache hit (exact)",
			zap.String("query", query),
			zap.Int("cached_results", len(cachedEntry.Results)))
		span.SetAttributes(
			attribute.Bool("cache.hit", true),
			attribute.String("cache.type", "exact"),
		)
		span.SetStatus(codes.Ok, "Semantic search completed (cached)")
		return cachedEntry.Results, nil
	}

	// Check embedding cache for query embedding
	embeddingKey := fmt.Sprintf("query:%s", query)
	var queryEmbedding []float32
	var err error

	if cachedEmbedding, found := middleware.Cache.Embeddings.Get(embeddingKey); found {
		queryEmbedding = cachedEmbedding
		l.Debug("Query embedding retrieved from cache", zap.String("query", query))
		span.SetAttributes(attribute.Bool("embedding.cached", true))
	} else {
		// Generate embedding for the query
		queryEmbedding, err = s.embeddingService.GenerateQueryEmbedding(ctx, query)
		if err != nil {
			l.Error("Failed to generate query embedding",
				zap.Any("error", err),
				zap.String("query", query))
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to generate query embedding")
			return nil, fmt.Errorf("failed to generate query embedding: %w", err)
		}

		// Cache the embedding for future use
		middleware.Cache.Embeddings.Set(embeddingKey, queryEmbedding, fmt.Sprintf("query: %s", query))
		span.SetAttributes(attribute.Bool("embedding.cached", false))
	}

	// Check for semantic cache hit (similar queries)
	if cachedEntry, similarity, found := middleware.Cache.VectorSearch.GetSimilar(queryEmbedding, "", nil); found {
		l.Info("Vector cache hit (semantic)",
			zap.String("query", query),
			zap.String("cached_query", cachedEntry.QueryText),
			zap.Float64("similarity", similarity),
			zap.Int("cached_results", len(cachedEntry.Results)))
		span.SetAttributes(
			attribute.Bool("cache.hit", true),
			attribute.String("cache.type", "semantic"),
			attribute.Float64("cache.similarity", similarity),
		)
		span.SetStatus(codes.Ok, "Semantic search completed (semantic cache)")
		return cachedEntry.Results, nil
	}

	// Cache miss - perform actual vector search
	l.Debug("Vector cache miss, performing database search", zap.String("query", query))
	span.SetAttributes(attribute.Bool("cache.hit", false))

	// Search for similar POIs
	pois, err := s.poiRepository.FindSimilarPOIs(ctx, queryEmbedding, limit)
	if err != nil {
		l.Error("Failed to find similar POIs", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to find similar POIs")
		return nil, fmt.Errorf("failed to find similar POIs: %w", err)
	}

	// Store results in vector cache
	cacheEntry := &middleware.VectorCacheEntry{
		QueryText:    query,
		Embedding:    queryEmbedding,
		Results:      pois,
		SearchParams: nil,
		CityID:       "",
	}
	middleware.Cache.VectorSearch.Set(cacheKey, cacheEntry)

	l.Info("Semantic search completed",
		zap.String("query", query),
		zap.Int("results", len(pois)))
	span.SetAttributes(
		attribute.String("query", query),
		attribute.Int("results.count", len(pois)),
	)
	span.SetStatus(codes.Ok, "Semantic search completed")

	return pois, nil
}

// SearchPOIsSemanticByCity performs semantic search for POIs within a specific city
func (s *ServiceImpl) SearchPOIsSemanticByCity(ctx context.Context, query string, cityID uuid.UUID, limit int) ([]models.POIDetailedInfo, error) {
	ctx, span := otel.Tracer("POIService").Start(ctx, "SearchPOIsSemanticByCity", trace.WithAttributes(
		attribute.String("query", query),
		attribute.String("city.id", cityID.String()),
		attribute.Int("limit", limit),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "SearchPOIsSemanticByCity"))

	if s.embeddingService == nil {
		err := fmt.Errorf("embedding service not available")
		l.Error("Embedding service not initialized", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Embedding service not available")
		return nil, err
	}

	// Build cache key for exact match (includes city filter)
	cacheKey := middleware.BuildVectorCacheKey(query, cityID.String(), nil)

	// Check for exact cache hit
	if cachedEntry, found := middleware.Cache.VectorSearch.Get(cacheKey); found {
		l.Info("Vector cache hit (exact) for city",
			zap.String("query", query),
			zap.String("city_id", cityID.String()),
			zap.Int("cached_results", len(cachedEntry.Results)))
		span.SetAttributes(
			attribute.Bool("cache.hit", true),
			attribute.String("cache.type", "exact"),
		)
		span.SetStatus(codes.Ok, "Semantic search by city completed (cached)")
		return cachedEntry.Results, nil
	}

	// Check embedding cache for query embedding
	embeddingKey := fmt.Sprintf("query:%s", query)
	var queryEmbedding []float32
	var err error

	if cachedEmbedding, found := middleware.Cache.Embeddings.Get(embeddingKey); found {
		queryEmbedding = cachedEmbedding
		l.Debug("Query embedding retrieved from cache",
			zap.String("query", query),
			zap.String("city_id", cityID.String()))
		span.SetAttributes(attribute.Bool("embedding.cached", true))
	} else {
		// Generate embedding for the query
		queryEmbedding, err = s.embeddingService.GenerateQueryEmbedding(ctx, query)
		if err != nil {
			l.Error("Failed to generate query embedding",
				zap.Any("error", err),
				zap.String("query", query))
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to generate query embedding")
			return nil, fmt.Errorf("failed to generate query embedding: %w", err)
		}

		// Cache the embedding for future use
		middleware.Cache.Embeddings.Set(embeddingKey, queryEmbedding, fmt.Sprintf("query: %s", query))
		span.SetAttributes(attribute.Bool("embedding.cached", false))
	}

	// Check for semantic cache hit (similar queries in same city)
	if cachedEntry, similarity, found := middleware.Cache.VectorSearch.GetSimilar(queryEmbedding, cityID.String(), nil); found {
		l.Info("Vector cache hit (semantic) for city",
			zap.String("query", query),
			zap.String("cached_query", cachedEntry.QueryText),
			zap.String("city_id", cityID.String()),
			zap.Float64("similarity", similarity),
			zap.Int("cached_results", len(cachedEntry.Results)))
		span.SetAttributes(
			attribute.Bool("cache.hit", true),
			attribute.String("cache.type", "semantic"),
			attribute.Float64("cache.similarity", similarity),
		)
		span.SetStatus(codes.Ok, "Semantic search by city completed (semantic cache)")
		return cachedEntry.Results, nil
	}

	// Cache miss - perform actual vector search
	l.Debug("Vector cache miss for city, performing database search",
		zap.String("query", query),
		zap.String("city_id", cityID.String()))
	span.SetAttributes(attribute.Bool("cache.hit", false))

	// Search for similar POIs in the specified city
	pois, err := s.poiRepository.FindSimilarPOIsByCity(ctx, queryEmbedding, cityID, limit)
	if err != nil {
		l.Error("Failed to find similar POIs by city", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to find similar POIs by city")
		return nil, fmt.Errorf("failed to find similar POIs by city: %w", err)
	}

	// Store results in vector cache
	cacheEntry := &middleware.VectorCacheEntry{
		QueryText:    query,
		Embedding:    queryEmbedding,
		Results:      pois,
		SearchParams: nil,
		CityID:       cityID.String(),
	}
	middleware.Cache.VectorSearch.Set(cacheKey, cacheEntry)

	l.Info("Semantic search by city completed",
		zap.String("query", query),
		zap.String("city_id", cityID.String()),
		zap.Int("results", len(pois)))
	span.SetAttributes(
		attribute.String("query", query),
		attribute.String("city.id", cityID.String()),
		attribute.Int("results.count", len(pois)),
	)
	span.SetStatus(codes.Ok, "Semantic search by city completed")

	return pois, nil
}

// SearchPOIsHybrid performs hybrid search combining spatial and semantic similarity
func (s *ServiceImpl) SearchPOIsHybrid(ctx context.Context, filter models.POIFilter, query string, semanticWeight float64) ([]models.POIDetailedInfo, error) {
	ctx, span := otel.Tracer("POIService").Start(ctx, "SearchPOIsHybrid", trace.WithAttributes(
		attribute.String("query", query),
		attribute.Float64("semantic.weight", semanticWeight),
		attribute.Float64("location.latitude", filter.Location.Latitude),
		attribute.Float64("location.longitude", filter.Location.Longitude),
		attribute.Float64("radius", filter.Radius),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "SearchPOIsHybrid"))

	if s.embeddingService == nil {
		err := fmt.Errorf("embedding service not available")
		l.Error("Embedding service not initialized", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Embedding service not available")
		return nil, err
	}

	// Validate semantic weight
	if semanticWeight < 0 || semanticWeight > 1 {
		err := fmt.Errorf("semantic weight must be between 0 and 1, got: %f", semanticWeight)
		l.Error("Invalid semantic weight", zap.Float64("semantic_weight", semanticWeight))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid semantic weight")
		return nil, err
	}

	// Build search params for cache key
	searchParams := map[string]interface{}{
		"latitude":        filter.Location.Latitude,
		"longitude":       filter.Location.Longitude,
		"radius":          filter.Radius,
		"semantic_weight": semanticWeight,
		"category":        filter.Category,
	}

	// Build cache key for exact match (no cityID in POIFilter)
	cacheKey := middleware.BuildVectorCacheKey(query, "", searchParams)

	// Check for exact cache hit
	if cachedEntry, found := middleware.Cache.VectorSearch.Get(cacheKey); found {
		l.Info("Vector cache hit (exact) for hybrid search",
			zap.String("query", query),
			zap.Float64("semantic_weight", semanticWeight),
			zap.Int("cached_results", len(cachedEntry.Results)))
		span.SetAttributes(
			attribute.Bool("cache.hit", true),
			attribute.String("cache.type", "exact"),
		)
		span.SetStatus(codes.Ok, "Hybrid search completed (cached)")
		return cachedEntry.Results, nil
	}

	// Check embedding cache for query embedding
	embeddingKey := fmt.Sprintf("query:%s", query)
	var queryEmbedding []float32
	var err error

	if cachedEmbedding, found := middleware.Cache.Embeddings.Get(embeddingKey); found {
		queryEmbedding = cachedEmbedding
		l.Debug("Query embedding retrieved from cache for hybrid search",
			zap.String("query", query))
		span.SetAttributes(attribute.Bool("embedding.cached", true))
	} else {
		// Generate embedding for the query
		queryEmbedding, err = s.embeddingService.GenerateQueryEmbedding(ctx, query)
		if err != nil {
			l.Error("Failed to generate query embedding",
				zap.Any("error", err),
				zap.String("query", query))
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to generate query embedding")
			return nil, fmt.Errorf("failed to generate query embedding: %w", err)
		}

		// Cache the embedding for future use
		middleware.Cache.Embeddings.Set(embeddingKey, queryEmbedding, fmt.Sprintf("query: %s", query))
		span.SetAttributes(attribute.Bool("embedding.cached", false))
	}

	// Check for semantic cache hit (similar queries with matching params)
	if cachedEntry, similarity, found := middleware.Cache.VectorSearch.GetSimilar(queryEmbedding, "", searchParams); found {
		l.Info("Vector cache hit (semantic) for hybrid search",
			zap.String("query", query),
			zap.String("cached_query", cachedEntry.QueryText),
			zap.Float64("similarity", similarity),
			zap.Float64("semantic_weight", semanticWeight),
			zap.Int("cached_results", len(cachedEntry.Results)))
		span.SetAttributes(
			attribute.Bool("cache.hit", true),
			attribute.String("cache.type", "semantic"),
			attribute.Float64("cache.similarity", similarity),
		)
		span.SetStatus(codes.Ok, "Hybrid search completed (semantic cache)")
		return cachedEntry.Results, nil
	}

	// Cache miss - perform actual hybrid search
	l.Debug("Vector cache miss for hybrid search, performing database search",
		zap.String("query", query),
		zap.Float64("semantic_weight", semanticWeight))
	span.SetAttributes(attribute.Bool("cache.hit", false))

	// Perform hybrid search
	pois, err := s.poiRepository.SearchPOIsHybrid(ctx, filter, queryEmbedding, semanticWeight)
	if err != nil {
		l.Error("Failed to perform hybrid search", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to perform hybrid search")
		return nil, fmt.Errorf("failed to perform hybrid search: %w", err)
	}

	// Store results in vector cache
	cacheEntry := &middleware.VectorCacheEntry{
		QueryText:    query,
		Embedding:    queryEmbedding,
		Results:      pois,
		SearchParams: searchParams,
		CityID:       "",
	}
	middleware.Cache.VectorSearch.Set(cacheKey, cacheEntry)

	l.Info("Hybrid search completed",
		zap.String("query", query),
		zap.Float64("semantic_weight", semanticWeight),
		zap.Int("results", len(pois)))
	span.SetAttributes(
		attribute.String("query", query),
		attribute.Float64("semantic.weight", semanticWeight),
		attribute.Int("results.count", len(pois)),
	)
	span.SetStatus(codes.Ok, "Hybrid search completed")

	return pois, nil
}

// GenerateEmbeddingForPOI generates and stores embedding for a specific POI
func (s *ServiceImpl) GenerateEmbeddingForPOI(ctx context.Context, poiID uuid.UUID) error {
	ctx, span := otel.Tracer("POIService").Start(ctx, "GenerateEmbeddingForPOI", trace.WithAttributes(
		attribute.String("poi.id", poiID.String()),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "GenerateEmbeddingForPOI"))

	if s.embeddingService == nil {
		err := fmt.Errorf("embedding service not available")
		l.Error("Embedding service not initialized", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Embedding service not available")
		return err
	}

	// Get POI details to generate embedding
	pois, err := s.poiRepository.GetPOIsWithoutEmbeddings(ctx, 1)
	if err != nil {
		l.Error("Failed to get POI details", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get POI details")
		return fmt.Errorf("failed to get POI details: %w", err)
	}

	if len(pois) == 0 {
		l.Info("No POI found for embedding generation", zap.String("poi_id", poiID.String()))
		span.SetStatus(codes.Ok, "No POI found")
		return fmt.Errorf("POI not found or already has embedding")
	}

	poi := pois[0]

	// Generate embedding using POI information
	embedding, err := s.embeddingService.GeneratePOIEmbedding(ctx, poi.Name, poi.DescriptionPOI, poi.Category)
	if err != nil {
		l.Error("Failed to generate POI embedding",
			zap.Any("error", err),
			zap.String("poi_id", poiID.String()))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to generate POI embedding")
		return fmt.Errorf("failed to generate POI embedding: %w", err)
	}

	// Update POI with generated embedding
	err = s.poiRepository.UpdatePOIEmbedding(ctx, poiID, embedding)
	if err != nil {
		l.Error("Failed to update POI embedding",
			zap.Any("error", err),
			zap.String("poi_id", poiID.String()))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update POI embedding")
		return fmt.Errorf("failed to update POI embedding: %w", err)
	}

	l.Info("POI embedding generated and stored successfully",
		zap.String("poi_id", poiID.String()),
		zap.String("poi_name", poi.Name))
	span.SetAttributes(
		attribute.String("poi.id", poiID.String()),
		attribute.String("poi.name", poi.Name),
	)
	span.SetStatus(codes.Ok, "POI embedding generated")

	return nil
}

// GenerateEmbeddingsForAllPOIs generates embeddings for all POIs that don't have them
func (s *ServiceImpl) GenerateEmbeddingsForAllPOIs(ctx context.Context, batchSize int) error {
	ctx, span := otel.Tracer("POIService").Start(ctx, "GenerateEmbeddingsForAllPOIs", trace.WithAttributes(
		attribute.Int("batch.size", batchSize),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "GenerateEmbeddingsForAllPOIs"))

	if s.embeddingService == nil {
		err := fmt.Errorf("embedding service not available")
		l.Error("Embedding service not initialized", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Embedding service not available")
		return err
	}

	if batchSize <= 0 {
		batchSize = 10 // Default batch size
	}

	totalProcessed := 0
	totalErrors := 0

	for {
		// Get batch of POIs without embeddings
		pois, err := s.poiRepository.GetPOIsWithoutEmbeddings(ctx, batchSize)
		if err != nil {
			l.Error("Failed to get POIs without embeddings", zap.Any("error", err))
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to get POIs without embeddings")
			return fmt.Errorf("failed to get POIs without embeddings: %w", err)
		}

		if len(pois) == 0 {
			// No more POIs to process
			break
		}

		l.Info("Processing batch of POIs", zap.Int("batch_size", len(pois)))

		// Process each POI in the batch
		for _, poi := range pois {
			// Generate embedding
			embedding, err := s.embeddingService.GeneratePOIEmbedding(ctx, poi.Name, poi.DescriptionPOI, poi.Category)
			if err != nil {
				l.Error("Failed to generate embedding for POI",
					zap.Any("error", err),
					zap.String("poi_id", poi.ID.String()),
					zap.String("poi_name", poi.Name))
				totalErrors++
				continue
			}

			// Update POI with embedding
			err = s.poiRepository.UpdatePOIEmbedding(ctx, poi.ID, embedding)
			if err != nil {
				l.Error("Failed to update POI embedding",
					zap.Any("error", err),
					zap.String("poi_id", poi.ID.String()),
					zap.String("poi_name", poi.Name))
				totalErrors++
				continue
			}

			totalProcessed++
			l.Debug("POI embedding generated successfully",
				zap.String("poi_id", poi.ID.String()),
				zap.String("poi_name", poi.Name))
		}

		// Break if we processed fewer POIs than the batch size (end of data)
		if len(pois) < batchSize {
			break
		}
	}

	l.Info("Batch embedding generation completed",
		zap.Int("total_processed", totalProcessed),
		zap.Int("total_errors", totalErrors))
	span.SetAttributes(
		attribute.Int("total.processed", totalProcessed),
		attribute.Int("total.errors", totalErrors),
	)

	if totalErrors > 0 {
		span.SetStatus(codes.Error, fmt.Sprintf("Completed with %d errors", totalErrors))
		return fmt.Errorf("embedding generation completed with %d errors out of %d total POIs", totalErrors, totalProcessed+totalErrors)
	}

	span.SetStatus(codes.Ok, "All POI embeddings generated successfully")
	return nil
}

func (s *ServiceImpl) GetGeneralPOIByDistance(ctx context.Context, userID uuid.UUID, lat, lon, distance float64) ([]models.POIDetailedInfo, error) {
	ctx, span := otel.Tracer("LlmInteractionService").Start(ctx, "GetGeneralPOIByDistance")
	defer span.End()

	cacheKey := generateFilteredPOICacheKey(lat, lon, distance, userID)
	span.SetAttributes(attribute.String("cache.key", cacheKey))

	if cached, found := s.cache.Get(cacheKey); found {
		if pois, ok := cached.([]models.POIDetailedInfo); ok {
			s.logger.Info("Serving POIs from cache", zap.String("key", cacheKey))
			return pois, nil
		}
	}

	s.logger.Info("Cache miss. Querying POIs from database.",
		zap.Float64("lat", lat),
		zap.Float64("lon", lon),
		zap.Float64("distance", distance),
	)

	poisFromDB, err := s.poiRepository.GetPOIsByLocationAndDistance(ctx, lat, lon, distance)
	if err == nil && len(poisFromDB) > 0 {
		for i := range poisFromDB {
			poisFromDB[i].Source = "points_of_interest"
		}
		s.cache.Set(cacheKey, poisFromDB, cache.DefaultExpiration)
		return poisFromDB, nil
	}

	s.logger.Info("No POIs found in database, falling back to LLM generation")
	span.AddEvent("database_miss_fallback_to_llm")

	genAIResponse, err := s.generatePOIsFromLLM(ctx, userID, lat, lon, distance)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	enrichedPOIs := s.enrichAndFilterLLMResponse(genAIResponse.GeneralPOI, lat, lon, distance)
	for i := range enrichedPOIs {
		enrichedPOIs[i].Source = "llm_suggested_pois"
	}

	if len(enrichedPOIs) > 0 {
		interaction := &models.LlmInteraction{
			UserID:    userID,
			ModelName: genAIResponse.ModelName,
			Prompt:    genAIResponse.Prompt,
			Response:  genAIResponse.Response,
			Latitude:  &lat,
			Longitude: &lon,
			Distance:  &distance,
		}

		llmInteractionID, err := s.poiRepository.SaveLlmInteraction(ctx, interaction)
		if err != nil {
			s.logger.Error("Failed to save LLM interaction", zap.Any("error", err))
			return nil, err
		}

		// Synchronous save to ensure POIs are available immediately
		if err := s.poiRepository.SaveLlmPoisToDatabase(ctx, userID, enrichedPOIs, genAIResponse, llmInteractionID); err != nil {
			s.logger.Warn("Failed to save LLM POIs to database", zap.Any("error", err))
		}
	}

	s.cache.Set(cacheKey, enrichedPOIs, cache.DefaultExpiration)
	span.SetStatus(codes.Ok, "POIs generated via LLM and cached")
	return enrichedPOIs, nil
}

func (s *ServiceImpl) generatePOIsFromLLM(ctx context.Context, userID uuid.UUID, lat, lon, distance float64) (*models.GenAIResponse, error) {
	resultCh := make(chan models.GenAIResponse, 1)
	var wg sync.WaitGroup
	wg.Go(func() {
		s.getGeneralPOIByDistance(&wg, ctx, userID, lat, lon, distance, resultCh, &genai.GenerateContentConfig{
			Temperature:     genai.Ptr[float32](0.7),
			MaxOutputTokens: 16384,
		})
	})
	go wg.Wait()
	close(resultCh)

	result := <-resultCh
	if result.Err != nil {
		return nil, result.Err
	}
	return &result, nil
}

func (s *ServiceImpl) getGeneralPOIByDistance(wg *sync.WaitGroup,
	ctx context.Context,
	userID uuid.UUID,
	lat, lon, distance float64,
	resultCh chan<- models.GenAIResponse,
	config *genai.GenerateContentConfig) {
	ctx, span := otel.Tracer("LlmInteractionService").Start(ctx, "GenerateGeneralPOIWorker", trace.WithAttributes(
		attribute.Float64("latitude", lat),
		attribute.Float64("longitude", lon),
		attribute.Float64("distance.km", distance),
		attribute.String("user.id", userID.String())))

	defer span.End()
	defer wg.Done()

	prompt := getGeneralPOIByDistance(lat, lon, distance)
	span.SetAttributes(attribute.Int("prompt.length", len(prompt)))

	if s.aiClient == nil {
		err := fmt.Errorf("AI client is not available - check API key configuration")
		span.RecordError(err)
		span.SetStatus(codes.Error, "AI client unavailable")
		resultCh <- models.GenAIResponse{Err: err}
		return
	}

	// Prepare LLM logging
	sessionID := uuid.New()
	intent := "nearby"
	searchType := "general"
	modelName := "gemini-2.0-flash"
	provider := "google"
	temperature := config.Temperature

	startTime := time.Now()
	response, err := s.aiClient.GenerateResponse(ctx, prompt, config)
	latencyMs := int64(time.Since(startTime).Milliseconds())
	span.SetAttributes(attribute.Int("response.latency_ms", int(latencyMs)))

	if err != nil {
		// Log failed LLM interaction
		s.logLLMInteractionAsync(ctx, userID, sessionID, intent, searchType, prompt, modelName, provider, "", err.Error(), temperature, 0, 0, 0, 500, latencyMs)

		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to generate general POIs")
		resultCh <- models.GenAIResponse{Err: fmt.Errorf("failed to generate general POIs: %w", err)}
		return
	}

	// Extract token counts
	promptTokens := 0
	completionTokens := 0
	totalTokens := 0

	var txt string
	for _, candidate := range response.Candidates {
		if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
			txt = candidate.Content.Parts[0].Text
			break
		}
	}

	// Extract token counts from response metadata
	if response.UsageMetadata != nil {
		promptTokens = int(response.UsageMetadata.PromptTokenCount)
		completionTokens = int(response.UsageMetadata.CandidatesTokenCount)
		totalTokens = int(response.UsageMetadata.TotalTokenCount)
	}

	if txt == "" {
		s.logLLMInteractionAsync(ctx, userID, sessionID, intent, searchType, prompt, modelName, provider, "", "no valid general POI content from AI", temperature, promptTokens, completionTokens, totalTokens, 500, latencyMs)

		err := fmt.Errorf("no valid general POI content from AI")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Empty response from AI")
		resultCh <- models.GenAIResponse{Err: err}
		return
	}
	span.SetAttributes(attribute.Int("response.length", len(txt)))

	cleanTxt := cleanJSONResponse(txt)
	var poiData struct {
		PointsOfInterest []models.POIDetailedInfo `json:"points_of_interest"`
	}
	if err := json.Unmarshal([]byte(cleanTxt), &poiData); err != nil {
		s.logLLMInteractionAsync(ctx, userID, sessionID, intent, searchType, prompt, modelName, provider, txt, fmt.Sprintf("Failed to parse JSON: %v", err), temperature, promptTokens, completionTokens, totalTokens, 500, latencyMs)

		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to parse general POI JSON")
		resultCh <- models.GenAIResponse{Err: fmt.Errorf("failed to parse general POI JSON: %w", err)}
		return
	}

	// Log successful LLM interaction
	s.logLLMInteractionAsync(ctx, userID, sessionID, intent, searchType, prompt, modelName, provider, txt, "", temperature, promptTokens, completionTokens, totalTokens, 200, latencyMs)

	fmt.Println(cleanTxt)

	span.SetAttributes(attribute.Int("pois.count", len(poiData.PointsOfInterest)))
	span.SetStatus(codes.Ok, "General POIs generated successfully")
	resultCh <- models.GenAIResponse{
		GeneralPOI: poiData.PointsOfInterest,
		ModelName:  s.aiClient.ModelName,
		Prompt:     prompt,
		Response:   cleanTxt,
	}
}

// FindOrCreateLLMPOI finds an existing LLM POI by name or creates a new one
func (s *ServiceImpl) FindOrCreateLLMPOI(ctx context.Context, poiData *models.POIDetailedInfo) (uuid.UUID, error) {
	ctx, span := otel.Tracer("POIService").Start(ctx, "FindOrCreateLLMPOI")
	defer span.End()

	if poiData == nil {
		return uuid.Nil, fmt.Errorf("POI data cannot be nil")
	}

	// First, try to find existing POI by name and city
	id, err := s.poiRepository.FindLLMPOIByNameAndCity(ctx, poiData.Name, poiData.City)
	if err == nil && id != uuid.Nil {
		s.logger.Info("Found existing LLM POI",
			zap.String("name", poiData.Name),
			zap.String("id", id.String()),
		)
		span.SetAttributes(attribute.String("operation", "found_existing"))
		return id, nil
	}

	s.logger.Info("Created new LLM POI",
		zap.String("name", poiData.Name),
		zap.String("city", poiData.City),
		zap.String("id", poiData.ID.String()),
	)
	span.SetAttributes(attribute.String("operation", "created_new"))
	return id, nil
}

// FindLLMPOIByName finds an LLM POI by name, searching across all cities
func (s *ServiceImpl) FindLLMPOIByName(ctx context.Context, poiName string) (uuid.UUID, error) {
	ctx, span := otel.Tracer("POIService").Start(ctx, "FindLLMPOIByName", trace.WithAttributes(
		attribute.String("poi.name", poiName),
	))
	defer span.End()

	// For removal purposes, we need to find the POI by name
	// Since we don't have city context, we'll search by name only
	// This could be enhanced later to include city context if needed
	return s.poiRepository.FindLLMPOIByName(ctx, poiName)
}

// GetNearbyRestaurants get nearby restaurants with optional filters
func (s *ServiceImpl) GetNearbyRestaurants(ctx context.Context, userID uuid.UUID, lat, lon, distance float64, cuisineType, priceRange string) ([]models.POIDetailedInfo, error) {
	ctx, span := otel.Tracer("POIService").Start(ctx, "GetNearbyRestaurants", trace.WithAttributes(
		attribute.Float64("location.lat", lat),
		attribute.Float64("location.lon", lon),
		attribute.Float64("distance", distance),
		attribute.String("cuisine_type", cuisineType),
		attribute.String("price_range", priceRange),
	))
	defer span.End()

	// Build cache key with domain-specific filters
	cacheKey := fmt.Sprintf("restaurants_%f_%f_%f_%s_%s_%s", lat, lon, distance, userID.String(), cuisineType, priceRange)

	if cached, found := s.cache.Get(cacheKey); found {
		if pois, ok := cached.([]models.POIDetailedInfo); ok {
			s.logger.Info("Serving restaurants from cache",
				zap.String("cache_key", cacheKey))
			return pois, nil
		}
	}

	s.logger.Info("Querying restaurants from database",
		zap.Float64("location.lat", lat),
		zap.Float64("location.lon", lon),
		zap.Float64("distance", distance))

	// Get restaurants from database with filters
	restaurants, err := s.poiRepository.GetPOIsByLocationAndDistanceWithCategory(ctx, lat, lon, distance, "restaurant")
	if err == nil && len(restaurants) > 0 {
		// Apply domain-specific filters
		filteredRestaurants := s.filterRestaurants(restaurants, cuisineType, priceRange)

		// Mark as database source
		for i := range filteredRestaurants {
			filteredRestaurants[i].Source = "points_of_interest"
		}

		s.cache.Set(cacheKey, filteredRestaurants, cache.DefaultExpiration)
		return filteredRestaurants, nil
	}

	s.logger.Info("No restaurants found in database, falling back to LLM generation")

	// Generate restaurants using LLM with domain-specific prompt
	genAIResponse, err := s.generateRestaurantsFromLLM(ctx, userID, lat, lon, distance, cuisineType, priceRange)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	enrichedRestaurants := s.enrichAndFilterLLMResponse(genAIResponse.GeneralPOI, lat, lon, distance)
	for i := range enrichedRestaurants {
		enrichedRestaurants[i].Source = "llm_suggested_pois"
	}

	s.cache.Set(cacheKey, enrichedRestaurants, cache.DefaultExpiration)
	return enrichedRestaurants, nil
}

// GetNearbyActivities get nearby activities with optional filters
func (s *ServiceImpl) GetNearbyActivities(ctx context.Context, userID uuid.UUID, lat, lon, distance float64, activityType, duration string) ([]models.POIDetailedInfo, error) {
	ctx, span := otel.Tracer("POIService").Start(ctx, "GetNearbyActivities", trace.WithAttributes(
		attribute.Float64("location.lat", lat),
		attribute.Float64("location.lon", lon),
		attribute.Float64("distance", distance),
		attribute.String("activity_type", activityType),
		attribute.String("duration", duration),
	))
	defer span.End()

	// Build cache key with domain-specific filters
	cacheKey := fmt.Sprintf("activities_%f_%f_%f_%s_%s_%s", lat, lon, distance, userID.String(), activityType, duration)

	if cached, found := s.cache.Get(cacheKey); found {
		if pois, ok := cached.([]models.POIDetailedInfo); ok {
			s.logger.Info("Serving activities from cache",
				zap.String("cache_key", cacheKey))
			return pois, nil
		}
	}

	s.logger.Info("Querying activities from database",
		zap.Float64("location.lat", lat),
		zap.Float64("location.lon", lon),
		zap.Float64("distance", distance))

	// Get activities from database with filters
	activities, err := s.poiRepository.GetPOIsByLocationAndDistanceWithCategory(ctx, lat, lon, distance, "activity")
	if err == nil && len(activities) > 0 {
		// Apply domain-specific filters
		filteredActivities := s.filterActivities(activities, activityType, duration)

		// Mark as database source
		for i := range filteredActivities {
			filteredActivities[i].Source = "points_of_interest"
		}

		s.cache.Set(cacheKey, filteredActivities, cache.DefaultExpiration)
		return filteredActivities, nil
	}

	s.logger.Info("No activities found in database, falling back to LLM generation")

	// Generate activities using LLM with domain-specific prompt
	genAIResponse, err := s.generateActivitiesFromLLM(ctx, userID, lat, lon, distance, activityType, duration)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	enrichedActivities := s.enrichAndFilterLLMResponse(genAIResponse.GeneralPOI, lat, lon, distance)
	for i := range enrichedActivities {
		enrichedActivities[i].Source = "llm_suggested_pois"
	}

	s.cache.Set(cacheKey, enrichedActivities, cache.DefaultExpiration)
	return enrichedActivities, nil
}

// GetNearbyHotels get nearby hotels with optional filters
func (s *ServiceImpl) GetNearbyHotels(ctx context.Context, userID uuid.UUID, lat, lon, distance float64, starRating, amenities string) ([]models.POIDetailedInfo, error) {
	ctx, span := otel.Tracer("POIService").Start(ctx, "GetNearbyHotels", trace.WithAttributes(
		attribute.Float64("location.lat", lat),
		attribute.Float64("location.lon", lon),
		attribute.Float64("distance", distance),
		attribute.String("star_rating", starRating),
		attribute.String("amenities", amenities),
	))
	defer span.End()

	// Build cache key with domain-specific filters
	cacheKey := fmt.Sprintf("hotels_%f_%f_%f_%s_%s_%s", lat, lon, distance, userID.String(), starRating, amenities)

	if cached, found := s.cache.Get(cacheKey); found {
		if pois, ok := cached.([]models.POIDetailedInfo); ok {
			s.logger.Info("Serving hotels from cache",
				zap.String("cache_key", cacheKey))
			return pois, nil
		}
	}

	s.logger.Info("Querying hotels from database",
		zap.Float64("location.lat", lat),
		zap.Float64("location.lon", lon),
		zap.Float64("distance", distance))

	// Get hotels from database with filters
	hotels, err := s.poiRepository.GetPOIsByLocationAndDistanceWithCategory(ctx, lat, lon, distance, "hotel")
	if err == nil && len(hotels) > 0 {
		// Apply domain-specific filters
		filteredHotels := s.filterHotels(hotels, starRating, amenities)

		// Mark as database source
		for i := range filteredHotels {
			filteredHotels[i].Source = "points_of_interest"
		}

		s.cache.Set(cacheKey, filteredHotels, cache.DefaultExpiration)
		return filteredHotels, nil
	}

	s.logger.Info("No hotels found in database, falling back to LLM generation")

	// Generate hotels using LLM with domain-specific prompt
	genAIResponse, err := s.generateHotelsFromLLM(ctx, userID, lat, lon, distance, starRating, amenities)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	enrichedHotels := s.enrichAndFilterLLMResponse(genAIResponse.GeneralPOI, lat, lon, distance)
	for i := range enrichedHotels {
		enrichedHotels[i].Source = "llm_suggested_pois"
	}

	s.cache.Set(cacheKey, enrichedHotels, cache.DefaultExpiration)
	return enrichedHotels, nil
}

// GetNearbyAttractions get nearby attractions with optional filters
func (s *ServiceImpl) GetNearbyAttractions(ctx context.Context, userID uuid.UUID, lat, lon, distance float64, attractionType, isOutdoor string) ([]models.POIDetailedInfo, error) {
	ctx, span := otel.Tracer("POIService").Start(ctx, "GetNearbyAttractions", trace.WithAttributes(
		attribute.Float64("location.lat", lat),
		attribute.Float64("location.lon", lon),
		attribute.Float64("distance", distance),
		attribute.String("attraction_type", attractionType),
		attribute.String("is_outdoor", isOutdoor),
	))
	defer span.End()

	// Build cache key with domain-specific filters
	cacheKey := fmt.Sprintf("attractions_%f_%f_%f_%s_%s_%s", lat, lon, distance, userID.String(), attractionType, isOutdoor)

	if cached, found := s.cache.Get(cacheKey); found {
		if pois, ok := cached.([]models.POIDetailedInfo); ok {
			s.logger.Info("Serving attractions from cache",
				zap.String("cache_key", cacheKey))
			return pois, nil
		}
	}

	s.logger.Info("Querying attractions from database",
		zap.Float64("location.lat", lat),
		zap.Float64("location.lon", lon),
		zap.Float64("distance", distance))

	// Get attractions from database with filters
	attractions, err := s.poiRepository.GetPOIsByLocationAndDistanceWithCategory(ctx, lat, lon, distance, "attraction")
	if err == nil && len(attractions) > 0 {
		// Apply domain-specific filters
		filteredAttractions := s.filterAttractions(attractions, attractionType, isOutdoor)

		// Mark as database source
		for i := range filteredAttractions {
			filteredAttractions[i].Source = "points_of_interest"
		}

		s.cache.Set(cacheKey, filteredAttractions, cache.DefaultExpiration)
		return filteredAttractions, nil
	}

	s.logger.Info("No attractions found in database, falling back to LLM generation")

	// Generate attractions using LLM with domain-specific prompt
	genAIResponse, err := s.generateAttractionsFromLLM(ctx, userID, lat, lon, distance, attractionType, isOutdoor)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	enrichedAttractions := s.enrichAndFilterLLMResponse(genAIResponse.GeneralPOI, lat, lon, distance)
	for i := range enrichedAttractions {
		enrichedAttractions[i].Source = "llm_suggested_pois"
	}

	s.cache.Set(cacheKey, enrichedAttractions, cache.DefaultExpiration)
	return enrichedAttractions, nil
}

// Helper functions for domain-specific filtering
func (s *ServiceImpl) filterRestaurants(restaurants []models.POIDetailedInfo, cuisineType, priceRange string) []models.POIDetailedInfo {
	if cuisineType == "" && priceRange == "" {
		return restaurants
	}

	filtered := make([]models.POIDetailedInfo, 0)
	for _, restaurant := range restaurants {
		// Filter by cuisine type
		if cuisineType != "" && restaurant.Category != cuisineType {
			continue
		}
		// Filter by price range
		if priceRange != "" && restaurant.PriceLevel != priceRange {
			continue
		}
		filtered = append(filtered, restaurant)
	}
	return filtered
}

func (s *ServiceImpl) filterActivities(activities []models.POIDetailedInfo, activityType, duration string) []models.POIDetailedInfo {
	if activityType == "" && duration == "" {
		return activities
	}

	filtered := make([]models.POIDetailedInfo, 0)
	for _, activity := range activities {
		// Filter by activity type
		if activityType != "" && activity.Category != activityType {
			continue
		}
		// Filter by duration (using description as proxy for duration since TimeToSpend field doesn't exist)
		if duration != "" && !strings.Contains(strings.ToLower(activity.Description), strings.ToLower(duration)) {
			continue
		}
		filtered = append(filtered, activity)
	}
	return filtered
}

func (s *ServiceImpl) filterHotels(hotels []models.POIDetailedInfo, starRating, amenities string) []models.POIDetailedInfo {
	if starRating == "" && amenities == "" {
		return hotels
	}

	filtered := make([]models.POIDetailedInfo, 0)
	for _, hotel := range hotels {
		// Filter by star rating
		if starRating != "" && hotel.PriceLevel != starRating {
			continue
		}
		// Filter by amenities (basic string matching)
		if amenities != "" {
			if !strings.Contains(strings.ToLower(hotel.Amenities), strings.ToLower(amenities)) {
				continue
			}
		}
		filtered = append(filtered, hotel)
	}
	return filtered
}

func (s *ServiceImpl) filterAttractions(attractions []models.POIDetailedInfo, attractionType, isOutdoor string) []models.POIDetailedInfo {
	if attractionType == "" && isOutdoor == "" {
		return attractions
	}

	filtered := make([]models.POIDetailedInfo, 0)
	for _, attraction := range attractions {
		// Filter by attraction type
		if attractionType != "" && attraction.Category != attractionType {
			continue
		}
		// Filter by outdoor/indoor (basic tag matching)
		if isOutdoor != "" {
			hasOutdoorTag := false
			for _, tag := range attraction.Tags {
				if (isOutdoor == "true" && tag == "outdoor") || (isOutdoor == "false" && tag == "indoor") {
					hasOutdoorTag = true
					break
				}
			}
			if !hasOutdoorTag {
				continue
			}
		}
		filtered = append(filtered, attraction)
	}
	return filtered
}

func (s *ServiceImpl) generateRestaurantsFromLLM(ctx context.Context, userID uuid.UUID, lat, lon, distance float64, _, _ string) (*models.GenAIResponse, error) {
	resultCh := make(chan models.GenAIResponse, 1)
	var wg sync.WaitGroup
	wg.Go(func() {
		s.getGeneralRestaurantByDistance(&wg, ctx, userID, lat, lon, distance, resultCh, &genai.GenerateContentConfig{
			Temperature:     genai.Ptr[float32](0.7),
			MaxOutputTokens: 16384,
		})
	})
	wg.Wait()
	close(resultCh)

	result := <-resultCh
	if result.Err != nil {
		return nil, result.Err
	}
	return &result, nil
}

func (s *ServiceImpl) getGeneralRestaurantByDistance(wg *sync.WaitGroup,
	ctx context.Context,
	userID uuid.UUID,
	lat, lon, distance float64,
	resultCh chan<- models.GenAIResponse,
	config *genai.GenerateContentConfig) {
	ctx, span := otel.Tracer("LlmInteractionService").Start(ctx, "GenerateGeneralPOIWorker", trace.WithAttributes(
		attribute.Float64("latitude", lat),
		attribute.Float64("longitude", lon),
		attribute.Float64("distance.km", distance),
		attribute.String("user.id", userID.String())))

	defer span.End()
	defer wg.Done()

	userLocation := models.UserLocation{
		UserLat:        lat,
		UserLon:        lon,
		SearchRadiusKm: distance,
	}
	prompt := getRestaurantsNearbyPrompt(userLocation)
	span.SetAttributes(attribute.Int("prompt.length", len(prompt)))

	if s.aiClient == nil {
		err := fmt.Errorf("AI client is not available - check API key configuration")
		span.RecordError(err)
		span.SetStatus(codes.Error, "AI client unavailable")
		resultCh <- models.GenAIResponse{Err: err}
		return
	}

	// Prepare LLM logging
	sessionID := uuid.New()
	intent := "nearby"
	searchType := "dining"
	modelName := "gemini-2.0-flash"
	provider := "google"
	temperature := config.Temperature

	startTime := time.Now()
	response, err := s.aiClient.GenerateResponse(ctx, prompt, config)
	latencyMs := int64(time.Since(startTime).Milliseconds())
	span.SetAttributes(attribute.Int("response.latency_ms", int(latencyMs)))

	// Extract token counts
	promptTokens := 0
	completionTokens := 0
	totalTokens := 0

	if err != nil {
		s.logLLMInteractionAsync(ctx, userID, sessionID, intent, searchType, prompt, modelName, provider, "", err.Error(), temperature, 0, 0, 0, 500, latencyMs)

		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to generate general POIs")
		resultCh <- models.GenAIResponse{Err: fmt.Errorf("failed to generate general POIs: %w", err)}
		return
	}

	var txt string
	for _, candidate := range response.Candidates {
		if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
			txt = candidate.Content.Parts[0].Text
			break
		}
	}

	// Extract token counts from response metadata
	if response.UsageMetadata != nil {
		promptTokens = int(response.UsageMetadata.PromptTokenCount)
		completionTokens = int(response.UsageMetadata.CandidatesTokenCount)
		totalTokens = int(response.UsageMetadata.TotalTokenCount)
	}

	if txt == "" {
		s.logLLMInteractionAsync(ctx, userID, sessionID, intent, searchType, prompt, modelName, provider, "", "no valid general POI content from AI", temperature, promptTokens, completionTokens, totalTokens, 500, latencyMs)

		err := fmt.Errorf("no valid general POI content from AI")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Empty response from AI")
		resultCh <- models.GenAIResponse{Err: err}
		return
	}
	span.SetAttributes(attribute.Int("response.length", len(txt)))

	cleanTxt := cleanJSONResponse(txt)
	var poiData struct {
		PointsOfInterest []models.POIDetailedInfo `json:"points_of_interest"`
	}
	if err := json.Unmarshal([]byte(cleanTxt), &poiData); err != nil {
		s.logLLMInteractionAsync(ctx, userID, sessionID, intent, searchType, prompt, modelName, provider, txt, fmt.Sprintf("Failed to parse JSON: %v", err), temperature, promptTokens, completionTokens, totalTokens, 500, latencyMs)

		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to parse general POI JSON")
		resultCh <- models.GenAIResponse{Err: fmt.Errorf("failed to parse general POI JSON: %w", err)}
		return
	}

	// Log successful LLM interaction
	s.logLLMInteractionAsync(ctx, userID, sessionID, intent, searchType, prompt, modelName, provider, txt, "", temperature, promptTokens, completionTokens, totalTokens, 200, latencyMs)

	fmt.Println(cleanTxt)

	span.SetAttributes(attribute.Int("pois.count", len(poiData.PointsOfInterest)))
	span.SetStatus(codes.Ok, "General POIs generated successfully")
	resultCh <- models.GenAIResponse{
		GeneralPOI: poiData.PointsOfInterest,
		ModelName:  s.aiClient.ModelName,
		Prompt:     prompt,
		Response:   cleanTxt,
	}
}

func (s *ServiceImpl) generateActivitiesFromLLM(ctx context.Context, userID uuid.UUID, lat, lon, distance float64, _, _ string) (*models.GenAIResponse, error) {
	resultCh := make(chan models.GenAIResponse, 1)
	var wg sync.WaitGroup
	wg.Go(func() {
		s.getGeneralActivitiesByDistance(&wg, ctx, userID, lat, lon, distance, resultCh, &genai.GenerateContentConfig{
			Temperature:     genai.Ptr[float32](0.7),
			MaxOutputTokens: 16384,
		})
	})
	wg.Wait()
	close(resultCh)

	result := <-resultCh
	if result.Err != nil {
		return nil, result.Err
	}
	return &result, nil
}

func (s *ServiceImpl) getGeneralActivitiesByDistance(wg *sync.WaitGroup,
	ctx context.Context,
	userID uuid.UUID,
	lat, lon, distance float64,
	resultCh chan<- models.GenAIResponse,
	config *genai.GenerateContentConfig) {
	ctx, span := otel.Tracer("LlmInteractionService").Start(ctx, "GenerateGeneralPOIWorker", trace.WithAttributes(
		attribute.Float64("latitude", lat),
		attribute.Float64("longitude", lon),
		attribute.Float64("distance.km", distance),
		attribute.String("user.id", userID.String())))

	defer span.End()
	defer wg.Done()

	userLocation := models.UserLocation{
		UserLat:        lat,
		UserLon:        lon,
		SearchRadiusKm: distance,
	}
	prompt := getActivitiesNearbyPrompt(userLocation)
	span.SetAttributes(attribute.Int("prompt.length", len(prompt)))

	if s.aiClient == nil {
		err := fmt.Errorf("AI client is not available - check API key configuration")
		span.RecordError(err)
		span.SetStatus(codes.Error, "AI client unavailable")
		resultCh <- models.GenAIResponse{Err: err}
		return
	}

	// Prepare LLM logging
	sessionID := uuid.New()
	intent := "nearby"
	searchType := "activities"
	modelName := "gemini-2.0-flash"
	provider := "google"
	temperature := config.Temperature

	startTime := time.Now()
	response, err := s.aiClient.GenerateResponse(ctx, prompt, config)
	latencyMs := int64(time.Since(startTime).Milliseconds())
	span.SetAttributes(attribute.Int("response.latency_ms", int(latencyMs)))

	// Extract token counts
	promptTokens := 0
	completionTokens := 0
	totalTokens := 0

	if err != nil {
		s.logLLMInteractionAsync(ctx, userID, sessionID, intent, searchType, prompt, modelName, provider, "", err.Error(), temperature, 0, 0, 0, 500, latencyMs)

		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to generate general POIs")
		resultCh <- models.GenAIResponse{Err: fmt.Errorf("failed to generate general POIs: %w", err)}
		return
	}

	var txt string
	for _, candidate := range response.Candidates {
		if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
			txt = candidate.Content.Parts[0].Text
			break
		}
	}

	// Extract token counts from response metadata
	if response.UsageMetadata != nil {
		promptTokens = int(response.UsageMetadata.PromptTokenCount)
		completionTokens = int(response.UsageMetadata.CandidatesTokenCount)
		totalTokens = int(response.UsageMetadata.TotalTokenCount)
	}

	if txt == "" {
		s.logLLMInteractionAsync(ctx, userID, sessionID, intent, searchType, prompt, modelName, provider, "", "no valid general POI content from AI", temperature, promptTokens, completionTokens, totalTokens, 500, latencyMs)

		err := fmt.Errorf("no valid general POI content from AI")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Empty response from AI")
		resultCh <- models.GenAIResponse{Err: err}
		return
	}
	span.SetAttributes(attribute.Int("response.length", len(txt)))

	cleanTxt := cleanJSONResponse(txt)
	var poiData struct {
		PointsOfInterest []models.POIDetailedInfo `json:"points_of_interest"`
	}
	if err := json.Unmarshal([]byte(cleanTxt), &poiData); err != nil {
		s.logLLMInteractionAsync(ctx, userID, sessionID, intent, searchType, prompt, modelName, provider, txt, fmt.Sprintf("Failed to parse JSON: %v", err), temperature, promptTokens, completionTokens, totalTokens, 500, latencyMs)

		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to parse general POI JSON")
		resultCh <- models.GenAIResponse{Err: fmt.Errorf("failed to parse general POI JSON: %w", err)}
		return
	}

	// Log successful LLM interaction
	s.logLLMInteractionAsync(ctx, userID, sessionID, intent, searchType, prompt, modelName, provider, txt, "", temperature, promptTokens, completionTokens, totalTokens, 200, latencyMs)

	fmt.Println(cleanTxt)

	span.SetAttributes(attribute.Int("pois.count", len(poiData.PointsOfInterest)))
	span.SetStatus(codes.Ok, "General POIs generated successfully")
	resultCh <- models.GenAIResponse{
		GeneralPOI: poiData.PointsOfInterest,
		ModelName:  s.aiClient.ModelName,
		Prompt:     prompt,
		Response:   cleanTxt,
	}
}

func (s *ServiceImpl) generateHotelsFromLLM(ctx context.Context, userID uuid.UUID, lat, lon, distance float64, _, _ string) (*models.GenAIResponse, error) {
	resultCh := make(chan models.GenAIResponse, 1)
	var wg sync.WaitGroup
	wg.Go(func() {
		s.getGeneralHotelsByDistance(&wg, ctx, userID, lat, lon, distance, resultCh, &genai.GenerateContentConfig{
			Temperature:     genai.Ptr[float32](0.7),
			MaxOutputTokens: 16384,
		})
	})
	wg.Wait()
	close(resultCh)

	result := <-resultCh
	if result.Err != nil {
		return nil, result.Err
	}
	return &result, nil
}

func (s *ServiceImpl) getGeneralHotelsByDistance(wg *sync.WaitGroup,
	ctx context.Context,
	userID uuid.UUID,
	lat, lon, distance float64,
	resultCh chan<- models.GenAIResponse,
	config *genai.GenerateContentConfig) {
	ctx, span := otel.Tracer("LlmInteractionService").Start(ctx, "GenerateGeneralPOIWorker", trace.WithAttributes(
		attribute.Float64("latitude", lat),
		attribute.Float64("longitude", lon),
		attribute.Float64("distance.km", distance),
		attribute.String("user.id", userID.String())))

	defer span.End()
	defer wg.Done()

	userLocation := models.UserLocation{
		UserLat:        lat,
		UserLon:        lon,
		SearchRadiusKm: distance,
	}
	prompt := getHotelsNeabyPrompt(userLocation)
	span.SetAttributes(attribute.Int("prompt.length", len(prompt)))

	// Prepare LLM logging
	sessionID := uuid.New()
	intent := "nearby"
	searchType := "accommodation"
	modelName := "gemini-2.0-flash"
	provider := "google"
	temperature := config.Temperature

	if s.aiClient == nil {
		err := fmt.Errorf("AI client is not available - check API key configuration")
		span.RecordError(err)
		span.SetStatus(codes.Error, "AI client unavailable")

		s.logLLMInteractionAsync(ctx, userID, sessionID, intent, searchType, prompt, modelName, provider, "", err.Error(), temperature, 0, 0, 0, 500, 0)

		resultCh <- models.GenAIResponse{Err: err}
		return
	}

	startTime := time.Now()
	response, err := s.aiClient.GenerateResponse(ctx, prompt, config)
	latencyMs := int64(time.Since(startTime).Milliseconds())
	span.SetAttributes(attribute.Int("response.latency_ms", int(latencyMs)))

	// Extract token counts
	promptTokens := 0
	completionTokens := 0
	totalTokens := 0

	if err != nil {
		s.logLLMInteractionAsync(ctx, userID, sessionID, intent, searchType, prompt, modelName, provider, "", err.Error(), temperature, 0, 0, 0, 500, int64(latencyMs))

		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to generate general POIs")

		resultCh <- models.GenAIResponse{Err: fmt.Errorf("failed to generate general POIs: %w", err)}
		return
	}

	var txt string
	for _, candidate := range response.Candidates {
		if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
			txt = candidate.Content.Parts[0].Text
			break
		}
	}

	// Extract token counts from response metadata
	if response.UsageMetadata != nil {
		promptTokens = int(response.UsageMetadata.PromptTokenCount)
		completionTokens = int(response.UsageMetadata.CandidatesTokenCount)
		totalTokens = int(response.UsageMetadata.TotalTokenCount)
	}

	if txt == "" {
		err := fmt.Errorf("no valid general POI content from AI")
		s.logLLMInteractionAsync(ctx, userID, sessionID, intent, searchType, prompt, modelName, provider, "", err.Error(), temperature, promptTokens, completionTokens, totalTokens, 500, int64(latencyMs))

		span.RecordError(err)
		span.SetStatus(codes.Error, "Empty response from AI")

		resultCh <- models.GenAIResponse{Err: err}
		return
	}
	span.SetAttributes(attribute.Int("response.length", len(txt)))

	cleanTxt := cleanJSONResponse(txt)
	var poiData struct {
		PointsOfInterest []models.POIDetailedInfo `json:"points_of_interest"`
	}
	if err := json.Unmarshal([]byte(cleanTxt), &poiData); err != nil {
		s.logLLMInteractionAsync(ctx, userID, sessionID, intent, searchType, prompt, modelName, provider, txt, fmt.Sprintf("Failed to parse JSON: %v", err), temperature, promptTokens, completionTokens, totalTokens, 500, int64(latencyMs))

		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to parse general POI JSON")

		resultCh <- models.GenAIResponse{Err: fmt.Errorf("failed to parse general POI JSON: %w", err)}
		return
	}

	// Log successful LLM interaction
	s.logLLMInteractionAsync(ctx, userID, sessionID, intent, searchType, prompt, modelName, provider, txt, "", temperature, promptTokens, completionTokens, totalTokens, 200, latencyMs)

	fmt.Println(cleanTxt)

	span.SetAttributes(attribute.Int("pois.count", len(poiData.PointsOfInterest)))
	span.SetStatus(codes.Ok, "General POIs generated successfully")
	resultCh <- models.GenAIResponse{
		GeneralPOI: poiData.PointsOfInterest,
		ModelName:  s.aiClient.ModelName,
		Prompt:     prompt,
		Response:   cleanTxt,
	}
}

func (s *ServiceImpl) generateAttractionsFromLLM(ctx context.Context, userID uuid.UUID, lat, lon, distance float64, _, _ string) (*models.GenAIResponse, error) {
	resultCh := make(chan models.GenAIResponse, 1)
	var wg sync.WaitGroup
	wg.Go(func() {
		s.getGeneralAttractionsByDistance(&wg, ctx, userID, lat, lon, distance, resultCh, &genai.GenerateContentConfig{
			Temperature:     genai.Ptr[float32](0.7),
			MaxOutputTokens: 16384,
		})
	})
	wg.Wait()
	close(resultCh)

	result := <-resultCh
	if result.Err != nil {
		return nil, result.Err
	}
	return &result, nil
}

func (s *ServiceImpl) getGeneralAttractionsByDistance(wg *sync.WaitGroup,
	ctx context.Context,
	userID uuid.UUID,
	lat, lon, distance float64,
	resultCh chan<- models.GenAIResponse,
	config *genai.GenerateContentConfig) {
	ctx, span := otel.Tracer("LlmInteractionService").Start(ctx, "GenerateGeneralPOIWorker", trace.WithAttributes(
		attribute.Float64("latitude", lat),
		attribute.Float64("longitude", lon),
		attribute.Float64("distance.km", distance),
		attribute.String("user.id", userID.String())))

	defer span.End()
	defer wg.Done()

	userLocation := models.UserLocation{
		UserLat:        lat,
		UserLon:        lon,
		SearchRadiusKm: distance,
	}
	prompt := getAttractionsNeabyPrompt(userLocation)
	span.SetAttributes(attribute.Int("prompt.length", len(prompt)))

	// Prepare LLM logging
	sessionID := uuid.New()
	intent := "nearby"
	searchType := "attractions"
	modelName := "gemini-2.0-flash"
	provider := "google"
	temperature := config.Temperature

	if s.aiClient == nil {
		err := fmt.Errorf("AI client is not available - check API key configuration")
		span.RecordError(err)
		span.SetStatus(codes.Error, "AI client unavailable")

		s.logLLMInteractionAsync(ctx, userID, sessionID, intent, searchType, prompt, modelName, provider, "", err.Error(), temperature, 0, 0, 0, 500, 0)

		resultCh <- models.GenAIResponse{Err: err}
		return
	}

	startTime := time.Now()
	response, err := s.aiClient.GenerateResponse(ctx, prompt, config)
	latencyMs := int64(time.Since(startTime).Milliseconds())
	span.SetAttributes(attribute.Int("response.latency_ms", int(latencyMs)))

	// Extract token counts
	promptTokens := 0
	completionTokens := 0
	totalTokens := 0

	if err != nil {
		s.logLLMInteractionAsync(ctx, userID, sessionID, intent, searchType, prompt, modelName, provider, "", err.Error(), temperature, 0, 0, 0, 500, int64(latencyMs))

		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to generate general POIs")

		resultCh <- models.GenAIResponse{Err: fmt.Errorf("failed to generate general POIs: %w", err)}
		return
	}

	var txt string
	for _, candidate := range response.Candidates {
		if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
			txt = candidate.Content.Parts[0].Text
			break
		}
	}

	// Extract token counts from response metadata
	if response.UsageMetadata != nil {
		promptTokens = int(response.UsageMetadata.PromptTokenCount)
		completionTokens = int(response.UsageMetadata.CandidatesTokenCount)
		totalTokens = int(response.UsageMetadata.TotalTokenCount)
	}

	if txt == "" {
		err := fmt.Errorf("no valid general POI content from AI")
		s.logLLMInteractionAsync(ctx, userID, sessionID, intent, searchType, prompt, modelName, provider, "", err.Error(), temperature, promptTokens, completionTokens, totalTokens, 500, int64(latencyMs))

		span.RecordError(err)
		span.SetStatus(codes.Error, "Empty response from AI")

		resultCh <- models.GenAIResponse{Err: err}
		return
	}
	span.SetAttributes(attribute.Int("response.length", len(txt)))

	cleanTxt := cleanJSONResponse(txt)
	var poiData struct {
		PointsOfInterest []models.POIDetailedInfo `json:"points_of_interest"`
	}
	if err := json.Unmarshal([]byte(cleanTxt), &poiData); err != nil {
		s.logLLMInteractionAsync(ctx, userID, sessionID, intent, searchType, prompt, modelName, provider, txt, fmt.Sprintf("Failed to parse JSON: %v", err), temperature, promptTokens, completionTokens, totalTokens, 500, int64(latencyMs))

		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to parse general POI JSON")

		resultCh <- models.GenAIResponse{Err: fmt.Errorf("failed to parse general POI JSON: %w", err)}
		return
	}

	// Log successful LLM interaction
	s.logLLMInteractionAsync(ctx, userID, sessionID, intent, searchType, prompt, modelName, provider, txt, "", temperature, promptTokens, completionTokens, totalTokens, 200, latencyMs)

	fmt.Println(cleanTxt)

	span.SetAttributes(attribute.Int("pois.count", len(poiData.PointsOfInterest)))
	span.SetStatus(codes.Ok, "General POIs generated successfully")
	resultCh <- models.GenAIResponse{
		GeneralPOI: poiData.PointsOfInterest,
		ModelName:  s.aiClient.ModelName,
		Prompt:     prompt,
		Response:   cleanTxt,
	}
}

// AddItineraryToBookmarks adds an itinerary to the user's bookmarks
// RemoveItineraryFromBookmarks removes an itinerary from the user's bookmarks
func (s *ServiceImpl) RemoveItineraryFromBookmarks(ctx context.Context, userID, itineraryID uuid.UUID) error {
	ctx, span := otel.Tracer("poi-service").Start(ctx, "RemoveItineraryFromBookmarks")
	defer span.End()

	err := s.poiRepository.RemoveItineraryFromBookmarks(ctx, userID, itineraryID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.logger.Error("Failed to remove itinerary from bookmarks",
			zap.String("user_id", userID.String()),
			zap.String("itinerary_id", itineraryID.String()),
			zap.Any("error", err))
		return fmt.Errorf("failed to remove itinerary from bookmarks: %w", err)
	}

	span.SetAttributes(
		attribute.String("user.id", userID.String()),
		attribute.String("itinerary.id", itineraryID.String()),
	)
	span.SetStatus(codes.Ok, "Bookmark removed successfully")

	s.logger.Info("Itinerary removed from bookmarks",
		zap.String("user_id", userID.String()),
		zap.String("itinerary_id", itineraryID.String()))

	return nil
}

func (s *ServiceImpl) CheckIsBookmarked(ctx context.Context, userID, itineraryID uuid.UUID) (bool, error) {
	isBookmarked, err := s.poiRepository.CheckIsBookmarked(ctx, userID, itineraryID)
	if err != nil {
		s.logger.Error("failed to check if itinerary is bookmarked", zap.Error(err))
		return false, err
	}
	return isBookmarked, nil
}

// GetBookmarksFiltered retrieves bookmarked itineraries with search and filter support
func (s *ServiceImpl) GetBookmarksFiltered(ctx context.Context, filter models.BookmarksFilter) ([]models.SavedItinerary, int, error) {
	ctx, span := otel.Tracer("poi-service").Start(ctx, "GetBookmarksFiltered")
	defer span.End()

	// Set default pagination if not provided
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	itineraries, total, err := s.poiRepository.GetBookmarksFiltered(ctx, filter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.logger.Error("Failed to get filtered bookmarks",
			zap.String("user_id", filter.UserID.String()),
			zap.Any("error", err))
		return nil, 0, fmt.Errorf("failed to get bookmarks: %w", err)
	}

	span.SetAttributes(
		attribute.String("user.id", filter.UserID.String()),
		attribute.Int("bookmarks.total", total),
		attribute.Int("bookmarks.returned", len(itineraries)),
		attribute.String("filter.search_text", filter.SearchText),
		attribute.String("filter.sort_by", filter.SortBy),
	)
	span.SetStatus(codes.Ok, "Bookmarks retrieved successfully")

	s.logger.Info("Retrieved filtered bookmarks",
		zap.String("user_id", filter.UserID.String()),
		zap.Int("total", total),
		zap.Int("returned", len(itineraries)))

	return itineraries, total, nil
}
