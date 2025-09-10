package middleware

import (
	"sync"
	"time"

	"github.com/FACorreiaa/go-templui/app/internal/models"
)

var ItineraryCache = NewResultCache(5 * time.Minute)
var CompleteItineraryCache = NewCompleteResultCache(5 * time.Minute)
var RestaurantsCache = NewRestaurantCache(5 * time.Minute)
var ActivitiesCache = NewActivityCache(5 * time.Minute)
var HotelsCache = NewHotelCache(5 * time.Minute)

type ResultCache struct {
	mu    sync.Mutex
	items map[string]cacheItem
	ttl   time.Duration
}

type cacheItem struct {
	value      models.AIItineraryResponse // The data you want to store
	expiration int64
}

func NewResultCache(ttl time.Duration) *ResultCache {
	c := &ResultCache{
		items: make(map[string]cacheItem),
		ttl:   ttl,
	}
	go c.cleanup()
	return c
}

// Set stores an item in the cache.
func (c *ResultCache) Set(key string, value models.AIItineraryResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = cacheItem{
		value:      value,
		expiration: time.Now().Add(c.ttl).UnixNano(),
	}
}

// Get retrieves an item without deleting it (persistent cache).
func (c *ResultCache) Get(key string) (models.AIItineraryResponse, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, found := c.items[key]
	if !found || time.Now().UnixNano() > item.expiration {
		return models.AIItineraryResponse{}, false
	}
	// Return the item without deleting it, allowing multiple reads
	return item.value, true
}

// cleanup runs periodically to remove expired items.
func (c *ResultCache) cleanup() {
	for {
		time.Sleep(c.ttl)
		c.mu.Lock()
		for key, item := range c.items {
			if time.Now().UnixNano() > item.expiration {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// CompleteResultCache for storing complete AiCityResponse (with city data, general POIs, and itinerary)
type CompleteResultCache struct {
	mu    sync.Mutex
	items map[string]completeCacheItem
	ttl   time.Duration
}

type completeCacheItem struct {
	value      models.AiCityResponse // Complete response with all three parts
	expiration int64
}

func NewCompleteResultCache(ttl time.Duration) *CompleteResultCache {
	c := &CompleteResultCache{
		items: make(map[string]completeCacheItem),
		ttl:   ttl,
	}
	go c.cleanupComplete()
	return c
}

// Set stores a complete item in the cache.
func (c *CompleteResultCache) Set(key string, value models.AiCityResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = completeCacheItem{
		value:      value,
		expiration: time.Now().Add(c.ttl).UnixNano(),
	}
}

// Get retrieves a complete item without deleting it (persistent cache).
func (c *CompleteResultCache) Get(key string) (models.AiCityResponse, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, found := c.items[key]
	if !found || time.Now().UnixNano() > item.expiration {
		return models.AiCityResponse{}, false
	}
	// Return the item without deleting it, allowing multiple reads
	return item.value, true
}

// cleanupComplete runs periodically to remove expired items.
func (c *CompleteResultCache) cleanupComplete() {
	for {
		time.Sleep(c.ttl)
		c.mu.Lock()
		for key, item := range c.items {
			if time.Now().UnixNano() > item.expiration {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// RestaurantCache for storing restaurant results
type RestaurantCache struct {
	mu    sync.Mutex
	items map[string]restaurantCacheItem
	ttl   time.Duration
}

type restaurantCacheItem struct {
	value      []models.RestaurantDetailedInfo
	expiration int64
}

func NewRestaurantCache(ttl time.Duration) *RestaurantCache {
	c := &RestaurantCache{
		items: make(map[string]restaurantCacheItem),
		ttl:   ttl,
	}
	go c.cleanupRestaurant()
	return c
}

func (c *RestaurantCache) Set(key string, value []models.RestaurantDetailedInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = restaurantCacheItem{
		value:      value,
		expiration: time.Now().Add(c.ttl).UnixNano(),
	}
}

func (c *RestaurantCache) Get(key string) ([]models.RestaurantDetailedInfo, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, found := c.items[key]
	if !found || time.Now().UnixNano() > item.expiration {
		return []models.RestaurantDetailedInfo{}, false
	}
	return item.value, true
}

func (c *RestaurantCache) cleanupRestaurant() {
	for {
		time.Sleep(c.ttl)
		c.mu.Lock()
		for key, item := range c.items {
			if time.Now().UnixNano() > item.expiration {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// ActivityCache for storing activity results
type ActivityCache struct {
	mu    sync.Mutex
	items map[string]activityCacheItem
	ttl   time.Duration
}

type activityCacheItem struct {
	value      []models.POIDetailedInfo
	expiration int64
}

func NewActivityCache(ttl time.Duration) *ActivityCache {
	c := &ActivityCache{
		items: make(map[string]activityCacheItem),
		ttl:   ttl,
	}
	go c.cleanupActivity()
	return c
}

func (c *ActivityCache) Set(key string, value []models.POIDetailedInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = activityCacheItem{
		value:      value,
		expiration: time.Now().Add(c.ttl).UnixNano(),
	}
}

func (c *ActivityCache) Get(key string) ([]models.POIDetailedInfo, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, found := c.items[key]
	if !found || time.Now().UnixNano() > item.expiration {
		return []models.POIDetailedInfo{}, false
	}
	return item.value, true
}

func (c *ActivityCache) cleanupActivity() {
	for {
		time.Sleep(c.ttl)
		c.mu.Lock()
		for key, item := range c.items {
			if time.Now().UnixNano() > item.expiration {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// HotelCache for storing hotel results
type HotelCache struct {
	mu    sync.Mutex
	items map[string]hotelCacheItem
	ttl   time.Duration
}

type hotelCacheItem struct {
	value      []models.HotelDetailedInfo
	expiration int64
}

func NewHotelCache(ttl time.Duration) *HotelCache {
	c := &HotelCache{
		items: make(map[string]hotelCacheItem),
		ttl:   ttl,
	}
	go c.cleanupHotel()
	return c
}

func (c *HotelCache) Set(key string, value []models.HotelDetailedInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = hotelCacheItem{
		value:      value,
		expiration: time.Now().Add(c.ttl).UnixNano(),
	}
}

func (c *HotelCache) Get(key string) ([]models.HotelDetailedInfo, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, found := c.items[key]
	if !found || time.Now().UnixNano() > item.expiration {
		return []models.HotelDetailedInfo{}, false
	}
	return item.value, true
}

func (c *HotelCache) cleanupHotel() {
	for {
		time.Sleep(c.ttl)
		c.mu.Lock()
		for key, item := range c.items {
			if time.Now().UnixNano() > item.expiration {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}
