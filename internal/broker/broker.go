package broker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cyberos/internal/contracts"
	domaingrant "cyberos/internal/domain/grant"
	"cyberos/internal/execution"
)

// Broker is the sole execution gateway of CYBEROS.
// All execution must pass through a validated ExecutionGrant.
//
// Phase 17A: after Grant validation, Broker resolves an allowlisted
// ExecutionHost (default: local) and dispatches. Host selection is NOT
// authorization and cannot bypass Grant checks.
type Broker struct {
	executor contracts.Executor // legacy direct path when Hosts is nil
	Hosts    *execution.HostRegistry
	// usedNonces provides basic replay protection within process lifetime.
	mu         sync.Mutex
	usedNonces map[string]time.Time
	// Now overrides clock for tests.
	Now func() time.Time
}

// New constructs a Tool Broker with the given executor (must be non-nil).
// Registers a LocalHost wrapping the executor as the default host.
func New(executor contracts.Executor) *Broker {
	reg := execution.NewHostRegistry()
	_ = reg.Register(execution.NewLocalHost(executor))
	return &Broker{
		executor:   executor,
		Hosts:      reg,
		usedNonces: make(map[string]time.Time),
		Now:        func() time.Time { return time.Now().UTC() },
	}
}

// NewWithHosts constructs a Broker that routes only through the provided registry.
// The registry must already contain a default host (typically LocalHost).
func NewWithHosts(hosts *execution.HostRegistry) *Broker {
	return &Broker{
		Hosts:      hosts,
		usedNonces: make(map[string]time.Time),
		Now:        func() time.Time { return time.Now().UTC() },
	}
}

// Execute validates the grant and dispatches to an allowlisted ExecutionHost.
//
// Invariant: NO VALID GRANT ⇒ NO EXECUTION
func (b *Broker) Execute(req contracts.ExecutionRequest) (contracts.ExecutionResult, error) {
	if b == nil {
		return rejected("broker not configured"), fmt.Errorf("%w: broker not configured", contracts.ErrInvalidArgument)
	}
	if b.Hosts == nil && b.executor == nil {
		return rejected("broker not configured"), fmt.Errorf("%w: broker not configured", contracts.ErrInvalidArgument)
	}

	now := b.Now()
	g := &domaingrant.Grant{ExecutionGrant: req.Grant}

	// Structural + temporal validation (Broker authority — NOT host)
	if err := g.Validate(); err != nil {
		return rejected("invalid grant: " + err.Error()), fmt.Errorf("%w: %v", contracts.ErrGrantInvalid, err)
	}
	if !g.IsUsable(now) {
		if g.Status != contracts.ExecutionGrantStatusIssued {
			return rejected("grant not in ISSUED status"), contracts.ErrGrantInvalid
		}
		return rejected("grant expired"), contracts.ErrGrantExpired
	}

	if err := g.EnsureUsable(now,
		g.Principal.ID,
		g.ProjectID,
		g.ScopeID,
		g.TaskID,
		g.CapabilityID,
	); err != nil {
		return rejected(err.Error()), err
	}

	// Replay protection via nonce (Broker-level only)
	if g.Nonce != "" {
		b.mu.Lock()
		if _, seen := b.usedNonces[g.Nonce]; seen {
			b.mu.Unlock()
			return rejected("grant nonce already used (replay)"), contracts.ErrGrantInvalid
		}
		b.usedNonces[g.Nonce] = now
		b.mu.Unlock()
	}

	if len(g.Constraints.AllowedExecutorHints) > 0 {
		_ = g.Constraints.AllowedExecutorHints
	}

	// Host resolution AFTER authorization. Unknown/invalid host → fail closed.
	if b.Hosts != nil {
		host, err := b.Hosts.Resolve(req)
		if err != nil {
			return rejected("unknown execution host: " + err.Error()), err
		}
		result, err := host.Execute(context.Background(), req)
		if err != nil {
			return result, err
		}
		if result.HostID == "" {
			result.HostID = string(host.ID())
		}
		return result, nil
	}

	// Legacy fallback (should not be used when New() registers Hosts)
	result, err := b.executor.Execute(req)
	if err != nil {
		return result, err
	}
	return result, nil
}

// ExecuteWithContext is like Execute but also verifies the grant matches
// the expected principal/project/scope/task/capability supplied by the caller.
func (b *Broker) ExecuteWithContext(
	ctx context.Context,
	req contracts.ExecutionRequest,
	principalID contracts.PrincipalID,
	projectID contracts.ProjectID,
	scopeID contracts.ScopeID,
	taskID contracts.TaskID,
	capabilityID contracts.CapabilityID,
) (contracts.ExecutionResult, error) {
	g := &domaingrant.Grant{ExecutionGrant: req.Grant}
	now := b.Now()
	if err := g.EnsureUsable(now, principalID, projectID, scopeID, taskID, capabilityID); err != nil {
		return rejected(err.Error()), err
	}
	return b.Execute(req)
}

func rejected(msg string) contracts.ExecutionResult {
	now := time.Now().UTC()
	return contracts.ExecutionResult{
		Status:       contracts.ExecutionStatusRejected,
		StartedAt:    now,
		FinishedAt:   now,
		ErrorMessage: msg,
		ExecutorID:   "broker",
	}
}

// Ensure interface compliance.
var _ contracts.ToolBroker = (*Broker)(nil)
