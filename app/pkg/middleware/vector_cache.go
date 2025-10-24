package middleware

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/app/internal/models"
	"github.com/FACorreiaa/go-templui/app/pkg/logger"
)

// VectorCacheEntry stores a vector embedding with its associated data and metadata
type VectorCacheEntry struct {
	QueryText    string                     `json:"query_text"`
	Embedding    []float32                  `json:"embedding"`
	Results      []models.POIDetailedInfo   `json:"results"`
	Expiration   int64                      `json:"expiration"`
	CityID       string                     `json:"city_id,omitempty"`
	SearchParams map[string]interface{}     `json:"search_params,omitempty"`
	CreatedAt    time.Time                  `json:"created_at"`
}

// VectorCache provides semantic caching for vector embeddings and search results
type VectorCache struct {
	mu                  sync.RWMutex
	entries             map[string]*VectorCacheEntry
	ttl                 time.Duration
	similarityThreshold float64 // For approximate matching (default: 0.95)
	name                string
	metrics             CacheMetrics
}

// NewVectorCache creates a new vector cache with configurable similarity threshold
func NewVectorCache(ttl time.Duration, similarityThreshold float64, name string) *VectorCache {
	vc := &VectorCache{
		entries:             make(map[string]*VectorCacheEntry),
		ttl:                 ttl,
		similarityThreshold: similarityThreshold,
		name:                name,
	}
	go vc.cleanup()
	return vc
}

// Set stores a vector embedding and its results
func (vc *VectorCache) Set(key string, entry *VectorCacheEntry) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	entry.Expiration = time.Now().Add(vc.ttl).UnixNano()
	entry.CreatedAt = time.Now()
	vc.entries[key] = entry
	vc.metrics.Sets++

	logger.Log.Debug("Vector cache set",
		zap.String("cache", vc.name),
		zap.String("key", key),
		zap.String("query", entry.QueryText),
		zap.Int("results", len(entry.Results)),
	)
}

// Get retrieves an exact cache hit
func (vc *VectorCache) Get(key string) (*VectorCacheEntry, bool) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	entry, found := vc.entries[key]
	if !found {
		vc.metrics.Misses++
		return nil, false
	}

	// Check expiration
	if time.Now().UnixNano() > entry.Expiration {
		vc.metrics.Misses++
		return nil, false
	}

	vc.metrics.Hits++
	logger.Log.Debug("Vector cache hit",
		zap.String("cache", vc.name),
		zap.String("key", key),
		zap.String("query", entry.QueryText),
	)
	return entry, true
}

// GetSimilar finds cache entries with similar embeddings (semantic cache hit)
// This enables reusing cached results for similar queries even if not exact matches
func (vc *VectorCache) GetSimilar(queryEmbedding []float32, cityID string, searchParams map[string]interface{}) (*VectorCacheEntry, float64, bool) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	var bestMatch *VectorCacheEntry
	var bestSimilarity float64 = 0

	now := time.Now().UnixNano()

	for _, entry := range vc.entries {
		// Skip expired entries
		if now > entry.Expiration {
			continue
		}

		// Filter by city if specified
		if cityID != "" && entry.CityID != cityID {
			continue
		}

		// Check if search params match (if specified)
		if !matchSearchParams(entry.SearchParams, searchParams) {
			continue
		}

		// Calculate cosine similarity
		similarity := cosineSimilarity(queryEmbedding, entry.Embedding)

		// Update best match if similarity exceeds threshold
		if similarity > bestSimilarity && similarity >= vc.similarityThreshold {
			bestSimilarity = similarity
			bestMatch = entry
		}
	}

	if bestMatch != nil {
		vc.metrics.Hits++
		logger.Log.Info("Vector cache semantic hit",
			zap.String("cache", vc.name),
			zap.String("cached_query", bestMatch.QueryText),
			zap.Float64("similarity", bestSimilarity),
			zap.String("city_id", cityID),
		)
		return bestMatch, bestSimilarity, true
	}

	vc.metrics.Misses++
	return nil, 0, false
}

// GetMetrics returns cache performance metrics
func (vc *VectorCache) GetMetrics() CacheMetrics {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return vc.metrics
}

// Size returns the number of entries in the cache
func (vc *VectorCache) Size() int {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return len(vc.entries)
}

// Clear removes all entries
func (vc *VectorCache) Clear() {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.entries = make(map[string]*VectorCacheEntry)
	logger.Log.Info("Vector cache cleared", zap.String("cache", vc.name))
}

