// Package config provides configuration loading for CYBEROS.
package config

import (
	"os"
	"strings"
)

// Config holds application configuration.
type Config struct {
	Env         string
	LogLevel    string
	LogFormat   string
	APIAddr     string
	DatabaseURL string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Env:         getEnv("CYBEROS_ENV", "development"),
		LogLevel:    getEnv("CYBEROS_LOG_LEVEL", "info"),
		LogFormat:   getEnv("CYBEROS_LOG_FORMAT", "json"),
		APIAddr:     getEnv("CYBEROS_API_ADDR", ":8080"),
		DatabaseURL: getEnv("CYBEROS_DATABASE_URL", "postgres://cyberos:cyberos@localhost:5432/cyberos?sslmode=disable"),
	}
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
