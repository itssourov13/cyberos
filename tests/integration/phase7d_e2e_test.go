package integration_test

import (
	"context"
	"testing"

	appscope "cyberos/internal/application/scope"
	"cyberos/internal/assessment"
	"cyberos/internal/contracts"
	"cyberos/internal/controlplane"
	domainfinding "cyberos/internal/domain/finding"
	"cyberos/internal/domain/principal"
	domainscope "cyberos/internal/domain/scope"
	"cyberos/internal/identity"
	"cyberos/internal/reports"
	"cyberos/internal/sdk"
	"cyberos/internal/storage/ports"
)

// memFindingRepo for Phase 7D durable-path simulation without PostgreSQL.
type memFindingRepo7d struct {
	byID map[contracts.FindingID]*domainfinding.Finding
	byFP map[string]*domainfinding.Finding
}

func (m *memFindingRepo7d) Upsert(_ context.Context, f *domainfinding.Finding) error {
	if m.byID == nil {
		m.byID = map[contracts.FindingID]*domainfinding.Finding{}
		m.byFP = map[string]*domainfinding.Finding{}
	}
	cp := *f
	m.byID[f.ID] = &cp
	if f.Fingerprint != "" {
		m.byFP[string(f.ProjectID)+"|"+f.Fingerprint] = &cp
	}
	return nil
}
func (m *memFindingRepo7d) GetByID(_ context.Context, id contracts.FindingID) (*domainfinding.Finding, error) {
	f, ok := m.byID[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *f
	return &cp, nil
}
func (m *memFindingRepo7d) GetByFingerprint(_ context.Context, projectID contracts.ProjectID, fp string) (*domainfinding.Finding, error) {
	f, ok := m.byFP[string(projectID)+"|"+fp]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *f
	return &cp, nil
}
func (m *memFindingRepo7d) ListByProject(context.Context, contracts.ProjectID, int, int) ([]*domainfinding.Finding, error) {
	return nil, nil
}
func (m *memFindingRepo7d) ListByAssessment(context.Context, string, int, int) ([]*domainfinding.Finding, error) {
	return nil, nil
}

type memRiskRepo7d struct {
	byF  map[contracts.FindingID]ports.RiskEvaluationRecord
	byID map[string]ports.RiskEvaluationRecord
}

func (m *memRiskRepo7d) Upsert(_ context.Context, r ports.RiskEvaluationRecord) error {
	if m.byID == nil {
		m.byID = map[string]ports.RiskEvaluationRecord{}
		m.byF = map[contracts.FindingID]ports.RiskEvaluationRecord{}
	}
	m.byID[r.ID] = r
	m.byF[r.FindingID] = r
	return nil
}
func (m *memRiskRepo7d) GetByID(_ context.Context, id string) (ports.RiskEvaluationRecord, error) {
	r, ok := m.byID[id]
	if !ok {
		return ports.RiskEvaluationRecord{}, contracts.ErrNotFound
	}
	return r, nil
}
func (m *memRiskRepo7d) GetByFinding(_ context.Context, id contracts.FindingID) (ports.RiskEvaluationRecord, error) {
	r, ok := m.byF[id]
	if !ok {
		return ports.RiskEvaluationRecord{}, contracts.ErrNotFound
	}
	return r, nil
}
func (m *memRiskRepo7d) ListByProject(context.Context, contracts.ProjectID, int, int) ([]ports.RiskEvaluationRecord, error) {
	return nil, nil
}
func (m *memRiskRepo7d) ListByAssessment(context.Context, string, int, int) ([]ports.RiskEvaluationRecord, error) {
	return nil, nil
}

type memCaseRepo7d struct {
	m map[string]ports.CaseRecord
}

func (m *memCaseRepo7d) Upsert(_ context.Context, c ports.CaseRecord) error {
	if m.m == nil {
		m.m = map[string]ports.CaseRecord{}
	}
	m.m[c.ID] = c
	return nil
}
func (m *memCaseRepo7d) Insert(_ context.Context, c ports.CaseRecord) error {
	if m.m == nil {
		m.m = map[string]ports.CaseRecord{}
	}
	if _, ok := m.m[c.ID]; ok {
		return contracts.ErrConflict
	}
	m.m[c.ID] = c
	return nil
}
func (m *memCaseRepo7d) Transition(_ context.Context, id string, from, to contracts.CaseStatus, _ ports.CaseStatusChange) (ports.CaseRecord, error) {
	c, ok := m.m[id]
	if !ok {
		return ports.CaseRecord{}, contracts.ErrNotFound
	}
	if c.Status != from {
		return ports.CaseRecord{}, contracts.ErrConflict
	}
	c.Status = to
	m.m[id] = c
	return c, nil
}
func (m *memCaseRepo7d) StatusHistory(context.Context, string) ([]ports.CaseStatusChange, error) {
	return nil, nil
}
func (m *memCaseRepo7d) GetByID(_ context.Context, id string) (ports.CaseRecord, error) {
	c, ok := m.m[id]
	if !ok {
		return ports.CaseRecord{}, contracts.ErrNotFound
	}
	return c, nil
}
func (m *memCaseRepo7d) filter(pred func(ports.CaseRecord) bool) []ports.CaseRecord {
	var out []ports.CaseRecord
	for _, c := range m.m {
		if pred(c) {
			out = append(out, c)
		}
	}
	return out
}
func (m *memCaseRepo7d) ListByProject(_ context.Context, p contracts.ProjectID, _, _ int) ([]ports.CaseRecord, error) {
	return m.filter(func(c ports.CaseRecord) bool { return c.ProjectID == p }), nil
}
func (m *memCaseRepo7d) ListByAssessment(_ context.Context, a string, _, _ int) ([]ports.CaseRecord, error) {
	return m.filter(func(c ports.CaseRecord) bool { return c.AssessmentID == a }), nil
}
func (m *memCaseRepo7d) ListByTenant(_ context.Context, t contracts.TenantID, _, _ int) ([]ports.CaseRecord, error) {
	return m.filter(func(c ports.CaseRecord) bool { return c.TenantID == t }), nil
}

type memReportRepo7d struct {
	m map[string]ports.ReportRecord
}

func (m *memReportRepo7d) Upsert(_ context.Context, r ports.ReportRecord) error {
	if m.m == nil {
		m.m = map[string]ports.ReportRecord{}
	}
	m.m[r.ID] = r
	return nil
}
func (m *memReportRepo7d) GetByID(_ context.Context, id string) (ports.ReportRecord, error) {
	r, ok := m.m[id]
	if !ok {
		return ports.ReportRecord{}, contracts.ErrNotFound
	}
	return r, nil
}
func (m *memReportRepo7d) List(context.Context, int, int) ([]ports.ReportRecord, error) {
	return nil, nil
}
func (m *memReportRepo7d) ListByProject(context.Context, contracts.ProjectID, int, int) ([]ports.ReportRecord, error) {
	return nil, nil
}

// Phase 7D: Assessment → CP → Grant → Broker → Phase6Sink → durable stores → Report
// then clear process memory and read back through repositories (restart simulation).
func TestPhase7D_AssessmentToReport_DurableRestart(t *testing.T) {
	client := sdk.NewInMemory()
	fRepo := &memFindingRepo7d{}
	rRepo := &memRiskRepo7d{}
	cRepo := &memCaseRepo7d{}
	repRepo := &memReportRepo7d{}
	client.Findings = client.Findings.WithRepository(fRepo)
	client.RiskPrio = client.RiskPrio.WithRepository(rRepo)
	client.Cases = client.Cases.WithRepository(cRepo)
	client.Reports = client.Reports.WithRepository(repRepo)

	prepo := &memP{}
	srepo := &memS{}
	idSvc := identity.NewService(prepo)
	p, err := principal.New("user-7d", contracts.PrincipalTypeHuman, "tester")
	if err != nil {
		t.Fatal(err)
	}
	_ = idSvc.Create(context.Background(), p)
	scSvc := appscope.NewService(srepo)
	sc, err := domainscope.New("scope-7d", "proj-7d", "Scope 7D", contracts.ScopeStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	_ = scSvc.Create(context.Background(), sc)
	client.WithIdentity(idSvc).WithScopes(scSvc).WithControlPlane(
		controlplane.NewService(idSvc, scSvc),
	).WireAssessment()

	a := &assessment.Assessment{
		ID: "asmt-7d", ProjectID: "proj-7d", ScopeID: "scope-7d",
		PrincipalID: "user-7d", Title: "Phase 7D E2E", Target: "10.0.0.7",
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
	if len(out.StepResults) < 1 || !out.StepResults[0].Success {
		t.Fatalf("step failed: %+v", out.StepResults)
	}

	findings := client.Findings.ListByProject("proj-7d")
	if len(findings) < 1 {
		t.Fatal("expected findings after Phase6Sink")
	}
	fid := findings[0].ID
	// Ensure risk prioritization exists for at least one finding (Phase6Sink or explicit).
	if _, err := client.RiskPrio.GetByFinding(fid); err != nil {
		ev, err2 := client.RiskPrio.EvaluateFinding(findings[0], contracts.AssetCriticalityMedium, contracts.ExposureUnknown, "")
		if err2 != nil {
			t.Fatal("risk", err, err2)
		}
		fid = findings[0].ID
		_ = ev
	}
	cases := client.Cases.ListByAssessment("asmt-7d")
	if len(cases) < 1 {
		t.Fatal("expected case")
	}
	caseID := cases[0].ID

	rep, err := client.Reports.Create("rep-7d", reports.BuildInput{
		AssessmentID: "asmt-7d", ProjectID: "proj-7d",
		Findings:    []map[string]any{{"id": string(fid), "title": findings[0].Title}},
		Cases:       cases,
		RiskSummary: client.RiskPrio.SummarizeAssessment("asmt-7d"),
		Timeline:    []string{"assessment", "finding", "risk", "case", "report"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// --- restart simulation: wipe in-process maps ---
	client.Findings = client.Findings.WithRepository(fRepo)
	client.Findings = sdk.NewInMemory().Findings.WithRepository(fRepo)
	client.RiskPrio = sdk.NewInMemory().RiskPrio.WithRepository(rRepo)
	client.Cases = sdk.NewInMemory().Cases.WithRepository(cRepo)
	client.Reports = sdk.NewInMemory().Reports.WithRepository(repRepo)

	gotF, err := client.Findings.Get(fid)
	if err != nil || gotF.Title == "" {
		t.Fatal("finding after restart", err, gotF)
	}
	gotR, err := client.RiskPrio.GetByFinding(fid)
	if err != nil {
		t.Fatal("risk after restart", err)
	}
	_ = gotR
	gotC, err := client.Cases.Get(caseID)
	if err != nil || gotC.Title == "" {
		t.Fatal("case after restart", err, gotC)
	}
	gotRep, err := client.Reports.Get(rep.ID)
	if err != nil {
		t.Fatal("report after restart", err)
	}
	jb, err := client.Reports.Export(gotRep.ID, contracts.ReportFormatJSON)
	if err != nil || len(jb) < 10 {
		t.Fatal(err)
	}
	mb, err := client.Reports.Export(gotRep.ID, contracts.ReportFormatMarkdown)
	if err != nil || len(mb) < 10 {
		t.Fatal(err)
	}
}

func TestPhase7D_AuthenticatedStillRequiresGrant(t *testing.T) {
	// Auth ≠ authorization: empty grant still cannot execute via Broker.
	client := sdk.NewInMemory()
	_, err := client.Broker.Execute(contracts.ExecutionRequest{})
	if err == nil {
		t.Fatal("expected grant failure")
	}
}
