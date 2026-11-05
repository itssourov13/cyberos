// Package sdk provides the in-process SDK foundation for CYBEROS.
// Future automation, plugins, agents, ZOYA/JARVIS, and UAB should depend on
// this surface (or the HTTP API), not on internal implementation packages.
package sdk

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cyberos/internal/ai"
	"cyberos/internal/ai/apply"
	"cyberos/internal/ai/autonomy"
	appscope "cyberos/internal/application/scope"
	"cyberos/internal/assessment"
	"cyberos/internal/assets"
	"cyberos/internal/audit"
	"cyberos/internal/auth"
	"cyberos/internal/broker"
	"cyberos/internal/capability"
	"cyberos/internal/cases"
	"cyberos/internal/contracts"
	"cyberos/internal/controlplane"
	domaincap "cyberos/internal/domain/capability"
	"cyberos/internal/events"
	"cyberos/internal/evidence"
	"cyberos/internal/execution"
	"cyberos/internal/execution/lifecycle"
	"cyberos/internal/execution/mock"
	"cyberos/internal/execution/sandbox"
	"cyberos/internal/findings"
	"cyberos/internal/identity"
	"cyberos/internal/provenance"
	"cyberos/internal/reports"
	"cyberos/internal/riskprio"
	"cyberos/internal/taskengine"
	"cyberos/internal/tools"
	"cyberos/internal/workflow"
)

// Client is the SDK entrypoint. Sensitive operations always go through
// Control Plane → Grant → Broker.
type Client struct {
	Identity                    *identity.Service
	Scopes                      *appscope.Service
	Caps                        *capability.Registry
	Tasks                       *taskengine.Engine
	Control                     *controlplane.Service
	Issuer                      *broker.Issuer
	Broker                      *broker.Broker
	Evidence                    *evidence.Service
	Findings                    *findings.Service
	Provenance                  *provenance.Service
	Audit                       *audit.Service
	Catalog                     *Catalog
	Assessment                  *assessment.Orchestrator
	Tools                       *tools.Manager
	Events                      *events.Store
	Hub                         *events.Hub
	ExecReg                     *execution.Registry
	Auth                        auth.Authenticator
	Platform                    *tools.Platform
	Assets                      *assets.Inventory
	RiskPrio                    *riskprio.Engine
	Cases                       *cases.Service
	Reports                     *reports.Service
	Lifecycle                   *lifecycle.Controller    // optional durable execution jobs (Phase 9D)
	Workflow                    *workflow.Orchestrator   // Phase 10B/10C orchestration
	AIApply                     *apply.Applicator        // Phase 12D plan → workflow (no execution)
	Autonomy                    *autonomy.Controller     // Phase 13/14 autonomy
	AutonomyHandoff             *autonomy.AdvanceHandoff // Phase 14B handoff
	Brain                       ai.Brain                 // Phase 15D optional external brain
	RequireExplicitExecutorHint bool                     // production must bind every grant to one registered executor
}

