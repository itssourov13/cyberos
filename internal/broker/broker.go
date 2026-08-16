package broker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cyberos/internal/contracts"
	domaingrant "cyberos/internal/domain/grant"
)

// Broker is the sole execution gateway of CYBEROS.
// All execution must pass through a validated ExecutionGrant.
type Broker struct {
	executor contracts.Executor
	// usedNonces provides basic replay protection within process lifetime.
	mu         sync.Mutex
	usedNonces map[string]time.Time
	// Now overrides clock for tests.
	Now func() time.Time
}

// New constructs a Tool Broker with the given executor (must be non-nil).
func New(executor contracts.Executor) *Broker {
	return &Broker{
		executor:   executor,
		usedNonces: make(map[string]time.Time),
		Now:        func() time.Time { return time.Now().UTC() },
	}
}

// Execute validates the grant and dispatches to the executor.
//
// Invariant: NO VALID GRANT ⇒ NO EXECUTION
func (b *Broker) Execute(req contracts.ExecutionRequest) (contracts.ExecutionResult, error) {
	if b == nil || b.executor == nil {
		return rejected("broker not configured"), fmt.Errorf("%w: broker not configured", contracts.ErrInvalidArgument)
	}

	now := b.Now()
	g := &domaingrant.Grant{ExecutionGrant: req.Grant}

	// Structural + temporal validation
	if err := g.Validate(); err != nil {
		return rejected("invalid grant: " + err.Error()), fmt.Errorf("%w: %v", contracts.ErrGrantInvalid, err)
	}
	if !g.IsUsable(now) {
		if g.Status != contracts.ExecutionGrantStatusIssued {
			return rejected("grant not in ISSUED status"), contracts.ErrGrantInvalid
		}
		return rejected("grant expired"), contracts.ErrGrantExpired
	}

	// Context consistency (principal/task/capability/scope/project)
	// When the request carries correlation metadata we still require grant fields.
	if err := g.EnsureUsable(now,
		g.Principal.ID,
		g.ProjectID,
		g.ScopeID,
		g.TaskID,
		g.CapabilityID,
	); err != nil {
		return rejected(err.Error()), err
	}

	// Replay protection via nonce
	if g.Nonce != "" {
		b.mu.Lock()
		if _, seen := b.usedNonces[g.Nonce]; seen {
			b.mu.Unlock()
			return rejected("grant nonce already used (replay)"), contracts.ErrGrantInvalid
		}
		b.usedNonces[g.Nonce] = now
		b.mu.Unlock()
	}

	// Optional executor hint constraint
	if len(g.Constraints.AllowedExecutorHints) > 0 {
		// Phase 1G only has mock; accept if "mock" is listed or list empty handled above.
		allowed := false
		for _, h := range g.Constraints.AllowedExecutorHints {
			if h == "mock" || h == "*" {
				allowed = true
				break
			}
		}
		if !allowed {
			return rejected("executor hint not allowed by grant"), contracts.ErrGrantInvalid
		}
	}

	// Dispatch — only path to execution
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
