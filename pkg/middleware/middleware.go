package middleware

import (
	"net/http"
	"time"

	"github.com/FACorreiaa/go-templui/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
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

		// Validate token (simplified for demo)
		if token == "" {
			logger.Log.Warn("Empty auth token", 
				zap.String("path", c.Request.URL.Path),
				zap.String("ip", c.ClientIP()),
			)
			c.Redirect(http.StatusFound, "/auth/signin")
			c.Abort()
			return
		}

		logger.Log.Debug("Valid auth token found",
			zap.String("path", c.Request.URL.Path),
			zap.String("user_token", token[:10]+"..."), // Log only first 10 chars for security
		)

		// Set user context (simplified - in real app, decode JWT)
		c.Set("user_id", "demo-user")
		c.Set("user_email", "demo@loci.app")
		c.Set("user_name", "Demo User")
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
		
		// Content Security Policy for HTMX
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' https://unpkg.com; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; " +
			"connect-src 'self'"
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