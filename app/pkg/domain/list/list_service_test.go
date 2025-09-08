package itinerarylist

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/FACorreiaa/go-templui/app/internal/models"
)

// MockListRepository is a mock implementation of Repository
type MockListRepository struct {
	mock.Mock
}

func (m *MockListRepository) CreateList(ctx context.Context, list models.List) error {
	args := m.Called(ctx, list)
	return args.Error(0)
}

func (m *MockListRepository) GetList(ctx context.Context, listID uuid.UUID) (models.List, error) {
	args := m.Called(ctx, listID)
	if args.Get(0) == nil {
		return models.List{}, args.Error(1)
	}
	return args.Get(0).(models.List), args.Error(1)
}

func (m *MockListRepository) UpdateList(ctx context.Context, list models.List) error {
	args := m.Called(ctx, list)
	return args.Error(0)
}

func (m *MockListRepository) DeleteList(ctx context.Context, listID uuid.UUID) error {
	args := m.Called(ctx, listID)
	return args.Error(0)
}

func (m *MockListRepository) GetListItems(ctx context.Context, listID uuid.UUID) ([]*models.ListItem, error) {
	args := m.Called(ctx, listID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ListItem), args.Error(1)
}

func (m *MockListRepository) AddListItem(ctx context.Context, item models.ListItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *MockListRepository) GetListItem(ctx context.Context, listID, itemID uuid.UUID, contentType string) (models.ListItem, error) {
	args := m.Called(ctx, listID, itemID, contentType)
	if args.Get(0) == nil {
		return models.ListItem{}, args.Error(1)
	}
	return args.Get(0).(models.ListItem), args.Error(1)
}

func (m *MockListRepository) UpdateListItem(ctx context.Context, item models.ListItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *MockListRepository) DeleteListItem(ctx context.Context, listID, itemID uuid.UUID, contentType string) error {
	args := m.Called(ctx, listID, itemID, contentType)
	return args.Error(0)
}

func (m *MockListRepository) GetUserLists(ctx context.Context, userID uuid.UUID, isItinerary bool) ([]*models.List, error) {
	args := m.Called(ctx, userID, isItinerary)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.List), args.Error(1)
}

func (m *MockListRepository) GetListItemByID(ctx context.Context, listID, itemID uuid.UUID) (models.ListItem, error) {
	args := m.Called(ctx, listID, itemID)
	if args.Get(0) == nil {
		return models.ListItem{}, args.Error(1)
	}
	return args.Get(0).(models.ListItem), args.Error(1)
}

func (m *MockListRepository) DeleteListItemByID(ctx context.Context, listID, itemID uuid.UUID) error {
	args := m.Called(ctx, listID, itemID)
	return args.Error(0)
}

func (m *MockListRepository) GetSubLists(ctx context.Context, parentListID uuid.UUID) ([]*models.List, error) {
	args := m.Called(ctx, parentListID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.List), args.Error(1)
}

// Saved Lists functionality
func (m *MockListRepository) SaveList(ctx context.Context, userID, listID uuid.UUID) error {
	args := m.Called(ctx, userID, listID)
	return args.Error(0)
}

func (m *MockListRepository) UnsaveList(ctx context.Context, userID, listID uuid.UUID) error {
	args := m.Called(ctx, userID, listID)
	return args.Error(0)
}

func (m *MockListRepository) GetUserSavedLists(ctx context.Context, userID uuid.UUID) ([]*models.List, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.List), args.Error(1)
}

// Content type specific methods
func (m *MockListRepository) GetListItemsByContentType(ctx context.Context, listID uuid.UUID, contentType models.ContentType) ([]*models.ListItem, error) {
	args := m.Called(ctx, listID, contentType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ListItem), args.Error(1)
}

// Search and filtering
func (m *MockListRepository) SearchLists(ctx context.Context, searchTerm, contentType string, cityID *uuid.UUID) ([]*models.List, error) {
	args := m.Called(ctx, searchTerm, contentType, cityID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.List), args.Error(1)
}

