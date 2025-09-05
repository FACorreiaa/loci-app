package llmchat

import (
	"context"

	"github.com/FACorreiaa/go-templui/app/lib/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/mock"
	"google.golang.org/genai" // For genai.GenerateContentConfig
)

// --- Mocks for Dependencies ---

// Mock AIClient
type MockAIClient struct {
	mock.Mock
}

// Ensure MockAIClient satisfies an interface if ServiceImpl uses one.
// For now, assuming direct use of *generativeAI.AIClient struct type.
// To make this more testable, ServiceImpl should ideally depend on an interface for AIClient.
// Let's define a minimal interface that AIClient should satisfy for our service's needs:
type AIClientInterface interface {
	GenerateResponse(ctx context.Context, prompt string, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error)
	// Add other methods used by ServiceImpl if any, e.g., StartChatSession
}

func (m *MockAIClient) GenerateResponse(ctx context.Context, prompt string, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	args := m.Called(ctx, prompt, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*genai.GenerateContentResponse), args.Error(1)
}

// Mock Repositories (Example for POIRepository, create similar for others)
type MockPOIRepository struct {
	mock.Mock
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

func (m *MockPOIRepository) RemovePoiFromFavourites(ctx context.Context, poiID uuid.UUID, userID uuid.UUID) error {
	args := m.Called(ctx, poiID, userID)
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

func (m *MockPOIRepository) FindPOIDetailedInfos(ctx context.Context, cityID uuid.UUID, lat, lon float64, tolerance float64) (*models.POIDetailedInfo, error) {
	args := m.Called(ctx, cityID, lat, lon, tolerance)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.POIDetailedInfo), args.Error(1)
}

func (m *MockPOIRepository) SavePOIDetailedInfos(ctx context.Context, poi models.POIDetailedInfo, cityID uuid.UUID) (uuid.UUID, error) {
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

type MockCityRepository struct{ mock.Mock }

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

func (m *MockCityRepository) GetCityIDByName(ctx context.Context, cityName string) (uuid.UUID, error) {
	args := m.Called(ctx, cityName)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockCityRepository) GetAllCities(ctx context.Context) ([]models.CityDetail, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.CityDetail), args.Error(1)
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

func (m *MockCityRepository) GetCity(ctx context.Context, lat, lon float64) (uuid.UUID, string, error) {
	args := m.Called(ctx, lat, lon)
	return args.Get(0).(uuid.UUID), args.Get(1).(string), args.Error(2)
}

type MockLLMInteractionRepository struct{ mock.Mock }

func (m *MockLLMInteractionRepository) SaveInteraction(ctx context.Context, interaction models.LlmInteraction) (uuid.UUID, error) {
	args := m.Called(ctx, interaction)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockLLMInteractionRepository) SaveLlmSuggestedPOIsBatch(ctx context.Context, pois []models.POIDetailedInfo, userID, searchProfileID, llmInteractionID, cityID uuid.UUID) error {
	args := m.Called(ctx, pois, userID, searchProfileID, llmInteractionID, cityID)
	return args.Error(0)
}

func (m *MockLLMInteractionRepository) GetLlmSuggestedPOIsByInteractionSortedByDistance(ctx context.Context, llmInteractionID uuid.UUID, cityID uuid.UUID, userLocation models.UserLocation) ([]models.POIDetailedInfo, error) {
	args := m.Called(ctx, llmInteractionID, cityID, userLocation)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.POIDetailedInfo), args.Error(1)
}

func (m *MockLLMInteractionRepository) AddChatToBookmark(ctx context.Context, itinerary *models.UserSavedItinerary) (uuid.UUID, error) {
	args := m.Called(ctx, itinerary)
	if args.Get(0) == nil {
		return uuid.Nil, args.Error(1)
	}
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockLLMInteractionRepository) RemoveChatFromBookmark(ctx context.Context, userID, itineraryID uuid.UUID) error {
	args := m.Called(ctx, userID, itineraryID)
	return args.Error(0)
}

func (m *MockLLMInteractionRepository) GetInteractionByID(ctx context.Context, interactionID uuid.UUID) (*models.LlmInteraction, error) {
	args := m.Called(ctx, interactionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LlmInteraction), args.Error(1)
}

func (m *MockLLMInteractionRepository) CreateSession(ctx context.Context, session models.ChatSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockLLMInteractionRepository) GetSession(ctx context.Context, sessionID uuid.UUID) (*models.ChatSession, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ChatSession), args.Error(1)
}

