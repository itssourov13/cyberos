// Package config provides configuration loading for CYBEROS.
package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"cyberos/internal/secrets"
)

// Config holds application configuration.
// Sensitive material is referenced by secrets.Ref, never stored as plaintext fields
// intended for events/audit/task state.
type Config struct {
	Env       string
	LogLevel  string
	LogFormat string
	APIAddr   string // listen address; development default is loopback-only

	// HTTP server timeouts (defaults chosen to keep SSE and slow clients safe; SSE clears its own write deadline).
	APIReadHeaderTimeout time.Duration
	APIReadTimeout       time.Duration
	APIWriteTimeout      time.Duration
	APIIdleTimeout       time.Duration
	DatabaseURL          string
	// DatabaseURLExplicit is true only when CYBEROS_DATABASE_URL was set. The
	// development fallback DSN never counts as explicit configuration.
	DatabaseURLExplicit bool
	// AllowDemoExecutors permits the mock/stub executor set. It is honoured only
	// outside production (CYBEROS_ALLOW_DEMO_EXECUTORS, default off).
	AllowDemoExecutors bool
	// SessionBackend selects "postgres" (durable, shared) or "memory"
	// (development only). Empty defaults by environment.
	SessionBackend   string
	AuthRequired     bool // production mandatory API auth (Phase 6H)
	TLSCertFile      string
	TLSKeyFile       string
	WorkerCACertFile string
	BootstrapKey     string // secret-manager supplied platform operator credential; never logged in production
	// GrantSigningKey is a hex-encoded secret (>=32 bytes) used to sign ExecutionGrants.
	// Required when Env=production; development falls back to an ephemeral per-process key.
	GrantSigningKey string
	// Session/cookie settings (Phase 24D). AllowedOrigins are extra trusted
	// origins for cookie-authenticated writes (same-origin is always allowed).
	SessionTTL     time.Duration
	SessionIdle    time.Duration
	CookieSecure   bool
	AllowedOrigins []string

	// Process runtime (Phase 9A) — fail-closed by default.
	ProcessEnabled     bool
	ProcessAllowShell  bool
	ProcessAllowedBins []string
	ProcessAllowedDirs []string
	ProcessTimeout     string // e.g. "10s"
	ProcessMaxOutput   int64  // bytes; 0 → runner default

	// SecretRefs maps logical names to secret references (env:NAME, file:path).
	// Resolved values are obtained only via a secrets.Resolver at use sites.
	SecretRefs map[string]secrets.Ref
}

