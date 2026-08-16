// Package config provides configuration loading for CYBEROS.
// Phase 0 Foundation: skeleton only.
package config

import (
	"os"
	"strings"
)

// Config holds application configuration.
// Expanded in later phases.
type Config struct {
	Env      string
	LogLevel string
	LogFormat string
	APIAddr  string
}

// Load reads configuration from environment variables with sensible defaults.
// No external config libraries in Phase 0.
func Load() *Config {
	return &Config{
		Env:       getEnv("CYBEROS_ENV", "development"),
		LogLevel:  getEnv("CYBEROS_LOG_LEVEL", "info"),
		LogFormat: getEnv("CYBEROS_LOG_FORMAT", "json"),
		APIAddr:   getEnv("CYBEROS_API_ADDR", ":8080"),
	}
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
