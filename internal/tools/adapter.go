package tools

import (
	"context"
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/execution/process"
)

// Adapter is a pluggable security-tool integration.
// Adapters must not expose unrestricted shell or bypass the broker.
type Adapter interface {
	Descriptor() Descriptor
	Health(ctx context.Context) HealthReport
	// Run produces a normalized result. Implementations may use process.Runner
	// under allowlist policy, or pure in-process simulation for stubs.
	Run(ctx context.Context, capability contracts.CapabilityID, params map[string]any) (NormalizedResult, error)
}

// Manager holds registered adapters and exposes ToolExecutors for the registry.
type Manager struct {
	adapters map[string]Adapter
	proc     *process.Runner
}

// NewManager constructs a tool adapter manager.
func NewManager(proc *process.Runner) *Manager {
	if proc == nil {
		proc = process.NewRunner(process.DefaultPolicy())
	}
	return &Manager{adapters: make(map[string]Adapter), proc: proc}
}

// Register adds an adapter (discovery only).
func (m *Manager) Register(a Adapter) error {
	if m == nil || a == nil {
		return fmt.Errorf("%w: adapter nil", contracts.ErrInvalidArgument)
	}
	d := a.Descriptor()
	if err := d.Validate(); err != nil {
		return err
	}
	if _, ok := m.adapters[d.ID]; ok {
		return contracts.ErrConflict
	}
	m.adapters[d.ID] = a
	return nil
}

// Get returns an adapter by id.
func (m *Manager) Get(id string) (Adapter, error) {
	a, ok := m.adapters[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	return a, nil
}

// List returns descriptors for all adapters.
func (m *Manager) List() []Descriptor {
	out := make([]Descriptor, 0, len(m.adapters))
	for _, a := range m.adapters {
		out = append(out, a.Descriptor())
	}
	return out
}

// HealthAll runs health checks.
func (m *Manager) HealthAll(ctx context.Context) map[string]HealthReport {
	out := make(map[string]HealthReport, len(m.adapters))
	for id, a := range m.adapters {
		out[id] = a.Health(ctx)
	}
	return out
}

// ExecutorFor returns a contracts.ToolExecutor wrapping the named adapter.
func (m *Manager) ExecutorFor(toolID string) (contracts.ToolExecutor, error) {
	a, err := m.Get(toolID)
	if err != nil {
		return nil, err
	}
	return &toolExecutor{adapter: a, manager: m}, nil
}

// RegisterAllExecutors registers each adapter into an execution registry.
func (m *Manager) RegisterAllExecutors(reg interface {
	Register(ex contracts.ToolExecutor) error
}) error {
	for id := range m.adapters {
		ex, err := m.ExecutorFor(id)
		if err != nil {
			return err
		}
		if err := reg.Register(ex); err != nil {
			return err
		}
	}
	return nil
}

type toolExecutor struct {
	adapter Adapter
	manager *Manager
}

func (e *toolExecutor) Manifest() contracts.ToolManifest {
	return e.adapter.Descriptor().ToolManifest()
}

func (e *toolExecutor) Execute(req contracts.ExecutionRequest) (contracts.ExecutionResult, error) {
	start := time.Now().UTC()
	if err := req.Validate(); err != nil {
		return fail(start, e.Manifest().ExecutorID, err.Error(), contracts.ExecutionStatusRejected), err
	}
	d := e.adapter.Descriptor()
	// Capability must be declared by the tool
	if len(d.Capabilities) > 0 {
		ok := false
		for _, c := range d.Capabilities {
			if c == req.Grant.CapabilityID {
				ok = true
				break
			}
		}
		if !ok {
			msg := fmt.Sprintf("capability %s not provided by tool %s", req.Grant.CapabilityID, d.ID)
			return fail(start, d.ExecutorID, msg, contracts.ExecutionStatusRejected),
				fmt.Errorf("%w: %s", contracts.ErrCapabilityUnknown, msg)
		}
	}
	nr, err := e.adapter.Run(context.Background(), req.Grant.CapabilityID, req.Parameters)
	finish := time.Now().UTC()
	status := contracts.ExecutionStatusSucceeded
	msg := ""
	if err != nil {
		status = contracts.ExecutionStatusFailed
		msg = err.Error()
	} else if !nr.Success {
		status = contracts.ExecutionStatusFailed
		msg = nr.Summary
	}
	out := map[string]any{
		"tool_id":     nr.ToolID,
		"capability":  nr.Capability,
		"success":     nr.Success,
		"summary":     nr.Summary,
		"findings":    nr.Findings,
		"metrics":     nr.Metrics,
		"normalized":  true,
		"executor_id": d.ExecutorID,
	}
	if nr.Stdout != "" {
		out["stdout"] = nr.Stdout
	}
	if nr.Stderr != "" {
		out["stderr"] = nr.Stderr
	}
	if nr.Raw != nil {
		out["raw"] = nr.Raw
	}
	return contracts.ExecutionResult{
		Status: status, StartedAt: start, FinishedAt: finish,
		Output: out, ErrorMessage: msg, CorrelationID: req.CorrelationID, ExecutorID: d.ExecutorID,
	}, err
}

func fail(start time.Time, id, msg string, st contracts.ExecutionStatus) contracts.ExecutionResult {
	return contracts.ExecutionResult{
		Status: st, StartedAt: start, FinishedAt: time.Now().UTC(),
		ErrorMessage: msg, ExecutorID: id,
	}
}

var _ contracts.ToolExecutor = (*toolExecutor)(nil)
