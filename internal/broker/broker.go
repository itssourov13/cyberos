package broker

import (
	"context"
	"errors"
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
	// Ledger is the shared durable grant lifecycle store (one-time claim,
	// revocation, execution state). Production wiring always sets it; without it
	// the Broker falls back to process-local nonce tracking (development only).
	Ledger GrantLedger
	// Signer verifies grant provenance. When set, grants not signed by the
	// issuer (forged, altered or unissued) are rejected before any other state
	// is touched. Production wiring always sets it.
	Signer *GrantSigner
}

// RevokeGrant marks an issued grant unusable for future Execute calls.
func (b *Broker) RevokeGrant(id contracts.ExecutionGrantID) {
	if b != nil {
		b.Signer.Revoke(id)
	}
}

// RevokeGrantDurable revokes a not-yet-claimed grant in the shared ledger so
// every instance observes it, and attributes the action to an actor.
func (b *Broker) RevokeGrantDurable(ctx context.Context, id contracts.ExecutionGrantID, actor contracts.PrincipalID, reason string) error {
	if b == nil || b.Ledger == nil {
		return fmt.Errorf("%w: grant ledger not configured", contracts.ErrInvalidArgument)
	}
	if err := b.Ledger.Revoke(ctx, id, actor, reason, b.Now()); err != nil {
		return err
	}
	if b.Signer != nil {
		b.Signer.Revoke(id)
	}
	return nil
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
	return b.ExecuteContext(context.Background(), req)
}

// ExecuteContext is Execute with caller cancellation/deadline propagated to the
// ledger and the execution host.
func (b *Broker) ExecuteContext(ctx context.Context, req contracts.ExecutionRequest) (contracts.ExecutionResult, error) {
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
	// Provenance: the grant must have been issued (signed) by this server.
	// Checked before nonce consumption so forged grants cannot burn nonces.
	if b.Signer != nil {
		if err := b.Signer.Verify(req.Grant); err != nil {
			return rejected("grant provenance check failed: " + err.Error()), err
		}
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

	if err := validateTargetBinding(req); err != nil {
		return rejected(err.Error()), err
	}

	// Phase 19A: pack binding before Host resolution.
	if g.PackContentDigest != "" {
		if req.PackContentDigest == "" {
			return rejected("pack-bound grant requires request pack_content_digest"), contracts.ErrGrantInvalid
		}
		if req.PackContentDigest != g.PackContentDigest {
			return rejected("pack content digest mismatch"), contracts.ErrGrantInvalid
		}
	}

	// Host resolution AFTER authorization. Unknown/invalid host → fail closed.
	// Resolve before claiming so a request for an unknown host cannot burn a grant.
	var host execution.ExecutionHost
	if b.Hosts != nil {
		h, err := b.Hosts.Resolve(req)
		if err != nil {
			return rejected("unknown execution host: " + err.Error()), err
		}
		host = h
	}

	// One-time claim. ALL pure validation (signature, window, binding, pack
	// digest, host) happens above so a request rejected for unrelated invalid
	// data never consumes a legitimate grant. The claim is the last gate before
	// dispatch and is atomic across processes when a durable ledger is wired.
	if err := b.claim(ctx, g, now); err != nil {
		return rejected(err.Error()), err
	}
	finish := func(res contracts.ExecutionResult, runErr error) error {
		if b.Ledger == nil {
			return nil
		}
		st := string(res.Status)
		if runErr != nil && (st == "" || st == string(contracts.ExecutionStatusSucceeded)) {
			st = string(contracts.ExecutionStatusFailed)
		}
		cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := b.Ledger.Complete(cctx, g.ID, st, b.Now()); err != nil {
			return fmt.Errorf("persist grant completion state: %w", err)
		}
		return nil
	}

	if host != nil {
		result, runErr := host.Execute(ctx, req)
		if completeErr := finish(result, runErr); completeErr != nil {
			return result, errors.Join(runErr, completeErr)
		}
		if runErr != nil {
			return result, runErr
		}
		if result.HostID == "" {
			result.HostID = string(host.ID())
		}
		return result, nil
	}

	// Legacy fallback (should not be used when New() registers Hosts)
	result, runErr := b.executor.Execute(req)
	if completeErr := finish(result, runErr); completeErr != nil {
		return result, errors.Join(runErr, completeErr)
	}
	if runErr != nil {
		return result, runErr
	}
	return result, nil
}

// claim consumes the grant exactly once. With a ledger the claim is durable and
// shared; otherwise it falls back to a process-local nonce set.
func (b *Broker) claim(ctx context.Context, g *domaingrant.Grant, now time.Time) error {
	if b.Ledger != nil {
		return b.Ledger.Claim(ctx, g.ID, g.Nonce, now)
	}
	if g.Nonce == "" {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, seen := b.usedNonces[g.Nonce]; seen {
		return fmt.Errorf("%w: grant nonce already used (replay)", contracts.ErrGrantInvalid)
	}
	b.usedNonces[g.Nonce] = now
	return nil
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

func validateTargetBinding(req contracts.ExecutionRequest) error {
	constraints := req.Grant.Constraints
	if len(constraints.AllowedAssetIDs) > 0 && len(constraints.AllowedTargets) == 0 {
		return fmt.Errorf("%w: asset-bound grant has no signed target binding", contracts.ErrGrantInvalid)
	}
	if len(constraints.AllowedTargets) == 0 {
		return nil
	}
	target := ""
	if raw, ok := req.Parameters["target"].(string); ok {
		target = raw
	}
	if target == "" {
		if raw, ok := req.Parameters["url"].(string); ok {
			target = raw
		}
	}
	if target == "" {
		return fmt.Errorf("%w: target-bound grant requires target or url parameter", contracts.ErrGrantInvalid)
	}
	for _, allowed := range constraints.AllowedTargets {
		if target == allowed {
			return nil
		}
	}
	return fmt.Errorf("%w: execution target does not match the signed grant target", contracts.ErrGrantInvalid)
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
