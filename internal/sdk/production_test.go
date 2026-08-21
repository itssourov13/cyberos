package sdk

import (
	"context"
	"testing"

	"cyberos/internal/config"
)

func TestNewProduction_RequiresDSN(t *testing.T) {
	_, err := NewProduction(context.Background(), &config.Config{Env: "production"}, "migrations")
	if err == nil {
		t.Fatal("expected error without database URL")
	}
}

func TestNewProduction_FailClosedBadDSN(t *testing.T) {
	cfg := &config.Config{
		Env:         "production",
		DatabaseURL: "postgres://invalid:invalid@127.0.0.1:1/none?sslmode=disable&connect_timeout=1",
	}
	_, err := NewProduction(context.Background(), cfg, "migrations")
	if err == nil {
		t.Fatal("expected open failure")
	}
}

func TestUseProduction(t *testing.T) {
	if UseProduction(&config.Config{Env: "development"}) {
		t.Fatal("dev")
	}
	if !UseProduction(&config.Config{Env: "production", DatabaseURL: "x"}) {
		t.Fatal("prod")
	}
}

func TestNewInMemory_StillWorks(t *testing.T) {
	c := NewInMemory()
	if c == nil || c.Broker == nil || c.Findings == nil {
		t.Fatal("inmemory")
	}
}
