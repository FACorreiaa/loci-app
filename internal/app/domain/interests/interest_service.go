package interests

import (
	"context"
	"fmt"
	"go.uber.org/zap"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/FACorreiaa/go-templui/internal/app/models"
)

// Ensure implementation satisfies the interface
var _ interestsService = (*interestsServiceImpl)(nil)

// interestsService defines the business logic contract for user operations.
type interestsService interface {
	//Removeinterests remove interests
	Removeinterests(ctx context.Context, userID uuid.UUID, interestID uuid.UUID) error
	GetAllInterests(ctx context.Context) ([]*models.Interest, error)
	CreateInterest(ctx context.Context, name string, description *string, isActive bool, userID string) (*models.Interest, error)
	Updateinterests(ctx context.Context, userID uuid.UUID, interestID uuid.UUID, params models.UpdateinterestsParams) error
}

// interestsServiceImpl provides the implementation for interestsService.
type interestsServiceImpl struct {
	logger *zap.Logger
	repo   Repository
}

// NewinterestsService creates a new user service instance.
func NewinterestsService(repo Repository, logger *zap.Logger) *interestsServiceImpl {
	return &interestsServiceImpl{
		logger: logger,
		repo:   repo,
	}
}

// CreateInterest create user interest
func (s *interestsServiceImpl) CreateInterest(ctx context.Context, name string, description *string, isActive bool, userID string) (*models.Interest, error) {
	ctx, span := otel.Tracer("interestsService").Start(ctx, "Createinterests", trace.WithAttributes(
		attribute.String("name", name),
		attribute.String("description", *description),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "Createinterests"),
		zap.String("name", name), zap.String("description", *description))
	l.Debug( "Adding user interest")

	interest, err := s.repo.CreateInterest(ctx, name, description, isActive, userID)
	if err != nil {
		l.Error( "Failed to add user interest", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to add user interest")
		return nil, fmt.Errorf("error adding user interest: %w", err)
	}

	l.Info( "User interest created successfully")
	span.SetStatus(codes.Ok, "User interest created successfully")
	return interest, nil
}

// Removeinterests removes an interest from a user's preferences.
func (s *interestsServiceImpl) Removeinterests(ctx context.Context, userID uuid.UUID, interestID uuid.UUID) error {
	ctx, span := otel.Tracer("interestsService").Start(ctx, "Removeinterests", trace.WithAttributes(
		attribute.String("user.id", userID.String()),
		attribute.String("interest.id", interestID.String()),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "Removeinterests"), zap.String("userID", userID.String()), zap.String("interestID", interestID.String()))
	l.Debug( "Removing user interest")

	err := s.repo.Removeinterests(ctx, userID, interestID)
	if err != nil {
		l.Error( "Failed to remove user interest", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to remove user interest")
		return fmt.Errorf("error removing user interest: %w", err)
	}

	l.Info( "User interest removed successfully")
	span.SetStatus(codes.Ok, "User interest removed successfully")
	return nil
}

// GetAllInterests retrieves all available interests.
func (s *interestsServiceImpl) GetAllInterests(ctx context.Context) ([]*models.Interest, error) {
	ctx, span := otel.Tracer("interestsService").Start(ctx, "GetAllInterests")
	defer span.End()

	l := s.logger.With(zap.String("method", "GetAllInterests"))
	l.Debug( "Fetching all interests")

	interests, err := s.repo.GetAllInterests(ctx)
	if err != nil {
		l.Error( "Failed to fetch all interests", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to fetch all interests")
		return nil, fmt.Errorf("error fetching all interests: %w", err)
	}

	l.Info( "All interests fetched successfully", zap.Int("count", len(interests)))
	span.SetStatus(codes.Ok, "All interests fetched successfully")
	return interests, nil
}

func (s *interestsServiceImpl) Updateinterests(ctx context.Context, userID uuid.UUID, interestID uuid.UUID, params models.UpdateinterestsParams) error {
	ctx, span := otel.Tracer("interestsService").Start(ctx, "Updateinterests", trace.WithAttributes(
		attribute.String("user.id", userID.String()),
		attribute.String("interest.id", interestID.String()),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "Updateinterests"), zap.String("userID", userID.String()), zap.String("interestID", interestID.String()))
	l.Debug( "Updating user interest")

	err := s.repo.Updateinterests(ctx, userID, interestID, params)
	if err != nil {
		l.Error( "Failed to update user interest", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update user interest")
		return fmt.Errorf("error updating user interest: %w", err)
	}
	return nil
}

// GetUserEnhancedInterests retrieves a user's enhanced interests.
//func (s *interestsServiceImpl) GetUserEnhancedInterests(ctx context.Context, userID uuid.UUID) ([]models.EnhancedInterest, error) {
//	ctx, span := otel.Tracer("interestsService").Start(ctx, "GetUserEnhancedInterests", trace.WithAttributes(
//		attribute.String("user.id", userID.String()),
//	))
//	defer span.End()
//
//	l := s.logger.With(zap.String("method", "GetUserEnhancedInterests"), zap.String("userID", userID.String()))
//	l.Debug( "Fetching user enhanced interests")
//
//	interests, err := s.repo.GetUserEnhancedInterests(ctx, userID)
//	if err != nil {
//		l.Error( "Failed to fetch user enhanced interests", zap.Any("error", err))
//		span.RecordError(err)
//		span.SetStatus(codes.Error, "Failed to fetch user enhanced interests")
//		return nil, fmt.Errorf("error fetching user enhanced interests: %w", err)
//	}
//
//	l.Info( "User enhanced interests fetched successfully", zap.Int("count", len(interests)))
//	span.SetStatus(codes.Ok, "User enhanced interests fetched successfully")
//	return interests, nil
//}
