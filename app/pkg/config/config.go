package config

import (
	"fmt"
	"os"
	"time"
)

type PostgresConfig struct {
	Host     string
	Port     string
	DB       string
	Username string
	Password string
	SSLMode  string
	MaxConns int32
	MinConns int32
}

type JWTConfig struct {
	SecretKey       string
	Issuer          string
	Audience        string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type RepositoriesConfig struct {
	Postgres PostgresConfig
}

type LLMConfig struct {
	StreamEndpoint string
}

type MapConfig struct {
	MapboxAPIKey string
}

type Config struct {
	Repositories RepositoriesConfig
	ServerPort   string
	JWT          JWTConfig
	LLM          LLMConfig
	Map          MapConfig
}

func Load() (*Config, error) {
	cfg := &Config{
		Repositories: RepositoriesConfig{
			Postgres: PostgresConfig{
				Host:     getEnvOrDefault("POSTGRES_HOST", "localhost"),
				Port:     getEnvOrDefault("POSTGRES_PORT", "5454"),
				DB:       getEnvOrDefault("POSTGRES_DB", "loci_templui"),
				Username: getEnvOrDefault("POSTGRES_USER", "postgres"),
				Password: getEnvOrDefault("POSTGRES_PASSWORD", ""),
				SSLMode:  getEnvOrDefault("POSTGRES_SSLMODE", "disable"),
				MaxConns: 30,
				MinConns: 5,
			},
		},
		ServerPort: getEnvOrDefault("SERVER_PORT", "8091"),
	}

	if cfg.Repositories.Postgres.Password == "" {
		return nil, fmt.Errorf("POSTGRES_PASSWORD environment variable is required")
	}

	// Load JWT configuration
	accessTTL, err := time.ParseDuration(getEnvOrDefault("JWT_ACCESS_TOKEN_TTL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_TOKEN_TTL: %v", err)
	}

	refreshTTL, err := time.ParseDuration(getEnvOrDefault("JWT_REFRESH_TOKEN_TTL", "168h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_TOKEN_TTL: %v", err)
	}

	cfg.JWT = JWTConfig{
		SecretKey:       getEnvOrDefault("JWT_SECRET_KEY", ""),
		Issuer:          getEnvOrDefault("JWT_ISSUER", "loci"),
		Audience:        getEnvOrDefault("JWT_AUDIENCE", "loci-app"),
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,
	}

	cfg.LLM = LLMConfig{
		StreamEndpoint: getEnvOrDefault("LLM_STREAM_ENDPOINT", "http://localhost:8000/api/v1/llm"),
	}

	cfg.Map = MapConfig{
		MapboxAPIKey: getEnvOrDefault("MAPBOX_API_KEY", ""),
	}

	if cfg.JWT.SecretKey == "" {
		return nil, fmt.Errorf("JWT_SECRET_KEY environment variable is required")
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
