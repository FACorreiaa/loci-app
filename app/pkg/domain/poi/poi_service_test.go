package poi

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	generativeAI "github.com/FACorreiaa/go-genai-sdk/lib"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/FACorreiaa/go-templui/app/lib/models"
)

type MockCityRepository struct {
	mock.Mock
}

func (m *MockCityRepository) GetCity(ctx context.Context, lat, lon float64) (uuid.UUID, string, error) {
	args := m.Called(ctx, lat, lon)
	return args.Get(0).(uuid.UUID), args.Get(1).(string), args.Error(2)
}

func (m *MockCityRepository) SaveCity(ctx context.Context, city models.CityDetail) (uuid.UUID, error) {
	args := m.Called(ctx, city)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockCityRepository) FindCityByNameAndCountry(ctx context.Context, name, country string) (*models.CityDetail, error) {
	args := m.Called(ctx, name, country)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CityDetail), args.Error(1)
}

func (m *MockCityRepository) GetCityByID(ctx context.Context, cityID uuid.UUID) (*models.CityDetail, error) {
	args := m.Called(ctx, cityID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CityDetail), args.Error(1)
}

func (m *MockCityRepository) FindSimilarCities(ctx context.Context, queryEmbedding []float32, limit int) ([]models.CityDetail, error) {
	args := m.Called(ctx, queryEmbedding, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.CityDetail), args.Error(1)
}

func (m *MockCityRepository) UpdateCityEmbedding(ctx context.Context, cityID uuid.UUID, embedding []float32) error {
	args := m.Called(ctx, cityID, embedding)
	return args.Error(0)
}

func (m *MockCityRepository) GetCitiesWithoutEmbeddings(ctx context.Context, limit int) ([]models.CityDetail, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.CityDetail), args.Error(1)
}

func (m *MockCityRepository) GetCityIDByName(ctx context.Context, cityName string) (uuid.UUID, error) {
	args := m.Called(ctx, cityName)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockCityRepository) FindCityByFuzzyName(ctx context.Context, name string) (*models.CityDetail, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CityDetail), args.Error(1)
}

func (m *MockCityRepository) GetAllCities(ctx context.Context) ([]models.CityDetail, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.CityDetail), args.Error(1)
}

// MockPOIRepository is a mock implementation of POIRepository
type MockPOIRepository struct {
	mock.Mock
}

func (m *MockPOIRepository) GetPOIsByLocationAndDistanceWithCategory(ctx context.Context, lat, lon, radiusMeters float64, category string) ([]models.POIDetailedInfo, error) {
	panic("implement me")
}

func (m *MockPOIRepository) SavePoi(ctx context.Context, poi models.POIDetailedInfo, cityID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, poi, cityID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockPOIRepository) FindPoiByNameAndCity(ctx context.Context, name string, cityID uuid.UUID) (*models.POIDetailedInfo, error) {
	args := m.Called(ctx, name, cityID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.POIDetailedInfo), args.Error(1)
}

func (m *MockPOIRepository) GetPOIsByCityAndDistance(ctx context.Context, cityID uuid.UUID, userLocation models.UserLocation) ([]models.POIDetailedInfo, error) {
	args := m.Called(ctx, cityID, userLocation)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.POIDetailedInfo), args.Error(1)
}

func (m *MockPOIRepository) GetPOIsByLocationAndDistance(ctx context.Context, lat, lon, radiusMeters float64) ([]models.POIDetailedInfo, error) {
	args := m.Called(ctx, lat, lon, radiusMeters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.POIDetailedInfo), args.Error(1)
}

func (m *MockPOIRepository) GetPOIsByLocationAndDistanceWithFilters(ctx context.Context, lat, lon, radiusMeters float64, filters map[string]string) ([]models.POIDetailedInfo, error) {
	args := m.Called(ctx, lat, lon, radiusMeters, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.POIDetailedInfo), args.Error(1)
}

func (m *MockPOIRepository) AddPoiToFavourites(ctx context.Context, userID, poiID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, userID, poiID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockPOIRepository) AddLLMPoiToFavourite(ctx context.Context, userID uuid.UUID, llmPoiID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, userID, llmPoiID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockPOIRepository) CheckPoiExists(ctx context.Context, poiID uuid.UUID) (bool, error) {
	args := m.Called(ctx, poiID)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockPOIRepository) RemovePoiFromFavourites(ctx context.Context, userID, poiID uuid.UUID) error {
	args := m.Called(ctx, poiID, userID)
	return args.Error(0)
}