// NewInMemory constructs a client with mock broker, issuer, evidence stack,
// and in-memory capability registry. Wire Identity/Scopes/Control/Tasks for
// full authorization flows.
func NewInMemory() *Client {
	ev := evidence.NewService()
	evStore := events.NewStore()
	hub := events.NewHub()
	evStore = evStore.WithHub(hub)
	execReg := execution.NewRegistry()
	toolMgr := tools.NewManager(nil)
	_ = toolMgr.Register(tools.NewStubScanner(nil))
	_ = toolMgr.RegisterAllExecutors(execReg)
	// Prefer tool router when available; fall back still safe via mock if registry empty.
	inner := sandbox.New(&execution.Router{Reg: execReg})
	// Also keep mock as default for grants without executor hints
	_ = execReg.Register(mockToolExecutor{})
	execReg.SetDefault("mock")
	brk := broker.New(inner)
	c := &Client{
		Caps:       capability.NewRegistry(nil),
		Issuer:     broker.NewIssuer(5 * time.Minute),
		Broker:     brk,
		Evidence:   ev,
		Findings:   findings.NewService(ev),
		Provenance: provenance.NewService(),
		Catalog:    NewCatalog(),
		Events:     evStore,
		Hub:        hub,
		Tools:      toolMgr,
		Platform:   tools.NewPlatform(toolMgr),
		Assets:     assets.NewInventory(),
		RiskPrio:   riskprio.NewEngine(),
		Cases:      cases.NewService(),
		Reports:    reports.NewService(),
		ExecReg:    execReg,
	}
	c.Lifecycle = lifecycle.New(brk,
		lifecycle.WithEvidence(ev),
		lifecycle.WithFindings(c.Findings),
		lifecycle.WithProvenance(c.Provenance),
	)
	if err := c.EnableGrantProvenance(nil); err != nil {
		panic(err) // crypto/rand failure is unrecoverable
	}
	c.UseGrantLedger(broker.NewMemoryGrantLedger()) // process-local; production swaps in PostgreSQL
	c.WireWorkflow()
	c.AIApply = apply.NewApplicator(nil)
	c.WireAutonomy(nil)
	// Phase 4A: no demo seed — catalog starts empty until RefreshFromRepos or explicit upserts.
	return c
}

// EnableGrantProvenance makes the Issuer sign every grant and the Broker reject
// any grant not signed by it. A nil/empty key selects an ephemeral per-process
// key (fail-closed across restarts).
func (c *Client) EnableGrantProvenance(key []byte) error {
	var (
		signer *broker.GrantSigner
		err    error
	)
	if len(key) == 0 {
		signer, err = broker.NewEphemeralGrantSigner()
	} else {
		signer, err = broker.NewGrantSigner(key)
	}
	if err != nil {
		return err
	}
	c.Issuer.Signer = signer
	c.Broker.Signer = signer
	return nil
}

// UseGrantLedger wires the shared grant ledger into both the Issuer (records
// every issued grant) and the Broker (atomic one-time claim, revocation).
func (c *Client) UseGrantLedger(l broker.GrantLedger) {
	c.Issuer.Ledger = l
	c.Broker.Ledger = l
}

// mockToolExecutor adapts mock.Executor to ToolExecutor for registry default.
type mockToolExecutor struct{}

func (mockToolExecutor) Manifest() contracts.ToolManifest {
	return contracts.ToolManifest{
		ID: "mock", Name: "Mock", Version: "1", Runtime: contracts.PluginRuntimeGo, ExecutorID: "mock",
	}
}
func (mockToolExecutor) Execute(req contracts.ExecutionRequest) (contracts.ExecutionResult, error) {
	return mock.New().Execute(req)
}

// WithIdentity sets the identity service.
func (c *Client) WithIdentity(s *identity.Service) *Client { c.Identity = s; return c }

// WithScopes sets the scope service.
func (c *Client) WithScopes(s *appscope.Service) *Client { c.Scopes = s; return c }

// WithControlPlane sets the control plane.
func (c *Client) WithControlPlane(s *controlplane.Service) *Client { c.Control = s; return c }

// WithTasks sets the task engine.
func (c *Client) WithTasks(e *taskengine.Engine) *Client { c.Tasks = e; return c }

// WithAudit sets the audit service.
func (c *Client) WithAudit(a *audit.Service) *Client { c.Audit = a; return c }

// WithEvidence shares an evidence service (and rebuilds findings on top).
func (c *Client) WithEvidence(e *evidence.Service) *Client {
	c.Evidence = e
	c.Findings = findings.NewService(e)
	return c
}

