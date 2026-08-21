//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"cyberos/internal/auth"
	"cyberos/internal/contracts"
	"cyberos/internal/storage/ports"
	"cyberos/internal/storage/postgres"
)

func TestPhase7B_CasesReportsKeysDurability(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	cases := postgres.NewCaseRepo(db)
	reports := postgres.NewReportRepo(db)
	keys := postgres.NewAPIKeyStore(db)

	now := time.Now().UTC()
	if err := cases.Upsert(ctx, ports.CaseRecord{
		ID: "case-7b", TenantID: "ten-a", ProjectID: "proj-7b", AssessmentID: "asmt",
		Title: "Case", Status: contracts.CaseStatusOpen, Priority: contracts.RiskPriorityHigh,
		CreatedAt: now, UpdatedAt: now, Metadata: map[string]string{},
	}); err != nil {
		t.Fatal(err)
	}
	if err := reports.Upsert(ctx, ports.ReportRecord{
		ID: "rep-7b", ProjectID: "proj-7b", AssessmentID: "asmt", GeneratedAt: now,
		SchemaVersion: "cyberos-report-v1", Status: "READY", SnapshotJSON: []byte(`{"schema_version":"cyberos-report-v1"}`),
	}); err != nil {
		t.Fatal(err)
	}
	if err := keys.Create(ctx, auth.APIKey{
		ID: "key-7b", Prefix: "cyb7b", Hash: "abc", PrincipalID: "user-1", TenantID: "ten-a",
		Active: true, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	_ = db.Close()
	db2 := setupDB(t)
	cases2 := postgres.NewCaseRepo(db2)
	reports2 := postgres.NewReportRepo(db2)
	keys2 := postgres.NewAPIKeyStore(db2)

	c, err := cases2.GetByID(ctx, "case-7b")
	if err != nil || c.Title != "Case" || c.TenantID != "ten-a" {
		t.Fatal(err, c)
	}
	// tenant isolation: list ten-b should not include ten-a if filter used
	listB, err := cases2.ListByTenant(ctx, "ten-b", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range listB {
		if x.ID == "case-7b" {
			t.Fatal("tenant leak")
		}
	}
	r, err := reports2.GetByID(ctx, "rep-7b")
	if err != nil || r.SchemaVersion != "cyberos-report-v1" {
		t.Fatal(err, r)
	}
	k, err := keys2.LookupByPrefix(ctx, "cyb7b")
	if err != nil || !k.Active {
		t.Fatal(err, k)
	}
	if err := keys2.Revoke(ctx, "key-7b"); err != nil {
		t.Fatal(err)
	}
	k2, err := keys2.LookupByPrefix(ctx, "cyb7b")
	if err != nil || k2.Active {
		t.Fatal("expected revoked", err, k2)
	}
}
