package city

import (
	"context"
	"fmt"
	"go.uber.org/zap"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/FACorreiaa/go-templui/internal/app/models"
)

type Service interface {
	GetAllCities(ctx context.Context) ([]models.CityDetail, error)
}

type ServiceImpl struct {
	logger *zap.Logger
	repo   Repository
}

func NewCityService(repo Repository, logger *zap.Logger) *ServiceImpl {
	return &ServiceImpl{
		logger: logger,
		repo:   repo,
	}
}

// GetAllCities retrieves all cities from the database
func (s *ServiceImpl) GetAllCities(ctx context.Context) ([]models.CityDetail, error) {
	ctx, span := otel.Tracer("CityService").Start(ctx, "GetAllCities")
	defer span.End()

	l := s.logger.With(zap.String("method", "GetAllCities"))

	l.Info( "Retrieving all cities from database")

	cities, err := s.repo.GetAllCities(ctx)
	if err != nil {
		l.Error( "Failed to retrieve cities from repository", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Repository operation failed")
		return nil, fmt.Errorf("failed to retrieve cities: %w", err)
	}

	l.Info( "Successfully retrieved cities", zap.Int("count", len(cities)))
	span.SetAttributes(attribute.Int("cities.count", len(cities)))
	span.SetStatus(codes.Ok, "Cities retrieved successfully")

	return cities, nil
}
