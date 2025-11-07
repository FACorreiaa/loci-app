package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// JWTConfig holds JWT authentication configuration
type JWTConfig struct {
	SecretKey       string
	TokenExpiration time.Duration
	Logger          *zap.Logger
	Optional        bool // If true, missing/invalid tokens won't block the request
}

// Claims represents the JWT claims
type Claims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email,omitempty"`
	Username string `json:"username,omitempty"`
	jwt.RegisteredClaims
}

// NewJWTService creates a new JWT service
func NewJWTService() *JWTService {
	return &JWTService{}
}

type JWTService struct{}

// GenerateToken generates a new JWT token
func (s *JWTService) GenerateToken(config JWTConfig, userID, email, username string) (string, error) {
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
		config.Logger.Error("Failed to sign token", zap.Error(err))
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken parses and validates a JWT token
func (s *JWTService) ValidateToken(config JWTConfig, tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// HashPassword hashes a password using bcrypt
func (s *JWTService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword compares a hashed password with a plaintext password
func (s *JWTService) CheckPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// JWTAuthMiddleware creates a middleware for JWT authentication
// Now checks cookies FIRST (for browser compatibility), then falls back to Authorization header
func JWTAuthMiddleware(config JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// Priority 1: Check cookie (primary method for browser sessions)
		if cookie, err := c.Cookie("auth_token"); err == nil && cookie != "" {
			tokenString = cookie
		}

		// Priority 2: Check Authorization header (for API clients)
		if tokenString == "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && parts[0] == "Bearer" {
					tokenString = parts[1]
				}
			}
		}

		// Priority 3: Check query parameter (for WebSocket connections)
		if tokenString == "" {
			queryToken := c.Query("token")
			if queryToken != "" {
				tokenString = queryToken
			}
		}

		// No token found
		if tokenString == "" {
			if config.Optional {
				c.Set("user_id", "anonymous")
				c.Set("authenticated", false)
				c.Next()
				return
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		// Validate token
		service := NewJWTService()
		claims, err := service.ValidateToken(config, tokenString)
		if err != nil {
			if config.Optional {
				c.Set("user_id", "anonymous")
				c.Set("authenticated", false)
				c.Next()
				return
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Set user context
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("username", claims.Username)
		c.Set("authenticated", true)
		c.Next()
	}
}