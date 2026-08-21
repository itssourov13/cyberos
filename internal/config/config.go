// Package config provides configuration loading for CYBEROS.
package config

import (
	"os"
	"strings"

	"cyberos/internal/secrets"
)

// Config holds application configuration.
// Sensitive material is referenced by secrets.Ref, never stored as plaintext fields
// intended for events/audit/task state.
type Config struct {
	Env          string
	LogLevel     string
	LogFormat    string
	APIAddr      string
	DatabaseURL  string
	AuthRequired bool   // production mandatory API auth (Phase 6H)
	BootstrapKey string // optional bootstrap API key plaintext for local/dev only

	// SecretRefs maps logical names to secret references (env:NAME, file:path).
	// Resolved values are obtained only via a secrets.Resolver at use sites.
	SecretRefs map[string]secrets.Ref
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	c := &Config{
		Env:          getEnv("CYBEROS_ENV", "development"),
		LogLevel:     getEnv("CYBEROS_LOG_LEVEL", "info"),
		LogFormat:    getEnv("CYBEROS_LOG_FORMAT", "json"),
		APIAddr:      getEnv("CYBEROS_API_ADDR", ":8080"),
		DatabaseURL:  getEnv("CYBEROS_DATABASE_URL", "postgres://cyberos:cyberos@localhost:5432/cyberos?sslmode=disable"),
		AuthRequired: getEnvBool("CYBEROS_AUTH_REQUIRED", false),
		BootstrapKey: getEnv("CYBEROS_BOOTSTRAP_API_KEY", ""),
		SecretRefs:   map[string]secrets.Ref{},
	}
	// Production environment defaults to auth required unless explicitly disabled.
	if c.Env == "production" && os.Getenv("CYBEROS_AUTH_REQUIRED") == "" {
		c.AuthRequired = true
	}
	// Optional named secret references (values stay in the environment, not here).
	if ref := getEnv("CYBEROS_SECRET_REF_API_TOKEN", ""); ref != "" {
		c.SecretRefs["api_token"] = secrets.Ref(ref)
	} else {
		// Default convention: reference env var name without reading its value into Config.
		c.SecretRefs["api_token"] = secrets.Ref("env:CYBEROS_API_TOKEN")
	}
	if ref := getEnv("CYBEROS_SECRET_REF_DB_PASSWORD", ""); ref != "" {
		c.SecretRefs["db_password"] = secrets.Ref(ref)
	}
	return c
}

// Resolver returns the default environment-backed secret resolver.
func (c *Config) Resolver() secrets.Resolver {
	return secrets.EnvResolver{}
}

// RedactedDatabaseURL returns the DSN with password material redacted for logs.
func (c *Config) RedactedDatabaseURL() string {
	return secrets.RedactString(c.DatabaseURL)
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}
