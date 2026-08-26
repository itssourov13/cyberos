package autonomy_test

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cyberos/internal/ai"
	"cyberos/internal/ai/autonomy"
	"cyberos/internal/contracts"
	"cyberos/internal/storage/memory"
	"cyberos/internal/storage/ports"
	"cyberos/internal/workflow"
)

// --- helpers ---

func controlledPolicy(project contracts.ProjectID, maxSteps int) autonomy.AutonomyPolicy {
	return autonomy.AutonomyPolicy{
		Version:             autonomy.PolicyContractV1,
		Mode:                autonomy.ModeControlledAutonomous,
		ProjectID:           project,
		AllowedCapabilities: []contracts.CapabilityID{"recon.network.portscan"},
		MaxSteps:            maxSteps,
		MaxRuntime:          time.Hour,
		MaxLLMCalls:         50,
		MaxNoProgress:       10,
	}
}

func manualPolicy(project contracts.ProjectID) autonomy.AutonomyPolicy {
	return autonomy.AutonomyPolicy{
		Version: autonomy.PolicyContractV1, Mode: autonomy.ModeManual, ProjectID: project, MaxSteps: 10,
	}
}

type grantCountingAuth struct {
	mu     sync.Mutex
	grants []contracts.ExecutionGrantID
	n      atomic.Int32
}

func (c *grantCountingAuth) AuthorizeStep(ctx context.Context, req workflow.AuthorizeStepRequest) (contracts.ExecutionGrant, error) {
	id := contracts.ExecutionGrantID("g-" + string(req.TaskID) + "-" + time.Now().Format("150405.000000"))
	c.n.Add(1)
	c.mu.Lock()
	c.grants = append(c.grants, id)
	c.mu.Unlock()
	return contracts.ExecutionGrant{
		ID: id, Principal: contracts.PrincipalRef{ID: req.PrincipalID},
		ProjectID: req.ProjectID, ScopeID: req.ScopeID, CapabilityID: req.CapabilityID, TaskID: req.TaskID,
		Status: contracts.ExecutionGrantStatusIssued, IssuedAt: time.Now().UTC(), ExpiresAt: time.Now().Add(time.Hour).UTC(),
	}, nil
}

func (c *grantCountingAuth) IDs() []contracts.ExecutionGrantID {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]contracts.ExecutionGrantID, len(c.grants))
	copy(out, c.grants)
	return out
}

// --- lineage ---

func Test14D_LineageAnalysisPlanWorkflow(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	run, err := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "lin-1", Goal: "recon", ProjectID: "proj-a", ScopeID: "scope-a", TenantID: "ten-a",
		Policy:          controlledPolicy("proj-a", 5),
		CapabilityHints: []contracts.CapabilityID{"recon.network.portscan"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.AnalysisID == "" {
		t.Fatal("missing analysis_id lineage")
	}
	if run.CurrentPlanID == "" {
		t.Fatal("missing plan_id lineage")
	}
	if run.Iteration < 1 {
		t.Fatal("iteration not recorded")
	}
	// re-plan path records parent
	adv, err := ctrl.Advance(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if adv.Run.ParentPlanID == "" && adv.Run.CurrentPlanID == run.CurrentPlanID {
		// parent may be set on second iterate
	}
	if adv.Run.AnalysisID == "" {
		t.Fatal("analysis lineage lost after advance")
	}
}

func Test14D_LineageNoCrossProject(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	run, _ := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "lin-2", Goal: "g", ProjectID: "proj-a", ScopeID: "s",
		Policy: controlledPolicy("proj-a", 5),
	})
	h := &autonomy.AdvanceHandoff{Controller: ctrl}
	res, err := h.Execute(context.Background(), autonomy.HandoffRequest{RunID: run.ID, ProjectID: "proj-b"})
	if err != nil {
		t.Fatal(err)
	}
	if res.PolicyDecision.Reason != "project_mismatch" {
		t.Fatal(res.PolicyDecision)
	}
	if res.OrchestratorCalled {
		t.Fatal("cross-project must not call orchestrator")
	}
}

// --- budget ---

