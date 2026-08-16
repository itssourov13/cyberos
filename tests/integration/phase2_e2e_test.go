package integration_test

import (
	"context"
	"testing"
	"time"

	appscope "cyberos/internal/application/scope"
	"cyberos/internal/audit"
	"cyberos/internal/broker"
	"cyberos/internal/capability"
	"cyberos/internal/contracts"
	"cyberos/internal/controlplane"
	"cyberos/internal/dependency"
	domaincap "cyberos/internal/domain/capability"
	"cyberos/internal/domain/principal"
	domainscope "cyberos/internal/domain/scope"
	"cyberos/internal/domain/task"
	"cyberos/internal/events"
	"cyberos/internal/evidence"
	"cyberos/internal/execution/lifecycle"
	"cyberos/internal/execution/mock"
	"cyberos/internal/execution/sandbox"
	"cyberos/internal/findings"
	"cyberos/internal/identity"
	"cyberos/internal/provenance"
	"cyberos/internal/scheduler"
	"cyberos/internal/taskengine"
)

// Phase 2 full path:
// Task → Schedule → (deps) → Promote → Control Plane → Grant →
// Broker → Sandbox → Mock → Lifecycle → Evidence → Finding → Provenance → Audit
func TestPhase2_FullLifecycleIntegration(t *testing.T) {
	ctx := context.Background()
	pr, sr, tr := &memP{}, &memS{}, &memT{}

	// Identity + scope
	p, _ := principal.NewHuman("user-1", "alice")
	_ = pr.Create(ctx, p)
	sc, _ := domainscope.New("scope-1", "proj-1", "Main", contracts.ScopeStatusActive)
	_ = sr.Create(ctx, sc)

	idSvc := identity.NewService(pr)
	scSvc := appscope.NewService(sr)
	capReg := capability.NewRegistry(nil)
	cap, _ := domaincap.New("recon.domain.enumerate", "Enum", contracts.CapabilityTypeRecon)
	_ = capReg.Register(ctx, cap)

	eng := taskengine.NewEngine(tr, idSvc, scSvc, capReg)
	cp := controlplane.NewService(idSvc, scSvc)
	issuer := broker.NewIssuer(5 * time.Minute)
	brk := broker.New(sandbox.New(mock.New()))
	evStore := events.NewStore()
	auditSvc := audit.NewService(evStore)
	evSvc := evidence.NewService()
	findSvc := findings.NewService(evSvc)
	provSvc := provenance.NewService()
	life := lifecycle.New(brk,
		lifecycle.WithEvidence(evSvc),
		lifecycle.WithFindings(findSvc),
		lifecycle.WithProvenance(provSvc),
		lifecycle.WithAudit(auditSvc),
		lifecycle.WithResourcePolicy(contracts.DefaultResourcePolicy()),
	)

	// Dependency: task-dep must succeed conceptually; for schedule we only need dep edge validation
	depTask, err := eng.Create(ctx, taskengine.CreateRequest{
		ID: "task-dep", ProjectID: "proj-1", ScopeID: "scope-1",
		PrincipalID: "user-1", CapabilityID: "recon.domain.enumerate",
		Type: contracts.TaskTypeRecon, Title: "Dependency",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Mark dependency succeeded in store for AssertRunnable
	depTask.State = contracts.TaskStateSucceeded
	_ = tr.Update(ctx, depTask)

	// Main task
	main, err := eng.Create(ctx, taskengine.CreateRequest{
		ID: "task-main", ProjectID: "proj-1", ScopeID: "scope-1",
		PrincipalID: "user-1", CapabilityID: "recon.domain.enumerate",
		Type: contracts.TaskTypeRecon, Title: "Main",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Dependencies
	lookup := &taskLookup{tr: tr}
	graph := dependency.New(lookup)
	if err := graph.ValidateEdges("task-main", []contracts.TaskID{"task-dep"}); err != nil {
		t.Fatal(err)
	}
	main, err = eng.SetDependencies(ctx, "task-main", []contracts.TaskID{"task-dep"})
	if err != nil {
		t.Fatal(err)
	}
	if err := graph.AssertRunnable(main); err != nil {
		t.Fatalf("deps should be satisfied: %v", err)
	}

	// Retry + timeout metadata
	_, err = eng.ApplyRetryPolicy(ctx, "task-main", 3, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = eng.SetTimeoutDeadline(ctx, "task-main", 30*time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Schedule → Tick → PENDING
	store := scheduler.NewMemoryStore()
	// sync main into scheduler store
	mt, _ := eng.Get(ctx, "task-main")
	store.Put(mt)
	sch := scheduler.New(store)
	now := time.Now().UTC()
	sch.Now = func() time.Time { return now }
	if _, err := sch.Schedule(scheduler.ScheduleRequest{
		TaskID: "task-main", NotBefore: now.Add(-time.Second), Priority: 5,
	}); err != nil {
		t.Fatal(err)
	}
	promoted, err := sch.Tick()
	if err != nil || len(promoted) != 1 {
		t.Fatalf("promote: %v len=%d", err, len(promoted))
	}
	if promoted[0].State != contracts.TaskStatePending {
		t.Fatalf("state=%s", promoted[0].State)
	}
	// Reflect PENDING into engine store
	_ = tr.Update(ctx, promoted[0])

	// Control Plane → Grant
	corr := contracts.CorrelationID("phase2-e2e-1")
	evalReq := controlplane.EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1",
		TaskID: "task-main", CapabilityID: "recon.domain.enumerate",
		CorrelationID: corr,
	}
	d, err := cp.Evaluate(ctx, evalReq)
	if err != nil || !d.Allowed() {
		t.Fatalf("evaluate: %v %s", err, d.Reason)
	}
	g, _, err := issuer.IssueAfterEvaluate(ctx, cp, evalReq)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = auditSvc.RecordDecision(d)
	_, _ = auditSvc.RecordGrantIssued(g.ExecutionGrant)

	// Lifecycle execution (Broker → Sandbox → Mock → Evidence → Finding)
	out, err := life.Run(lifecycle.RunRequest{
		Grant: g.ExecutionGrant, Parameters: map[string]any{"target": "example.test"},
		FindingTitle: "Phase2 observation", FindingSeverity: contracts.FindingSeverityInfo,
		CorrelationID: corr,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Record.State != contracts.ExecutionLifecycleSucceeded {
		t.Fatalf("lifecycle state=%s reason=%s", out.Record.State, out.Record.FailureReason)
	}
	if out.Evidence == nil || out.Finding == nil {
		t.Fatal("expected evidence and finding")
	}
	if out.Record.GrantID != g.ID || out.Record.TaskID != "task-main" {
		t.Fatalf("attribution: %+v", out.Record)
	}

	// Provenance + audit chain
	if _, err := provSvc.Get("evidence", string(*out.Evidence)); err != nil {
		t.Fatalf("evidence provenance: %v", err)
	}
	chain := auditSvc.ByCorrelation(corr)
	if len(chain) < 2 {
		t.Fatalf("audit chain too short: %d", len(chain))
	}
	evts := evStore.ByCorrelation(corr)
	if len(evts) < 2 {
		t.Fatalf("event chain too short: %d", len(evts))
	}
}

func TestPhase2_ScheduleDoesNotBypassAuthorization(t *testing.T) {
	// Promote to PENDING without CP still cannot get a grant without capability evaluation success
	// Denied path: missing capability
	pr, sr := &memP{}, &memS{}
	p, _ := principal.NewHuman("user-1", "a")
	_ = pr.Create(context.Background(), p)
	sc, _ := domainscope.New("scope-1", "proj-1", "M", contracts.ScopeStatusActive)
	_ = sr.Create(context.Background(), sc)
	cp := controlplane.NewService(identity.NewService(pr), appscope.NewService(sr))
	issuer := broker.NewIssuer(time.Minute)
	_, _, err := issuer.IssueAfterEvaluate(context.Background(), cp, controlplane.EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1",
		CorrelationID: "no-cap",
	})
	if err == nil {
		t.Fatal("grant without capability must fail")
	}
}

func TestPhase2_NoGrantNoLifecycleExecution(t *testing.T) {
	life := lifecycle.New(broker.New(sandbox.New(mock.New())), lifecycle.WithEvidence(evidence.NewService()))
	_, err := life.Run(lifecycle.RunRequest{})
	if err == nil {
		t.Fatal("empty grant must fail")
	}
}

func TestPhase2_DependencyBlocksRunnable(t *testing.T) {
	tr := &memT{}
	a, _ := task.New("a", "p", "s", "u", "c", contracts.TaskTypeCustom, "A")
	b, _ := task.New("b", "p", "s", "u", "c", contracts.TaskTypeCustom, "B")
	b.DependsOn = []contracts.TaskID{"a"}
	_ = tr.Create(context.Background(), a)
	_ = tr.Create(context.Background(), b)
	g := dependency.New(&taskLookup{tr: tr})
	if err := g.AssertRunnable(b); err == nil {
		t.Fatal("unsatisfied dep must block")
	}
}

type taskLookup struct{ tr *memT }

func (l *taskLookup) GetByID(id contracts.TaskID) (*task.Task, error) {
	return l.tr.GetByID(context.Background(), id)
}
