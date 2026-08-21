package integration_test

import (
	"context"
	"testing"

	appscope "cyberos/internal/application/scope"
	"cyberos/internal/assessment"
	"cyberos/internal/contracts"
	"cyberos/internal/controlplane"
	"cyberos/internal/domain/principal"
	domainscope "cyberos/internal/domain/scope"
	"cyberos/internal/identity"
	"cyberos/internal/reports"
	"cyberos/internal/sdk"
)

func TestPhase6H_AssessmentToReportPipeline(t *testing.T) {
	client := sdk.NewInMemory()
	prepo := &memP{}
	srepo := &memS{}
	idSvc := identity.NewService(prepo)
	p, err := principal.New("user-6h", contracts.PrincipalTypeHuman, "tester")
	if err != nil {
		t.Fatal(err)
	}
	if err := idSvc.Create(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	scSvc := appscope.NewService(srepo)
	sc, err := domainscope.New("scope-6h", "proj-6h", "Scope 6H", contracts.ScopeStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	if err := scSvc.Create(context.Background(), sc); err != nil {
		t.Fatal(err)
	}
	client.WithIdentity(idSvc).WithScopes(scSvc).WithControlPlane(
		controlplane.NewService(idSvc, scSvc),
	).WireAssessment()

	a := &assessment.Assessment{
		ID: "asmt-6h", ProjectID: "proj-6h", ScopeID: "scope-6h",
		PrincipalID: "user-6h", Title: "Phase 6H E2E", Target: "10.0.0.5",
		State: contracts.AssessmentStatePlanned,
		Steps: []contracts.AssessmentStep{{
			ID: "step-1", CapabilityID: "assessment.stub.probe",
			Parameters: map[string]any{},
		}},
	}
	if err := client.Assessment.Save(a); err != nil {
		t.Fatal(err)
	}
	out, err := client.Assessment.Run(context.Background(), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if out == nil {
		t.Fatal("nil assessment")
	}

	assets := client.Assets.ListByProject("proj-6h")
	if len(assets) < 1 {
		t.Fatalf("expected asset, state=%s results=%+v", out.State, out.StepResults)
	}
	findings := client.Findings.ListByAssessment("asmt-6h")
	if len(findings) < 1 {
		findings = client.Findings.ListByProject("proj-6h")
	}
	if len(findings) < 1 {
		t.Fatalf("expected findings, results=%+v", out.StepResults)
	}
	caseList := client.Cases.ListByAssessment("asmt-6h")
	if len(caseList) < 1 {
		t.Fatal("expected case")
	}
	if _, err := client.RiskPrio.GetByFinding(findings[0].ID); err != nil {
		t.Fatalf("risk: %v", err)
	}

	rep, err := client.Reports.Create("rep-6h", reports.BuildInput{
		AssessmentID: "asmt-6h", ProjectID: "proj-6h",
		Findings:    []map[string]any{{"id": string(findings[0].ID), "title": findings[0].Title}},
		Assets:      []map[string]any{{"id": string(assets[0].ID), "value": assets[0].Value}},
		Cases:       caseList,
		RiskSummary: client.RiskPrio.SummarizeAssessment("asmt-6h"),
		Timeline:    []string{"assessment ran", "finding ingested", "case opened"},
	})
	if err != nil {
		t.Fatal(err)
	}
	jb, err := client.Reports.Export(rep.ID, contracts.ReportFormatJSON)
	if err != nil || len(jb) < 10 {
		t.Fatal(err, string(jb))
	}
	mb, err := client.Reports.Export(rep.ID, contracts.ReportFormatMarkdown)
	if err != nil || len(mb) < 10 {
		t.Fatal(err, string(mb))
	}
}
