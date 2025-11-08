package middleware

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FACorreiaa/go-templui/internal/app/domain/auth"
	"github.com/FACorreiaa/go-templui/internal/app/models"
)

// Define typed context keys
type contextKey string

const UserContextKey contextKey = "user"

const UserIDKey contextKey = "userID"
const UserRoleKey contextKey = "userRole"
const UserPlanKey contextKey = "userPlan"
const UserEmail contextKey = "userEmail"
const userName contextKey = "userName"
const UserSubStatusKey contextKey = "userSubStatus"

// CORSMiddleware handles CORS headers
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, HX-Request, HX-Target, HX-Current-URL")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// AuthMiddleware validates authentication tokens
// Note: Logging is handled by ginzap middleware
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("auth_token")
		if err != nil {
			handleAuthRedirect(c, "/auth/signin")
			return
		}

		// Validate JWT token
		if token == "" {
			handleAuthRedirect(c, "/auth/signin")
			return
		}

		// Get JWT secret from environment
		jwtSecret := os.Getenv("JWT_SECRET_KEY")
		if jwtSecret == "" {
			jwtSecret = "default-secret-key-change-in-production-min-32-chars"
		}

		// Create JWT service and validate token
		jwtService := auth.NewJWTService()
		config := auth.JWTConfig{
			SecretKey:       jwtSecret,
			TokenExpiration: time.Hour * 24,
			Logger:          nil, // Logger will be injected elsewhere
		}
		claims, err := jwtService.ValidateToken(config, token)
		if err != nil {
			handleAuthRedirect(c, "/auth/signin")
			return
		}

		// Create User object from JWT claims
		user := &models.User{
			ID:       claims.UserID,
			Name:     claims.Username,
			Email:    claims.Email,
			IsActive: true,
		}

		// Set full user object in context - this is what GetUserFromContext looks for
		c.Set(string(UserContextKey), user)

		// Also set individual fields for backwards compatibility
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_name", claims.Username)
		c.Next()
	}
}

// handleAuthRedirect handles redirects for both regular and HTMX requests
func handleAuthRedirect(c *gin.Context, redirectURL string) {
	// Check if this is an HTMX request
	if c.GetHeader("HX-Request") == "true" {
		// For HTMX requests, use HX-Redirect header to trigger client-side redirect
		c.Header("HX-Redirect", redirectURL)
		c.AbortWithStatus(http.StatusUnauthorized)
	} else {
		// For regular requests, use standard HTTP redirect
		c.Redirect(http.StatusFound, redirectURL)
		c.Abort()
	}
}

// SecurityMiddleware adds security headers
func SecurityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Basic security headers
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy for HTMX, Mapbox and external resources
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://unpkg.com https://cdn.jsdelivr.net https://api.mapbox.com; " +
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://api.mapbox.com https://cdnjs.cloudflare.com; " +
			"font-src 'self' https://fonts.gstatic.com https://cdnjs.cloudflare.com; " +
			"img-src 'self' data: https: blob:; " +
			"connect-src 'self' https://unpkg.com https://cdn.jsdelivr.net https://fonts.googleapis.com https://fonts.gstatic.com https://api.mapbox.com https://*.tiles.mapbox.com https://events.mapbox.com; " +
			"worker-src 'self' blob:"
		c.Writer.Header().Set("Content-Security-Policy", csp)

		c.Next()
	}
}

// GetUserFromContext extracts user information from Gin context
func GetUserFromContext(c *gin.Context) *models.User {
	user, exists := c.Get(string(UserContextKey))
	if !exists {
		return nil
	}

	userModel, ok := user.(*models.User)
	if !ok {
		return nil
	}

	return userModel
}

