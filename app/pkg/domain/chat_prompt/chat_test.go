package llmchat

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/FACorreiaa/go-templui/app/lib/models"
)

type mockAIClient struct{}

type mockCityRepo struct{}

func (m *mockCityRepo) FindCityByNameAndCountry(_ context.Context, name, country string) (*models.CityDetail, error) {
	return &models.CityDetail{ID: uuid.New(), Name: name, Country: country}, nil
}

type mockPOIRepo struct {
	pois map[string]*models.POIDetailedInfo
}

func (m *mockPOIRepo) FindPOIDetailedInfos(_ context.Context, cityID uuid.UUID, lat, lon float64, tolerance float64) (*models.POIDetailedInfo, error) {
	key := fmt.Sprintf("%s:%.6f:%.6f", cityID.String(), lat, lon)
	if poi, exists := m.pois[key]; exists {
		return poi, nil
	}
	return nil, nil
}

func (m *mockPOIRepo) SavePOIDetailedInfos(_ context.Context, poi models.POIDetailedInfo, cityID uuid.UUID) (uuid.UUID, error) {
	key := fmt.Sprintf("%s:%.6f:%.6f", cityID.String(), poi.Latitude, poi.Longitude)
	m.pois[key] = &poi
	return uuid.New(), nil
}
