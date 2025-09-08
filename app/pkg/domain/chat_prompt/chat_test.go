package llmchat

import (
	"context"
	"fmt"
	"testing"

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

func (m *mockPOIRepo) FindPOIDetailedInfos(_ context.Context, cityID uuid.UUID, lat, lon float64, _ float64) (*models.POIDetailedInfo, error) {
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

func TestMocksCompile(t *testing.T) {
	ctx := context.Background()
	m := &mockPOIRepo{pois: map[string]*models.POIDetailedInfo{}}
	cityRepo := &mockCityRepo{}
	if _, err := cityRepo.FindCityByNameAndCountry(ctx, "TestCity", "TC"); err != nil {
		t.Fatalf("FindCityByNameAndCountry error: %v", err)
	}
	if _, err := m.SavePOIDetailedInfos(ctx, models.POIDetailedInfo{Latitude: 1.23, Longitude: 4.56}, uuid.New()); err != nil {
		t.Fatalf("SavePOIDetailedInfos error: %v", err)
	}
	if _, err := m.FindPOIDetailedInfos(ctx, uuid.New(), 1.23, 4.56, 0.1); err != nil {
		t.Fatalf("FindPOIDetailedInfos error: %v", err)
	}
	_ = mockAIClient{}
}
