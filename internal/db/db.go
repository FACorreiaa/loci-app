// Package database app/platform/database/db.go (or your db package path)
package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/internal/pkg/config"
)

//go:embed migrations
var migrationFS embed.FS

const defaultRetries = 5 // Renamed from retries to avoid conflict if needed elsewhere

type DatabaseConfig struct {
	ConnectionURL string
}

// WaitForDB waits for the database connection pool to be available..
func WaitForDB(ctx context.Context, pgpool *pgxpool.Pool, logger *zap.Logger) bool {
	maxAttempts := defaultRetries
	for attempts := 1; attempts <= maxAttempts; attempts++ {
		err := pgpool.Ping(ctx)
		if err == nil {
			logger.Info("Database connection successful")
			return true // Connection successful
		}

		waitDuration := time.Duration(attempts) * 200 * time.Millisecond // Increased base wait time slightly
		logger.Warn("Database ping failed, retrying...",
			zap.Int("attempt", attempts),
			zap.Int("max_attempts", maxAttempts),
			zap.Duration("wait_duration", waitDuration),
			zap.Error(err),
		)
		// Don't wait after the last attempt
		if attempts < maxAttempts {
			time.Sleep(waitDuration)
		}
	}
	logger.Error("Database connection failed after multiple retries")
	return false // Failed to connect after retries
}

func RunMigrations(databaseURL string, logger *zap.Logger) error {
	logger.Info("Running database migrations...")

	// Set up goose to use embedded migrations
	goose.SetBaseFS(migrationFS)

	if err := goose.SetDialect(string(goose.DialectPostgres)); err != nil {
		logger.Error("Failed to set goose dialect", zap.Error(err))
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	// Open a standard database connection for goose
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		logger.Error("Failed to open database for migrations", zap.Error(err))
		return fmt.Errorf("sql.Open failed: %w", err)
	}
	defer db.Close()

	// Log available migrations
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		logger.Error("Failed to read embedded migrations directory", zap.Error(err))
		return fmt.Errorf("failed to read embedded migrations directory: %w", err)
	}
	if len(entries) == 0 {
		logger.Warn("No migration files found in embedded migrations directory")
	} else {
		logger.Info("Found migration files", zap.Int("count", len(entries)))
		for _, entry := range entries {
			logger.Debug("Migration file", zap.String("name", entry.Name()))
		}
	}

	// Run migrations
	if err := goose.Up(db, "migrations"); err != nil {
		logger.Error("Failed to run migrations", zap.Error(err))
		return fmt.Errorf("goose.Up failed: %w", err)
	}

	logger.Info("Database migrations completed successfully")
	return nil
}

// NewDatabaseConfig generates the database connection URL from configuration.
func NewDatabaseConfig(cfg *config.Config, logger *zap.Logger) (*DatabaseConfig, error) {
	// Assume cfg is already loaded and valid if passed here
	if cfg == nil || cfg.Repositories.Postgres.Host == "" {
		errMsg := "Postgres configuration is missing or invalid"
		logger.Error(errMsg)
		return nil, fmt.Errorf("%s", errMsg)
	}

	// schema := os.Getenv("POSTGRES_SCHEMA") // Get schema if needed, maybe from cfg instead?
	// if schema != "" {
	// 	query.Add("search_path", schema)
	// }

	query := url.Values{}
	query.Set("sslmode", "disable") // Or get from config: cfg.Repositories.Postgres.SSLMode
	query.Set("timezone", "utc")

	connURL := url.URL{
		Scheme:   "postgresql", // Use postgresql:// for migrate compatibility
		User:     url.UserPassword(cfg.Repositories.Postgres.Username, cfg.Repositories.Postgres.Password),
		Host:     fmt.Sprintf("%s:%s", cfg.Repositories.Postgres.Host, cfg.Repositories.Postgres.Port),
		Path:     cfg.Repositories.Postgres.DB,
		RawQuery: query.Encode(),
	}

	connStr := connURL.String()
	// Avoid logging password in production logs if possible
	logger.Info("Database connection URL generated", zap.String("host", connURL.Host), zap.String("database", connURL.Path))
	// fmt.Printf("Connection URL: %s\n", connStr) // Keep for local dev if helpful

	return &DatabaseConfig{
		ConnectionURL: connStr,
	}, nil
}

// Init initializes the pgxpool connection pool.
func Init(connectionURL string, logger *zap.Logger) (*pgxpool.Pool, error) {
	logger.Info("Initializing database connection pool...")
	cfg, err := pgxpool.ParseConfig(connectionURL)
	if err != nil {
		logger.Error("Failed to parse database config", zap.Error(err))
		return nil, fmt.Errorf("failed parsing db config: %w", err)
	}

	// Consider adjusting pool settings from config
	// cfg.MaxConns = cfg.Repositories.Postgres.MaxConns
	// cfg.MinConns = cfg.Repositories.Postgres.MinConns
	// cfg.MaxConnLifetime = ...
	// cfg.MaxConnIdleTime = ...

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		logger.Error("Failed to create database connection pool", zap.Error(err))
		return nil, fmt.Errorf("failed creating db pool: %w", err)
	}

	logger.Info("Database connection pool initialized")
	return pool, nil
}