func (m *MockLLMInteractionRepository) GetUserChatSessions(ctx context.Context, userID uuid.UUID) ([]models.ChatSession, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.ChatSession), args.Error(1)
}

func (m *MockLLMInteractionRepository) UpdateSession(ctx context.Context, session models.ChatSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockLLMInteractionRepository) AddMessageToSession(ctx context.Context, sessionID uuid.UUID, message models.ConversationMessage) error {
	args := m.Called(ctx, sessionID, message)
	return args.Error(0)
}

func (m *MockLLMInteractionRepository) SaveSinglePOI(ctx context.Context, poi models.POIDetailedInfo, userID, cityID uuid.UUID, llmInteractionID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, poi, userID, cityID, llmInteractionID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockLLMInteractionRepository) GetPOIsBySessionSortedByDistance(ctx context.Context, sessionID, cityID uuid.UUID, userLocation models.UserLocation) ([]models.POIDetailedInfo, error) {
	args := m.Called(ctx, sessionID, cityID, userLocation)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.POIDetailedInfo), args.Error(1)
}

func (m *MockLLMInteractionRepository) CalculateDistancePostGIS(ctx context.Context, userLat, userLon, poiLat, poiLon float64) (float64, error) {
	args := m.Called(ctx, userLat, userLon, poiLat, poiLon)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockLLMInteractionRepository) GetOrCreatePOI(ctx context.Context, tx pgx.Tx, POIDetailedInfo models.POIDetailedInfo, cityID uuid.UUID, sourceInteractionID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, tx, POIDetailedInfo, cityID, sourceInteractionID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

type MockinterestsRepo struct{ mock.Mock }

func (m *MockinterestsRepo) CreateInterest(ctx context.Context, name string, description *string, isActive bool, userID string) (*models.Interest, error) {
	args := m.Called(ctx, name, description, isActive, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Interest), args.Error(1)
}

func (m *MockinterestsRepo) Removeinterests(ctx context.Context, userID uuid.UUID, interestID uuid.UUID) error {
	args := m.Called(ctx, userID, interestID)
	return args.Error(0)
}

func (m *MockinterestsRepo) GetAllInterests(ctx context.Context) ([]*models.Interest, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Interest), args.Error(1)
}

func (m *MockinterestsRepo) GetInterest(ctx context.Context, interestID uuid.UUID) (*models.Interest, error) {
	args := m.Called(ctx, interestID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Interest), args.Error(1)
}

func (m *MockinterestsRepo) Updateinterests(ctx context.Context, userID uuid.UUID, interestID uuid.UUID, params models.UpdateinterestsParams) error {
	args := m.Called(ctx, userID, interestID, params)
	return args.Error(0)
}

func (m *MockinterestsRepo) AddInterestToProfile(ctx context.Context, profileID, interestID uuid.UUID) error {
	args := m.Called(ctx, profileID, interestID)
	return args.Error(0)
}

func (m *MockinterestsRepo) GetInterestsForProfile(ctx context.Context, profileID uuid.UUID) ([]*models.Interest, error) {
	args := m.Called(ctx, profileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Interest), args.Error(1)
}

type MockSearchProfileRepo struct{ mock.Mock }

func (m *MockSearchProfileRepo) GetSearchProfiles(ctx context.Context, userID uuid.UUID) ([]models.UserPreferenceProfileResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.UserPreferenceProfileResponse), args.Error(1)
}

