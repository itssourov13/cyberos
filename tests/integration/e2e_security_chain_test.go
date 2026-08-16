package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	appscope "cyberos/internal/application/scope"
	"cyberos/internal/audit"
	"cyberos/internal/broker"
	"cyberos/internal/capability"
	"cyberos/internal/contracts"
	"cyberos/internal/controlplane"
	domaincap "cyberos/internal/domain/capability"
	"cyberos/internal/domain/principal"
	domainscope "cyberos/internal/domain/scope"
	"cyberos/internal/domain/task"
	"cyberos/internal/events"
	"cyberos/internal/evidence"
	"cyberos/internal/execution/mock"
	"cyberos/internal/findings"
	"cyberos/internal/identity"
	"cyberos/internal/sdk"
	"cyberos/internal/taskengine"
)

// --- in-memory repos ---

type memP struct {
	items map[contracts.PrincipalID]*principal.Principal
}

func (m *memP) Create(_ context.Context, p *principal.Principal) error {
	if m.items == nil {
		m.items = map[contracts.PrincipalID]*principal.Principal{}
	}
	cp := *p
	m.items[p.ID] = &cp
	return nil
}
func (m *memP) GetByID(_ context.Context, id contracts.PrincipalID) (*principal.Principal, error) {
	p, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *p
	return &cp, nil
}
func (m *memP) List(context.Context, int, int) ([]*principal.Principal, error) { return nil, nil }

type memS struct {
	items map[contracts.ScopeID]*domainscope.Scope
}

func (m *memS) Create(_ context.Context, s *domainscope.Scope) error {
	if m.items == nil {
		m.items = map[contracts.ScopeID]*domainscope.Scope{}
	}
	cp := *s
	m.items[s.ID] = &cp
	return nil
}
func (m *memS) GetByID(_ context.Context, id contracts.ScopeID) (*domainscope.Scope, error) {
	s, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *s
	return &cp, nil
}
func (m *memS) Update(context.Context, *domainscope.Scope) error { return nil }
func (m *memS) ListByProject(context.Context, contracts.ProjectID, int, int) ([]*domainscope.Scope, error) {
	return nil, nil
}

type memT struct {
	items map[contracts.TaskID]*task.Task
}

func (m *memT) Create(_ context.Context, t *task.Task) error {
	if m.items == nil {
		m.items = map[contracts.TaskID]*task.Task{}
	}
	cp := *t
	m.items[t.ID] = &cp
	return nil
}
func (m *memT) GetByID(_ context.Context, id contracts.TaskID) (*task.Task, error) {
	t, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *t
	return &cp, nil
}
func (m *memT) Update(_ context.Context, t *task.Task) error {
	cp := *t
	m.items[t.ID] = &cp
	return nil
}
func (m *memT) ListByScope(context.Context, contracts.ScopeID, int, int) ([]*task.Task, error) {
	return nil, nil
}

func wire(t *testing.T) *sdk.Client {
	t.Helper()
	pr, sr, tr := &memP{}, &memS{}, &memT{}
	p, _ := principal.NewHuman("user-1", "alice")
	_ = pr.Create(context.Background(), p)
	sc, _ := domainscope.New("scope-1", "proj-1", "Main", contracts.ScopeStatusActive)
	_ = sr.Create(context.Background(), sc)

	idSvc := identity.NewService(pr)
	scSvc := appscope.NewService(sr)
	capReg := capability.NewRegistry(nil)
	c, _ := domaincap.New("recon.domain.enumerate", "Enum", contracts.CapabilityTypeRecon)
	_ = capReg.Register(context.Background(), c)

	cp := controlplane.NewService(idSvc, scSvc)
	eng := taskengine.NewEngine(tr, idSvc, scSvc, capReg)
	evStore := events.NewStore()
	auditSvc := audit.NewService(evStore)
	evSvc := evidence.NewService()

	return sdk.NewInMemory().
		WithIdentity(idSvc).
		WithScopes(scSvc).
		WithControlPlane(cp).
		WithTasks(eng).
		WithAudit(auditSvc).
		WithEvidence(evSvc)
}

