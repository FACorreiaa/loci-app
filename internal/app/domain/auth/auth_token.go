package auth

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/FACorreiaa/go-templui/internal/pkg/middleware"
)

// AuthTokenHandler handles JWT token generation for testing/development
type AuthTokenHandler struct {
	logger    *slog.Logger
	jwtConfig middleware.JWTConfig
}

// NewAuthTokenHandler creates a new auth token handler
func NewAuthTokenHandler(logger *slog.Logger, jwtConfig middleware.JWTConfig) *AuthTokenHandler {
	return &AuthTokenHandler{
		logger:    logger,
		jwtConfig: jwtConfig,
	}
}

// GenerateTokenRequest represents the request body for token generation
type GenerateTokenRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

// GenerateTokenResponse represents the token response
type GenerateTokenResponse struct {
	Token     string `json:"token"`
	ExpiresIn string `json:"expires_in"`
	UserID    string `json:"user_id"`
}

// GenerateToken generates a JWT token for a user
func (h *AuthTokenHandler) GenerateToken(c *gin.Context) {
	var req GenerateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid token request", slog.Any("error", err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request. user_id is required",
		})
		return
	}

	// Generate token
	token, err := middleware.GenerateToken(
		h.jwtConfig,
		req.UserID,
		req.Email,
		req.Username,
	)
	if err != nil {
		h.logger.Error("Failed to generate token",
			slog.String("user_id", req.UserID),
			slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate token",
		})
		return
	}

	h.logger.Info("Token generated",
		slog.String("user_id", req.UserID),
		slog.String("email", req.Email))

	c.JSON(http.StatusOK, GenerateTokenResponse{
		Token:     token,
		ExpiresIn: h.jwtConfig.TokenExpiration.String(),
		UserID:    req.UserID,
	})
}

// VerifyToken verifies a JWT token and returns user info
func (h *AuthTokenHandler) VerifyToken(c *gin.Context) {
	// User info is already set by JWT middleware
	userID, _ := c.Get("user_id")
	email, _ := c.Get("email")
	username, _ := c.Get("username")
	authenticated, _ := c.Get("authenticated")

	if !authenticated.(bool) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Not authenticated",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":       userID,
		"email":         email,
		"username":      username,
		"authenticated": true,
	})
}

// GetTokenExample returns an example of how to use the token API
func (h *AuthTokenHandler) GetTokenExample(c *gin.Context) {
	example := map[string]interface{}{
		"description": "Generate and use JWT tokens for WebSocket authentication",
		"endpoints": map[string]interface{}{
			"generate_token": map[string]interface{}{
				"method":      "POST",
				"path":        "/api/auth/token",
				"description": "Generate a JWT token for a user",
				"request": map[string]interface{}{
					"user_id":  "user-123",
					"email":    "user@example.com",
					"username": "johndoe",
				},
				"response": map[string]interface{}{
					"token":      "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
					"expires_in": "24h0m0s",
					"user_id":    "user-123",
				},
			},
			"verify_token": map[string]interface{}{
				"method":      "GET",
				"path":        "/api/auth/verify",
				"description": "Verify a JWT token",
				"headers": map[string]string{
					"Authorization": "Bearer <your-token>",
				},
				"response": map[string]interface{}{
					"user_id":       "user-123",
					"email":         "user@example.com",
					"username":      "johndoe",
					"authenticated": true,
				},
			},
		},
		"websocket_usage": map[string]interface{}{
			"description": "Use the token with WebSocket connections",
			"example":     "ws://localhost:8080/ws/nearby?token=<your-token>",
			"javascript": map[string]string{
				"connect": "const ws = new WebSocket(`ws://localhost:8080/ws/nearby?token=${token}`);",
			},
		},
		"curl_examples": map[string]interface{}{
			"generate_token": `curl -X POST http://localhost:8080/api/auth/token \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-123", "email": "user@example.com", "username": "johndoe"}'`,
			"verify_token": `curl -X GET http://localhost:8080/api/auth/verify \
  -H "Authorization: Bearer <your-token>"`,
		},
	}

	c.JSON(http.StatusOK, example)
}