func (m *MockSearchProfileRepo) GetSearchProfile(ctx context.Context, userID, profileID uuid.UUID) (*models.UserPreferenceProfileResponse, error) {
	args := m.Called(ctx, userID, profileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserPreferenceProfileResponse), args.Error(1)
}

func (m *MockSearchProfileRepo) GetDefaultSearchProfile(ctx context.Context, userID uuid.UUID) (*models.UserPreferenceProfileResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserPreferenceProfileResponse), args.Error(1)
}

func (m *MockSearchProfileRepo) CreateSearchProfile(ctx context.Context, userID uuid.UUID, params models.CreateUserPreferenceProfileParams) (*models.UserPreferenceProfileResponse, error) {
	args := m.Called(ctx, userID, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserPreferenceProfileResponse), args.Error(1)
}

func (m *MockSearchProfileRepo) UpdateSearchProfile(ctx context.Context, userID, profileID uuid.UUID, params models.UpdateSearchProfileParams) error {
	args := m.Called(ctx, userID, profileID, params)
	return args.Error(0)
}

func (m *MockSearchProfileRepo) DeleteSearchProfile(ctx context.Context, userID, profileID uuid.UUID) error {
	args := m.Called(ctx, userID, profileID)
	return args.Error(0)
}

func (m *MockSearchProfileRepo) SetDefaultSearchProfile(ctx context.Context, userID, profileID uuid.UUID) error {
	args := m.Called(ctx, userID, profileID)
	return args.Error(0)
}

type MockTagsRepo struct{ mock.Mock }

func (m *MockTagsRepo) GetAll(ctx context.Context, userID uuid.UUID) ([]*models.Tags, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Tags), args.Error(1)
}

func (m *MockTagsRepo) Get(ctx context.Context, userID, tagID uuid.UUID) (*models.Tags, error) {
	args := m.Called(ctx, userID, tagID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Tags), args.Error(1)
}

func (m *MockTagsRepo) Create(ctx context.Context, userID uuid.UUID, params models.CreatePersonalTagParams) (*models.PersonalTag, error) {
	args := m.Called(ctx, userID, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PersonalTag), args.Error(1)
}

func (m *MockTagsRepo) Delete(ctx context.Context, userID uuid.UUID, tagID uuid.UUID) error {
	args := m.Called(ctx, userID, tagID)
	return args.Error(0)
}

func (m *MockTagsRepo) Update(ctx context.Context, userID, tagsID uuid.UUID, params models.UpdatePersonalTagParams) error {
	args := m.Called(ctx, userID, tagsID, params)
	return args.Error(0)
}

func (m *MockTagsRepo) GetTagByName(ctx context.Context, name string) (*models.Tags, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Tags), args.Error(1)
}

func (m *MockTagsRepo) LinkPersonalTagToProfile(ctx context.Context, userID, profileID uuid.UUID, tagID uuid.UUID) error {
	args := m.Called(ctx, userID, profileID, tagID)
	return args.Error(0)
}

func (m *MockTagsRepo) GetTagsForProfile(ctx context.Context, profileID uuid.UUID) ([]*models.Tags, error) {
	args := m.Called(ctx, profileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Tags), args.Error(1)
}

// Helper to setup service with mocks for each test

// Example for GetItinerary (simpler, as it's mostly a direct repo call)

// Add similar unit tests for:
// - GetItineraries
// - UpdateItinerary
// - SaveItenerary
// - RemoveItenerary
// - GetHotelsByPreferenceResponse (mocking repo's FindHotelDetails, and AI call if fallback)
// - GetRestaurantsByPreferencesResponse (mocking repo's FindRestaurantDetails, and AI call if fallback)
// - etc.

// --- Integration Tests for llmInteraction (Example for GetPOIDetailedInfosResponse) ---
// These would require a running database instance and potentially a configured AI client.