// WireAssessment attaches ControlPlaneIssuer + orchestrator using this client's services.
// Call after WithIdentity/WithScopes/WithControlPlane.
func (c *Client) WireAssessment() *Client {
	if c.Control == nil || c.Issuer == nil || c.Broker == nil {
		return c
	}
	issuer := sdkAssessmentIssuer{client: c}
	executor := sdkAssessmentExecutor{client: c}
	orch := assessment.NewOrchestrator(issuer, executor)
	if c.Evidence != nil {
		orch = orch.WithEvidence(c.Evidence, c.Findings, c.Provenance, c.Events)
	}
	orch = orch.WithPhase6(&assessment.Phase6Sink{
		Assets: c.Assets, Findings: c.Findings, Risk: c.RiskPrio, Cases: c.Cases,
	})
	c.Assessment = orch
	return c
}

// WireWorkflow attaches Materializer + Orchestrator using Tasks, Platform, Lifecycle.
// Step authorization uses IssueGrant (CP → Decision → Grant) per step — no grant caching.
func (c *Client) WireWorkflow() *Client {
	if c == nil {
		return c
	}
	if c.Tasks == nil {
		c.Tasks = taskengine.NewEngine(nil, c.Identity, c.Scopes, c.Caps)
	}
	if c.Platform == nil && c.Tools != nil {
		c.Platform = tools.NewPlatform(c.Tools)
	}
	if c.Lifecycle == nil || c.Platform == nil {
		return c
	}
	mat := workflow.NewMaterializer(c.Tasks, workflow.NewCapabilityResolver(c.Platform))
	auth := workflow.FuncAuthorizer{Fn: func(ctx context.Context, req workflow.AuthorizeStepRequest) (contracts.ExecutionGrant, error) {
		if c.Control == nil || c.Issuer == nil {
			return contracts.ExecutionGrant{}, fmt.Errorf("%w: control/issuer required for workflow steps", contracts.ErrInvalidArgument)
		}
		eval := controlplane.EvaluateRequest{
			PrincipalID: req.PrincipalID, ProjectID: req.ProjectID, ScopeID: req.ScopeID,
			TaskID: req.TaskID, CapabilityID: req.CapabilityID, CorrelationID: req.CorrelationID,
		}
		constraints := contracts.ExecutionGrantConstraints{}
		if req.ExecutorHint != "" {
			constraints.AllowedExecutorHints = []string{req.ExecutorHint}
		}
		if req.Target != "" {
			constraints.AllowedTargets = []string{req.Target}
			if c.Assets != nil {
				asset, assetErr := c.Assets.GetByValue(req.ProjectID, req.Target)
				if assetErr == nil {
					eval.AssetID = asset.ID
					constraints.AllowedTargets = []string{asset.Value}
				} else if !errors.Is(assetErr, contracts.ErrNotFound) {
					return contracts.ExecutionGrant{}, fmt.Errorf("resolve workflow target: %w", assetErr)
				}
			}
		}
		grant, _, err := c.IssueGrantWithConstraints(ctx, eval, constraints)
		return grant, err
	}}
	c.Workflow = workflow.NewOrchestrator(mat, c.Tasks, c.Lifecycle, auth)
	return c
}

// StartWorkflow materializes and registers a run (orchestration only).
func (c *Client) StartWorkflow(ctx context.Context, def workflow.Definition, principalID contracts.PrincipalID, runID string) (*workflow.Run, error) {
	if c == nil || c.Workflow == nil {
		return nil, fmt.Errorf("%w: workflow not configured", contracts.ErrInvalidArgument)
	}
	return c.Workflow.Start(ctx, workflow.StartRequest{Definition: def, PrincipalID: principalID, RunID: runID})
}

// AdvanceWorkflow runs ready steps through CP → Grant → Lifecycle (fresh grant each step).
func (c *Client) AdvanceWorkflow(ctx context.Context, runID string, def workflow.Definition) (*workflow.Run, error) {
	if c == nil || c.Workflow == nil {
		return nil, fmt.Errorf("%w: workflow not configured", contracts.ErrInvalidArgument)
	}
	return c.Workflow.Advance(ctx, runID, def)
}

