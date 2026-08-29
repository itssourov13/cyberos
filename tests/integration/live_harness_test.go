//go:build integration

package integration_test

import (
	"os"
	"testing"
)

// Live infrastructure gates. Do not claim PASS without real env.
func TestLivePostgres_Required(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" && os.Getenv("CYBEROS_DATABASE_URL") == "" {
		t.Skip("SKIPPED — DATABASE_URL / CYBEROS_DATABASE_URL unset")
	}
	t.Log("DSN present — use existing postgres integration tests for durability")
}

func TestLiveMTLS_Required(t *testing.T) {
	if os.Getenv("CYBEROS_TEST_CA") == "" {
		t.Skip("SKIPPED — CYBEROS_TEST_CA unset")
	}
	t.Log("CA present — run execution/mtls suite against live peer")
}