// DefaultAPIAddr binds to loopback only so a development server is not reachable from the network.
const (
	DefaultAPIAddr = "127.0.0.1:8080"
	// DevDatabaseURL is deliberately limited to non-production environments.
	DevDatabaseURL = "postgres://cyberos:cyberos@localhost:5432/cyberos?sslmode=disable"
)

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	c := &Config{
		Env:                  getEnv("CYBEROS_ENV", "development"),
		LogLevel:             getEnv("CYBEROS_LOG_LEVEL", "info"),
		LogFormat:            getEnv("CYBEROS_LOG_FORMAT", "json"),
		APIAddr:              getEnv("CYBEROS_API_ADDR", DefaultAPIAddr),
		APIReadHeaderTimeout: getEnvDuration("CYBEROS_API_READ_HEADER_TIMEOUT", 10*time.Second),
		APIReadTimeout:       getEnvDuration("CYBEROS_API_READ_TIMEOUT", 30*time.Second),
		APIWriteTimeout:      getEnvDuration("CYBEROS_API_WRITE_TIMEOUT", 120*time.Second),
		APIIdleTimeout:       getEnvDuration("CYBEROS_API_IDLE_TIMEOUT", 120*time.Second),
		DatabaseURL:          strings.TrimSpace(os.Getenv("CYBEROS_DATABASE_URL")),
		SessionBackend:       strings.ToLower(getEnv("CYBEROS_SESSION_BACKEND", "")),
		TLSCertFile:          getEnv("CYBEROS_TLS_CERT_FILE", ""),
		TLSKeyFile:           getEnv("CYBEROS_TLS_KEY_FILE", ""),
		WorkerCACertFile:     getEnv("CYBEROS_WORKER_CA_CERT_FILE", ""),
		AllowDemoExecutors:   getEnvBool("CYBEROS_ALLOW_DEMO_EXECUTORS", false),
		AuthRequired:         getEnvBool("CYBEROS_AUTH_REQUIRED", false),
		BootstrapKey:         getEnv("CYBEROS_BOOTSTRAP_API_KEY", ""),
		GrantSigningKey:      getEnv("CYBEROS_GRANT_SIGNING_KEY", ""),
		SessionTTL:           getEnvDuration("CYBEROS_SESSION_TTL", 12*time.Hour),
		SessionIdle:          getEnvDuration("CYBEROS_SESSION_IDLE", 30*time.Minute),
		CookieSecure:         getEnvBool("CYBEROS_COOKIE_SECURE", false),
		AllowedOrigins:       splitCSV(getEnv("CYBEROS_ALLOWED_ORIGINS", "")),
		ProcessEnabled:       getEnvBool("CYBEROS_PROCESS_ENABLED", false),
		ProcessAllowShell:    getEnvBool("CYBEROS_PROCESS_ALLOW_SHELL", false),
		ProcessAllowedBins:   splitCSV(getEnv("CYBEROS_PROCESS_ALLOWED_BINS", "")),
		ProcessAllowedDirs:   splitCSV(getEnv("CYBEROS_PROCESS_ALLOWED_DIRS", "")),
		ProcessTimeout:       getEnv("CYBEROS_PROCESS_TIMEOUT", "10s"),
		ProcessMaxOutput:     getEnvInt64("CYBEROS_PROCESS_MAX_OUTPUT_BYTES", 0),
		SecretRefs:           map[string]secrets.Ref{},
	}
	// The insecure local DSN and in-memory sessions are development-only.
	// Every other named environment uses the durable path and fails closed when
	// PostgreSQL, auth, or secure-cookie configuration is missing.
	c.DatabaseURLExplicit = c.DatabaseURL != ""
	productionLike := c.Env != "" && c.Env != "development"
	if c.DatabaseURL == "" && !productionLike {
		c.DatabaseURL = DevDatabaseURL
	}
	if c.SessionBackend == "" {
		if productionLike {
			c.SessionBackend = "postgres"
		} else {
			c.SessionBackend = "memory"
		}
	}
	if productionLike {
		c.AuthRequired = true
		if os.Getenv("CYBEROS_COOKIE_SECURE") == "" {
			c.CookieSecure = true
		}
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

// ValidateProduction rejects unsafe or incomplete production configuration.
func (c *Config) ValidateProduction() error {
	if c == nil {
		return fmt.Errorf("config required")
	}
	if c.Env == "development" || c.Env == "" {
		return nil
	}
	if !c.DatabaseURLExplicit || c.DatabaseURL == "" {
		return fmt.Errorf("CYBEROS_DATABASE_URL must be explicitly configured in production")
	}
	u, err := url.Parse(c.DatabaseURL)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Host == "" {
		return fmt.Errorf("CYBEROS_DATABASE_URL must be a valid PostgreSQL URL")
	}
	if u.Query().Get("sslmode") != "verify-full" {
		return fmt.Errorf("production PostgreSQL requires sslmode=verify-full")
	}
	if !c.AuthRequired {
		return fmt.Errorf("authentication cannot be disabled in production")
	}
	if !c.CookieSecure {
		return fmt.Errorf("secure session cookies are mandatory in production")
	}
	if c.SessionBackend != "postgres" {
		return fmt.Errorf("production requires CYBEROS_SESSION_BACKEND=postgres")
	}
	if c.AllowDemoExecutors {
		return fmt.Errorf("demo executors cannot be enabled in production")
	}
	if c.GrantSigningKey == "" {
		return fmt.Errorf("CYBEROS_GRANT_SIGNING_KEY required in production")
	}
	if (c.TLSCertFile == "") != (c.TLSKeyFile == "") {
		return fmt.Errorf("CYBEROS_TLS_CERT_FILE and CYBEROS_TLS_KEY_FILE must be configured together")
	}
	if c.WorkerCACertFile != "" && c.TLSCertFile == "" {
		return fmt.Errorf("CYBEROS_WORKER_CA_CERT_FILE requires direct TLS certificate/key configuration")
	}
	if c.ProcessEnabled && len(c.ProcessAllowedBins) == 0 {
		return fmt.Errorf("CYBEROS_PROCESS_ENABLED requires an explicit CYBEROS_PROCESS_ALLOWED_BINS allowlist")
	}
	if c.ProcessAllowShell {
		return fmt.Errorf("CYBEROS_PROCESS_ALLOW_SHELL must remain false in production")
	}
	if c.ProcessMaxOutput > 4*1024*1024 {
		return fmt.Errorf("CYBEROS_PROCESS_MAX_OUTPUT_BYTES cannot exceed 4 MiB")
	}
	return nil
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

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getEnvInt64(key string, fallback int64) int64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	var n int64
	for _, r := range v {
		if r < '0' || r > '9' {
			return fallback
		}
		n = n*10 + int64(r-'0')
	}
	return n
}

func getEnvBool(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}

// IsLoopbackAddr reports whether a host:port listen address binds only to loopback.
// Wildcard hosts (empty, 0.0.0.0, ::) and any non-IP hostname other than "localhost"
// are treated as network-accessible. Unparseable addresses return an error.
func IsLoopbackAddr(addr string) (bool, error) {
	host, _, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		return false, fmt.Errorf("invalid listen address %q: %w", addr, err)
	}
	if i := strings.IndexByte(host, '%'); i >= 0 { // IPv6 zone
		host = host[:i]
	}
	if host == "" {
		return false, nil // wildcard
	}
	if strings.EqualFold(host, "localhost") {
		return true, nil
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false, nil // unknown hostname: assume reachable
	}
	if ip.IsUnspecified() {
		return false, nil
	}
	return ip.IsLoopback(), nil
}

// ValidateListenSecurity fails closed when a network-accessible listen address is
// configured without mandatory authentication.
func ValidateListenSecurity(addr string, authRequired bool) error {
	loop, err := IsLoopbackAddr(addr)
	if err != nil {
		return err
	}
	if !loop && !authRequired {
		return fmt.Errorf("refusing to listen on non-loopback address %q without authentication: set CYBEROS_AUTH_REQUIRED=true or bind to 127.0.0.1", addr)
	}
	return nil
}