// CancelWorkflow cancels a run (orchestration only).
func (c *Client) CancelWorkflow(runID string) (*workflow.Run, error) {
	if c == nil || c.Workflow == nil {
		return nil, fmt.Errorf("%w: workflow not configured", contracts.ErrInvalidArgument)
	}
	return c.Workflow.Cancel(runID)
}

// GetWorkflowRun returns durable/memory run state.
func (c *Client) GetWorkflowRun(runID string) (*workflow.Run, error) {
	if c == nil || c.Workflow == nil {
		return nil, fmt.Errorf("%w: workflow not configured", contracts.ErrInvalidArgument)
	}
	return c.Workflow.Get(runID)
}

// Evaluate runs the Control Plane decision chain.
func (c *Client) Evaluate(ctx context.Context, req controlplane.EvaluateRequest) (contracts.Decision, error) {
	if c.Control == nil {
		return contracts.Decision{}, fmt.Errorf("%w: control plane not configured", contracts.ErrInvalidArgument)
	}
	if req.CorrelationID == "" {
		req.CorrelationID = contracts.CorrelationID(fmt.Sprintf("decision-%d", time.Now().UTC().UnixNano()))
	}
	d, err := c.Control.Evaluate(ctx, req)
	if err != nil {
		return d, err
	}
	if c.Audit != nil {
		if _, auditErr := c.Audit.RecordDecision(d); auditErr != nil {
			return d, fmt.Errorf("decision completed but audit persistence failed: %w", auditErr)
		}
	}
	return d, nil
}

// IssueGrant evaluates then issues a grant only if allowed.
func (c *Client) IssueGrant(ctx context.Context, req controlplane.EvaluateRequest) (contracts.ExecutionGrant, contracts.Decision, error) {
	return c.IssueGrantWithConstraints(ctx, req, contracts.ExecutionGrantConstraints{})
}

// IssueGrantWithConstraints derives asset bindings from the authoritative inventory
// before the Issuer signs the grant. Client-provided target strings are never used
// to replace a canonical stored asset value when an asset ID is supplied.
func (c *Client) IssueGrantWithConstraints(ctx context.Context, req controlplane.EvaluateRequest, constraints contracts.ExecutionGrantConstraints) (contracts.ExecutionGrant, contracts.Decision, error) {
	if c.Issuer == nil || c.Control == nil {
		return contracts.ExecutionGrant{}, contracts.Decision{}, fmt.Errorf("%w: issuer/control not configured", contracts.ErrInvalidArgument)
	}
	if c.RequireExplicitExecutorHint && len(constraints.AllowedExecutorHints) == 0 {
		return contracts.ExecutionGrant{}, contracts.Decision{}, fmt.Errorf("%w: production grant requires a registered executor hint", contracts.ErrCapabilityUnknown)
	}
	if c.RequireExplicitExecutorHint {
		if c.Tools == nil {
			return contracts.ExecutionGrant{}, contracts.Decision{}, fmt.Errorf("%w: production tool manager unavailable", contracts.ErrInvalidArgument)
		}
		for _, hint := range constraints.AllowedExecutorHints {
			adapter, err := c.Tools.Get(hint)
			if err != nil {
				return contracts.ExecutionGrant{}, contracts.Decision{}, fmt.Errorf("%w: executor hint %q is not registered", contracts.ErrCapabilityUnknown, hint)
			}
			targetBound, ok := adapter.(tools.TargetBoundAdapter)
			if ok && targetBound.RequiresTargetBinding() {
				if req.AssetID == "" || len(constraints.AllowedTargets) != 1 || strings.TrimSpace(constraints.AllowedTargets[0]) == "" {
					return contracts.ExecutionGrant{}, contracts.Decision{}, fmt.Errorf("%w: target-bound production executors require one canonical registered asset", contracts.ErrInvalidArgument)
				}
			}
		}
	}
	if req.AssetID != "" {
		if c.Assets == nil {
			return contracts.ExecutionGrant{}, contracts.Decision{}, fmt.Errorf("%w: asset inventory not configured", contracts.ErrInvalidArgument)
		}
		asset, err := c.Assets.Get(req.AssetID)
		if err != nil {
			return contracts.ExecutionGrant{}, contracts.Decision{}, fmt.Errorf("resolve grant asset: %w", err)
		}
		if asset.ProjectID != req.ProjectID {
			return contracts.ExecutionGrant{}, contracts.Decision{}, fmt.Errorf("%w: asset does not belong to project", contracts.ErrForbidden)
		}
		if len(constraints.AllowedTargets) > 0 {
			for _, target := range constraints.AllowedTargets {
				if target != asset.Value {
					return contracts.ExecutionGrant{}, contracts.Decision{}, fmt.Errorf("%w: grant target differs from canonical asset value", contracts.ErrForbidden)
				}
			}
		}
		constraints.AllowedTargets = []string{asset.Value}
		if !containsGrantAsset(constraints.AllowedAssetIDs, asset.ID) {
			constraints.AllowedAssetIDs = append(constraints.AllowedAssetIDs, asset.ID)
		}
	}
	g, d, err := c.Issuer.IssueAfterEvaluateWith(ctx, c.Control, req, constraints)
	if err != nil {
		return contracts.ExecutionGrant{}, d, err
	}
	if c.Audit != nil {
		if _, auditErr := c.Audit.RecordGrantIssued(g.ExecutionGrant); auditErr != nil {
			// A grant that cannot be durably audited must not remain executable.
			if c.Broker != nil {
				revokeErr := c.Broker.RevokeGrantDurable(ctx, g.ExecutionGrant.ID, g.ExecutionGrant.Principal.ID, "grant audit persistence failed")
				if revokeErr != nil {
					return contracts.ExecutionGrant{}, d, errors.Join(
						fmt.Errorf("grant audit persistence failed: %w", auditErr),
						fmt.Errorf("grant revocation also failed: %w", revokeErr),
					)
				}
			}
			return contracts.ExecutionGrant{}, d, fmt.Errorf("grant issued but revoked because audit persistence failed: %w", auditErr)
		}
	}
	return g.ExecutionGrant, d, nil
}