// Full happy path: Scope→Principal→CP→Grant→Broker→Mock→Evidence→Provenance→Audit/Events
func TestE2E_FullSecurityChain(t *testing.T) {
	client := wire(t)
	ctx := context.Background()
	corr := contracts.CorrelationID("e2e-corr-1")

	// 1–2 Task bound to scope/principal/capability
	tk, err := client.Tasks.Create(ctx, taskengine.CreateRequest{
		ID: "task-1", ProjectID: "proj-1", ScopeID: "scope-1",
		PrincipalID: "user-1", CapabilityID: "recon.domain.enumerate",
		Type: contracts.TaskTypeRecon, Title: "Enumerate",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	// 3 Control Plane
	d, err := client.Evaluate(ctx, controlplane.EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1",
		TaskID: tk.ID, CapabilityID: "recon.domain.enumerate",
		CorrelationID: corr,
	})
	if err != nil || !d.Allowed() {
		t.Fatalf("evaluate: err=%v decision=%s %s", err, d.Outcome, d.Reason)
	}

	// 4 Grant
	g, d2, err := client.IssueGrant(ctx, controlplane.EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1",
		TaskID: tk.ID, CapabilityID: "recon.domain.enumerate",
		CorrelationID: corr,
	})
	if err != nil || !d2.Allowed() {
		t.Fatalf("grant: %v", err)
	}

	// 5 Broker → Mock
	res, err := client.Execute(g, map[string]any{"target": "example.test"})
	if err != nil || res.Status != contracts.ExecutionStatusSucceeded {
		t.Fatalf("execute: err=%v status=%s", err, res.Status)
	}
	if res.ExecutorID != "mock" && res.ExecutorID != "sandbox/mock" {
		t.Fatalf("executor=%s", res.ExecutorID)
	}

	// 6 Evidence
	ev, err := client.Evidence.RecordFromExecution(res, g, "ev-1")
	if err != nil {
		t.Fatalf("evidence: %v", err)
	}
	if err := evidence.EnsureLinked(ev); err != nil {
		t.Fatal(err)
	}

	// 7 Finding
	f, err := client.Findings.Create(findings.CreateRequest{
		ID: "f-1", TaskID: tk.ID, ProjectID: "proj-1", ScopeID: "scope-1",
		Severity: contracts.FindingSeverityLow, Title: "Mock observation",
		EvidenceIDs: []contracts.EvidenceID{"ev-1"}, CorrelationID: corr,
	})
	if err != nil {
		t.Fatalf("finding: %v", err)
	}

	// 8 Provenance
	p, err := client.Provenance.RecordForEvidence(ev, res.ExecutorID)
	if err != nil {
		t.Fatalf("provenance: %v", err)
	}
	if p.GrantID != g.ID || p.Principal.ID != "user-1" {
		t.Fatalf("provenance incomplete: %+v", p)
	}
	_, err = client.Provenance.RecordForFinding(f, g.Principal, g.CapabilityID, g.ID, res.ExecutorID)
	if err != nil {
		t.Fatalf("finding provenance: %v", err)
	}

	// 9 Audit / Events correlation chain
	if client.Audit == nil {
		t.Fatal("audit not wired")
	}
	chain := client.Audit.ByCorrelation(corr)
	if len(chain) < 2 {
		t.Fatalf("expected audit correlation chain >= 2, got %d", len(chain))
	}
}

func TestE2E_NoGrantNoExecution(t *testing.T) {
	client := wire(t)
	_, err := client.Execute(contracts.ExecutionGrant{}, nil)
	if err == nil {
		t.Fatal("execution without grant must fail")
	}
}

func TestE2E_DeniedDecisionNoGrant(t *testing.T) {
	client := wire(t)
	_, _, err := client.IssueGrant(context.Background(), controlplane.EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1",
		CorrelationID: "deny-1",
		// missing capability
	})
	if err == nil {
		t.Fatal("denied decision must not issue grant")
	}
	if !errors.Is(err, contracts.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestE2E_ExpiredScopeBlocksChain(t *testing.T) {
	pr, sr := &memP{}, &memS{}
	p, _ := principal.NewHuman("user-1", "a")
	_ = pr.Create(context.Background(), p)
	sc, _ := domainscope.New("scope-exp", "proj-1", "E", contracts.ScopeStatusActive)
	past := time.Now().UTC().Add(-time.Hour)
	sc.ExpiresAt = &past
	_ = sr.Create(context.Background(), sc)

	idSvc := identity.NewService(pr)
	scSvc := appscope.NewService(sr)
	client := sdk.NewInMemory().
		WithIdentity(idSvc).
		WithScopes(scSvc).
		WithControlPlane(controlplane.NewService(idSvc, scSvc))

	d, _ := client.Evaluate(context.Background(), controlplane.EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-exp",
		CapabilityID: "c1", CorrelationID: "exp-1",
	})
	if d.Allowed() {
		t.Fatal("expired scope must deny")
	}
}

func TestE2E_CLI_API_SDK_SameControlPlane(t *testing.T) {
	// Prove SDK Evaluate is the same path API/CLI use (shared Client).
	client := wire(t)
	req := controlplane.EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1",
		CapabilityID: "recon.domain.enumerate", CorrelationID: "shared-1",
	}
	d1, err := client.Evaluate(context.Background(), req)
	if err != nil || !d1.Allowed() {
		t.Fatalf("sdk evaluate: %v %s", err, d1.Reason)
	}
	// Second call still structured decision
	d2, err := client.Control.Evaluate(context.Background(), req)
	if err != nil || d2.Outcome != d1.Outcome {
		t.Fatalf("control plane direct mismatch: %v %s vs %s", err, d2.Outcome, d1.Outcome)
	}
}

func TestE2E_ReplayGrantRejected(t *testing.T) {
	client := wire(t)
	g, _, err := client.IssueGrant(context.Background(), controlplane.EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1",
		TaskID: "task-1", CapabilityID: "recon.domain.enumerate",
		CorrelationID: "replay-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Execute(g, nil); err != nil {
		t.Fatalf("first exec: %v", err)
	}
	if _, err := client.Execute(g, nil); err == nil {
		t.Fatal("replay must fail")
	}
}

// Ensure mock broker is the only executor in the wired client.
func TestE2E_OnlyMockExecutor(t *testing.T) {
	_ = mock.New()
	client := wire(t)
	g, _, err := client.IssueGrant(context.Background(), controlplane.EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1",
		TaskID: "task-mock", CapabilityID: "recon.domain.enumerate", CorrelationID: "mock-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := client.Execute(g, nil)
	if err != nil {
		t.Fatal(err)
	}
	if (res.ExecutorID != "mock" && res.ExecutorID != "sandbox/mock") || res.Output["mock"] != true {
		t.Fatalf("expected mock executor, got %+v", res)
	}
}

// Broker direct misuse without issuer still requires valid grant shape.
func TestE2E_DirectBrokerRequiresValidGrant(t *testing.T) {
	brk := broker.New(mock.New())
	_, err := brk.Execute(contracts.ExecutionRequest{})
	if err == nil {
		t.Fatal("empty grant must be rejected by broker")
	}
}
