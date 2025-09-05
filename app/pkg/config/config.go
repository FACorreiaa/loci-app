package config

import (
	"fmt"
	"os"
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

type RepositoriesConfig struct {
	Postgres PostgresConfig
}

type Config struct {
	Repositories RepositoriesConfig
	ServerPort   string
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

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}