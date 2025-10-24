# Unified Cache System Documentation

## Overview

The new unified cache system replaces the old domain-specific caches (RestaurantCache, HotelCache, etc.) with a single generic implementation that:

- ✅ **Eliminates code duplication** - One cache implementation for all types
- ✅ **Provides consistent caching behavior** - Same logic everywhere
- ✅ **Includes metrics** - Track hit/miss rates for performance monitoring
- ✅ **Supports flexible TTLs** - Different expiration times per cache
- ✅ **Thread-safe** - Uses RWMutex for concurrent access
- ✅ **Auto-cleanup** - Automatically removes expired entries

## Architecture

### Core Components

1. **UnifiedCache[T]** - Generic cache that works with any type
2. **CacheManager** - Holds all application caches in one place
3. **CacheKeyBuilder** - Consistent cache key generation

## Usage Examples

### Basic Cache Operations

```go
import "github.com/FACorreiaa/go-templui/app/pkg/middleware"

// Access caches through the global Cache manager
cache := middleware.Cache

// Set a value
restaurants := []models.RestaurantDetailedInfo{...}
cache.Restaurants.Set("cache_key_123", restaurants)

// Get a value
if data, found := cache.Restaurants.Get("cache_key_123"); found {
    // Use cached data
    fmt.Println("Found", len(data), "restaurants in cache")
} else {
    // Cache miss - fetch from database or LLM
}

// Delete a value
cache.Restaurants.Delete("cache_key_123")

// Clear entire cache
cache.Restaurants.Clear()
```

### Building Cache Keys

The `CacheKeyBuilder` ensures consistent key generation:

```go
// Simple key generation
key := middleware.NewCacheKeyBuilder().
    AddCity("Lisbon").
    AddDomain("restaurants").
    BuildOrDefault()

// Key with user context
key := middleware.NewCacheKeyBuilder().
    AddCity("Paris").
    AddDomain("hotels").
    AddUserContext(userID.String(), profileID.String()).
    BuildOrDefault()

// Key with preferences
key := middleware.NewCacheKeyBuilder().
    AddCity("Tokyo").
    AddDomain("activities").
    AddPreferences(userPrefs).
    BuildOrDefault()

// Custom key components
key := middleware.NewCacheKeyBuilder().
    Add("city", "Berlin").
    Add("domain", "restaurants").
    Add("cuisine", "italian").
    Add("price_range", "$$").
    Add("rating_min", 4.0).
    BuildOrDefault()
```

### Handler Example (Restaurants)

**Old Way (Confusing):**
```go
// Check restaurants cache
if restaurantsData, found := middleware.RestaurantsCache.Get(cacheKey); found {
    // Also check complete cache for city data
    var cityData models.GeneralCityData
    if completeData, found := middleware.CompleteItineraryCache.Get(cacheKey); found {
        cityData = completeData.GeneralCityData
    } else {
        // Fallback to database
        cityData = h.loadCityDataFromDatabase(sessionID)
    }
    return results.RestaurantsResults(cityData, restaurantsData, ...)
}
```

**New Way (Clean):**
```go
// Build cache key
cacheKey := middleware.NewCacheKeyBuilder().
    AddCity(city).
    AddDomain("restaurants").
    AddPreferences(preferences).
    BuildOrDefault()

// Check restaurants cache
if restaurants, found := middleware.Cache.Restaurants.Get(cacheKey); found {
    // Check city data cache separately
    if cityData, found := middleware.Cache.CityData.Get(cacheKey); found {
        return results.RestaurantsResults(cityData, restaurants, ...)
    }
}

// Cache miss - fetch from source
restaurants, cityData := h.fetchRestaurantsFromLLM(city, preferences)

// Store both in respective caches
middleware.Cache.Restaurants.Set(cacheKey, restaurants)
middleware.Cache.CityData.Set(cacheKey, cityData)
```

### Cache Metrics

Monitor cache performance:

```go
// Get metrics for specific cache
metrics := middleware.Cache.Restaurants.GetMetrics()
fmt.Printf("Hits: %d, Misses: %d, Sets: %d\n",
    metrics.Hits, metrics.Misses, metrics.Sets)

// Calculate hit rate
hitRate := float64(metrics.Hits) / float64(metrics.Hits + metrics.Misses) * 100
fmt.Printf("Hit Rate: %.2f%%\n", hitRate)

// Get all metrics
allMetrics := middleware.Cache.GetAllMetrics()
for name, m := range allMetrics {
    fmt.Printf("%s: Hits=%d Misses=%d\n", name, m.Hits, m.Misses)
}

// Get cache size
size := middleware.Cache.Restaurants.Size()
fmt.Printf("Cache contains %d items\n", size)
```

## Use Cases

### Use Case 1: Same City, Same Preferences = Cache Hit

```go
// User 1 searches: Lisbon + preferences [vegetarian, budget]
key1 := middleware.NewCacheKeyBuilder().
    AddCity("Lisbon").
    AddPreferences(map[string]interface{}{
        "dietary": "vegetarian",
        "budget": "low",
    }).
    BuildOrDefault()

// First search - cache miss, fetch from LLM
restaurants1 := fetchFromLLM("Lisbon", prefs)
middleware.Cache.Restaurants.Set(key1, restaurants1)

// User 2 (or same user) searches: Lisbon + same preferences
key2 := middleware.NewCacheKeyBuilder().
    AddCity("Lisbon").
    AddPreferences(map[string]interface{}{
        "dietary": "vegetarian",
        "budget": "low",
    }).
    BuildOrDefault()

// key1 == key2 → Cache hit! No LLM call needed
if restaurants, found := middleware.Cache.Restaurants.Get(key2); found {
    return restaurants // Instant response
}
```

### Use Case 2: Same City, Different Preferences = New Search

