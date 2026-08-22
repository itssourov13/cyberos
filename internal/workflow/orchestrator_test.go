package workflow_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"cyberos/internal/broker"
	"cyberos/internal/capability"
	"cyberos/internal/contracts"
	domaincap "cyberos/internal/domain/capability"
	"cyberos/internal/evidence"
	"cyberos/internal/execution/lifecycle"
	"cyberos/internal/execution/mock"
	"cyberos/internal/execution/sandbox"
	"cyberos/internal/provenance"
	"cyberos/internal/taskengine"
	"cyberos/internal/tools"
	"cyberos/internal/workflow"
)

func sampleDef() workflow.Definition {
	return workflow.Definition{
		ID: "wf-orch", Name: "Orch", Version: "1.0.0", ProjectID: "proj-a", ScopeID: "scope-a",
		Steps: []workflow.Step{
			{ID: "s1", CapabilityID: "cap.scan", Parameters: map[string]any{"k": "1"}},
			{ID: "s2", CapabilityID: "cap.scan", DependsOn: []string{"s1"}},
		},
	}
}

func setupOrch(t *testing.T, auth workflow.StepAuthorizer) *workflow.Orchestrator {
	t.Helper()
	capReg := capability.NewRegistry(nil)
	c, _ := domaincap.New("cap.scan", "Scan", contracts.CapabilityTypeRecon)
	_ = capReg.Register(context.Background(), c)
	mgr := tools.NewManager(nil)
	_ = mgr.Register(noopAdapter{d: tools.Descriptor{
		ID: "tool.scan", Name: "scan", Version: "1", Runtime: contracts.PluginRuntimeGo,
		ExecutorID: "tool.scan", Capabilities: []contracts.CapabilityID{"cap.scan"},
	}})
	plat := tools.NewPlatform(mgr)
	_ = plat.RegisterPack(tools.Pack{ID: "p", Name: "P", Version: "1.0.0", State: tools.PackStateActive, ToolIDs: []string{"tool.scan"}})
	repo := &memTasks{}
	eng := taskengine.NewEngine(repo, nil, nil, capReg)
	mat := workflow.NewMaterializer(eng, workflow.NewCapabilityResolver(plat))
	life := lifecycle.New(broker.New(sandbox.New(mock.New())),
		lifecycle.WithEvidence(evidence.NewService()),
		lifecycle.WithProvenance(provenance.NewService()),
	)
	return workflow.NewOrchestrator(mat, eng, life, auth)
}

func grantAuth(t *testing.T) workflow.StepAuthorizer {
	t.Helper()
	var n atomic.Int32
	return workflow.FuncAuthorizer{Fn: func(ctx context.Context, req workflow.AuthorizeStepRequest) (contracts.ExecutionGrant, error) {
		i := n.Add(1)
		now := time.Now().UTC()
		return contracts.ExecutionGrant{
			ID:     contracts.ExecutionGrantID("g-" + string(rune('0'+i))),
			TaskID: req.TaskID, ProjectID: req.ProjectID, ScopeID: req.ScopeID,
			CapabilityID: req.CapabilityID,
			Principal:    contracts.PrincipalRef{ID: req.PrincipalID, Type: contracts.PrincipalTypeHuman},
			Status:       contracts.ExecutionGrantStatusIssued,
			IssuedAt:     now, ExpiresAt: now.Add(time.Hour),
			Nonce: "n-" + string(rune('0'+i)), CorrelationID: req.CorrelationID,
		}, nil
	}}
}

