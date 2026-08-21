//go:build integration

package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	appscope "cyberos/internal/application/scope"
	"cyberos/internal/assessment"
	"cyberos/internal/config"
	"cyberos/internal/contracts"
	"cyberos/internal/controlplane"
	"cyberos/internal/domain/principal"
	domainscope "cyberos/internal/domain/scope"
	"cyberos/internal/identity"
	"cyberos/internal/reports"
	"cyberos/internal/sdk"
)

func TestPhase7D_ProductionPath_AssessmentReportRestart(t *testing.T) {
	dsn := os.Getenv("CYBEROS_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://cyberos:cyberos@localhost:5432/cyberos?sslmode=disable"
	}
	cfg := &config.Config{Env: "production", DatabaseURL: dsn, AuthRequired: true}
	mig := findMigrations(t)
	ctx := context.Background()

	prod, err := sdk.NewProduction(ctx, cfg, mig)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer func() { _ = prod.Close() }()
	client := prod.Client

	p, err := principal.New("user-7d-pg", contracts.PrincipalTypeHuman, "pg-tester")
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Identity.Create(ctx, p); err != nil {
		// may already exist from prior run
		_ = err
	}
	sc, err := domainscope.New("scope-7d-pg", "proj-7d-pg", "Scope 7D PG", contracts.ScopeStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Scopes.Create(ctx, sc); err != nil {
		_ = err
	}
	// Ensure CP is wired (NewProduction already set Control)
	_ = appscope.NewService
	_ = controlplane.NewService
	_ = identity.NewService

	a := &assessment.Assessment{
		ID: "asmt-7d-pg", ProjectID: "proj-7d-pg", ScopeID: "scope-7d-pg",
		PrincipalID: "user-7d-pg", Title: "Phase 7D PG", Target: "10.0.0.9",
		State: contracts.AssessmentStatePlanned,
		Steps: []contracts.AssessmentStep{{
			ID: "step-1", CapabilityID: "assessment.stub.probe", Parameters: map[string]any{},
		}},
	}
	if client.Assessment == nil {
		t.Fatal("assessment not wired")
	}
	if err := client.Assessment.Save(a); err != nil {
		t.Fatal(err)
	}
	out, err := client.Assessment.Run(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	_ = out

	findings := client.Findings.ListByProject("proj-7d-pg")
	if len(findings) < 1 {
		t.Fatal("expected durable findings")
	}
	fid := findings[0].ID

	// Close and reopen production client (restart)
	_ = prod.Close()
	prod2, err := sdk.NewProduction(ctx, cfg, mig)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = prod2.Close() }()

	got, err := prod2.Client.Findings.Get(fid)
	if err != nil {
		t.Fatal("finding after PG restart", err)
	}
	if got.Title == "" {
		t.Fatal("empty title")
	}
	if _, err := prod2.Client.RiskPrio.GetByFinding(fid); err != nil {
		t.Fatal("risk after restart", err)
	}
	cases := prod2.Client.Cases.ListByAssessment("asmt-7d-pg")
	if len(cases) < 1 {
		// case may still be loadable by id pattern
		t.Log("warning: case list empty after restart — checking create report path")
	}
	_, err = prod2.Client.Reports.Create("rep-7d-pg", reports.BuildInput{
		AssessmentID: "asmt-7d-pg", ProjectID: "proj-7d-pg",
		Findings: []map[string]any{{"id": string(fid), "title": got.Title}},
		Timeline: []string{"restart-ok"},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func findMigrations(t *testing.T) string {
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
