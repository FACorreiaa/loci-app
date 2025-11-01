# WebSocket Security Implementation Guide

This document explains how to add rate limiting and JWT authentication to the `/nearby` WebSocket endpoint.

## Overview

The `/nearby` WebSocket endpoint now supports:
- ✅ **Rate Limiting**: Prevents abuse by limiting connections and messages per user/IP
- ✅ **Optional JWT Authentication**: Can require authentication or allow anonymous access
- ✅ **Message-level Rate Limiting**: Limits location update messages (30 per minute by default)
- ✅ **User Tracking**: Tracks authenticated users by ID, anonymous users by IP

## Quick Start

### Option 1: Rate Limiting Only (No Authentication Required)

This is the simplest setup - users can access without logging in, but are rate-limited by IP address:

```go
// In your routes setup (e.g., app/routes/routes.go)
import (
    "log/slog"
    "time"
    "github.com/FACorreiaa/go-templui/app/pkg/middleware"
)

func SetupRoutes(router *gin.Engine, handlers *handlers.Handlers, logger *slog.Logger) {
    // Create rate limiter
    rateLimiter := middleware.NewRateLimiter(
        logger,
        10,              // Max 10 WebSocket connections
        1*time.Minute,   // Per minute
    )

    // WebSocket endpoint with rate limiting only
    router.GET("/ws/nearby",
        middleware.WebSocketRateLimitMiddleware(rateLimiter),
        handlers.Nearby.HandleWebSocket,
    )
}
```

**Benefits:**
- Simple to implement
- No authentication infrastructure needed
- Prevents basic abuse/DoS
- Good for public-facing features

**Limitations:**
- Users can bypass IP-based limits using VPNs/proxies
- Can't track individual users across sessions
- Less control over who accesses the endpoint

### Option 2: Optional JWT Authentication + Rate Limiting (Recommended)

This allows both authenticated and anonymous users, but tracks authenticated users better:

```go
func SetupRoutes(router *gin.Engine, handlers *handlers.Handlers, logger *slog.Logger) {
    // JWT Configuration
    jwtConfig := middleware.JWTConfig{
        SecretKey:       os.Getenv("JWT_SECRET_KEY"), // Set this in your environment
        TokenExpiration: 24 * time.Hour,
        Logger:          logger,
        Optional:        true, // Allow anonymous users
    }

    // Rate Limiter
    rateLimiter := middleware.NewRateLimiter(logger, 10, 1*time.Minute)

    // WebSocket endpoint with optional auth + rate limiting
    router.GET("/ws/nearby",
        middleware.JWTAuthMiddleware(jwtConfig),
        middleware.WebSocketRateLimitMiddleware(rateLimiter),
        handlers.Nearby.HandleWebSocket,
    )

    // Login endpoint (optional - implement if you have user accounts)
    router.POST("/auth/login", handlers.Auth.Login)
}
```

**Benefits:**
- Authenticated users get better rate limit tracking
- Anonymous users still have access
- Can provide enhanced features to authenticated users
- Better abuse prevention (can ban user accounts, not just IPs)

**Limitations:**
- Requires user account system
- More complex setup

### Option 3: Required JWT Authentication + Rate Limiting (Most Secure)

This requires all users to authenticate before accessing the WebSocket:

```go
func SetupRoutes(router *gin.Engine, handlers *handlers.Handlers, logger *slog.Logger) {
    // JWT Configuration
    jwtConfig := middleware.JWTConfig{
        SecretKey:       os.Getenv("JWT_SECRET_KEY"),
        TokenExpiration: 24 * time.Hour,
        Logger:          logger,
        Optional:        false, // Require authentication
    }

    // Rate Limiter
    rateLimiter := middleware.NewRateLimiter(logger, 10, 1*time.Minute)

    // WebSocket endpoint - authentication required
    router.GET("/ws/nearby",
        middleware.JWTAuthMiddleware(jwtConfig),
        middleware.RequireAuthMiddleware(), // Extra check
        middleware.WebSocketRateLimitMiddleware(rateLimiter),
        handlers.Nearby.HandleWebSocket,
    )

    // Must implement login endpoint
    router.POST("/auth/login", handlers.Auth.Login)
}
```

**Benefits:**
- Maximum security
- Know exactly who is using your service
- Can track user behavior accurately
- Can ban abusive users by account

**Limitations:**
- Requires user registration/login
- May reduce adoption (signup friction)
- Need to implement full auth system

## Client-Side Implementation

### Without Authentication

```javascript
// Simple connection
const ws = new WebSocket('ws://localhost:8080/ws/nearby');

ws.onopen = () => {
    console.log('Connected');

    // Send location update
    ws.send(JSON.stringify({
        latitude: 38.7223,
        longitude: -9.1393,
        radius: 5.0
    }));
};

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);

    if (data.type === 'pois') {
        // Display POIs
        console.log('Received POIs:', data.pois);
    } else if (data.type === 'error') {
        // Handle error (e.g., rate limit exceeded)
        console.error('Error:', data.message);
    }
};
```

### With JWT Authentication

```javascript
// Get token from login or localStorage
const token = localStorage.getItem('jwt_token');

// Connect with token in query parameter
const ws = new WebSocket(`ws://localhost:8080/ws/nearby?token=${token}`);

