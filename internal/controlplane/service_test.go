package controlplane

import (
	"context"
	"testing"
	"time"

	"cyberos/internal/application/scope"
	"cyberos/internal/contracts"
	"cyberos/internal/domain/principal"
	domainscope "cyberos/internal/domain/scope"
	"cyberos/internal/identity"
)

// --- in-memory repos for tests ---

type memPrincipalRepo struct {
	items map[contracts.PrincipalID]*principal.Principal
}

func (m *memPrincipalRepo) Create(_ context.Context, p *principal.Principal) error {
	if m.items == nil {
		m.items = map[contracts.PrincipalID]*principal.Principal{}
	}
	cp := *p
	m.items[p.ID] = &cp
	return nil
}
func (m *memPrincipalRepo) GetByID(_ context.Context, id contracts.PrincipalID) (*principal.Principal, error) {
	p, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *p
	return &cp, nil
}
func (m *memPrincipalRepo) List(context.Context, int, int) ([]*principal.Principal, error) {
	return nil, nil
}

type memScopeRepo struct {
	items map[contracts.ScopeID]*domainscope.Scope
}

func (m *memScopeRepo) Create(_ context.Context, s *domainscope.Scope) error {
	if m.items == nil {
		m.items = map[contracts.ScopeID]*domainscope.Scope{}
	}
	cp := *s
	m.items[s.ID] = &cp
	return nil
}
func (m *memScopeRepo) GetByID(_ context.Context, id contracts.ScopeID) (*domainscope.Scope, error) {
	s, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *s
	return &cp, nil
}
func (m *memScopeRepo) Update(context.Context, *domainscope.Scope) error { return nil }
func (m *memScopeRepo) ListByProject(context.Context, contracts.ProjectID, int, int) ([]*domainscope.Scope, error) {
	return nil, nil
}

func setupCP(t *testing.T) (*Service, *memPrincipalRepo, *memScopeRepo) {
	t.Helper()
	pr := &memPrincipalRepo{}
	sr := &memScopeRepo{}
	idSvc := identity.NewService(pr)
	scSvc := scope.NewService(sr)
	cp := NewService(idSvc, scSvc)
	return cp, pr, sr
}

func seedActive(t *testing.T, pr *memPrincipalRepo, sr *memScopeRepo) {
	t.Helper()
	p, _ := principal.NewHuman("user-1", "alice")
	_ = pr.Create(context.Background(), p)
	sc, _ := domainscope.New("scope-1", "proj-1", "Main", contracts.ScopeStatusActive)
	_ = sr.Create(context.Background(), sc)
}

