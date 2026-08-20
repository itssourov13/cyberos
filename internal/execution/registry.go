package execution

import (
	"fmt"
	"sync"

	"cyberos/internal/contracts"
)

// Registry maps executor IDs / hints to ToolExecutor implementations.
// The Tool Broker remains the only public execution gateway; this registry
// only supplies backends after grant validation.
type Registry struct {
	mu        sync.RWMutex
	executors map[string]contracts.ToolExecutor
	defaultID string
}

// NewRegistry constructs an empty executor registry.
func NewRegistry() *Registry {
	return &Registry{executors: make(map[string]contracts.ToolExecutor)}
}

// Register adds a tool executor. Registration is discovery of a backend, not authorization.
func (r *Registry) Register(ex contracts.ToolExecutor) error {
	if r == nil || ex == nil {
		return fmt.Errorf("%w: executor registry not configured", contracts.ErrInvalidArgument)
	}
	m := ex.Manifest()
	if err := m.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.executors[m.ExecutorID]; exists {
		return contracts.ErrConflict
	}
	r.executors[m.ExecutorID] = ex
	if r.defaultID == "" {
		r.defaultID = m.ExecutorID
	}
	return nil
}

// SetDefault selects the default executor id.
func (r *Registry) SetDefault(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.executors[id]; !ok {
		return contracts.ErrNotFound
	}
	r.defaultID = id
	return nil
}

// Get returns an executor by id.
func (r *Registry) Get(id string) (contracts.ToolExecutor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ex, ok := r.executors[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	return ex, nil
}

// Resolve picks an executor from grant hints or the default.
func (r *Registry) Resolve(grant contracts.ExecutionGrant) (contracts.ToolExecutor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, h := range grant.Constraints.AllowedExecutorHints {
		if ex, ok := r.executors[h]; ok {
			return ex, nil
		}
	}
	if r.defaultID != "" {
		if ex, ok := r.executors[r.defaultID]; ok {
			return ex, nil
		}
	}
	return nil, fmt.Errorf("%w: no executor available", contracts.ErrNotFound)
}

// ListManifests returns registered tool manifests.
func (r *Registry) ListManifests() []contracts.ToolManifest {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]contracts.ToolManifest, 0, len(r.executors))
	for _, ex := range r.executors {
		out = append(out, ex.Manifest())
	}
	return out
}

// Router is an Executor that dispatches to registry backends.
type Router struct {
	Reg *Registry
}

// Execute implements contracts.Executor.
func (rt *Router) Execute(req contracts.ExecutionRequest) (contracts.ExecutionResult, error) {
	if rt == nil || rt.Reg == nil {
		return contracts.ExecutionResult{}, fmt.Errorf("%w: router not configured", contracts.ErrInvalidArgument)
	}
	if err := req.Validate(); err != nil {
		return contracts.ExecutionResult{}, err
	}
	ex, err := rt.Reg.Resolve(req.Grant)
	if err != nil {
		return contracts.ExecutionResult{}, err
	}
	return ex.Execute(req)
}

var _ contracts.Executor = (*Router)(nil)
