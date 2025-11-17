package llmchat

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/internal/app/models"
	cache2 "github.com/FACorreiaa/go-templui/internal/pkg/cache"
)

func newTestService() *ServiceImpl {
	return &ServiceImpl{
		logger:       zap.NewNop(),
		deadLetterCh: make(chan models.StreamEvent, 1),
	}
}

func drainEvents(ch <-chan models.StreamEvent) []models.StreamEvent {
	var events []models.StreamEvent
	for event := range ch {
		events = append(events, event)
	}
	return events
}

func TestBuildCacheKeyUsesPreferencesAndCity(t *testing.T) {
	service := newTestService()

	key1 := service.buildCacheKey("Madrid", models.DomainAccommodation, "Budget Traveler", "hotel request")
	require.NotEmpty(t, key1)

	key2 := service.buildCacheKey("Madrid", models.DomainAccommodation, "Budget Traveler", "different message")
	assert.Equal(t, key1, key2, "same preferences and city should yield identical cache keys")

	key3 := service.buildCacheKey("Madrid", models.DomainAccommodation, "Luxury Traveler", "hotel request")
	assert.NotEqual(t, key1, key3, "changing preferences should produce a different cache key")

	key4 := service.buildCacheKey("Barcelona", models.DomainAccommodation, "Budget Traveler", "hotel request")
	assert.NotEqual(t, key1, key4, "changing city should produce a different cache key")
}

func TestServeFromCacheIfAvailableItinerary(t *testing.T) {
	cache2.Cache.ClearAll()
	service := newTestService()

	cacheKey := service.buildCacheKey("Madrid", models.DomainItinerary, "Family pace", "trip to madrid")
	complete := models.AiCityResponse{
		GeneralCityData: models.GeneralCityData{City: "Madrid"},
		PointsOfInterest: []models.POIDetailedInfo{
			{Name: "Museum Prado", Category: "Museum"},
		},
		AIItineraryResponse: models.AIItineraryResponse{
			ItineraryName: "Madrid Highlights",
			PointsOfInterest: []models.POIDetailedInfo{
				{Name: "Park Retiro", Category: "Park"},
			},
		},
	}
	cache2.CompleteItineraryCache.Set(cacheKey, complete)

	eventCh := make(chan models.StreamEvent, 10)
	served, err := service.serveFromCacheIfAvailable(context.Background(), eventCh, cacheKey, "Madrid", models.DomainItinerary, uuid.New())
	require.NoError(t, err)
	require.True(t, served)

	events := drainEvents(eventCh)
	require.GreaterOrEqual(t, len(events), 3)
	assert.Equal(t, models.EventTypeStart, events[0].Type)
	assert.Equal(t, models.EventTypeComplete, events[len(events)-1].Type)
}

func TestServeFromCacheIfAvailableHotels(t *testing.T) {
	cache2.Cache.ClearAll()
	service := newTestService()

	cacheKey := service.buildCacheKey("Lisbon", models.DomainAccommodation, "Romantic getaway", "lisbon hotels")
	cache2.CompleteItineraryCache.Set(cacheKey, models.AiCityResponse{
		GeneralCityData: models.GeneralCityData{City: "Lisbon"},
	})
	cache2.HotelsCache.Set(cacheKey, []models.HotelDetailedInfo{
		{Name: "Riverside Suites", Latitude: 1.0, Longitude: 1.0},
	})

	eventCh := make(chan models.StreamEvent, 10)
	served, err := service.serveFromCacheIfAvailable(context.Background(), eventCh, cacheKey, "Lisbon", models.DomainAccommodation, uuid.New())
	require.NoError(t, err)
	require.True(t, served)

	events := drainEvents(eventCh)
	require.GreaterOrEqual(t, len(events), 3)

	var chunkParts []string
	for _, ev := range events {
		if ev.Type == models.EventTypeChunk {
			data, ok := ev.Data.(map[string]interface{})
			if ok {
				if part, has := data["part"].(string); has {
					chunkParts = append(chunkParts, part)
				}
			}
		}
	}
	assert.Contains(t, chunkParts, "hotels")
}
