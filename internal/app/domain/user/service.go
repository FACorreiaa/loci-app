package user

import (
	"context"
	"fmt"
	"go.uber.org/zap"

	"github.com/google/uuid"

	"github.com/FACorreiaa/go-templui/internal/app/models"
)

// Ensure implementation satisfies the interface
var _ UserService = (*ServiceUserImpl)(nil)

// UserService defines the business logic contract for user operations.
type UserService interface {
	// GetUserProfile Profile Management
	GetUserProfile(ctx context.Context, userID uuid.UUID) (*models.UserProfile, error)
	UpdateUserProfile(ctx context.Context, userID uuid.UUID, params models.UpdateProfileParams) error

	// UpdateLastLogin Status & Activity
	UpdateLastLogin(ctx context.Context, userID uuid.UUID) error
	MarkEmailAsVerified(ctx context.Context, userID uuid.UUID) error
	DeactivateUser(ctx context.Context, userID uuid.UUID) error
	ReactivateUser(ctx context.Context, userID uuid.UUID) error
}

// ServiceUserImpl provides the implementation for UserService.
type ServiceUserImpl struct {
	logger *zap.Logger
	repo   UserRepo
}

// NewUserService creates a new user service instance.
func NewUserService(repo UserRepo, logger *zap.Logger) *ServiceUserImpl {
	return &ServiceUserImpl{
		logger: logger,
		repo:   repo,
	}
}

// GetUserProfile retrieves a user's profile by ID.
func (s *ServiceUserImpl) GetUserProfile(ctx context.Context, userID uuid.UUID) (*models.UserProfile, error) {
	l := s.logger.With(zap.String("method", "GetUserProfile"), zap.String("userID", userID.String()))
	l.Debug( "Fetching user profile")

	profile, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		l.Error( "Failed to fetch user profile", zap.Any("error", err))
		return nil, fmt.Errorf("error fetching user profile: %w", err)
	}

	l.Info( "User profile fetched successfully")
	return profile, nil
}

// UpdateUserProfile updates a user's profile.
func (s *ServiceUserImpl) UpdateUserProfile(ctx context.Context, userID uuid.UUID, params models.UpdateProfileParams) error {
	l := s.logger.With(zap.String("method", "UpdateUserProfile"), zap.String("userID", userID.String()))
	l.Debug( "Updating user profile")

	err := s.repo.UpdateProfile(ctx, userID, params)
	if err != nil {
		l.Error( "Failed to update user profile", zap.Any("error", err))
		return fmt.Errorf("error updating user profile: %w", err)
	}

	l.Info( "User profile updated successfully")
	return nil
}

// UpdateLastLogin updates the last login timestamp for a user.
func (s *ServiceUserImpl) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	l := s.logger.With(zap.String("method", "UpdateLastLogin"), zap.String("userID", userID.String()))
	l.Debug( "Updating user last login timestamp")

	err := s.repo.UpdateLastLogin(ctx, userID)
	if err != nil {
		l.Error( "Failed to update user last login timestamp", zap.Any("error", err))
		return fmt.Errorf("error updating user last login timestamp: %w", err)
	}

	l.Info( "User last login timestamp updated successfully")
	return nil
}

// MarkEmailAsVerified marks a user's email as verified.
func (s *ServiceUserImpl) MarkEmailAsVerified(ctx context.Context, userID uuid.UUID) error {
	l := s.logger.With(zap.String("method", "MarkEmailAsVerified"), zap.String("userID", userID.String()))
	l.Debug( "Marking user email as verified")

	err := s.repo.MarkEmailAsVerified(ctx, userID)
	if err != nil {
		l.Error( "Failed to mark user email as verified", zap.Any("error", err))
		return fmt.Errorf("error marking user email as verified: %w", err)
	}

	l.Info( "User email marked as verified successfully")
	return nil
}

// DeactivateUser deactivates a user.
func (s *ServiceUserImpl) DeactivateUser(ctx context.Context, userID uuid.UUID) error {
	l := s.logger.With(zap.String("method", "DeactivateUser"), zap.String("userID", userID.String()))
	l.Debug( "Deactivating user")

	err := s.repo.DeactivateUser(ctx, userID)
	if err != nil {
		l.Error( "Failed to deactivate user", zap.Any("error", err))
		return fmt.Errorf("error deactivating user: %w", err)
	}

	l.Info( "User deactivated successfully")
	return nil
}

// ReactivateUser reactivates a user.
func (s *ServiceUserImpl) ReactivateUser(ctx context.Context, userID uuid.UUID) error {
	l := s.logger.With(zap.String("method", "ReactivateUser"), zap.String("userID", userID.String()))
	l.Debug( "Reactivating user")

	err := s.repo.ReactivateUser(ctx, userID)
	if err != nil {
		l.Error( "Failed to reactivate user", zap.Any("error", err))
		return fmt.Errorf("error reactivating user: %w", err)
	}

	l.Info( "User reactivated successfully")
	return nil
}
