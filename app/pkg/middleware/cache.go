package middleware

import (
	"sync"
	"time"

	"github.com/FACorreiaa/go-templui/app/internal/models"
)

var ItineraryCache = NewResultCache(5 * time.Minute)
var CompleteItineraryCache = NewCompleteResultCache(5 * time.Minute)

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