```go
// Search 1: Lisbon + [vegetarian, budget]
key1 := middleware.NewCacheKeyBuilder().
    AddCity("Lisbon").
    AddPreferences(map[string]interface{}{
        "dietary": "vegetarian",
        "budget": "low",
    }).
    BuildOrDefault()

// Search 2: Lisbon + [seafood, luxury]
key2 := middleware.NewCacheKeyBuilder().
    AddCity("Lisbon").
    AddPreferences(map[string]interface{}{
        "dietary": "seafood",
        "budget": "high",
    }).
    BuildOrDefault()

// key1 != key2 → Different cache keys → New LLM search
```

### Use Case 3: Different City = New Search

```go
// Search 1: Lisbon + preferences
key1 := middleware.NewCacheKeyBuilder().
    AddCity("Lisbon").
    AddPreferences(prefs).
    BuildOrDefault()

// Search 2: Paris + same preferences
key2 := middleware.NewCacheKeyBuilder().
    AddCity("Paris").
    AddPreferences(prefs).
    BuildOrDefault()

// key1 != key2 → Different cities → New LLM search
```

## Migration Guide

### Before (Old System)

```go
// Multiple cache types
var RestaurantsCache = NewRestaurantCache(5 * time.Minute)
var HotelsCache = NewHotelCache(5 * time.Minute)
var ActivitiesCache = NewActivityCache(5 * time.Minute)
var ItineraryCache = NewResultCache(5 * time.Minute)
var CompleteItineraryCache = NewCompleteResultCache(5 * time.Minute)

// Each had its own methods
restaurantsData, found := RestaurantsCache.Get(key)
hotelsData, found := HotelsCache.Get(key)
```

### After (New System)

```go
// Single cache manager
var Cache = NewCacheManager()

// Unified interface
restaurantsData, found := Cache.Restaurants.Get(key)
hotelsData, found := Cache.Hotels.Get(key)
activitiesData, found := Cache.Activities.Get(key)

// Backward compatibility maintained
restaurantsData, found := RestaurantsCache.Get(key) // Still works!
```

## Configuration

### Default TTLs

```go
Complete:    10 minutes  // Longest - complete city responses
City Data:   15 minutes  // Longest - city info changes rarely
Restaurants:  5 minutes  // Standard
Hotels:       5 minutes  // Standard
Activities:   5 minutes  // Standard
Itineraries:  5 minutes  // Standard
```

### Custom TTLs

```go
// Create cache manager with custom TTLs
customCache := &middleware.CacheManager{
    Restaurants: middleware.NewUnifiedCache[[]models.RestaurantDetailedInfo](
        30*time.Minute,  // Longer TTL
        "restaurants",
    ),
    // ... other caches
}
```

## Performance Benefits

1. **Reduced LLM Calls** - Identical queries reuse cached results
2. **Lower Latency** - Cache hits return in microseconds vs seconds for LLM
3. **Cost Savings** - Fewer API calls = lower costs
4. **Better UX** - Instant results for repeated searches

## Monitoring

### Log Output

The cache automatically logs operations:

```
DEBUG Cache set cache=restaurants key=abc123 ttl=5m0s
DEBUG Cache hit cache=restaurants key=abc123
DEBUG Cache miss cache=hotels key=xyz789
INFO  Cache cleanup cache=activities expired_items=12 remaining_items=45
```

### Metrics Endpoint (Example)

```go
// Create metrics endpoint
func CacheMetricsHandler(c *gin.Context) {
    metrics := middleware.Cache.GetAllMetrics()
    c.JSON(200, metrics)
}

// Response:
{
  "restaurants": {"hits": 1523, "misses": 234, "sets": 234},
  "hotels": {"hits": 892, "misses": 156, "sets": 156},
  "activities": {"hits": 2341, "misses": 421, "sets": 421},
  ...
}
```

## Best Practices

1. **Always use CacheKeyBuilder** - Ensures consistent key generation
2. **Include all relevant parameters** - City, domain, preferences, etc.
3. **Cache separately** - Don't mix domain-specific data with city data
4. **Monitor metrics** - Track hit rates to optimize TTLs
5. **Clear on updates** - Clear cache when data sources change

## Troubleshooting

### Cache not hitting

```go
// Debug cache keys
key1 := builder1.BuildOrDefault()
key2 := builder2.BuildOrDefault()
fmt.Println("Key 1:", key1)
fmt.Println("Key 2:", key2)
// If different, check what components differ
```

### Memory concerns

```go
// Check cache sizes
for name, cache := range map[string]interface{}{
    "restaurants": middleware.Cache.Restaurants,
    "hotels": middleware.Cache.Hotels,
    // ...
} {
    size := cache.Size()
    if size > 1000 {
        log.Warn("Large cache", "name", name, "size", size)
    }
}

// Clear if needed
middleware.Cache.ClearAll()
```

## Vector/Embedding Cache System

For semantic search optimization using PGVector embeddings, see **[VECTOR_CACHE_SYSTEM.md](VECTOR_CACHE_SYSTEM.md)**.

The vector cache system provides:
- ✅ **Semantic similarity matching** - Reuses cached results for similar queries (95% threshold)
- ✅ **Embedding caching** - Avoids regenerating embeddings for repeated queries
- ✅ **User profile embeddings** - Caches user preference vectors (60min TTL)
- ✅ **Three-tier caching** - VectorSearch, Embeddings, UserProfiles caches

## Future Enhancements

- [ ] Add Redis backend for distributed caching
- [ ] Implement cache warming for popular cities
- [ ] Add cache invalidation strategies
- [ ] Implement tiered caching (L1/L2)
- [ ] Add Prometheus metrics export
- [x] **Vector/Embedding cache for semantic search** ✓ Implemented