func (m *MockPOIRepository) CheckLlmPoiExists(ctx context.Context, llmPoiID uuid.UUID) (bool, error) {
	args := m.Called(ctx, llmPoiID)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockPOIRepository) RemoveLLMPoiFromFavourite(ctx context.Context, userID, llmPoiID uuid.UUID) error {
	args := m.Called(ctx, userID, llmPoiID)
	return args.Error(0)
}

func (m *MockPOIRepository) GetFavouritePOIsByUserID(ctx context.Context, userID uuid.UUID) ([]models.POIDetailedInfo, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.POIDetailedInfo), args.Error(1)
}

func (m *MockPOIRepository) GetFavouritePOIsByUserIDPaginated(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.POIDetailedInfo, int, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]models.POIDetailedInfo), args.Get(1).(int), args.Error(2)
}

func (m *MockPOIRepository) GetPOIsByCityID(ctx context.Context, cityID uuid.UUID) ([]models.POIDetailedInfo, error) {
	args := m.Called(ctx, cityID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.POIDetailedInfo), args.Error(1)
}

func (m *MockPOIRepository) FindPOIDetails(ctx context.Context, cityID uuid.UUID, lat, lon float64, tolerance float64) (*models.POIDetailedInfo, error) {
	args := m.Called(ctx, cityID, lat, lon, tolerance)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.POIDetailedInfo), args.Error(1)
}

func (m *MockPOIRepository) SavePOIDetails(ctx context.Context, poi models.POIDetailedInfo, cityID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, poi, cityID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockPOIRepository) SearchPOIs(ctx context.Context, filter models.POIFilter) ([]models.POIDetailedInfo, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.POIDetailedInfo), args.Error(1)
}

func (m *MockPOIRepository) FindSimilarPOIs(ctx context.Context, queryEmbedding []float32, limit int) ([]models.POIDetailedInfo, error) {
	args := m.Called(ctx, queryEmbedding, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.POIDetailedInfo), args.Error(1)
}

func (m *MockPOIRepository) FindSimilarPOIsByCity(ctx context.Context, queryEmbedding []float32, cityID uuid.UUID, limit int) ([]models.POIDetailedInfo, error) {
	args := m.Called(ctx, queryEmbedding, cityID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.POIDetailedInfo), args.Error(1)
}

func (m *MockPOIRepository) SearchPOIsHybrid(ctx context.Context, filter models.POIFilter, queryEmbedding []float32, semanticWeight float64) ([]models.POIDetailedInfo, error) {
	args := m.Called(ctx, filter, queryEmbedding, semanticWeight)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.POIDetailedInfo), args.Error(1)
}

func (m *MockPOIRepository) UpdatePOIEmbedding(ctx context.Context, poiID uuid.UUID, embedding []float32) error {
	args := m.Called(ctx, poiID, embedding)
	return args.Error(0)
}

func (m *MockPOIRepository) GetPOIsWithoutEmbeddings(ctx context.Context, limit int) ([]models.POIDetailedInfo, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.POIDetailedInfo), args.Error(1)
}

func (m *MockPOIRepository) FindHotelDetails(ctx context.Context, cityID uuid.UUID, lat, lon, tolerance float64) ([]models.HotelDetailedInfo, error) {
	args := m.Called(ctx, cityID, lat, lon, tolerance)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.HotelDetailedInfo), args.Error(1)
}

func (m *MockPOIRepository) SaveHotelDetails(ctx context.Context, hotel models.HotelDetailedInfo, cityID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, hotel, cityID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockPOIRepository) GetHotelByID(ctx context.Context, hotelID uuid.UUID) (*models.HotelDetailedInfo, error) {
	args := m.Called(ctx, hotelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.HotelDetailedInfo), args.Error(1)
}

func (m *MockPOIRepository) FindRestaurantDetails(ctx context.Context, cityID uuid.UUID, lat, lon, tolerance float64, preferences *models.RestaurantUserPreferences) ([]models.RestaurantDetailedInfo, error) {
	args := m.Called(ctx, cityID, lat, lon, tolerance, preferences)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.RestaurantDetailedInfo), args.Error(1)
}

func (m *MockPOIRepository) SaveRestaurantDetails(ctx context.Context, restaurant models.RestaurantDetailedInfo, cityID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, restaurant, cityID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockPOIRepository) GetRestaurantByID(ctx context.Context, restaurantID uuid.UUID) (*models.RestaurantDetailedInfo, error) {
	args := m.Called(ctx, restaurantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RestaurantDetailedInfo), args.Error(1)
}

func (m *MockPOIRepository) GetItinerary(ctx context.Context, userID, itineraryID uuid.UUID) (*models.UserSavedItinerary, error) {
	args := m.Called(ctx, userID, itineraryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserSavedItinerary), args.Error(1)
}