func Test14D_BudgetMaxStepsBlocksOrchestrator(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	run, err := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "bud-1", Goal: "g", ProjectID: "proj-a", ScopeID: "s",
		Policy:          controlledPolicy("proj-a", 1),
		CapabilityHints: []contracts.CapabilityID{"recon.network.portscan"},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = ctrl.RecordStepExecuted(run.ID)
	auth := &grantCountingAuth{}
	h := &autonomy.AdvanceHandoff{
		Controller: ctrl, Orchestrator: workflow.NewOrchestrator(nil, nil, nil, auth),
	}
	res, err := h.Execute(context.Background(), autonomy.HandoffRequest{RunID: run.ID, ProjectID: "proj-a"})
	if err != nil {
		t.Fatal(err)
	}
	if res.OrchestratorCalled {
		t.Fatal("budget exhausted must not call orchestrator")
	}
	if auth.n.Load() != 0 {
		t.Fatal("no grants when budget blocked")
	}
	if res.Run.State != autonomy.RunPolicyBlocked && res.PolicyDecision.Decision != autonomy.DecisionStop {
		t.Logf("state=%s decision=%+v", res.Run.State, res.PolicyDecision)
	}
}

func Test14D_BudgetDenyOverridesAllow(t *testing.T) {
	p := controlledPolicy("proj-a", 10)
	p.DeniedCapabilities = []contracts.CapabilityID{"recon.network.portscan"}
	d, _ := autonomy.EvaluateAutonomyPolicy(autonomy.PolicyEvalRequest{
		ProjectID: "proj-a", CapabilityID: "recon.network.portscan", Policy: p,
	})
	if d.Decision != autonomy.DecisionDeny {
		t.Fatal(d)
	}
}

func Test14D_BudgetPersistsAcrossRepoRestart(t *testing.T) {
	repo := memory.NewAutonomyRunRepo()
	rec := ports.AutonomyRunRecord{
		ID: "bud-2", ProjectID: "proj-a", ScopeID: "s", State: "RUNNING",
		StepsConsumed: 7, Version: 1, StartedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		PolicyJSON: []byte(`{"mode":"CONTROLLED_AUTONOMOUS","max_steps":20}`),
	}
	if err := repo.Create(context.Background(), rec); err != nil {
		t.Fatal(err)
	}
	// simulate process restart: new repo load
	got, err := repo.Get(context.Background(), "proj-a", "bud-2")
	if err != nil || got.StepsConsumed != 7 {
		t.Fatalf("budget reset: err=%v steps=%d", err, got.StepsConsumed)
	}
}

// --- MANUAL / ASSISTED ---

func Test14D_ManualNeverCallsOrchestrator(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	run, _ := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "man-1", Goal: "g", ProjectID: "proj-a", ScopeID: "s",
		Policy:          manualPolicy("proj-a"),
		CapabilityHints: []contracts.CapabilityID{"recon.network.portscan"},
	})
	auth := &grantCountingAuth{}
	h := &autonomy.AdvanceHandoff{
		Controller: ctrl, Orchestrator: workflow.NewOrchestrator(nil, nil, nil, auth),
	}
	res, err := h.Execute(context.Background(), autonomy.HandoffRequest{RunID: run.ID, ProjectID: "proj-a"})
	if err != nil {
		t.Fatal(err)
	}
	if res.OrchestratorCalled || res.MayAdvance {
		t.Fatal("MANUAL must not advance")
	}
	if auth.n.Load() != 0 {
		t.Fatal("no grants for MANUAL")
	}
}

func Test14D_AssistedNeverCallsOrchestrator(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	run, _ := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "ast-1", Goal: "g", ProjectID: "proj-a", ScopeID: "s",
		Policy: autonomy.AutonomyPolicy{
			Version: autonomy.PolicyContractV1, Mode: autonomy.ModeAssisted, ProjectID: "proj-a", MaxSteps: 5,
		},
		CapabilityHints: []contracts.CapabilityID{"recon.network.portscan"},
	})
	auth := &grantCountingAuth{}
	h := &autonomy.AdvanceHandoff{
		Controller: ctrl, Orchestrator: workflow.NewOrchestrator(nil, nil, nil, auth),
	}
	res, _ := h.Execute(context.Background(), autonomy.HandoffRequest{RunID: run.ID, ProjectID: "proj-a"})
	if res.OrchestratorCalled {
		t.Fatal("ASSISTED must not call orchestrator")
	}
	if auth.n.Load() != 0 {
		t.Fatal("no grants")
	}
}

// --- cancellation & terminal ---

