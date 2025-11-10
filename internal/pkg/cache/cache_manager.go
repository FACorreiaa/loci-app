package cache

import (
	"time"

	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/internal/app/models"
)

// CacheManager holds all application caches
type CacheManager struct {
	// Complete responses (includes city data, POIs, itinerary)
	Complete *UnifiedCache[models.AiCityResponse]

	// Domain-specific results
	Restaurants *UnifiedCache[[]models.RestaurantDetailedInfo]
	Hotels      *UnifiedCache[[]models.HotelDetailedInfo]
	Activities  *UnifiedCache[[]models.POIDetailedInfo]
	Itineraries *UnifiedCache[models.AIItineraryResponse]

	// City data cache (can be shared across domains)
	CityData *UnifiedCache[models.GeneralCityData]

	// Vector/Embedding caches for semantic search
	VectorSearch *VectorCache    // Caches vector search results with semantic matching
	Embeddings   *EmbeddingCache // Caches raw embeddings (user profiles, queries)
	UserProfiles *EmbeddingCache // User preference profile embeddings
}

// NewCacheManager creates a new cache manager with default TTLs
func NewCacheManager(logger *zap.Logger) *CacheManager {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &CacheManager{
		// Complete responses have longer TTL (10 minutes)
		Complete: NewUnifiedCache[models.AiCityResponse](10*time.Minute, "complete", logger),

		// Domain-specific caches (5 minutes)
		Restaurants: NewUnifiedCache[[]models.RestaurantDetailedInfo](5*time.Minute, "restaurants", logger),
		Hotels:      NewUnifiedCache[[]models.HotelDetailedInfo](5*time.Minute, "hotels", logger),
		Activities:  NewUnifiedCache[[]models.POIDetailedInfo](5*time.Minute, "activities", logger),
		Itineraries: NewUnifiedCache[models.AIItineraryResponse](5*time.Minute, "itineraries", logger),

		// City data has longer TTL since it changes less frequently (15 minutes)
		CityData: NewUnifiedCache[models.GeneralCityData](15*time.Minute, "city_data", logger),

		// Vector caches for semantic search (longer TTL since embeddings are expensive to compute)
		VectorSearch: NewVectorCache(20*time.Minute, 0.95, "vector_search", logger), // 95% similarity threshold
		Embeddings:   NewEmbeddingCache(30 * time.Minute),                           // Query embeddings cache
		UserProfiles: NewEmbeddingCache(60 * time.Minute),                           // User profile embeddings (longest TTL)
	}
}

// Global cache manager instance
var Cache = NewCacheManager(nil)

// GetAllMetrics returns metrics for all caches
func (cm *CacheManager) GetAllMetrics() map[string]CacheMetrics {
	return map[string]CacheMetrics{
		"complete":      cm.Complete.GetMetrics(),
		"restaurants":   cm.Restaurants.GetMetrics(),
		"hotels":        cm.Hotels.GetMetrics(),
		"activities":    cm.Activities.GetMetrics(),
		"itineraries":   cm.Itineraries.GetMetrics(),
		"city_data":     cm.CityData.GetMetrics(),
		"vector_search": cm.VectorSearch.GetMetrics(),
		"embeddings":    cm.Embeddings.GetMetrics(),
		"user_profiles": cm.UserProfiles.GetMetrics(),
	}
}

// ClearAll clears all caches
func (cm *CacheManager) ClearAll() {
	cm.Complete.Clear()
	cm.Restaurants.Clear()
	cm.Hotels.Clear()
	cm.Activities.Clear()
	cm.Itineraries.Clear()
	cm.CityData.Clear()
	cm.VectorSearch.Clear()
	// Note: We intentionally don't clear Embeddings and UserProfiles
	// as they are expensive to regenerate. Clear them manually if needed.
}

// Backward compatibility - expose old cache interfaces
// These will be deprecated in favor of Cache.Restaurants, Cache.Hotels, etc.
var (
	RestaurantsCache       = Cache.Restaurants
	HotelsCache            = Cache.Hotels
	ActivitiesCache        = Cache.Activities
	ItineraryCache         = Cache.Itineraries
	CompleteItineraryCache = Cache.Complete
)