func (m *MockPOIRepository) GetItineraries(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]models.UserSavedItinerary, int, error) {
	args := m.Called(ctx, userID, page, pageSize)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]models.UserSavedItinerary), args.Get(1).(int), args.Error(2)
}

func (m *MockPOIRepository) GetItineraryByUserIDAndCityID(ctx context.Context, userID, cityID uuid.UUID) (*models.UserSavedItinerary, error) {
	args := m.Called(ctx, userID, cityID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserSavedItinerary), args.Error(1)
}

func (m *MockPOIRepository) UpdateItinerary(ctx context.Context, userID uuid.UUID, itineraryID uuid.UUID, updates models.UpdateItineraryRequest) (*models.UserSavedItinerary, error) {
	args := m.Called(ctx, userID, itineraryID, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserSavedItinerary), args.Error(1)
}

func (m *MockPOIRepository) SaveItinerary(ctx context.Context, userID, cityID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, userID, cityID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockPOIRepository) SaveItineraryPOIs(ctx context.Context, itineraryID uuid.UUID, pois []models.POIDetailedInfo) error {
	args := m.Called(ctx, itineraryID, pois)
	return args.Error(0)
}

func (m *MockPOIRepository) SavePOItoPointsOfInterest(ctx context.Context, poi models.POIDetailedInfo, cityID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, poi, cityID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockPOIRepository) CityExists(ctx context.Context, cityID uuid.UUID) (bool, error) {
	args := m.Called(ctx, cityID)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockPOIRepository) CalculateDistancePostGIS(ctx context.Context, userLat, userLon, poiLat, poiLon float64) (float64, error) {
	args := m.Called(ctx, userLat, userLon, poiLat, poiLon)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockPOIRepository) SaveLlmPoisToDatabase(ctx context.Context, userID uuid.UUID, pois []models.POIDetailedInfo, genAIResponse *models.GenAIResponse, llmInteractionID uuid.UUID) error {
	args := m.Called(ctx, userID, pois, genAIResponse, llmInteractionID)
	return args.Error(0)
}

func (m *MockPOIRepository) SaveLlmInteraction(ctx context.Context, interaction *models.LlmInteraction) (uuid.UUID, error) {
	args := m.Called(ctx, interaction)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockPOIRepository) CreateLLMPOI(ctx context.Context, poiData *models.POIDetailedInfo) (uuid.UUID, error) {
	args := m.Called(ctx, poiData)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockPOIRepository) FindLLMPOIByNameAndCity(ctx context.Context, name, city string) (uuid.UUID, error) {
	args := m.Called(ctx, name, city)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockPOIRepository) FindLLMPOIByName(ctx context.Context, name string) (uuid.UUID, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockPOIRepository) RemoveLLMPoiFromFavouriteByName(ctx context.Context, userID uuid.UUID, poiName string) error {
	args := m.Called(ctx, userID, poiName)
	return args.Error(0)
}

// Helper to setup service with mock repository
func setupPOIServiceTest() (*ServiceImpl, *MockPOIRepository, *MockCityRepository) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})) // or io.Discard
	mockRepo := new(MockPOIRepository)
	mockCityRepo := new(MockCityRepository)
	embeddingService := &generativeAI.EmbeddingService{} // Mock or nil
	service := NewServiceImpl(mockRepo, embeddingService, mockCityRepo, logger)
	return service, mockRepo, mockCityRepo
}

func TestPOIServiceImpl_AddPoiToFavourites(t *testing.T) {
	service, mockRepo, _ := setupPOIServiceTest()
	ctx := context.Background()
	userID := uuid.New()
	poiID := uuid.New()
	expectedFavouriteID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mockRepo.On("AddLLMPoiToFavourite", ctx, userID, poiID).Return(expectedFavouriteID, nil).Once()

		favID, err := service.AddPoiToFavourites(ctx, userID, poiID, true)
		require.NoError(t, err)
		assert.Equal(t, expectedFavouriteID, favID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		expectedErr := errors.New("db error")
		mockRepo.On("AddLLMPoiToFavourite", ctx, userID, poiID).Return(uuid.Nil, expectedErr).Once()

		_, err := service.AddPoiToFavourites(ctx, userID, poiID, true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error()) // Service just passes through the error
		mockRepo.AssertExpectations(t)
	})
}

func TestMain(m *testing.M) {
	// Load .env file for tests
	if err := godotenv.Load(); err != nil {
		slog.Error("Error loading .env file in tests", "error", err)
	}

	// Run tests
	os.Exit(m.Run())
}
