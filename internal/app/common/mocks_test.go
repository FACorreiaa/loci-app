package common

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/FACorreiaa/go-templui/internal/app/models"
)

// Mock Repository for testing
type MockRepository struct{}

func (m *MockRepository) GetLatestInteractionBySessionID(_ context.Context, sessionID uuid.UUID) (*models.LlmInteraction, error) {
	return &models.LlmInteraction{
		ID:           uuid.New(),
		UserID:       uuid.New(),
		Prompt:       "Test message",
		ResponseText: `{"generalCityData":{"city":"Test City"},"pointsOfInterest":[]}`,
		SessionID:    sessionID,
	}, nil
}

func (m *MockRepository) GetOrCreatePOI(_ context.Context, _ pgx.Tx, _ models.POIDetailedInfo, _ uuid.UUID, _ uuid.UUID) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (m *MockRepository) SaveSinglePOI(_ context.Context, _ models.POIDetailedInfo, _ uuid.UUID, _ uuid.UUID, _ uuid.UUID) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (m *MockRepository) GetPOIsBySessionSortedByDistance(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ models.UserLocation) ([]models.POIDetailedInfo, error) {
	return []models.POIDetailedInfo{}, nil
}

func (m *MockRepository) AddMessageToSession(_ context.Context, _ uuid.UUID, _ models.ConversationMessage) error {
	return nil
}

func (m *MockRepository) CreateSession(_ context.Context, _ models.ChatSession) error {
	return nil
}

func (m *MockRepository) DeleteInteraction(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *MockRepository) GetUserChatSessions(_ context.Context, _ uuid.UUID, _ int, _ int) (*models.ChatSessionsResponse, error) {
	return &models.ChatSessionsResponse{}, nil
}

func (m *MockRepository) GetRecentChatSessions(_ context.Context, _ uuid.UUID, _ int) ([]models.ChatSession, error) {
	return []models.ChatSession{}, nil
}

func (m *MockRepository) GetRecentChatSessionsByType(_ context.Context, _ uuid.UUID, _ models.SearchType, _ int) ([]models.ChatSession, error) {
	return []models.ChatSession{}, nil
}

func (m *MockRepository) AddChatToBookmark(_ context.Context, _ *models.UserSavedItinerary) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (m *MockRepository) GetBookmarkedItineraries(_ context.Context, _ uuid.UUID, _ int, _ int) (*models.PaginatedUserItinerariesResponse, error) {
	return &models.PaginatedUserItinerariesResponse{}, nil
}

func (m *MockRepository) RemoveChatFromBookmark(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

func (m *MockRepository) GetInteractionByID(_ context.Context, _ uuid.UUID) (*models.LlmInteraction, error) {
	return &models.LlmInteraction{}, nil
}

func (m *MockRepository) SaveInteraction(_ context.Context, _ models.LlmInteraction) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (m *MockRepository) SaveLlmSuggestedPOIsBatch(_ context.Context, _ []models.POIDetailedInfo, _ uuid.UUID, _ uuid.UUID, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

func (m *MockRepository) GetLlmSuggestedPOIsByInteractionSortedByDistance(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ models.UserLocation) ([]models.POIDetailedInfo, error) {
	return []models.POIDetailedInfo{}, nil
}

func (m *MockRepository) GetSession(_ context.Context, _ uuid.UUID) (*models.ChatSession, error) {
	return &models.ChatSession{}, nil
}

func (m *MockRepository) UpdateSession(_ context.Context, _ models.ChatSession) error {
	return nil
}

func (m *MockRepository) SaveItineraryPOIs(_ context.Context, _ uuid.UUID, _ []models.POIDetailedInfo) error {
	return nil
}

// Mock LLM Service
type MockLlmService struct{}

func (m *MockLlmService) SaveItenerary(_ context.Context, _ uuid.UUID, _ models.BookmarkRequest) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (m *MockLlmService) GetBookmarkedItineraries(_ context.Context, _ uuid.UUID, _ int, _ int) (*models.PaginatedUserItinerariesResponse, error) {
	return &models.PaginatedUserItinerariesResponse{}, nil
}

func (m *MockLlmService) RemoveItenerary(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

func (m *MockLlmService) GetPOIDetailedInfosResponse(_ context.Context, _ uuid.UUID, _ string, _ float64, _ float64) (*models.POIDetailedInfo, error) {
	return &models.POIDetailedInfo{}, nil
}

func (m *MockLlmService) ContinueSessionStreamed(_ context.Context, _ uuid.UUID, _ string, _ *models.UserLocation, eventCh chan<- models.StreamEvent) error {
	close(eventCh)
	return nil
}

func (m *MockLlmService) ProcessUnifiedChatMessageStream(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string, _ string, _ *models.UserLocation, eventCh chan<- models.StreamEvent) error {
	close(eventCh)
	return nil
}

func (m *MockLlmService) GetUserChatSessions(_ context.Context, _ uuid.UUID, _ int, _ int) (*models.ChatSessionsResponse, error) {
	return &models.ChatSessionsResponse{}, nil
}

func (m *MockLlmService) GetRecentDiscoveries(_ context.Context, _ uuid.UUID, _ int) (*models.ChatSessionsResponse, error) {
	return &models.ChatSessionsResponse{}, nil
}

func (m *MockLlmService) ProcessUnifiedChatMessageStreamFree(_ context.Context, _ string, _ string, _ *models.UserLocation, eventCh chan<- models.StreamEvent) error {
	close(eventCh)
	return nil
}

// Mock Profile Service
type MockProfileService struct{}

func (m *MockProfileService) GetSearchProfiles(_ context.Context, _ uuid.UUID) ([]models.UserPreferenceProfileResponse, error) {
	return []models.UserPreferenceProfileResponse{}, nil
}

func (m *MockProfileService) GetSearchProfile(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*models.UserPreferenceProfileResponse, error) {
	return &models.UserPreferenceProfileResponse{}, nil
}

func (m *MockProfileService) GetDefaultSearchProfile(_ context.Context, _ uuid.UUID) (*models.UserPreferenceProfileResponse, error) {
	return &models.UserPreferenceProfileResponse{}, nil
}

func (m *MockProfileService) CreateSearchProfile(_ context.Context, _ uuid.UUID, _ models.CreateUserPreferenceProfileParams) (*models.UserPreferenceProfileResponse, error) {
	return &models.UserPreferenceProfileResponse{}, nil
}

func (m *MockProfileService) UpdateSearchProfile(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ models.UpdateSearchProfileParams) error {
	return nil
}

func (m *MockProfileService) DeleteSearchProfile(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

func (m *MockProfileService) SetDefaultSearchProfile(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}
