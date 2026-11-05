package sdk

import (
	"testing"

	"cyberos/internal/config"
)

func TestNonDevelopmentEnvironmentsAlwaysUseProductionStorage(t *testing.T) {
	for _, env := range []string{"staging", "preview", "production"} {
		cfg := &config.Config{Env: env}
		if !UseProduction(cfg) {
			t.Fatalf("environment %q must never fall back to in-memory storage", env)
		}
	}
	if UseProduction(&config.Config{Env: "development", AuthRequired: true}) {
		t.Fatal("development remains an explicit local-only mode")
	}
}