// Helper to setup service with mock repository
func setupListServiceTest() (*ServiceImpl, *MockListRepository) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	mockRepo := new(MockListRepository)
	service := NewServiceImpl(mockRepo, logger)
	return service, mockRepo
}

func TestServiceImpl_CreateTopLevelList(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	cityID := uuid.New()

	t.Run("success - with city", func(t *testing.T) {
		service, mockRepo := setupListServiceTest()

		mockRepo.On("CreateList", mock.Anything, mock.AnythingOfType("models.List")).Return(nil).Once()

		result, err := service.CreateTopLevelList(ctx, userID, "Test List", "Test Description", &cityID, false, true)

		require.NoError(t, err)
		assert.Equal(t, userID, result.UserID)
		assert.Equal(t, "Test List", result.Name)
		assert.Equal(t, "Test Description", result.Description)
		assert.Equal(t, &cityID, result.CityID)
		assert.False(t, result.IsItinerary)
		assert.True(t, result.IsPublic)
		mockRepo.AssertExpectations(t)
	})

	t.Run("success - without city", func(t *testing.T) {
		service, mockRepo := setupListServiceTest()

		mockRepo.On("CreateList", mock.Anything, mock.AnythingOfType("models.List")).Return(nil).Once()

		result, err := service.CreateTopLevelList(ctx, userID, "Test List", "Test Description", nil, true, false)

		require.NoError(t, err)
		assert.Equal(t, userID, result.UserID)
		assert.True(t, result.IsItinerary)
		assert.False(t, result.IsPublic)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		service, mockRepo := setupListServiceTest()

		repoErr := errors.New("db error")
		mockRepo.On("CreateList", mock.Anything, mock.AnythingOfType("models.List")).Return(repoErr).Once()

		_, err := service.CreateTopLevelList(ctx, userID, "Test List", "Test Description", nil, false, false)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create list")
		mockRepo.AssertExpectations(t)
	})
}