// Or use Authorization header (not supported in all browsers for WebSocket)
// Note: In browser, you must use query parameter

ws.onopen = () => {
    console.log('Connected as authenticated user');

    ws.send(JSON.stringify({
        latitude: 38.7223,
        longitude: -9.1393,
        radius: 5.0
    }));
};

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);

    if (data.type === 'pois') {
        console.log('Received POIs:', data.pois);
    } else if (data.type === 'error') {
        if (data.message.includes('rate limit')) {
            // Show rate limit warning to user
            showNotification('Please slow down - too many requests');
        }
    }
};

ws.onerror = (error) => {
    console.error('WebSocket error:', error);
};

ws.onclose = (event) => {
    if (event.code === 1008) {
        // Policy violation (e.g., authentication failed)
        console.error('Authentication failed');
        // Redirect to login
        window.location.href = '/login';
    }
};
```

## Rate Limit Details

### Connection Rate Limit
- **Default**: 10 connections per minute per user/IP
- **Behavior**: After exceeding limit, connection attempts return HTTP 429
- **Reset**: Sliding window - old connections count drops off after 1 minute

### Message Rate Limit
- **Default**: 30 messages per minute per WebSocket connection
- **Behavior**: After exceeding limit, server sends error message, doesn't close connection
- **Reset**: Sliding window - old messages count drops off after 1 minute

### Customizing Rates

```go
// Adjust connection rate limit
rateLimiter := middleware.NewRateLimiter(
    logger,
    20,              // Allow 20 connections instead of 10
    2*time.Minute,   // Per 2 minutes instead of 1
)

// Adjust message rate limit (in nearby.go handler)
// Edit NewNearbyHandler function:
messageLimiter: &MessageRateLimiter{
    maxMessages: 60,              // 60 messages instead of 30
    window:      2 * time.Minute, // Per 2 minutes
    logger:      logger,
},
```

## Error Handling

### Client receives rate limit error:

```json
{
    "type": "error",
    "message": "Too many requests. Please slow down."
}
```

**What to do:**
1. Show user-friendly message
2. Disable location updates temporarily
3. Re-enable after waiting period (e.g., 10 seconds)

### Client receives authentication error:

HTTP 401 with:
```json
{
    "error": "Invalid or expired token"
}
```

**What to do:**
1. Redirect to login page
2. Clear stored token
3. Ask user to re-authenticate

## Environment Variables

Create a `.env` file:

```bash
JWT_SECRET_KEY=your-super-secret-key-min-32-chars-long-please
```

Load in your app:

```go
import "github.com/joho/godotenv"

func init() {
    godotenv.Load()
}
```

## Testing

### Test Rate Limiting

```bash
# Install wscat for testing
npm install -g wscat

# Connect and spam messages
wscat -c ws://localhost:8080/ws/nearby

# Send many messages quickly
> {"latitude": 38.7223, "longitude": -9.1393, "radius": 5.0}
> {"latitude": 38.7223, "longitude": -9.1393, "radius": 5.0}
# ... (repeat 30+ times)

# You should see rate limit error after 30 messages
< {"type":"error","message":"Too many requests. Please slow down."}
```

### Test Authentication

```bash
# Connect without token (should work if Optional: true)
wscat -c ws://localhost:8080/ws/nearby

# Connect with invalid token (should fail if Optional: false)
wscat -c "ws://localhost:8080/ws/nearby?token=invalid"

# Connect with valid token
wscat -c "ws://localhost:8080/ws/nearby?token=eyJhbGc..."
```

## Monitoring

Add logging to track abuse:

```go
// In middleware or handler
logger.Warn("Potential abuse detected",
    slog.String("user_id", userID),
    slog.String("ip", clientIP),
    slog.Int("violations", violationCount),
)
```

## Recommendations

**For Development:**
- Use Option 1 (Rate Limiting Only)
- Higher rate limits for testing

**For Production (Public App):**
- Use Option 2 (Optional JWT Auth + Rate Limiting)
- Implement analytics to track anonymous vs authenticated usage
- Consider requiring auth for heavy users

**For Production (Private/Enterprise App):**
- Use Option 3 (Required JWT Auth + Rate Limiting)
- Implement proper user management
- Lower rate limits (quality over quantity)

## Next Steps

1. Choose your security option (1, 2, or 3)
2. Add middleware to your routes
3. Update frontend to handle rate limit errors
4. If using auth, implement login endpoint
5. Set environment variables (JWT_SECRET_KEY)
6. Test thoroughly
7. Monitor logs for abuse patterns

## Security Checklist

- [ ] Use HTTPS/WSS in production
- [ ] Set JWT_SECRET_KEY from environment (not hardcoded)
- [ ] Implement proper origin validation in WebSocket upgrader
- [ ] Add logging for rate limit violations
- [ ] Consider adding IP-based blocking for repeat offenders
- [ ] Implement token refresh mechanism for long sessions
- [ ] Add rate limiting to login endpoint
- [ ] Use secure, httpOnly cookies if storing tokens client-side
- [ ] Implement CORS properly
- [ ] Add input validation for location data (lat/lon ranges, radius limits)