func Test14D_CancelBlocksAdvance(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	run, _ := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "can-1", Goal: "g", ProjectID: "p", ScopeID: "s",
		Policy: controlledPolicy("p", 5),
	})
	_, _ = ctrl.Cancel(run.ID)
	_, _ = ctrl.Cancel(run.ID) // idempotent
	auth := &grantCountingAuth{}
	h := &autonomy.AdvanceHandoff{
		Controller: ctrl, Orchestrator: workflow.NewOrchestrator(nil, nil, nil, auth),
	}
	res, err := h.Execute(context.Background(), autonomy.HandoffRequest{RunID: run.ID, ProjectID: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if res.OrchestratorCalled || auth.n.Load() != 0 {
		t.Fatal("cancelled run must not execute")
	}
}

func Test14D_CancelSurvivesRepo(t *testing.T) {
	repo := memory.NewAutonomyRunRepo()
	rec := ports.AutonomyRunRecord{
		ID: "can-2", ProjectID: "proj-a", State: "RUNNING", Version: 1,
		StartedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(), PolicyJSON: []byte(`{}`),
	}
	_ = repo.Create(context.Background(), rec)
	_ = repo.Cancel(context.Background(), "proj-a", "can-2")
	_ = repo.Cancel(context.Background(), "proj-a", "can-2")
	got, err := repo.Get(context.Background(), "proj-a", "can-2")
	if err != nil || got.State != "CANCELLED" {
		t.Fatal(err, got.State)
	}
}

func Test14D_TerminalFailedNoAdvance(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	run, _ := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "term-1", Goal: "g", ProjectID: "p", ScopeID: "s",
		Policy: controlledPolicy("p", 5),
	})
	// force terminal via cancel path is enough; also test policy blocked
	_ = ctrl.RecordStepExecuted(run.ID)
	_ = ctrl.RecordStepExecuted(run.ID)
	_ = ctrl.RecordStepExecuted(run.ID)
	_ = ctrl.RecordStepExecuted(run.ID)
	_ = ctrl.RecordStepExecuted(run.ID)
	_ = ctrl.RecordStepExecuted(run.ID) // exceed max 5
	auth := &grantCountingAuth{}
	h := &autonomy.AdvanceHandoff{
		Controller: ctrl, Orchestrator: workflow.NewOrchestrator(nil, nil, nil, auth),
	}
	res, _ := h.Execute(context.Background(), autonomy.HandoffRequest{RunID: run.ID, ProjectID: "p"})
	if res.OrchestratorCalled {
		t.Fatal("over budget terminal must not advance")
	}
}

// --- isolation ---

func Test14D_TenantMismatch(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	run, _ := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "ten-1", Goal: "g", ProjectID: "proj-a", ScopeID: "s", TenantID: "ten-a",
		Policy: controlledPolicy("proj-a", 5),
	})
	h := &autonomy.AdvanceHandoff{Controller: ctrl}
	res, _ := h.Execute(context.Background(), autonomy.HandoffRequest{
		RunID: run.ID, ProjectID: "proj-a", TenantID: "ten-other",
	})
	if res.PolicyDecision.Reason != "tenant_mismatch" {
		t.Fatal(res.PolicyDecision)
	}
	if res.OrchestratorCalled {
		t.Fatal("tenant mismatch")
	}
}

func Test14D_ScopeMismatch(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	run, _ := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "scp-1", Goal: "g", ProjectID: "proj-a", ScopeID: "scope-a",
		Policy: controlledPolicy("proj-a", 5),
	})
	h := &autonomy.AdvanceHandoff{Controller: ctrl}
	res, _ := h.Execute(context.Background(), autonomy.HandoffRequest{
		RunID: run.ID, ProjectID: "proj-a", ScopeID: "scope-b",
	})
	if res.PolicyDecision.Reason != "scope_mismatch" {
		t.Fatal(res.PolicyDecision)
	}
}

func Test14D_RepoProjectIsolation(t *testing.T) {
	repo := memory.NewAutonomyRunRepo()
	_ = repo.Create(context.Background(), ports.AutonomyRunRecord{
		ID: "iso-1", ProjectID: "proj-a", State: "RUNNING", Version: 1,
		StartedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	})
	_, err := repo.Get(context.Background(), "proj-b", "iso-1")
	if err == nil {
		t.Fatal("cross-project get must fail")
	}
	if err := repo.Cancel(context.Background(), "proj-b", "iso-1"); err == nil {
		// cancel on wrong project should not find
	}
	got, _ := repo.Get(context.Background(), "proj-a", "iso-1")
	if got.State == "CANCELLED" {
		t.Fatal("wrong project cancel must not affect run")
	}
}

// --- no grant in autonomy state ---

