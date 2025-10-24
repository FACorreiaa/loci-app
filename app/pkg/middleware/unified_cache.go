package middleware

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/app/pkg/logger"
)

// CacheMetrics tracks cache performance
type CacheMetrics struct {
	Hits   int64
	Misses int64
	Sets   int64
}

// UnifiedCache is a generic cache that works with any type
type UnifiedCache[T any] struct {
	mu      sync.RWMutex
	items   map[string]cacheEntry[T]
	ttl     time.Duration
	name    string // For logging/debugging
	metrics CacheMetrics
}

type cacheEntry[T any] struct {
	value      T
	expiration int64
}

// NewUnifiedCache creates a new generic cache with specified TTL and name
func NewUnifiedCache[T any](ttl time.Duration, name string) *UnifiedCache[T] {
	c := &UnifiedCache[T]{
		items: make(map[string]cacheEntry[T]),
		ttl:   ttl,
		name:  name,
	}
	go c.cleanup()
	return c
}

// Set stores an item in the cache with the given key
func (c *UnifiedCache[T]) Set(key string, value T) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = cacheEntry[T]{
		value:      value,
		expiration: time.Now().Add(c.ttl).UnixNano(),
	}
	c.metrics.Sets++

	logger.Log.Debug("Cache set",
		zap.String("cache", c.name),
		zap.String("key", key),
		zap.Duration("ttl", c.ttl),
	)
}

// Get retrieves an item from the cache
func (c *UnifiedCache[T]) Get(key string) (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		c.metrics.Misses++
		var zero T
		logger.Log.Debug("Cache miss",
			zap.String("cache", c.name),
			zap.String("key", key),
		)
		return zero, false
	}

	// Check if expired
	if time.Now().UnixNano() > item.expiration {
		c.metrics.Misses++
		var zero T
		logger.Log.Debug("Cache expired",
			zap.String("cache", c.name),
			zap.String("key", key),
		)
		return zero, false
	}

	c.metrics.Hits++
	logger.Log.Debug("Cache hit",
		zap.String("cache", c.name),
		zap.String("key", key),
	)
	return item.value, true
}

// Delete removes an item from the cache
func (c *UnifiedCache[T]) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
	logger.Log.Debug("Cache delete",
		zap.String("cache", c.name),
		zap.String("key", key),
	)
}

// Clear removes all items from the cache
func (c *UnifiedCache[T]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]cacheEntry[T])
	logger.Log.Info("Cache cleared",
		zap.String("cache", c.name),
	)
}

// GetMetrics returns current cache metrics
func (c *UnifiedCache[T]) GetMetrics() CacheMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.metrics
}

// Size returns the number of items in the cache
func (c *UnifiedCache[T]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// cleanup runs periodically to remove expired items
func (c *UnifiedCache[T]) cleanup() {
	ticker := time.NewTicker(c.ttl / 2) // Run cleanup twice per TTL period
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now().UnixNano()
		expiredCount := 0

		for key, item := range c.items {
			if now > item.expiration {
				delete(c.items, key)
				expiredCount++
			}
		}

		if expiredCount > 0 {
			logger.Log.Info("Cache cleanup",
				zap.String("cache", c.name),
				zap.Int("expired_items", expiredCount),
				zap.Int("remaining_items", len(c.items)),
			)
		}
		c.mu.Unlock()
	}
}

// CacheKeyBuilder helps build consistent cache keys
type CacheKeyBuilder struct {
	components []interface{}
}

// NewCacheKeyBuilder creates a new cache key builder
func NewCacheKeyBuilder() *CacheKeyBuilder {
	return &CacheKeyBuilder{
		components: make([]interface{}, 0, 8),
	}
}

// Add adds a component to the cache key
func (b *CacheKeyBuilder) Add(key string, value interface{}) *CacheKeyBuilder {
	b.components = append(b.components, map[string]interface{}{key: value})
	return b
}

// AddCity adds city name to the cache key
func (b *CacheKeyBuilder) AddCity(city string) *CacheKeyBuilder {
	return b.Add("city", city)
}

// AddDomain adds domain to the cache key
func (b *CacheKeyBuilder) AddDomain(domain string) *CacheKeyBuilder {
	return b.Add("domain", domain)
}

// AddPreferences adds user preferences to the cache key
func (b *CacheKeyBuilder) AddPreferences(prefs interface{}) *CacheKeyBuilder {
	return b.Add("preferences", prefs)
}

// AddUserContext adds user ID and profile ID to the cache key
func (b *CacheKeyBuilder) AddUserContext(userID, profileID string) *CacheKeyBuilder {
	return b.Add("user_id", userID).Add("profile_id", profileID)
}

// Build generates the final cache key as an MD5 hash
func (b *CacheKeyBuilder) Build() (string, error) {
	// Marshal components to JSON for consistent hashing
	jsonBytes, err := json.Marshal(b.components)
	if err != nil {
		return "", fmt.Errorf("failed to marshal cache key components: %w", err)
	}

	// Generate MD5 hash
	hash := md5.Sum(jsonBytes)
	key := hex.EncodeToString(hash[:])

	logger.Log.Debug("Cache key built",
		zap.String("key", key),
		zap.String("components", string(jsonBytes)),
	)

	return key, nil
}

// BuildOrDefault builds the cache key, returns empty string on error
func (b *CacheKeyBuilder) BuildOrDefault() string {
	key, err := b.Build()
	if err != nil {
		logger.Log.Error("Failed to build cache key", zap.Error(err))
		return ""
	}
	return key
}
