package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/internal/db"
	"github.com/FACorreiaa/go-templui/internal/pkg/config"
)

// Server holds the dependencies for the HTTP server
type Server struct {
	cfg    *config.Config
	logger *zap.Logger
	dbPool *pgxpool.Pool
	router http.Handler
}

// New creates a new Server instance with all dependencies
func New(cfg *config.Config, logger *zap.Logger) (*Server, error) {
	s := &Server{
		cfg:    cfg,
		logger: logger,
	}

	// Setup database
	ctx := context.Background()
	dbPool, err := s.setupDatabase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to setup database: %w", err)
	}
	s.dbPool = dbPool

	return s, nil
}

// setupDatabase initializes the database connection and runs migrations
func (s *Server) setupDatabase(ctx context.Context) (*pgxpool.Pool, error) {
	s.logger.Info("Setting up database connection and migrations")

	dbConfig, err := database.NewDatabaseConfig(s.cfg, s.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database configuration: %w", err)
	}

	pool, err := database.Init(dbConfig.ConnectionURL, s.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database pool: %w", err)
	}

	database.WaitForDB(ctx, pool, s.logger)
	s.logger.Info("Connected to Postgres",
		zap.String("host", s.cfg.Repositories.Postgres.Host),
		zap.String("port", s.cfg.Repositories.Postgres.Port),
		zap.String("database", s.cfg.Repositories.Postgres.DB))

	if err = database.RunMigrations(dbConfig.ConnectionURL, s.logger); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	s.logger.Info("Database setup completed successfully")
	return pool, nil
}

// HTTPServer creates and configures the HTTP server
func (s *Server) HTTPServer() *http.Server {
	return &http.Server{
		Addr:         ":" + s.cfg.ServerPort,
		Handler:      s.router,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
}

// SetRouter sets the HTTP router/handler
func (s *Server) SetRouter(router http.Handler) {
	s.router = router
}

// GetDBPool returns the database connection pool
func (s *Server) GetDBPool() *pgxpool.Pool {
	return s.dbPool
}

// GetLogger returns the logger instance
func (s *Server) GetLogger() *zap.Logger {
	return s.logger
}

// GetConfig returns the configuration
func (s *Server) GetConfig() *config.Config {
	return s.cfg
}

// Close closes all server resources
func (s *Server) Close() {
	if s.dbPool != nil {
		s.dbPool.Close()
	}
}
