# WebSocket Rate Limiting and JWT Authentication

This package provides middleware for protecting WebSocket endpoints with rate limiting and optional JWT authentication.

## Features

### Rate Limiting
- **Connection Rate Limiting**: Limits the number of WebSocket connections per client
- **Message Rate Limiting**: Limits the number of messages a client can send over WebSocket
- **IP-based and User-based**: Works with both anonymous (IP-based) and authenticated users
- **Automatic Cleanup**: Removes stale client entries periodically

### JWT Authentication
- **Optional or Required**: Can be configured to allow anonymous access or require authentication
- **Token from Header or Query**: Supports Authorization header or query parameter (useful for WebSockets)
- **User Context**: Automatically sets user information in Gin context

## Usage

### 1. Basic Rate Limiting (No Auth)

```go
import (
    "log/slog"
    "time"
    "github.com/FACorreiaa/go-templui/app/pkg/middleware"
)

// Create rate limiter
logger := slog.Default()
rateLimiter := middleware.NewRateLimiter(
    logger,
    10,              // max 10 connections
    1*time.Minute,   // per minute
)

// Apply to WebSocket route
router.GET("/ws/nearby",
    middleware.WebSocketRateLimitMiddleware(rateLimiter),
    handler.HandleWebSocket,
)
```

### 2. Rate Limiting with Optional JWT Auth

This allows both authenticated and anonymous users, with rate limiting applied to both:

```go
// Create JWT config
jwtConfig := middleware.JWTConfig{
    SecretKey:       "your-secret-key-here", // Use env variable in production
    TokenExpiration: 24 * time.Hour,
    Logger:          logger,
    Optional:        true, // Allow anonymous access
}

// Create rate limiter
rateLimiter := middleware.NewRateLimiter(logger, 10, 1*time.Minute)

// Apply both middlewares
router.GET("/ws/nearby",
    middleware.JWTAuthMiddleware(jwtConfig),      // Optional auth
    middleware.WebSocketRateLimitMiddleware(rateLimiter), // Rate limit
    handler.HandleWebSocket,
)
```

With this configuration:
- Users with valid JWT tokens get their rate limit tracked by user ID
- Anonymous users get rate limited by IP address
- Both can access the endpoint

### 3. Rate Limiting with Required JWT Auth

To require authentication:

```go
jwtConfig := middleware.JWTConfig{
    SecretKey:       "your-secret-key-here",
    TokenExpiration: 24 * time.Hour,
    Logger:          logger,
    Optional:        false, // Require authentication
}

router.GET("/ws/nearby",
    middleware.JWTAuthMiddleware(jwtConfig),
    middleware.RequireAuthMiddleware(), // Extra check for authenticated users only
    middleware.WebSocketRateLimitMiddleware(rateLimiter),
    handler.HandleWebSocket,
)
```

### 4. WebSocket Client Example

#### JavaScript client with JWT token:

```javascript
// Token in query parameter (recommended for WebSocket)
const token = "your-jwt-token";
const ws = new WebSocket(`ws://localhost:8080/ws/nearby?token=${token}`);

// Or with Authorization header (not all browsers support this)
const ws = new WebSocket('ws://localhost:8080/ws/nearby');
// Note: WebSocket API doesn't support setting custom headers in browser
// Use query parameter instead
```

#### JavaScript client without authentication:

```javascript
const ws = new WebSocket('ws://localhost:8080/ws/nearby');
```

### 5. Generating JWT Tokens

```go
import "github.com/FACorreiaa/go-templui/app/pkg/middleware"

// Generate token for user
token, err := middleware.GenerateToken(
    jwtConfig,
    "user-123",           // User ID
    "user@example.com",   // Email
    "username",           // Username
)
if err != nil {
    // Handle error
}

