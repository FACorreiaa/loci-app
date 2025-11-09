package server

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	middleware2 "github.com/FACorreiaa/go-templui/internal/app/middleware"
	"github.com/FACorreiaa/go-templui/internal/routes"
)

const (
	userkey = "user"   // key used to store the username in the session
	secret  = "secret" // random and secure key used to encrypt the session cookie
)

// SetupRouter configures and returns the Gin router with all middleware and routes
func SetupRouter(dbPool *pgxpool.Pool, logger *zap.Logger) *gin.Engine {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	// Create Gin router
	r := gin.New()

	// Setup middleware
	r.Use(ginzap.GinzapWithConfig(logger, &ginzap.Config{
		UTC:        true,
		TimeFormat: time.RFC3339,
		Context:    zapContextFunc(),
	}))
	r.Use(ginzap.RecoveryWithZap(logger, true))
	r.Use(middleware2.OTELGinMiddleware("loci-templui"))
	r.Use(gin.Recovery())
	r.Use(middleware2.CORSMiddleware())
	r.Use(middleware2.SecurityMiddleware())
	r.Use(func(c *gin.Context) {
		c.Set("db", dbPool)
		c.Next()
	})
	r.Use(sessions.Sessions("mysession", cookie.NewStore([]byte(secret))))

	// Setup routes
	routes.Setup(r, dbPool, logger)

	return r
}

// zapContextFunc returns the Zap context function for logging
func zapContextFunc() ginzap.Fn {
	return func(c *gin.Context) []zapcore.Field {
		fields := []zapcore.Field{}

		// Request ID (from header; customize key if needed)
		if requestID := c.Writer.Header().Get("X-Request-Id"); requestID != "" {
			fields = append(fields, zap.String("request_id", requestID))
		}

		// OTEL trace/span IDs (from context)
		if span := trace.SpanFromContext(c.Request.Context()); span.SpanContext().IsValid() {
			fields = append(fields,
				zap.String("trace_id", span.SpanContext().TraceID().String()),
				zap.String("span_id", span.SpanContext().SpanID().String()),
			)
		}

		// Request body (reads and restores; skip for large/streaming bodies to avoid perf hit)
		if c.Request.Body != nil {
			var buf bytes.Buffer
			tee := io.TeeReader(c.Request.Body, &buf)
			body, _ := io.ReadAll(tee)
			c.Request.Body = io.NopCloser(&buf)
			if len(body) > 0 {
				fields = append(fields, zap.String("body", string(body)))
			}
		}

		return fields
	}
}
