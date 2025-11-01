package middleware

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter tracks request rates per client
type RateLimiter struct {
	clients map[string]*ClientLimit
	mu      sync.RWMutex
	logger  *slog.Logger

	// Configuration
	maxRequests     int           // Maximum requests allowed
	window          time.Duration // Time window for rate limiting
	cleanupInterval time.Duration // How often to clean up old entries
}

// ClientLimit tracks requests for a single client
type ClientLimit struct {
	requests  []time.Time
	mu        sync.Mutex
	lastSeen  time.Time
	websocket bool // Track if this is a WebSocket connection
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(logger *slog.Logger, maxRequests int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients:         make(map[string]*ClientLimit),
		logger:          logger,
		maxRequests:     maxRequests,
		window:          window,
		cleanupInterval: window * 2, // Cleanup twice as slow as the window
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// cleanup removes old client entries periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for clientID, limit := range rl.clients {
			limit.mu.Lock()
			// Remove clients that haven't been seen in 2x the window and don't have active WebSocket
			if !limit.websocket && now.Sub(limit.lastSeen) > rl.window*2 {
				delete(rl.clients, clientID)
			}
			limit.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// getClientID extracts a unique identifier for the client
func getClientID(c *gin.Context) string {
	// Try to get user ID from context (if authenticated)
	if userID, exists := c.Get("user_id"); exists {
		return userID.(string)
	}

	// Fall back to IP address
	return c.ClientIP()
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(c *gin.Context) bool {
	clientID := getClientID(c)

	rl.mu.RLock()
	client, exists := rl.clients[clientID]
	rl.mu.RUnlock()

	if !exists {
		// Create new client entry
		rl.mu.Lock()
		client = &ClientLimit{
			requests: make([]time.Time, 0, rl.maxRequests),
			lastSeen: time.Now(),
		}
		rl.clients[clientID] = client
		rl.mu.Unlock()
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	now := time.Now()
	client.lastSeen = now

	// Remove requests outside the time window
	validRequests := make([]time.Time, 0, len(client.requests))
	cutoff := now.Add(-rl.window)
	for _, reqTime := range client.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	client.requests = validRequests

	// Check if limit is exceeded
	if len(client.requests) >= rl.maxRequests {
		rl.logger.Warn("Rate limit exceeded",
			slog.String("client_id", clientID),
			slog.Int("requests", len(client.requests)),
			slog.Int("max_requests", rl.maxRequests),
			slog.Duration("window", rl.window))
		return false
	}

	// Add current request
	client.requests = append(client.requests, now)
	return true
}

// MarkWebSocket marks a client as having an active WebSocket connection
func (rl *RateLimiter) MarkWebSocket(clientID string, active bool) {
	rl.mu.RLock()
	client, exists := rl.clients[clientID]
	rl.mu.RUnlock()

	if !exists {
		if active {
			rl.mu.Lock()
			rl.clients[clientID] = &ClientLimit{
				requests:  make([]time.Time, 0),
				lastSeen:  time.Now(),
				websocket: true,
			}
			rl.mu.Unlock()
		}
		return
	}

	client.mu.Lock()
	client.websocket = active
	client.mu.Unlock()
}

// WebSocketRateLimitMiddleware returns a Gin middleware for rate limiting WebSocket connections
func WebSocketRateLimitMiddleware(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.Allow(c) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Please try again later.",
			})
			c.Abort()
			return
		}

		// Mark as WebSocket connection
		clientID := getClientID(c)
		rl.MarkWebSocket(clientID, true)

		// Cleanup on connection close
		c.Next()

		// Mark WebSocket as closed
		rl.MarkWebSocket(clientID, false)
	}
}

// MessageRateLimiter tracks message rates for WebSocket connections
type MessageRateLimiter struct {
	maxMessages int           // Maximum messages per window
	window      time.Duration // Time window
	logger      *slog.Logger
}

// NewMessageRateLimiter creates a limiter for WebSocket messages
func NewMessageRateLimiter(logger *slog.Logger, maxMessages int, window time.Duration) *MessageRateLimiter {
	return &MessageRateLimiter{
		maxMessages: maxMessages,
		window:      window,
		logger:      logger,
	}
}

// AllowMessage checks if a WebSocket message should be allowed
func (mrl *MessageRateLimiter) AllowMessage(client *ClientLimit, clientID string) bool {
	client.mu.Lock()
	defer client.mu.Unlock()

	now := time.Now()
	client.lastSeen = now

	// Remove old messages
	validRequests := make([]time.Time, 0, len(client.requests))
	cutoff := now.Add(-mrl.window)
	for _, reqTime := range client.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	client.requests = validRequests

	// Check limit
	if len(client.requests) >= mrl.maxMessages {
		mrl.logger.Warn("Message rate limit exceeded",
			slog.String("client_id", clientID),
			slog.Int("messages", len(client.requests)),
			slog.Int("max_messages", mrl.maxMessages),
			slog.Duration("window", mrl.window))
		return false
	}

	// Add current message
	client.requests = append(client.requests, now)
	return true
}
