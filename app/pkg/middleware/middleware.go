package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/FACorreiaa/go-templui/app/observability/metrics"
	"github.com/FACorreiaa/go-templui/app/pkg/domain/auth"
	"github.com/FACorreiaa/go-templui/app/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

// LoggerMiddleware logs all HTTP requests using zap
func LoggerMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log request
		param := gin.LogFormatterParams{
			Request:      c.Request,
			TimeStamp:    time.Now(),
			Latency:      time.Since(start),
			ClientIP:     c.ClientIP(),
			Method:       c.Request.Method,
			StatusCode:   c.Writer.Status(),
			ErrorMessage: c.Errors.ByType(gin.ErrorTypePrivate).String(),
		}

		if raw != "" {
			param.Path = path + "?" + raw
		} else {
			param.Path = path
		}

		// Log with appropriate level based on status code
		fields := []zap.Field{
			zap.String("method", param.Method),
			zap.String("path", param.Path),
			zap.String("ip", param.ClientIP),
			zap.Int("status", param.StatusCode),
			zap.Duration("latency", param.Latency),
			zap.String("user_agent", c.GetHeader("User-Agent")),
		}

		if param.ErrorMessage != "" {
			fields = append(fields, zap.String("error", param.ErrorMessage))
		}

		switch {
		case param.StatusCode >= 500:
			logger.Log.Error("HTTP Request", fields...)
		case param.StatusCode >= 400:
			logger.Log.Warn("HTTP Request", fields...)
		default:
			logger.Log.Info("HTTP Request", fields...)
		}
	})
}

// CORSMiddleware handles CORS headers
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		logger.Log.Debug("CORS request",
			zap.String("origin", origin),
			zap.String("method", c.Request.Method),
		)

		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, HX-Request, HX-Target, HX-Current-URL")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			logger.Log.Debug("CORS preflight request handled")
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// AuthMiddleware validates authentication tokens
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("auth_token")
		if err != nil {
			logger.Log.Warn("No auth token found",
				zap.String("path", c.Request.URL.Path),
				zap.String("ip", c.ClientIP()),
			)
			c.Redirect(http.StatusFound, "/auth/signin")
			c.Abort()
			return
		}

		// Validate JWT token
		if token == "" {
			logger.Log.Warn("Empty auth token",
				zap.String("path", c.Request.URL.Path),
				zap.String("ip", c.ClientIP()),
			)
			c.Redirect(http.StatusFound, "/auth/signin")
			c.Abort()
			return
		}

		// Create JWT service and validate token
		jwtService := auth.NewJWTService()
		claims, err := jwtService.ValidateToken(token)
		if err != nil {
			logger.Log.Warn("Invalid auth token",
				zap.String("path", c.Request.URL.Path),
				zap.String("ip", c.ClientIP()),
				zap.Error(err),
			)
			c.Redirect(http.StatusFound, "/auth/signin")
			c.Abort()
			return
		}

		logger.Log.Debug("Valid auth token found",
			zap.String("path", c.Request.URL.Path),
			zap.String("user_id", claims.UserID),
		)

		// Set user context from JWT claims
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_name", claims.Name)
		c.Next()
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
func GetUserFromContext(c *gin.Context) (string, string, string) {
	userID := "anonymous"
	userEmail := ""
	userName := ""

	if id, exists := c.Get("user_id"); exists {
		userID = id.(string)
	}
	if email, exists := c.Get("user_email"); exists {
		userEmail = email.(string)
	}
	if name, exists := c.Get("user_name"); exists {
		userName = name.(string)
	}

	return userID, userEmail, userName
}

// OptionalAuthMiddleware sets user context if token exists, but doesn't require auth
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("auth_token")
		if err == nil && token != "" {
			// Create JWT service and validate token
			jwtService := auth.NewJWTService()
			claims, err := jwtService.ValidateToken(token)
			if err == nil {
				// Set user context from JWT claims
				c.Set("user_id", claims.UserID)
				c.Set("user_email", claims.Email)
				c.Set("user_name", claims.Name)
			}
		}
		c.Next()
	}
}

// GetUserIDFromContext extracts just the user ID from context
func GetUserIDFromContext(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		return userID.(string)
	}
	return "anonymous"
}

// GetDBFromContext extracts the database pool from context
func GetDBFromContext(c *gin.Context) *pgxpool.Pool {
	if db, exists := c.Get("db"); exists {
		return db.(*pgxpool.Pool)
	}
	return nil
}

// ObservabilityMiddleware adds OpenTelemetry tracing and metrics to HTTP requests
func ObservabilityMiddleware() gin.HandlerFunc {
	tracer := otel.Tracer("loci-templui")
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()

		// Start tracing span
		ctx, span := tracer.Start(c.Request.Context(), c.Request.Method+" "+c.Request.URL.Path)
		defer span.End()

		// Set span attributes
		span.SetAttributes(
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.url", c.Request.URL.String()),
			attribute.String("http.user_agent", c.Request.UserAgent()),
			attribute.String("client.ip", c.ClientIP()),
		)

		// Replace context with traced context
		c.Request = c.Request.WithContext(ctx)

		// Process request
		c.Next()

		// Record metrics
		duration := time.Since(start).Seconds()
		statusCode := c.Writer.Status()

		// Add status code to span
		span.SetAttributes(
			attribute.Int("http.status_code", statusCode),
			attribute.Float64("http.duration", duration),
		)

		// Record HTTP metrics
		m := metrics.Get()
		m.HTTPRequestsTotal.Add(context.Background(), 1,
			metric.WithAttributes(
				attribute.String("method", c.Request.Method),
				attribute.String("path", c.Request.URL.Path),
				attribute.String("status", strconv.Itoa(statusCode)),
			))

		m.HTTPRequestDuration.Record(context.Background(), duration,
			metric.WithAttributes(
				attribute.String("method", c.Request.Method),
				attribute.String("path", c.Request.URL.Path),
			))

		// Record auth-specific metrics
		if c.Request.URL.Path == "/auth/signin" || c.Request.URL.Path == "/auth/signup" {
			m.AuthRequestsTotal.Add(context.Background(), 1,
				metric.WithAttributes(
					attribute.String("endpoint", c.Request.URL.Path),
					attribute.String("status", strconv.Itoa(statusCode)),
				))
		}

		// Record search-specific metrics
		if c.Request.URL.Path == "/discover/search" || c.Request.URL.Path == "/search" {
			m.SearchRequestsTotal.Add(context.Background(), 1,
				metric.WithAttributes(
					attribute.String("endpoint", c.Request.URL.Path),
				))
		}
	})
}
