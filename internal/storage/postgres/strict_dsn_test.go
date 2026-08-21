//go:build integration

package postgres_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cyberos/internal/config"
	"cyberos/internal/contracts"
	"cyberos/internal/findings"
	"cyberos/internal/sdk"
	"cyberos/internal/storage/postgres"
)

// When CYBEROS_DATABASE_URL is explicitly set, PostgreSQL must be reachable.
// Unset DSN → SKIP (caller may still run other integration tests via setupDB default).
func TestPhase8D_StrictDSN_MustConnectWhenConfigured(t *testing.T) {
	dsn := os.Getenv("CYBEROS_DATABASE_URL")
	if dsn == "" {
		t.Skip("CYBEROS_DATABASE_URL unset — PG strict check skipped (not a pass for durability)")
	}
	db, err := postgres.Open(dsn)
	if err != nil {
		t.Fatalf("CYBEROS_DATABASE_URL set but PostgreSQL unreachable: %v", err)
	}
	_ = db.Close()
}

func TestPhase8D_ProductionRestartReadBack(t *testing.T) {
	dsn := os.Getenv("CYBEROS_DATABASE_URL")
	if dsn == "" {
		// Optional: try default only if open works; else skip without claiming verified.
		dsn = "postgres://cyberos:cyberos@localhost:5432/cyberos?sslmode=disable"
		if _, err := postgres.Open(dsn); err != nil {
			t.Skipf("PostgreSQL not available: %v", err)
		}
	}
	cfg := &config.Config{Env: "production", DatabaseURL: dsn, AuthRequired: true}
	mig := findMigDir(t)
	ctx := context.Background()
	prod, err := sdk.NewProduction(ctx, cfg, mig)
	if err != nil {
		if os.Getenv("CYBEROS_DATABASE_URL") != "" {
			t.Fatalf("production open failed with DSN set: %v", err)
		}
		t.Skipf("production open: %v", err)
	}
	defer func() { _ = prod.Close() }()

	f, err := prod.Client.Findings.Create(findings.CreateRequest{
		ID: "f-8d", TaskID: "t-8d", ProjectID: "p-8d", ScopeID: "s-8d",
		Severity: contracts.FindingSeverityLow, Title: "8d",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = prod.Close()

	prod2, err := sdk.NewProduction(ctx, cfg, mig)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = prod2.Close() }()
	got, err := prod2.Client.Findings.Get(f.ID)
	if err != nil || got.Title != "8d" {
		t.Fatal(err, got)
	}
	// Bootstrap hygiene: EnsureBootstrap should not create when keys exist
	n, err := prod2.Keys.CountActive(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, created, err := prod2.Auth.EnsureBootstrap(ctx, "", "")
	if err != nil || created {
		t.Fatalf("should not re-issue: created=%v n=%d err=%v", created, n, err)
	}
	_ = time.Now()
}

func findMigDir(t *testing.T) string {
	t.Helper()
	dir, _ := os.Getwd()
	for i := 0; i < 8; i++ {
		p := filepath.Join(dir, "migrations")
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("migrations not found")
	return ""
}