func TestEvaluateAllowed(t *testing.T) {
	cp, pr, sr := setupCP(t)
	seedActive(t, pr, sr)

	d, err := cp.Evaluate(context.Background(), EvaluateRequest{
		PrincipalID:  "user-1",
		ProjectID:    "proj-1",
		ScopeID:      "scope-1",
		CapabilityID: "recon.domain.enumerate",
		TaskID:       "task-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !d.Allowed() {
		t.Fatalf("expected ALLOWED, got %s: %s", d.Outcome, d.Reason)
	}
	if d.Reason == "" || d.ID == "" {
		t.Error("decision must have id and reason")
	}
	if d.Policy == nil || d.Permission == nil || d.Risk == nil || d.Approval == nil {
		t.Error("allowed decision must include evaluation sub-results")
	}
}

func TestEvaluateMissingScope(t *testing.T) {
	cp, pr, sr := setupCP(t)
	seedActive(t, pr, sr)

	d, err := cp.Evaluate(context.Background(), EvaluateRequest{
		PrincipalID:  "user-1",
		CapabilityID: "cap-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.Allowed() {
		t.Fatal("missing scope must deny")
	}
	if d.Outcome != contracts.DecisionDenied {
		t.Errorf("outcome = %s", d.Outcome)
	}
}

func TestEvaluateExpiredScope(t *testing.T) {
	cp, pr, sr := setupCP(t)
	p, _ := principal.NewHuman("user-1", "alice")
	_ = pr.Create(context.Background(), p)
	sc, _ := domainscope.New("scope-exp", "proj-1", "Exp", contracts.ScopeStatusActive)
	past := time.Now().UTC().Add(-time.Hour)
	sc.ExpiresAt = &past
	_ = sr.Create(context.Background(), sc)

	d, _ := cp.Evaluate(context.Background(), EvaluateRequest{
		PrincipalID:  "user-1",
		ProjectID:    "proj-1",
		ScopeID:      "scope-exp",
		CapabilityID: "cap-1",
	})
	if d.Allowed() {
		t.Fatal("expired scope must deny")
	}
}

func TestEvaluateInactivePrincipal(t *testing.T) {
	cp, pr, sr := setupCP(t)
	p, _ := principal.NewHuman("user-inactive", "bob")
	p.Active = false
	_ = pr.Create(context.Background(), p)
	sc, _ := domainscope.New("scope-1", "proj-1", "Main", contracts.ScopeStatusActive)
	_ = sr.Create(context.Background(), sc)

	d, _ := cp.Evaluate(context.Background(), EvaluateRequest{
		PrincipalID:  "user-inactive",
		ProjectID:    "proj-1",
		ScopeID:      "scope-1",
		CapabilityID: "cap-1",
	})
	if d.Allowed() {
		t.Fatal("inactive principal must deny")
	}
}

func TestEvaluateOutOfScopeAsset(t *testing.T) {
	cp, pr, sr := setupCP(t)
	seedActive(t, pr, sr)
	sc, _ := sr.GetByID(context.Background(), "scope-1")
	sc.Constraints.AllowedAssetIDs = []contracts.AssetID{"only-this"}
	_ = sr.Create(context.Background(), sc) // overwrite

	// re-seed properly
	sr.items["scope-1"] = sc

	d, _ := cp.Evaluate(context.Background(), EvaluateRequest{
		PrincipalID:  "user-1",
		ProjectID:    "proj-1",
		ScopeID:      "scope-1",
		CapabilityID: "cap-1",
		AssetID:      "other-asset",
	})
	if d.Allowed() {
		t.Fatal("out-of-scope asset must deny")
	}
}

func TestEvaluateCriticalRiskDenied(t *testing.T) {
	cp, pr, sr := setupCP(t)
	seedActive(t, pr, sr)

	d, _ := cp.Evaluate(context.Background(), EvaluateRequest{
		PrincipalID:   "user-1",
		ProjectID:     "proj-1",
		ScopeID:       "scope-1",
		CapabilityID:  "cap-1",
		RiskLevelHint: contracts.RiskLevelCritical,
	})
	if d.Allowed() {
		t.Fatal("critical risk must deny by default")
	}
	if d.Risk == nil || d.Risk.Accepted {
		t.Error("risk result should show rejection")
	}
}

func TestEvaluateApprovalRequired(t *testing.T) {
	cp, pr, sr := setupCP(t)
	seedActive(t, pr, sr)

	d, _ := cp.Evaluate(context.Background(), EvaluateRequest{
		PrincipalID:     "user-1",
		ProjectID:       "proj-1",
		ScopeID:         "scope-1",
		CapabilityID:    "cap-1",
		RequireApproval: true,
	})
	if d.Allowed() {
		t.Fatal("explicit approval requirement must deny until approved")
	}
	if d.Approval == nil || d.Approval.Status != contracts.ApprovalStatusPending {
		t.Errorf("approval status = %v", d.Approval)
	}
}

func TestEvaluateMissingCapability(t *testing.T) {
	cp, pr, sr := setupCP(t)
	seedActive(t, pr, sr)

	d, _ := cp.Evaluate(context.Background(), EvaluateRequest{
		PrincipalID: "user-1",
		ProjectID:   "proj-1",
		ScopeID:     "scope-1",
	})
	if d.Allowed() {
		t.Fatal("missing capability must deny")
	}
}

func TestDecisionNeverBareBool(t *testing.T) {
	cp, pr, sr := setupCP(t)
	seedActive(t, pr, sr)

	d, _ := cp.Evaluate(context.Background(), EvaluateRequest{
		PrincipalID:  "user-1",
		ProjectID:    "proj-1",
		ScopeID:      "scope-1",
		CapabilityID: "cap-1",
	})
	// Structured fields must be present
	if d.ID == "" || d.Reason == "" || d.Timestamp.IsZero() {
		t.Error("decision missing required structured fields")
	}
	if !d.Outcome.Valid() {
		t.Error("outcome must be valid enum")
	}
}