func TestServiceImpl_CreateItineraryForList(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	parentListID := uuid.New()
	cityID := uuid.New()

	parentList := models.List{
		ID:          parentListID,
		UserID:      userID,
		Name:        "Parent List",
		CityID:      &cityID,
		IsItinerary: false,
		IsPublic:    true,
	}

	t.Run("success", func(t *testing.T) {
		service, mockRepo := setupListServiceTest()

		mockRepo.On("GetList", mock.Anything, parentListID).Return(parentList, nil).Once()
		mockRepo.On("CreateList", mock.Anything, mock.AnythingOfType("models.List")).Return(nil).Once()

		result, err := service.CreateItineraryForList(ctx, userID, parentListID, "Test Itinerary", "Test Description", false)

		require.NoError(t, err)
		assert.Equal(t, userID, result.UserID)
		assert.Equal(t, "Test Itinerary", result.Name)
		assert.True(t, result.IsItinerary)
		assert.Equal(t, parentListID, *result.ParentListID)
		assert.Equal(t, &cityID, result.CityID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("parent list not found", func(t *testing.T) {
		service, mockRepo := setupListServiceTest()

		repoErr := errors.New("list not found")
		mockRepo.On("GetList", mock.Anything, parentListID).Return(nil, repoErr).Once()

		_, err := service.CreateItineraryForList(ctx, userID, parentListID, "Test Itinerary", "Test Description", false)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "parent list not found")
		mockRepo.AssertExpectations(t)
	})

	t.Run("user does not own parent list", func(t *testing.T) {
		service, mockRepo := setupListServiceTest()

		otherUserList := models.List{
			ID:          parentListID,
			UserID:      uuid.New(), // Different user
			Name:        "Parent List",
			CityID:      &cityID,
			IsItinerary: false,
			IsPublic:    true,
		}

		mockRepo.On("GetList", mock.Anything, parentListID).Return(otherUserList, nil).Once()

		_, err := service.CreateItineraryForList(ctx, userID, parentListID, "Test Itinerary", "Test Description", false)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "user does not own parent list")
		mockRepo.AssertExpectations(t)
	})
}

func TestServiceImpl_GetListDetails(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	listID := uuid.New()

	list := models.List{
		ID:          listID,
		UserID:      userID,
		Name:        "Test List",
		IsItinerary: true,
		IsPublic:    false,
	}

	items := []*models.ListItem{
		{
			ListID:      listID,
			ItemID:      uuid.New(),
			ContentType: models.ContentTypePOI,
			Position:    1,
			Notes:       "First POI",
		},
		{
			ListID:      listID,
			ItemID:      uuid.New(),
			ContentType: models.ContentTypePOI,
			Position:    2,
			Notes:       "Second POI",
		},
	}

	t.Run("success - owner access", func(t *testing.T) {
		service, mockRepo := setupListServiceTest()

		mockRepo.On("GetList", mock.Anything, listID).Return(list, nil).Once()
		mockRepo.On("GetListItems", mock.Anything, listID).Return(items, nil).Once()

		result, err := service.GetListDetails(ctx, listID, userID)

		require.NoError(t, err)
		assert.Equal(t, list, result.List)
		assert.Equal(t, items, result.Items)
		assert.Len(t, result.Items, 2)
		mockRepo.AssertExpectations(t)
	})

	t.Run("success - public list access", func(t *testing.T) {
		service, mockRepo := setupListServiceTest()

		publicList := models.List{
			ID:          listID,
			UserID:      uuid.New(), // Different user
			Name:        "Public Test List",
			IsItinerary: true,
			IsPublic:    true, // Public list
		}

		mockRepo.On("GetList", mock.Anything, listID).Return(publicList, nil).Once()
		mockRepo.On("GetListItems", mock.Anything, listID).Return(items, nil).Once()

		result, err := service.GetListDetails(ctx, listID, userID)

		require.NoError(t, err)
		assert.Equal(t, publicList, result.List)
		mockRepo.AssertExpectations(t)
	})

	t.Run("access denied - private list", func(t *testing.T) {
		service, mockRepo := setupListServiceTest()

		privateList := models.List{
			ID:          listID,
			UserID:      uuid.New(), // Different user
			Name:        "Private Test List",
			IsItinerary: true,
			IsPublic:    false, // Private list
		}

		mockRepo.On("GetList", mock.Anything, listID).Return(privateList, nil).Once()

		_, err := service.GetListDetails(ctx, listID, userID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied to list")
		mockRepo.AssertExpectations(t)
	})

	t.Run("list not found", func(t *testing.T) {
		service, mockRepo := setupListServiceTest()

		repoErr := errors.New("list not found")
		mockRepo.On("GetList", mock.Anything, listID).Return(nil, repoErr).Once()

		_, err := service.GetListDetails(ctx, listID, userID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "list not found")
		mockRepo.AssertExpectations(t)
	})
}

func TestServiceImpl_UpdateListDetails(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	listID := uuid.New()

	list := models.List{
		ID:          listID,
		UserID:      userID,
		Name:        "Old Name",
		Description: "Old Description",
		IsPublic:    false,
	}

	t.Run("success", func(t *testing.T) {
		service, mockRepo := setupListServiceTest()
		newName := "New Name"
		newDescription := "New Description"
		isPublic := true

		params := models.UpdateListRequest{
			Name:        &newName,
			Description: &newDescription,
			IsPublic:    &isPublic,
		}

		mockRepo.On("GetList", mock.Anything, listID).Return(list, nil).Once()
		mockRepo.On("UpdateList", mock.Anything, mock.AnythingOfType("models.List")).Return(nil).Once()

		result, err := service.UpdateListDetails(ctx, listID, userID, params)

		require.NoError(t, err)
		assert.Equal(t, newName, result.Name)
		assert.Equal(t, newDescription, result.Description)
		assert.True(t, result.IsPublic)
		mockRepo.AssertExpectations(t)
	})

	t.Run("user does not own list", func(t *testing.T) {
		service, mockRepo := setupListServiceTest()

		otherUserList := models.List{
			ID:     listID,
			UserID: uuid.New(), // Different user
			Name:   "Test List",
		}

		mockRepo.On("GetList", mock.Anything, listID).Return(otherUserList, nil).Once()

		_, err := service.UpdateListDetails(ctx, listID, userID, models.UpdateListRequest{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "user does not own list")
		mockRepo.AssertExpectations(t)
	})
}

func TestServiceImpl_AddPOIListItem(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	listID := uuid.New()
	poiID := uuid.New()

	list := models.List{
		ID:          listID,
		UserID:      userID,
		Name:        "Test Itinerary",
		IsItinerary: true,
	}

	t.Run("success", func(t *testing.T) {
		service, mockRepo := setupListServiceTest()
		morningTime := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
		params := models.AddListItemRequest{
			Position:        1,
			Notes:           "Test POI",
			DayNumber:       &[]int{1}[0],
			TimeSlot:        &morningTime,
			DurationMinutes: &[]int{60}[0],
		}

		mockRepo.On("GetList", mock.Anything, listID).Return(list, nil).Once()
		mockRepo.On("AddListItem", mock.Anything, mock.AnythingOfType("models.ListItem")).Return(nil).Once()

		result, err := service.AddPOIListItem(ctx, userID, listID, poiID, params)

		require.NoError(t, err)
		assert.Equal(t, listID, result.ListID)
		assert.Equal(t, poiID, result.ItemID)
		assert.Equal(t, 1, result.Position)
		mockRepo.AssertExpectations(t)
	})

	t.Run("list is not an itinerary", func(t *testing.T) {
		service, mockRepo := setupListServiceTest()

		nonItineraryList := models.List{
			ID:          listID,
			UserID:      userID,
			Name:        "Regular List",
			IsItinerary: false,
		}

		mockRepo.On("GetList", mock.Anything, listID).Return(nonItineraryList, nil).Once()

		_, err := service.AddPOIListItem(ctx, userID, listID, poiID, models.AddListItemRequest{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "list is not an itinerary")
		mockRepo.AssertExpectations(t)
	})
}

func TestServiceImpl_GetUserLists(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	expectedLists := []*models.List{
		{
			ID:          uuid.New(),
			UserID:      userID,
			Name:        "List 1",
			IsItinerary: false,
		},
		{
			ID:          uuid.New(),
			UserID:      userID,
			Name:        "List 2",
			IsItinerary: false,
		},
	}

	t.Run("success", func(t *testing.T) {
		service, mockRepo := setupListServiceTest()

		mockRepo.On("GetUserLists", mock.Anything, userID, false).Return(expectedLists, nil).Once()

		result, err := service.GetUserLists(ctx, userID, false)

		require.NoError(t, err)
		assert.Equal(t, expectedLists, result)
		assert.Len(t, result, 2)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		service, mockRepo := setupListServiceTest()

		repoErr := errors.New("db error")
		mockRepo.On("GetUserLists", mock.Anything, userID, true).Return(nil, repoErr).Once()

		_, err := service.GetUserLists(ctx, userID, true)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get user lists")
		mockRepo.AssertExpectations(t)
	})
}

func TestServiceImpl_DeleteUserList(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	listID := uuid.New()

	list := models.List{
		ID:     listID,
		UserID: userID,
		Name:   "Test List",
	}

	t.Run("success", func(t *testing.T) {
		service, mockRepo := setupListServiceTest()

		mockRepo.On("GetList", mock.Anything, listID).Return(list, nil).Once()
		mockRepo.On("DeleteList", mock.Anything, listID).Return(nil).Once()

		err := service.DeleteUserList(ctx, listID, userID)

		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("user does not own list", func(t *testing.T) {
		service, mockRepo := setupListServiceTest()

		otherUserList := models.List{
			ID:     listID,
			UserID: uuid.New(), // Different user
			Name:   "Test List",
		}

		mockRepo.On("GetList", mock.Anything, listID).Return(otherUserList, nil).Once()

		err := service.DeleteUserList(ctx, listID, userID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "user does not own list")
		mockRepo.AssertExpectations(t)
	})

	t.Run("list not found", func(t *testing.T) {
		service, mockRepo := setupListServiceTest()

		repoErr := errors.New("list not found")
		mockRepo.On("GetList", mock.Anything, listID).Return(nil, repoErr).Once()

		err := service.DeleteUserList(ctx, listID, userID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "list not found")
		mockRepo.AssertExpectations(t)
	})
}