// Return token to client
c.JSON(http.StatusOK, gin.H{
    "token": token,
})
```

## Configuration Options

### Rate Limiter

```go
type RateLimiter struct {
    maxRequests     int           // Maximum requests allowed
    window          time.Duration // Time window for rate limiting
    cleanupInterval time.Duration // How often to clean up old entries
}
```

**Recommended values:**
- **WebSocket connections**: 10 connections per minute
- **WebSocket messages**: 30 messages per minute

### JWT Config

```go
type JWTConfig struct {
    SecretKey       string        // Secret key for signing tokens
    TokenExpiration time.Duration // How long tokens are valid
    Logger          *slog.Logger  // Logger for auth events
    Optional        bool          // Allow unauthenticated access
}
```

**Recommended values:**
- **SecretKey**: Use environment variable, minimum 32 characters
- **TokenExpiration**: 24 hours for regular users, 1 hour for sensitive operations
- **Optional**: `true` for public endpoints, `false` for protected resources

## Rate Limit Response

When rate limit is exceeded, the middleware returns:

```json
{
    "error": "Rate limit exceeded. Please try again later."
}
```

HTTP Status: `429 Too Many Requests`

## Authentication Response

When authentication fails (if `Optional: false`):

```json
{
    "error": "Invalid or expired token"
}
```

HTTP Status: `401 Unauthorized`

## Security Best Practices

1. **Use Environment Variables**: Never hardcode JWT secret keys
   ```go
   SecretKey: os.Getenv("JWT_SECRET_KEY")
   ```

2. **Use HTTPS**: Always use TLS/SSL in production for WebSocket connections
   ```
   wss://yourdomain.com/ws/nearby
   ```

3. **Validate Origin**: Update the WebSocket upgrader to validate origins:
   ```go
   var upgrader = websocket.Upgrader{
       CheckOrigin: func(r *http.Request) bool {
           origin := r.Header.Get("Origin")
           return origin == "https://yourdomain.com"
       },
   }
   ```

4. **Rate Limit Tuning**: Adjust rate limits based on your use case
   - Lower limits for expensive operations (AI calls)
   - Higher limits for cheap operations (static data)

5. **Token Rotation**: Implement token refresh mechanism for long-lived sessions

6. **Monitoring**: Log rate limit violations and failed auth attempts

## Example: Complete Setup

```go
package main

import (
    "log/slog"
    "os"
    "time"

    "github.com/FACorreiaa/go-templui/app/pkg/middleware"
    "github.com/gin-gonic/gin"
)

func main() {
    router := gin.Default()
    logger := slog.Default()

    // JWT Config (from environment)
    jwtConfig := middleware.JWTConfig{
        SecretKey:       os.Getenv("JWT_SECRET_KEY"),
        TokenExpiration: 24 * time.Hour,
        Logger:          logger,
        Optional:        true, // Allow both auth and anon users
    }

    // Rate Limiter
    rateLimiter := middleware.NewRateLimiter(
        logger,
        10,            // 10 connections
        1*time.Minute, // per minute
    )

    // Protected WebSocket endpoint
    router.GET("/ws/nearby",
        middleware.JWTAuthMiddleware(jwtConfig),
        middleware.WebSocketRateLimitMiddleware(rateLimiter),
        nearbyHandler.HandleWebSocket,
    )

    // Login endpoint to get tokens
    router.POST("/auth/login", func(c *gin.Context) {
        // Validate credentials (implement your logic)
        userID := "user-123"
        email := "user@example.com"
        username := "john"

        token, err := middleware.GenerateToken(jwtConfig, userID, email, username)
        if err != nil {
            c.JSON(500, gin.H{"error": "Failed to generate token"})
            return
        }

        c.JSON(200, gin.H{"token": token})
    })

    router.Run(":8080")
}
```

## Monitoring

The middleware logs important events:

- **Rate limit exceeded**: `WARN` level
- **Invalid authentication**: `WARN` level
- **Successful authentication**: `DEBUG` level
- **WebSocket errors**: `ERROR` level

Monitor these logs to detect:
- Abuse attempts (high rate limit violations)
- Authentication issues
- DoS attacks

## Performance

- **Memory**: ~200 bytes per active client
- **CPU**: Minimal overhead (~0.1ms per request)
- **Cleanup**: Runs every 2x the rate limit window
- **Thread-safe**: Uses sync.RWMutex for concurrent access