func containsGrantAsset(items []contracts.AssetID, target contracts.AssetID) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

// Execute runs work through the Tool Broker under a grant.
func (c *Client) Execute(grant contracts.ExecutionGrant, params map[string]any) (contracts.ExecutionResult, error) {
	if c.Broker == nil {
		return contracts.ExecutionResult{}, fmt.Errorf("%w: broker not configured", contracts.ErrInvalidArgument)
	}
	res, err := c.Broker.Execute(contracts.ExecutionRequest{
		Grant: grant, Parameters: params, CorrelationID: grant.CorrelationID,
	})
	if c.Audit != nil {
		if _, auditErr := c.Audit.RecordExecution(grant, res); auditErr != nil {
			return res, errors.Join(err, fmt.Errorf("execution result could not be durably audited: %w", auditErr))
		}
	}
	return res, err
}

// RegisterCapability registers a capability for discovery only.
func (c *Client) RegisterCapability(ctx context.Context, id, name string, typ contracts.CapabilityType) error {
	if c.Caps == nil {
		return fmt.Errorf("%w: capability registry not configured", contracts.ErrInvalidArgument)
	}
	cap, err := domaincap.New(contracts.CapabilityID(id), name, typ)
	if err != nil {
		return err
	}
	return c.Caps.Register(ctx, cap)
}

// ListCapabilities returns registered capability descriptors.
func (c *Client) ListCapabilities(ctx context.Context) ([]contracts.CapabilityDescriptor, error) {
	if c.Caps == nil {
		return nil, fmt.Errorf("%w: capability registry not configured", contracts.ErrInvalidArgument)
	}
	list, err := c.Caps.List(ctx, 100, 0)
	if err != nil {
		return nil, err
	}
	out := make([]contracts.CapabilityDescriptor, 0, len(list))
	for _, cap := range list {
		out = append(out, cap.Descriptor())
	}
	return out, nil
}
