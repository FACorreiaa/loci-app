package handlers

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/FACorreiaa/go-templui/app/internal/models"
)

// Mock Repository for testing
type MockRepository struct{}

func (m *MockRepository) CreateNewInteraction(ctx context.Context, userID uuid.UUID, message, response string, sessionID *uuid.UUID, provider, interactionType string) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (m *MockRepository) GetLatestInteractionBySessionID(ctx context.Context, sessionID uuid.UUID) (*models.LlmInteraction, error) {
	return &models.LlmInteraction{
		ID:           uuid.New(),
		UserID:       uuid.New(),
		Prompt:       "Test message",
		ResponseText: `{"generalCityData":{"city":"Test City"},"pointsOfInterest":[]}`,
		SessionID:    sessionID,
	}, nil
}

func (m *MockRepository) GetMessagesBySessionID(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]*models.LlmInteraction, error) {
	return []*models.LlmInteraction{}, nil
}

func (m *MockRepository) GetOrCreatePOI(ctx context.Context, tx pgx.Tx, poi models.POIDetailedInfo, cityID uuid.UUID, sourceInteractionID uuid.UUID) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (m *MockRepository) SaveSinglePOI(ctx context.Context, poi models.POIDetailedInfo, userID, cityID uuid.UUID, llmInteractionID uuid.UUID) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (m *MockRepository) GetPOIsBySessionSortedByDistance(ctx context.Context, sessionID, cityID uuid.UUID, userLocation models.UserLocation) ([]models.POIDetailedInfo, error) {
	return []models.POIDetailedInfo{}, nil
}

func (m *MockRepository) AddMessageToSession(ctx context.Context, sessionID uuid.UUID, message models.ConversationMessage) error {
	return nil
}

func (m *MockRepository) CreateSession(ctx context.Context, session models.ChatSession) error {
	return nil
}

func (m *MockRepository) DeleteInteraction(ctx context.Context, sessionID uuid.UUID) error {
	return nil
}

func (m *MockRepository) GetUserChatSessions(ctx context.Context, userID uuid.UUID, page, limit int) (*models.ChatSessionsResponse, error) {
	return &models.ChatSessionsResponse{}, nil
}

func (m *MockRepository) AddChatToBookmark(ctx context.Context, itinerary *models.UserSavedItinerary) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (m *MockRepository) GetBookmarkedItineraries(ctx context.Context, userID uuid.UUID, page, limit int) (*models.PaginatedUserItinerariesResponse, error) {
	return &models.PaginatedUserItinerariesResponse{}, nil
}

func (m *MockRepository) RemoveChatFromBookmark(ctx context.Context, userID, itineraryID uuid.UUID) error {
	return nil
}

func (m *MockRepository) GetInteractionByID(ctx context.Context, interactionID uuid.UUID) (*models.LlmInteraction, error) {
	return &models.LlmInteraction{}, nil
}

func (m *MockRepository) SaveInteraction(ctx context.Context, interaction models.LlmInteraction) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (m *MockRepository) SaveLlmSuggestedPOIsBatch(ctx context.Context, pois []models.POIDetailedInfo, userID, searchProfileID, llmInteractionID, cityID uuid.UUID) error {
	return nil
}

func (m *MockRepository) GetLlmSuggestedPOIsByInteractionSortedByDistance(ctx context.Context, llmInteractionID uuid.UUID, cityID uuid.UUID, userLocation models.UserLocation) ([]models.POIDetailedInfo, error) {
	return []models.POIDetailedInfo{}, nil
}

func (m *MockRepository) GetSession(ctx context.Context, sessionID uuid.UUID) (*models.ChatSession, error) {
	return &models.ChatSession{}, nil
}

func (m *MockRepository) UpdateSession(ctx context.Context, session models.ChatSession) error {
	return nil
}

func (m *MockRepository) SaveItineraryPOIs(ctx context.Context, itineraryID uuid.UUID, pois []models.POIDetailedInfo) error {
	return nil
}

// Mock LLM Service
type MockLlmService struct{}

func (m *MockLlmService) SaveItenerary(ctx context.Context, userID uuid.UUID, req models.BookmarkRequest) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (m *MockLlmService) GetBookmarkedItineraries(ctx context.Context, userID uuid.UUID, page, limit int) (*models.PaginatedUserItinerariesResponse, error) {
	return &models.PaginatedUserItinerariesResponse{}, nil
}

func (m *MockLlmService) RemoveItenerary(ctx context.Context, userID, itineraryID uuid.UUID) error {
	return nil
}

func (m *MockLlmService) GetPOIDetailedInfosResponse(ctx context.Context, userID uuid.UUID, city string, lat, lon float64) (*models.POIDetailedInfo, error) {
	return &models.POIDetailedInfo{}, nil
}

func (m *MockLlmService) ContinueSessionStreamed(ctx context.Context, sessionID uuid.UUID, message string, userLocation *models.UserLocation, eventCh chan<- models.StreamEvent) error {
	close(eventCh)
	return nil
}

func (m *MockLlmService) ProcessUnifiedChatMessageStream(ctx context.Context, userID, profileID uuid.UUID, cityName, message string, userLocation *models.UserLocation, eventCh chan<- models.StreamEvent) error {
	close(eventCh)
	return nil
}

func (m *MockLlmService) GetUserChatSessions(ctx context.Context, userID uuid.UUID, page, limit int) (*models.ChatSessionsResponse, error) {
	return &models.ChatSessionsResponse{}, nil
}

func (m *MockLlmService) ProcessUnifiedChatMessageStreamFree(ctx context.Context, cityName, message string, userLocation *models.UserLocation, eventCh chan<- models.StreamEvent) error {
	close(eventCh)
	return nil
}

// Mock Profile Service
type MockProfileService struct{}

func (m *MockProfileService) GetSearchProfiles(ctx context.Context, userID uuid.UUID) ([]models.UserPreferenceProfileResponse, error) {
	return []models.UserPreferenceProfileResponse{}, nil
}

func (m *MockProfileService) GetSearchProfile(ctx context.Context, userID, profileID uuid.UUID) (*models.UserPreferenceProfileResponse, error) {
	return &models.UserPreferenceProfileResponse{}, nil
}

func (m *MockProfileService) GetDefaultSearchProfile(ctx context.Context, userID uuid.UUID) (*models.UserPreferenceProfileResponse, error) {
	return &models.UserPreferenceProfileResponse{}, nil
}

func (m *MockProfileService) CreateSearchProfile(ctx context.Context, userID uuid.UUID, params models.CreateUserPreferenceProfileParams) (*models.UserPreferenceProfileResponse, error) {
	return &models.UserPreferenceProfileResponse{}, nil
}

func (m *MockProfileService) UpdateSearchProfile(ctx context.Context, userID, profileID uuid.UUID, params models.UpdateSearchProfileParams) error {
	return nil
}

func (m *MockProfileService) DeleteSearchProfile(ctx context.Context, userID, profileID uuid.UUID) error {
	return nil
}

func (m *MockProfileService) SetDefaultSearchProfile(ctx context.Context, userID, profileID uuid.UUID) error {
	return nil
}