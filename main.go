package main

import (
	"context"
	"fmt"
	"log"

	"github.com/joho/godotenv"

	database "github.com/FACorreiaa/go-templui/app/db"
	"github.com/FACorreiaa/go-templui/app/observability/metrics"
	"github.com/FACorreiaa/go-templui/app/observability/tracer"
	"github.com/FACorreiaa/go-templui/app/pkg/config"
	"github.com/FACorreiaa/go-templui/app/pkg/logger"
	"github.com/FACorreiaa/go-templui/app/pkg/middleware"

	"github.com/FACorreiaa/go-templui/app/routes"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Initialize logger
	err = logger.Init(zapcore.InfoLevel, zap.String("service", "loci-templui"))
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}

	logger.Log.Info("Starting Loci TemplUI application")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Log.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Initialize observability (OpenTelemetry)
	otelShutdown, err := tracer.InitOtelProviders("loci-templui", ":9092")
	if err != nil {
		logger.Log.Fatal("Failed to initialize OpenTelemetry", zap.Error(err))
	}
	defer func() {
		if err := otelShutdown(context.Background()); err != nil {
			logger.Log.Error("Failed to shutdown OpenTelemetry", zap.Error(err))
		}
	}()

	// Initialize application metrics
	metrics.InitAppMetrics()
	logger.Log.Info("Observability initialized", zap.String("metrics_endpoint", ":9092/metrics"))

	// Setup database
	ctx := context.Background()
	dbPool, err := setupDatabase(ctx, cfg)
	if err != nil {
		logger.Log.Fatal("Failed to setup database", zap.Error(err))
	}
	defer dbPool.Close()

	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	// Create Gin router
	r := gin.New()

	// Setup middleware (inject database pool into context)
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.OTELGinMiddleware("loci-templui"))
	r.Use(middleware.ObservabilityMiddleware())
	r.Use(gin.Recovery())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.SecurityMiddleware())
	r.Use(func(c *gin.Context) {
		c.Set("db", dbPool)
		c.Next()
	})

	// Setup routes with database pool
	routes.Setup(r, dbPool)

	// Start server
	serverPort := ":" + cfg.ServerPort
	logger.Log.Info("Server starting", zap.String("port", cfg.ServerPort))
	if err := r.Run(serverPort); err != nil {
		logger.Log.Fatal("Failed to start server", zap.Error(err))
	}
}

func setupDatabase(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	logger.Log.Info("Setting up database connection and migrations")

	dbConfig, err := database.NewDatabaseConfig(cfg, logger.Log)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database configuration: %w", err)
	}

	pool, err := database.Init(dbConfig.ConnectionURL, logger.Log)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database pool: %w", err)
	}

	database.WaitForDB(ctx, pool, logger.Log)
	logger.Log.Info("Connected to Postgres",
		zap.String("host", cfg.Repositories.Postgres.Host),
		zap.String("port", cfg.Repositories.Postgres.Port),
		zap.String("database", cfg.Repositories.Postgres.DB))

	if err = database.RunMigrations(dbConfig.ConnectionURL, logger.Log); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	logger.Log.Info("Database setup completed successfully")
	return pool, nil
}