// cleanup runs periodically to remove expired entries
func (vc *VectorCache) cleanup() {
	ticker := time.NewTicker(vc.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		vc.mu.Lock()
		now := time.Now().UnixNano()
		expiredCount := 0

		for key, entry := range vc.entries {
			if now > entry.Expiration {
				delete(vc.entries, key)
				expiredCount++
			}
		}

		if expiredCount > 0 {
			logger.Log.Info("Vector cache cleanup",
				zap.String("cache", vc.name),
				zap.Int("expired_entries", expiredCount),
				zap.Int("remaining_entries", len(vc.entries)),
			)
		}
		vc.mu.Unlock()
	}
}

// Helper: Calculate cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// Helper: Check if search params match
func matchSearchParams(cached, query map[string]interface{}) bool {
	// If no query params specified, match any cached entry
	if len(query) == 0 {
		return true
	}

	// If cached has no params but query does, no match
	if len(cached) == 0 && len(query) > 0 {
		return false
	}

	// Check if all query params match cached params
	for key, queryVal := range query {
		cachedVal, exists := cached[key]
		if !exists {
			return false
		}

		// Simple equality check (you may want to make this more sophisticated)
		if fmt.Sprintf("%v", cachedVal) != fmt.Sprintf("%v", queryVal) {
			return false
		}
	}

	return true
}

// BuildVectorCacheKey generates a cache key from query parameters
func BuildVectorCacheKey(queryText string, cityID string, searchParams map[string]interface{}) string {
	data := map[string]interface{}{
		"query":         queryText,
		"city_id":       cityID,
		"search_params": searchParams,
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		logger.Log.Error("Failed to marshal vector cache key", zap.Error(err))
		return ""
	}

	hash := md5.Sum(jsonBytes)
	return hex.EncodeToString(hash[:])
}

// EmbeddingCache stores raw embeddings for reuse (e.g., user profile embeddings)
type EmbeddingCache struct {
	mu      sync.RWMutex
	entries map[string]embeddingEntry
	ttl     time.Duration
	metrics CacheMetrics
}

type embeddingEntry struct {
	embedding  []float32
	expiration int64
	context    string // What this embedding represents
}

// NewEmbeddingCache creates a cache for storing embeddings
func NewEmbeddingCache(ttl time.Duration) *EmbeddingCache {
	ec := &EmbeddingCache{
		entries: make(map[string]embeddingEntry),
		ttl:     ttl,
	}
	go ec.cleanup()
	return ec
}

// Set stores an embedding
func (ec *EmbeddingCache) Set(key string, embedding []float32, context string) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.entries[key] = embeddingEntry{
		embedding:  embedding,
		expiration: time.Now().Add(ec.ttl).UnixNano(),
		context:    context,
	}
	ec.metrics.Sets++
}

// Get retrieves an embedding
func (ec *EmbeddingCache) Get(key string) ([]float32, bool) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	entry, found := ec.entries[key]
	if !found {
		ec.metrics.Misses++
		return nil, false
	}

	if time.Now().UnixNano() > entry.expiration {
		ec.metrics.Misses++
		return nil, false
	}

	ec.metrics.Hits++
	return entry.embedding, true
}

// GetMetrics returns cache metrics
func (ec *EmbeddingCache) GetMetrics() CacheMetrics {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.metrics
}

// cleanup removes expired entries
func (ec *EmbeddingCache) cleanup() {
	ticker := time.NewTicker(ec.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		ec.mu.Lock()
		now := time.Now().UnixNano()
		for key, entry := range ec.entries {
			if now > entry.expiration {
				delete(ec.entries, key)
			}
		}
		ec.mu.Unlock()
	}
}

// BlendEmbeddings combines user profile and query embeddings with configurable weights
func BlendEmbeddings(profileEmbedding, queryEmbedding []float32, profileWeight, queryWeight float64) ([]float32, error) {
	if len(profileEmbedding) != len(queryEmbedding) {
		return nil, fmt.Errorf("embedding dimensions mismatch: profile=%d, query=%d", len(profileEmbedding), len(queryEmbedding))
	}

	// Validate weights
	if profileWeight+queryWeight != 1.0 {
		return nil, fmt.Errorf("weights must sum to 1.0, got: %.2f", profileWeight+queryWeight)
	}

	blended := make([]float32, len(profileEmbedding))
	var normSquared float64

	// Weighted average
	for i := 0; i < len(profileEmbedding); i++ {
		blended[i] = float32(float64(profileEmbedding[i])*profileWeight + float64(queryEmbedding[i])*queryWeight)
		normSquared += float64(blended[i]) * float64(blended[i])
	}

	// Normalize to unit vector (important for cosine similarity)
	norm := math.Sqrt(normSquared)
	if norm > 0 {
		for i := range blended {
			blended[i] /= float32(norm)
		}
	}

	return blended, nil
}
