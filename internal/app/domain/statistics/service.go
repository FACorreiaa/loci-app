package statistics

import (
	"context"

	"go.uber.org/zap"

	"github.com/google/uuid"

	"github.com/FACorreiaa/go-templui/internal/app/models"
)

var _ Service = (*ServiceImpl)(nil)

type Service interface {
	GetMainPageStatistics(ctx context.Context, userID uuid.UUID) (*models.MainPageStatistics, error)
	GetDetailedPOIStatistics(ctx context.Context, userID uuid.UUID) (*models.DetailedPOIStatistics, error)
	GetLandingPageStatistics(ctx context.Context, userID uuid.UUID) (*models.LandingPageUserStats, error)
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

func (s *ServiceImpl) GetMainPageStatistics(ctx context.Context, userID uuid.UUID) (*models.MainPageStatistics, error) {
	l := s.logger.With(zap.String("method", "GetMainPageStatistics"))
	stats, err := s.repo.GetMainPageStatistics(ctx, userID)
	if err != nil {
		l.Error("Failed to get main page statistics", zap.Error(err))
		return nil, err
	}

	l.Info("Successfully retrieved main page statistics")
	return stats, nil
}

func (s *ServiceImpl) GetDetailedPOIStatistics(ctx context.Context, userID uuid.UUID) (*models.DetailedPOIStatistics, error) {
	l := s.logger.With(zap.String("method", "GetDetailedPOIStatistics"))
	stats, err := s.repo.GetDetailedPOIStatistics(ctx, userID)
	if err != nil {
		l.Error("Failed to get detailed POI statistics", zap.Error(err))
		return nil, err
	}

	l.Info("Successfully retrieved detailed POI statistics")
	return stats, nil
}

func (s *ServiceImpl) GetLandingPageStatistics(ctx context.Context, userID uuid.UUID) (*models.LandingPageUserStats, error) {
	l := s.logger.With(zap.String("method", "GetLandingPageStatistics"))
	stats, err := s.repo.LandingPageStatistics(ctx, userID)
	if err != nil {
		l.Error("Failed to get landing page statistics", zap.Error(err))
		return nil, err
	}

	l.Info("Successfully retrieved landing page statistics")
	return stats, nil
}
