// Package database internal/platform/database/db.go (or your db package path)
package database

import (
	"context"
	"embed"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // Required for postgres driver registration
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	uuid "github.com/vgarvardt/pgx-google-uuid/v5"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/app/pkg/config"
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

	entries, err := migrationFS.ReadDir(".")
	if err != nil {
		logger.Error("Failed to read embedded migrations directory", zap.Error(err))
		return fmt.Errorf("failed to read embedded migrations directory: %w", err)
	}
	if len(entries) == 0 {
		logger.Warn("No migration files found in embedded migrations directory")
	}
	for _, entry := range entries {
		logger.Info("Found embedded migration file", zap.String("name", entry.Name()))
	}

	sourceDriver, err := iofs.New(migrationFS, "migrations")
	if err != nil {
		logger.Error("Failed to create migration source driver", zap.Error(err))
		return fmt.Errorf("failed to create migration source driver: %w", err)
	}

	if !strings.HasPrefix(databaseURL, "postgres://") && !strings.HasPrefix(databaseURL, "postgresql://") {
		errMsg := "invalid database URL scheme for migrate, ensure it starts with postgresql://"
		logger.Error(errMsg, zap.String("url", databaseURL))
		return fmt.Errorf("%s", errMsg)
	}

	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, databaseURL)
	if err != nil {
		logger.Error("Failed to initialize migrate instance", zap.Error(err))
		return fmt.Errorf("failed to initialize migrate instance: %w", err)
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		logger.Error("Failed to apply migrations", zap.Error(err))
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil {
		logger.Warn("Could not determine migration version", zap.Error(err))
	} else if dirty {
		logger.Error("DATABASE MIGRATION STATE IS DIRTY!", zap.Uint64("version", uint64(version)))
	} else if err == migrate.ErrNoChange {
		logger.Info("No new migrations to apply.", zap.Uint64("current_version", uint64(version)))
	} else {
		logger.Info("Database migrations applied successfully.", zap.Uint64("new_version", uint64(version)))
	}

	srcErr, dbErr := m.Close()
	if srcErr != nil {
		logger.Warn("Error closing migration source", zap.Error(srcErr))
	}
	if dbErr != nil {
		logger.Warn("Error closing migration database connection", zap.Error(dbErr))
	}

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

	// Register UUID type HandlerImpl after connecting
	cfg.AfterConnect = func(_ context.Context, conn *pgx.Conn) error {
		uuid.Register(conn.TypeMap())
		logger.Debug("Registered UUID type for database connection")
		return nil
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
