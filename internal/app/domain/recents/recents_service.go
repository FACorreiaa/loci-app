package recents

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/FACorreiaa/go-templui/internal/app/models"
)

var _ Service = (*ServiceImpl)(nil)

type Service interface {
	GetUserRecentInteractions(ctx context.Context, userID uuid.UUID, page, limit int, filterOptions *models.RecentInteractionsFilter) (*models.RecentInteractionsResponse, error)
	GetCityDetailsForUser(ctx context.Context, userID uuid.UUID, cityName string) (*models.CityInteractions, error)
}

type ServiceImpl struct {
	repo   Repository
	logger *zap.Logger
}

func NewService(repo Repository, logger *zap.Logger) *ServiceImpl {
	return &ServiceImpl{
		repo:   repo,
		logger: logger,
	}
}

// GetUserRecentInteractions retrieves recent interactions for a user
func (s *ServiceImpl) GetUserRecentInteractions(ctx context.Context, userID uuid.UUID, page, limit int, filterOptions *models.RecentInteractionsFilter) (*models.RecentInteractionsResponse, error) {
	ctx, span := otel.Tracer("RecentsService").Start(ctx, "GetUserRecentInteractions", trace.WithAttributes(
		attribute.String("user_id", userID.String()),
		attribute.Int("page", page),
		attribute.Int("limit", limit),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "GetUserRecentInteractions"))

	// Validate page
	if page <= 0 {
		page = 1
	}

	// Validate limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	l.Info( "Getting user recent interactions",
		zap.String("user_id", userID.String()),
		zap.Int("page", page),
		zap.Int("limit", limit))

	// Get recent interactions from repository
	response, err := s.repo.GetUserRecentInteractions(ctx, userID, page, limit, filterOptions)
	if err != nil {
		l.Error( "Failed to get recent interactions", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get recent interactions")
		return nil, fmt.Errorf("failed to get recent interactions: %w", err)
	}

	l.Info( "Successfully retrieved recent interactions",
		zap.String("user_id", userID.String()),
		zap.Int("cities_count", len(response.Cities)))

	span.SetAttributes(attribute.Int("results.cities", len(response.Cities)))
	span.SetStatus(codes.Ok, "Recent interactions retrieved")

	return response, nil
}

