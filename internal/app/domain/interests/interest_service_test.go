package interests

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/FACorreiaa/go-templui/internal/app/models"
)

// MockinterestsRepo is a mock implementation of interestsRepo
type MockinterestsRepo struct {
	mock.Mock
}

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

func (m *MockinterestsRepo) Updateinterests(ctx context.Context, userID uuid.UUID, interestID uuid.UUID, params models.UpdateinterestsParams) error {
	args := m.Called(ctx, userID, interestID, params)
	return args.Error(0)
}

func (m *MockinterestsRepo) GetInterest(ctx context.Context, interestID uuid.UUID) (*models.Interest, error) {
	args := m.Called(ctx, interestID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Interest), args.Error(1)
}

func (m *MockinterestsRepo) AddInterestToProfile(ctx context.Context, profileID uuid.UUID, interestID uuid.UUID) error {
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

func TestCreateInterest(t *testing.T) {
	// Test data
	name := "Test Interest"
	description := "Test Description"
	isActive := true
	userID := "user123"
	active := isActive
	ctx := context.Background()

	// Test cases
	tests := []struct {
		name          string
		setupMock     func(*MockinterestsRepo) *models.Interest
		expectedError bool
	}{
		{
			name: "Success",
			setupMock: func(mockRepo *MockinterestsRepo) *models.Interest {
				expectedInterest := &models.Interest{
					ID:          uuid.New(),
					Name:        name,
					Description: &description,
					Active:      &active,
					CreatedAt:   time.Now(),
					Source:      "test",
				}
				mockRepo.On("CreateInterest", mock.Anything, name, &description, isActive, userID).Return(expectedInterest, nil).Once()
				return expectedInterest
			},
			expectedError: false,
		},
		{
			name: "Repository Error",
			setupMock: func(mockRepo *MockinterestsRepo) *models.Interest {
				mockRepo.On("CreateInterest", mock.Anything, name, &description, isActive, userID).Return(nil, errors.New("repository error")).Once()
				return nil
			},
			expectedError: true,
		},
	}

	// Run tests
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup - create fresh mock and service for each test
			mockRepo := new(MockinterestsRepo)
			logger := slog.Default()
			service := NewinterestsService(mockRepo, logger)

			// Setup mock
			expectedInterest := tc.setupMock(mockRepo)

			// Call the method
			interest, err := service.CreateInterest(ctx, name, &description, isActive, userID)

			// Assertions
			if tc.expectedError {
				assert.Error(t, err)
				assert.Nil(t, interest)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, expectedInterest, interest)
			}

			// Verify mock
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestRemoveinterests(t *testing.T) {
	// Test data
	userID := uuid.New()
	interestID := uuid.New()
	ctx := context.Background()

	// Test cases
	tests := []struct {
		name          string
		setupMock     func(*MockinterestsRepo)
		expectedError bool
	}{
		{
			name: "Success",
			setupMock: func(mockRepo *MockinterestsRepo) {
				mockRepo.On("Removeinterests", mock.Anything, userID, interestID).Return(nil).Once()
			},
			expectedError: false,
		},
		{
			name: "Repository Error",
			setupMock: func(mockRepo *MockinterestsRepo) {
				mockRepo.On("Removeinterests", mock.Anything, userID, interestID).Return(errors.New("repository error")).Once()
			},
			expectedError: true,
		},
	}

	// Run tests
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup - create fresh mock and service for each test
			mockRepo := new(MockinterestsRepo)
			logger := slog.Default()
			service := NewinterestsService(mockRepo, logger)

			// Setup mock
			tc.setupMock(mockRepo)

			// Call the method
			err := service.Removeinterests(ctx, userID, interestID)

			// Assertions
			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify mock
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetAllInterests(t *testing.T) {
	// Test data
	active1 := true
	active2 := true
	now := time.Now()
	ctx := context.Background()

	// Test cases
	tests := []struct {
		name          string
		setupMock     func(*MockinterestsRepo) []*models.Interest
		expectedError bool
	}{
		{
			name: "Success",
			setupMock: func(mockRepo *MockinterestsRepo) []*models.Interest {
				expectedInterests := []*models.Interest{
					{
						ID:        uuid.New(),
						Name:      "Interest 1",
						Active:    &active1,
						CreatedAt: now,
						Source:    "test",
					},
					{
						ID:        uuid.New(),
						Name:      "Interest 2",
						Active:    &active2,
						CreatedAt: now,
						Source:    "test",
					},
				}
				mockRepo.On("GetAllInterests", mock.Anything).Return(expectedInterests, nil).Once()
				return expectedInterests
			},
			expectedError: false,
		},
		{
			name: "Repository Error",
			setupMock: func(mockRepo *MockinterestsRepo) []*models.Interest {
				mockRepo.On("GetAllInterests", mock.Anything).Return(nil, errors.New("repository error")).Once()
				return nil
			},
			expectedError: true,
		},
	}

	// Run tests
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup - create fresh mock and service for each test
			mockRepo := new(MockinterestsRepo)
			logger := slog.Default()
			service := NewinterestsService(mockRepo, logger)

			// Setup mock
			expectedInterests := tc.setupMock(mockRepo)

			// Call the method
			interests, err := service.GetAllInterests(ctx)

			// Assertions
			if tc.expectedError {
				assert.Error(t, err)
				assert.Nil(t, interests)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, expectedInterests, interests)
			}

			// Verify mock
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUpdateinterests(t *testing.T) {
	// Test data
	userID := uuid.New()
	interestID := uuid.New()
	name := "Updated Interest"
	description := "Updated Description"
	active := true
	params := models.UpdateinterestsParams{
		Name:        &name,
		Description: &description,
		Active:      &active,
	}
	ctx := context.Background()

	// Test cases
	tests := []struct {
		name          string
		setupMock     func(*MockinterestsRepo)
		expectedError bool
	}{
		{
			name: "Success",
			setupMock: func(mockRepo *MockinterestsRepo) {
				mockRepo.On("Updateinterests", mock.Anything, userID, interestID, params).Return(nil).Once()
			},
			expectedError: false,
		},
		{
			name: "Repository Error",
			setupMock: func(mockRepo *MockinterestsRepo) {
				mockRepo.On("Updateinterests", mock.Anything, userID, interestID, params).Return(errors.New("repository error")).Once()
			},
			expectedError: true,
		},
	}

	// Run tests
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup - create fresh mock and service for each test
			mockRepo := new(MockinterestsRepo)
			logger := slog.Default()
			service := NewinterestsService(mockRepo, logger)

			// Setup mock
			tc.setupMock(mockRepo)

			// Call the method
			err := service.Updateinterests(ctx, userID, interestID, params)

			// Assertions
			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify mock
			mockRepo.AssertExpectations(t)
		})
	}
}