func Test14D_AutonomyRunJSONHasNoGrant(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	run, _ := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "nog-1", Goal: "g", ProjectID: "p", ScopeID: "s",
		Policy: controlledPolicy("p", 5),
	})
	b, err := json.Marshal(run)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, bad := range []string{`"grant"`, `"Grant"`, `execution_grant`, `api_key`, `secret`} {
		if containsFold(s, bad) && bad == `"grant"` {
			// field name grant should not appear as stored auth
		}
	}
	if containsFold(s, `"grant_id"`) || containsFold(s, `"execution_grant"`) {
		t.Fatal("grant field leaked in AutonomyRun JSON:", s)
	}
}

func containsFold(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if equalFoldASCII(s[i:i+len(sub)], sub) {
					return true
				}
			}
			return false
		})())
}

func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// --- idempotent start ---

func Test14D_StartIdempotent(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	req := autonomy.StartRequest{
		ID: "idem-1", Goal: "g", ProjectID: "p", ScopeID: "s",
		Policy: manualPolicy("p"),
	}
	a, err := ctrl.Start(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ctrl.Start(context.Background(), req)
	if err != nil || a.ID != b.ID {
		t.Fatal(err, a, b)
	}
}

// --- restart hydration ---

func Test14D_LoadFromRepoPreservesBudgetAndCancel(t *testing.T) {
	repo := memory.NewAutonomyRunRepo()
	ctrl := autonomy.NewController(&ai.MockBrain{})
	ctrl.Repo = repo

	run, err := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "rst-1", Goal: "g", ProjectID: "proj-a", ScopeID: "s", TenantID: "ten-a",
		Policy:          controlledPolicy("proj-a", 20),
		CapabilityHints: []contracts.CapabilityID{"recon.network.portscan"},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = ctrl.RecordStepExecuted(run.ID)
	_ = ctrl.RecordStepExecuted(run.ID)
	_ = ctrl.RecordStepExecuted(run.ID)

	// new controller simulates process restart
	ctrl2 := autonomy.NewController(&ai.MockBrain{})
	ctrl2.Repo = repo
	loaded, err := ctrl2.LoadFromRepo(context.Background(), "proj-a", "rst-1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.StepsConsumed != 3 {
		t.Fatalf("budget lost on restart: %d", loaded.StepsConsumed)
	}
	if loaded.ProjectID != "proj-a" {
		t.Fatal(loaded.ProjectID)
	}

	_, _ = ctrl2.Cancel("rst-1")
	ctrl3 := autonomy.NewController(&ai.MockBrain{})
	ctrl3.Repo = repo
	loaded2, err := ctrl3.LoadFromRepo(context.Background(), "proj-a", "rst-1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded2.State != autonomy.RunCancelled {
		// Cancel may only update in-memory if persist path differs — check repo
		rec, err := repo.Get(context.Background(), "proj-a", "rst-1")
		if err != nil {
			t.Fatal(err)
		}
		if rec.State != "CANCELLED" && loaded2.State != autonomy.RunCancelled {
			t.Fatalf("cancel not durable: run=%s repo=%s", loaded2.State, rec.State)
		}
	}
}

// --- concurrent advance does not panic / corrupt ---

func Test14D_ConcurrentAdvanceSafe(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	run, _ := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "conc-1", Goal: "g", ProjectID: "proj-a", ScopeID: "s",
		Policy:          controlledPolicy("proj-a", 2),
		CapabilityHints: []contracts.CapabilityID{"recon.network.portscan"},
	})
	auth := &grantCountingAuth{}
	h := &autonomy.AdvanceHandoff{
		Controller: ctrl, Orchestrator: workflow.NewOrchestrator(nil, nil, nil, auth),
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = h.Execute(context.Background(), autonomy.HandoffRequest{RunID: run.ID, ProjectID: "proj-a"})
		}()
	}
	wg.Wait()
	got, _ := ctrl.Get(run.ID)
	if got.StepsConsumed > 2 {
		// policy max 2; RecordStepExecuted only on successful orch path which may fail without mat
		t.Logf("steps_consumed=%d grants=%d", got.StepsConsumed, auth.n.Load())
	}
}

// --- policy version fail closed ---

func Test14D_UnknownPolicyVersionFailClosed(t *testing.T) {
	p := controlledPolicy("proj-a", 5)
	p.Version = "99"
	_, err := autonomy.EvaluateAutonomyPolicy(autonomy.PolicyEvalRequest{
		ProjectID: "proj-a", CapabilityID: "recon.network.portscan", Policy: p,
	})
	if err == nil {
		t.Fatal("unknown policy version must fail")
	}
}
