package config

import (
	"testing"
)

func TestLoadProductionFailsClosedWithoutDatabaseURL(t *testing.T) {
	t.Setenv("CYBEROS_ENV", "production")
	t.Setenv("CYBEROS_DATABASE_URL", "")
	t.Setenv("CYBEROS_AUTH_REQUIRED", "false")
	t.Setenv("CYBEROS_COOKIE_SECURE", "")
	t.Setenv("CYBEROS_SESSION_BACKEND", "")
	t.Setenv("CYBEROS_GRANT_SIGNING_KEY", "")
	c := Load()
	if c.DatabaseURL != "" || c.DatabaseURLExplicit {
		t.Fatalf("production must not inherit a development DSN: %+v", c)
	}
	if !c.AuthRequired || !c.CookieSecure || c.SessionBackend != "postgres" {
		t.Fatalf("production security defaults not enforced: %+v", c)
	}
	if err := c.ValidateProduction(); err == nil {
		t.Fatal("expected missing database configuration to be rejected")
	}
}

func TestLoadStagingUsesDurableSecurityDefaults(t *testing.T) {
	t.Setenv("CYBEROS_ENV", "staging")
	t.Setenv("CYBEROS_DATABASE_URL", "")
	t.Setenv("CYBEROS_AUTH_REQUIRED", "false")
	t.Setenv("CYBEROS_COOKIE_SECURE", "")
	t.Setenv("CYBEROS_SESSION_BACKEND", "")
	c := Load()
	if c.DatabaseURL != "" || c.DatabaseURLExplicit {
		t.Fatalf("staging must not inherit the development DSN: %+v", c)
	}
	if !c.AuthRequired || !c.CookieSecure || c.SessionBackend != "postgres" {
		t.Fatalf("staging must use production security defaults: %+v", c)
	}
}

func TestValidateProductionRequiresVerifiedPostgresTLS(t *testing.T) {
	c := &Config{
		Env: "production", DatabaseURL: "postgres://user:pass@db.example/cyberos?sslmode=require",
		DatabaseURLExplicit: true, AuthRequired: true, CookieSecure: true,
		SessionBackend: "postgres", GrantSigningKey: "0123456789abcdef0123456789abcdef",
	}
	if err := c.ValidateProduction(); err == nil {
		t.Fatal("sslmode=require must not satisfy verified TLS")
	}
	c.DatabaseURL = "postgres://user:pass@db.example/cyberos?sslmode=verify-full"
	if err := c.ValidateProduction(); err != nil {
		t.Fatalf("valid production config rejected: %v", err)
	}
	c.AllowDemoExecutors = true
	if err := c.ValidateProduction(); err == nil {
		t.Fatal("demo executors must be rejected in production")
	}
}