// OptionalAuthMiddleware sets user context if token exists, but doesn't require auth
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("auth_token")

		// Debug: Log cookie status
		if err != nil {
			// No cookie found
			c.Next()
			return
		}

		if token == "" {
			// Empty token
			c.Next()
			return
		}

		// Get JWT secret from environment
		jwtSecret := os.Getenv("JWT_SECRET_KEY")
		if jwtSecret == "" {
			jwtSecret = "default-secret-key-change-in-production-min-32-chars"
		}

		// Create JWT service and validate token
		jwtService := auth.NewJWTService()
		config := auth.JWTConfig{
			SecretKey:       jwtSecret,
			TokenExpiration: time.Hour * 24,
			Logger:          nil, // Logger will be injected elsewhere
		}
		claims, err := jwtService.ValidateToken(config, token)
		if err == nil {
			// Create User object from JWT claims
			user := &models.User{
				ID:       claims.UserID,
				Name:     claims.Username,
				Email:    claims.Email,
				IsActive: true,
			}

			// Set full user object in context - this is what GetUserFromContext looks for
			c.Set(string(UserContextKey), user)

			// Also set individual fields for backwards compatibility
			c.Set("user_id", claims.UserID)
			c.Set("user_email", claims.Email)
			c.Set("user_name", claims.Username)
		}

		c.Next()
	}
}

// GetUserIDFromContext extracts just the user ID from context
func GetUserIDFromContext(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if idStr, ok := userID.(string); ok {
			return idStr
		}
	}
	return "anonymous"
}

// CreateContextWithUser creates a context.Context with user information for LLM services
func CreateContextWithUser(c *gin.Context) context.Context {
	ctx := c.Request.Context()

	// Add user information to context for LLM services that expect the old pattern
	if userID, exists := c.Get("user_id"); exists {
		ctx = context.WithValue(ctx, UserIDKey, userID.(string))
	}
	if userEmail, exists := c.Get("user_email"); exists {
		ctx = context.WithValue(ctx, userEmail, userEmail.(string))
	}
	if un, exists := c.Get("user_name"); exists {
		ctx = context.WithValue(ctx, userName, un.(string))
	}

	return ctx
}

// GetDBFromContext extracts the database pool from context
func GetDBFromContext(c *gin.Context) *pgxpool.Pool {
	if db, exists := c.Get("db"); exists {
		return db.(*pgxpool.Pool)
	}
	return nil
}

// ObservabilityMiddleware adds OpenTelemetry tracing and metrics to HTTP requests
//func ObservabilityMiddleware() gin.HandlerFunc {
//	tracer := otel.Tracer("loci-templui")
//	return gin.HandlerFunc(func(c *gin.Context) {
//		start := time.Now()
//
//		// Start tracing span
//		ctx, span := tracer.Start(c.Request.Context(), c.Request.Method+" "+c.Request.URL.Path)
//		defer span.End()
//
//		// Set span attributes
//		span.SetAttributes(
//			attribute.String("http.method", c.Request.Method),
//			attribute.String("http.url", c.Request.URL.String()),
//			attribute.String("http.user_agent", c.Request.UserAgent()),
//			attribute.String("client.ip", c.ClientIP()),
//		)
//
//		// Replace context with traced context
//		c.Request = c.Request.WithContext(ctx)
//
//		// Process request
//		c.Next()
//
//		// Record metrics
//		duration := time.Since(start).Seconds()
//		statusCode := c.Writer.Status()
//
//		// Add status code to span
//		span.SetAttributes(
//			attribute.Int("http.status_code", statusCode),
//			attribute.Float64("http.duration", duration),
//		)
//
//		// Record HTTP metrics
//		m := metrics.Get()
//		m.HTTPRequestsTotal.Add(context.Background(), 1,
//			metric.WithAttributes(
//				attribute.String("method", c.Request.Method),
//				attribute.String("path", c.Request.URL.Path),
//				attribute.String("status", strconv.Itoa(statusCode)),
//			))
//
//		m.HTTPRequestDuration.Record(context.Background(), duration,
//			metric.WithAttributes(
//				attribute.String("method", c.Request.Method),
//				attribute.String("path", c.Request.URL.Path),
//			))
//
//		// Record auth-specific metrics
//		if c.Request.URL.Path == "/auth/signin" || c.Request.URL.Path == "/auth/signup" {
//			m.AuthRequestsTotal.Add(context.Background(), 1,
//				metric.WithAttributes(
//					attribute.String("endpoint", c.Request.URL.Path),
//					attribute.String("status", strconv.Itoa(statusCode)),
//				))
//		}
//
//		// Record search-specific metrics
//		if c.Request.URL.Path == "/discover/search" || c.Request.URL.Path == "/search" {
//			m.SearchRequestsTotal.Add(context.Background(), 1,
//				metric.WithAttributes(
//					attribute.String("endpoint", c.Request.URL.Path),
//				))
//		}
//	})
//}