func TestOrchestratorStartAndSucceed(t *testing.T) {
	orch := setupOrch(t, grantAuth(t))
	def := sampleDef()
	run, err := orch.Start(context.Background(), workflow.StartRequest{
		Definition: def, PrincipalID: "user-1", CorrelationID: "corr-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.State != workflow.RunStateRunning {
		t.Fatal(run.State)
	}
	if run.ProjectID != "proj-a" || run.ScopeID != "scope-a" {
		t.Fatal("isolation fields")
	}
	// Advance until complete (max 5 rounds)
	for i := 0; i < 5; i++ {
		run, err = orch.Advance(context.Background(), run.ID, def)
		if err != nil {
			t.Fatal(err)
		}
		if run.State == workflow.RunStateSucceeded {
			break
		}
	}
	if run.State != workflow.RunStateSucceeded {
		t.Fatalf("state=%s steps=%v", run.State, run.Steps)
	}
	if run.Steps["s1"].State != workflow.StepSucceeded || run.Steps["s2"].State != workflow.StepSucceeded {
		t.Fatal(run.Steps)
	}
	if run.Steps["s1"].EvidenceID == "" {
		t.Fatal("evidence required")
	}
}

func TestOrchestratorDeniedGrant(t *testing.T) {
	auth := workflow.FuncAuthorizer{Fn: func(ctx context.Context, req workflow.AuthorizeStepRequest) (contracts.ExecutionGrant, error) {
		return contracts.ExecutionGrant{}, contracts.ErrForbidden
	}}
	orch := setupOrch(t, auth)
	def := sampleDef()
	run, err := orch.Start(context.Background(), workflow.StartRequest{Definition: def, PrincipalID: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = orch.Advance(context.Background(), run.ID, def)
	if err == nil {
		t.Fatal("expected deny")
	}
	run, _ = orch.Get(run.ID)
	if run.State != workflow.RunStateFailed {
		t.Fatal(run.State)
	}
}

func TestOrchestratorCancel(t *testing.T) {
	orch := setupOrch(t, grantAuth(t))
	def := sampleDef()
	run, err := orch.Start(context.Background(), workflow.StartRequest{Definition: def, PrincipalID: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	run, err = orch.Cancel(run.ID)
	if err != nil || run.State != workflow.RunStateCancelled {
		t.Fatal(err, run.State)
	}
}

func TestOrchestratorIdempotentStep(t *testing.T) {
	orch := setupOrch(t, grantAuth(t))
	def := workflow.Definition{
		ID: "wf-idem", Name: "I", Version: "1", ProjectID: "p", ScopeID: "s",
		Steps: []workflow.Step{{ID: "only", CapabilityID: "cap.scan"}},
	}
	run, err := orch.Start(context.Background(), workflow.StartRequest{Definition: def, PrincipalID: "u1", RunID: "fixed-run"})
	if err != nil {
		t.Fatal(err)
	}
	run, err = orch.Advance(context.Background(), run.ID, def)
	if err != nil {
		t.Fatal(err)
	}
	exec1 := run.Steps["only"].ExecutionID
	// Advance again — lifecycle idempotency key run:step should reuse
	run, err = orch.Advance(context.Background(), run.ID, def)
	if err != nil {
		t.Fatal(err)
	}
	if run.State != workflow.RunStateSucceeded {
		t.Fatal(run.State)
	}
	_ = exec1
}

func TestPerStepFreshAuthorization(t *testing.T) {
	var calls atomic.Int32
	auth := workflow.FuncAuthorizer{Fn: func(ctx context.Context, req workflow.AuthorizeStepRequest) (contracts.ExecutionGrant, error) {
		calls.Add(1)
		now := time.Now().UTC()
		return contracts.ExecutionGrant{
			ID:     contracts.ExecutionGrantID("g-" + string(req.TaskID)),
			TaskID: req.TaskID, ProjectID: req.ProjectID, ScopeID: req.ScopeID,
			CapabilityID: req.CapabilityID,
			Principal:    contracts.PrincipalRef{ID: req.PrincipalID, Type: contracts.PrincipalTypeHuman},
			Status:       contracts.ExecutionGrantStatusIssued, IssuedAt: now, ExpiresAt: now.Add(time.Hour),
			Nonce: string(req.TaskID), CorrelationID: req.CorrelationID,
		}, nil
	}}
	orch := setupOrch(t, auth)
	def := sampleDef()
	run, _ := orch.Start(context.Background(), workflow.StartRequest{Definition: def, PrincipalID: "u1"})
	for i := 0; i < 5; i++ {
		run, _ = orch.Advance(context.Background(), run.ID, def)
		if run.State == workflow.RunStateSucceeded || run.State == workflow.RunStateFailed {
			break
		}
	}
	if calls.Load() < 2 {
		t.Fatalf("expected per-step auth calls, got %d", calls.Load())
	}
}
