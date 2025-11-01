package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig holds JWT authentication configuration
type JWTConfig struct {
	SecretKey       string
	TokenExpiration time.Duration
	Logger          *slog.Logger
	Optional        bool // If true, missing/invalid tokens won't block the request
}

// Claims represents the JWT claims
type Claims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email,omitempty"`
	Username string `json:"username,omitempty"`
	jwt.RegisteredClaims
}

// JWTAuthMiddleware creates a middleware for JWT authentication
func JWTAuthMiddleware(config JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// Check query parameter (useful for WebSocket connections)
			authHeader = c.Query("token")
			if authHeader != "" {
				authHeader = "Bearer " + authHeader
			}
		}

		if authHeader == "" {
			if config.Optional {
				// Set anonymous user
				c.Set("user_id", "anonymous")
				c.Set("authenticated", false)
				c.Next()
				return
			}

			config.Logger.Warn("Missing authorization header", slog.String("path", c.Request.URL.Path))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>" format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			if config.Optional {
				c.Set("user_id", "anonymous")
				c.Set("authenticated", false)
				c.Next()
				return
			}

			config.Logger.Warn("Invalid authorization header format", slog.String("path", c.Request.URL.Path))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(config.SecretKey), nil
		})

		if err != nil || !token.Valid {
			if config.Optional {
				config.Logger.Debug("Invalid token, using anonymous",
					slog.String("path", c.Request.URL.Path),
					slog.Any("error", err))
				c.Set("user_id", "anonymous")
				c.Set("authenticated", false)
				c.Next()
				return
			}

			config.Logger.Warn("Invalid token",
				slog.String("path", c.Request.URL.Path),
				slog.Any("error", err))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Check expiration
		if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
			if config.Optional {
				c.Set("user_id", "anonymous")
				c.Set("authenticated", false)
				c.Next()
				return
			}

			config.Logger.Warn("Expired token",
				slog.String("path", c.Request.URL.Path),
				slog.String("user_id", claims.UserID))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Token has expired",
			})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("username", claims.Username)
		c.Set("authenticated", true)

		config.Logger.Debug("Authenticated request",
			slog.String("user_id", claims.UserID),
			slog.String("path", c.Request.URL.Path))

		c.Next()
	}
}

// GenerateToken generates a new JWT token
func GenerateToken(config JWTConfig, userID, email, username string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Email:    email,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(config.TokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.SecretKey))
	if err != nil {
		config.Logger.Error("Failed to sign token", slog.Any("error", err))
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return tokenString, nil
}

// RequireAuthMiddleware ensures the user is authenticated (not anonymous)
func RequireAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authenticated, exists := c.Get("authenticated")
		if !exists || !authenticated.(bool) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