// GetCityDetailsForUser retrieves detailed information for a specific city
func (s *ServiceImpl) GetCityDetailsForUser(ctx context.Context, userID uuid.UUID, cityName string) (*models.CityInteractions, error) {
	ctx, span := otel.Tracer("RecentsService").Start(ctx, "GetCityDetailsForUser", trace.WithAttributes(
		attribute.String("user_id", userID.String()),
		attribute.String("city_name", cityName),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "GetCityDetailsForUser"))

	if cityName == "" {
		err := fmt.Errorf("city name is required")
		l.Error( "City name is required")
		span.RecordError(err)
		span.SetStatus(codes.Error, "City name is required")
		return nil, err
	}

	l.Info( "Getting city details for user",
		zap.String("user_id", userID.String()),
		zap.String("city_name", cityName))

	// Get recent interactions to find the city data
	defaultFilter := &models.RecentInteractionsFilter{
		SortBy:    "last_activity",
		SortOrder: "desc",
	}
	recentResponse, err := s.repo.GetUserRecentInteractions(ctx, userID, 1, 50, defaultFilter) // Get more to find the city
	if err != nil {
		l.Error( "Failed to get recent interactions", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get recent interactions")
		return nil, fmt.Errorf("failed to get recent interactions: %w", err)
	}

	// Find the city in recent interactions
	var cityInteractions *models.CityInteractions
	for _, city := range recentResponse.Cities {
		if city.CityName == cityName {
			cityInteractions = &city
			break
		}
	}

	if cityInteractions == nil {
		err := fmt.Errorf("no interactions found for city: %s", cityName)
		l.Warn( "No interactions found for city", zap.String("city_name", cityName))
		span.SetStatus(codes.Error, "No interactions found")
		return nil, err
	}

	interactions := cityInteractions.Interactions
	poiCount := cityInteractions.POICount

	// Get POIs for the city
	pois, err := s.repo.GetCityPOIsByInteraction(ctx, userID, cityName)
	if err != nil {
		l.Warn( "Failed to get POIs for city",
			zap.String("city_name", cityName),
			zap.Any("error", err))
		pois = []models.POIDetailedInfo{} // Set to empty slice if we can't get POIs
	}

	// Get hotels for the city
	hotels, err := s.repo.GetCityHotelsByInteraction(ctx, userID, cityName)
	if err != nil {
		l.Warn( "Failed to get hotels for city",
			zap.String("city_name", cityName),
			zap.Any("error", err))
		hotels = []models.HotelDetailedInfo{} // Set to empty slice if we can't get hotels
	}

	// Get restaurants for the city
	restaurants, err := s.repo.GetCityRestaurantsByInteraction(ctx, userID, cityName)
	if err != nil {
		l.Warn( "Failed to get restaurants for city",
			zap.String("city_name", cityName),
			zap.Any("error", err))
		restaurants = []models.RestaurantDetailedInfo{} // Set to empty slice if we can't get restaurants
	}

	// Get saved itineraries for the city
	itineraries, err := s.repo.GetCityItinerariesByInteraction(ctx, userID, cityName)
	if err != nil {
		l.Warn( "Failed to get itineraries for city",
			zap.String("city_name", cityName),
			zap.Any("error", err))
		itineraries = []models.UserSavedItinerary{} // Set to empty slice if we can't get itineraries
	}

	// Get favorite POIs for the city
	favorites, err := s.repo.GetCityFavorites(ctx, userID, cityName)
	if err != nil {
		l.Warn( "Failed to get favorites for city",
			zap.String("city_name", cityName),
			zap.Any("error", err))
		favorites = []models.POIDetailedInfo{} // Set to empty slice if we can't get favorites
	}

	// Enrich interactions with POI/hotel/restaurant data
	for i := range interactions {
		interactions[i].POIs = convertPOIsToDetail(pois)
		interactions[i].Hotels = hotels
		interactions[i].Restaurants = restaurants
	}

	// Get the last activity timestamp
	var lastActivity time.Time
	if len(interactions) > 0 {
		lastActivity = interactions[0].CreatedAt // Interactions are ordered by created_at DESC
	}

	cityDetails := &models.CityInteractions{
		CityName:     cityName,
		Interactions: interactions,
		POICount:     poiCount,
		LastActivity: lastActivity,
		//SavedItineraries:  itineraries,
		//FavoritePOIs:      favorites,
		//TotalInteractions: len(interactions),
		//TotalFavorites:    len(favorites),
		//TotalItineraries:  len(itineraries),
	}

	l.Info( "Successfully retrieved city details",
		zap.String("user_id", userID.String()),
		zap.String("city_name", cityName),
		zap.Int("interaction_count", len(interactions)),
		zap.Int("poi_count", len(pois)),
		zap.Int("hotel_count", len(hotels)),
		zap.Int("restaurant_count", len(restaurants)),
		zap.Int("itinerary_count", len(itineraries)),
		zap.Int("favorite_count", len(favorites)))

	span.SetAttributes(
		attribute.Int("results.interactions", len(interactions)),
		attribute.Int("results.pois", len(pois)),
		attribute.Int("results.hotels", len(hotels)),
		attribute.Int("results.restaurants", len(restaurants)),
		attribute.Int("results.itineraries", len(itineraries)),
		attribute.Int("results.favorites", len(favorites)),
	)
	span.SetStatus(codes.Ok, "City details retrieved")

	return cityDetails, nil
}

// Helper function to convert POIDetailedInfo to POIDetail for consistency with existing types
func convertPOIsToDetail(detailedPOIs []models.POIDetailedInfo) []models.POIDetailedInfo {
	var pois []models.POIDetailedInfo
	for _, poi := range detailedPOIs {
		detail := models.POIDetailedInfo{
			ID:               poi.ID,
			LlmInteractionID: poi.LlmInteractionID,
			City:             poi.City,
			CityID:           poi.CityID,
			Name:             poi.Name,
			Latitude:         poi.Latitude,
			Longitude:        poi.Longitude,
			Category:         poi.Category,
			DescriptionPOI:   poi.Description,
			Address:          poi.Address,
			Website:          poi.Website,
			Distance:         poi.Distance,
		}
		pois = append(pois, detail)
	}
	return pois
}
