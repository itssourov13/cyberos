package broker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cyberos/internal/contracts"
)

// GrantState is the durable lifecycle state of an issued grant.
type GrantState string

const (
	GrantStateIssued    GrantState = "ISSUED"
	GrantStateClaimed   GrantState = "CLAIMED"
	GrantStateCompleted GrantState = "COMPLETED"
	GrantStateFailed    GrantState = "FAILED"
	GrantStateRevoked   GrantState = "REVOKED"
)

// GrantRecord is the server-side record created when a grant is issued.
// It is the authority for one-time claim, revocation and execution state; the
// HMAC signature only proves provenance, never freshness.
type GrantRecord struct {
	GrantID       contracts.ExecutionGrantID
	Nonce         string
	PrincipalID   contracts.PrincipalID
	ProjectID     contracts.ProjectID
	ScopeID       contracts.ScopeID
	TaskID        contracts.TaskID
	CapabilityID  contracts.CapabilityID
	CorrelationID contracts.CorrelationID
	IssuedAt      time.Time
	ExpiresAt     time.Time

	State        GrantState
	ClaimedAt    *time.Time
	CompletedAt  *time.Time
	ResultStatus string
	RevokedAt    *time.Time
	RevokedBy    contracts.PrincipalID
	RevokeReason string
}

// GrantLedger is the shared, durable grant lifecycle store. Implementations MUST
// make Claim atomic across all callers and processes (PostgreSQL: a single
// conditional UPDATE plus UNIQUE grant_id and nonce).
type GrantLedger interface {
	// Record stores a newly issued grant. Duplicate grant id or nonce → ErrConflict.
	Record(ctx context.Context, r GrantRecord) error
	// Claim atomically moves ISSUED → CLAIMED. It fails with ErrGrantInvalid when
	// the grant is unknown, already claimed (replay), revoked, or expired.
	Claim(ctx context.Context, id contracts.ExecutionGrantID, nonce string, now time.Time) error
	// Complete records the execution outcome of a claimed grant.
	Complete(ctx context.Context, id contracts.ExecutionGrantID, resultStatus string, at time.Time) error
	// Revoke makes a not-yet-claimed grant permanently unusable. Revoking an
	// already revoked grant is a no-op; a claimed grant returns ErrConflict.
	Revoke(ctx context.Context, id contracts.ExecutionGrantID, actor contracts.PrincipalID, reason string, at time.Time) error
	// Get returns a grant record.
	Get(ctx context.Context, id contracts.ExecutionGrantID) (GrantRecord, error)
	// Cleanup deletes records whose grants expired before the cutoff and returns the count.
	Cleanup(ctx context.Context, expiredBefore time.Time) (int64, error)
}

// ClaimError converts a ledger-observed state into the canonical claim error.
func ClaimError(state GrantState, expired bool) error {
	switch {
	case state == GrantStateRevoked:
		return fmt.Errorf("%w: grant revoked", contracts.ErrGrantInvalid)
	case state != GrantStateIssued:
		return fmt.Errorf("%w: grant already used (replay)", contracts.ErrGrantInvalid)
	case expired:
		return contracts.ErrGrantExpired
	}
	return nil
}

// MemoryGrantLedger is a process-local ledger for development and tests. It is
// NOT cluster-safe and must never back a production deployment.
type MemoryGrantLedger struct {
	mu      sync.Mutex
	byID    map[contracts.ExecutionGrantID]*GrantRecord
	byNonce map[string]contracts.ExecutionGrantID
}

// NewMemoryGrantLedger constructs an empty in-memory ledger.
func NewMemoryGrantLedger() *MemoryGrantLedger {
	return &MemoryGrantLedger{
		byID:    map[contracts.ExecutionGrantID]*GrantRecord{},
		byNonce: map[string]contracts.ExecutionGrantID{},
	}
}

func (l *MemoryGrantLedger) Record(_ context.Context, r GrantRecord) error {
	if r.GrantID == "" || r.Nonce == "" {
		return fmt.Errorf("%w: grant id and nonce required", contracts.ErrInvalidArgument)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.byID[r.GrantID]; ok {
		return contracts.ErrConflict
	}
	if _, ok := l.byNonce[r.Nonce]; ok {
		return contracts.ErrConflict
	}
	r.State = GrantStateIssued
	cp := r
	l.byID[r.GrantID] = &cp
	l.byNonce[r.Nonce] = r.GrantID
	return nil
}

func (l *MemoryGrantLedger) Claim(_ context.Context, id contracts.ExecutionGrantID, nonce string, now time.Time) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	rec, ok := l.byID[id]
	if !ok || rec.Nonce != nonce {
		return fmt.Errorf("%w: grant was not issued by this control plane", contracts.ErrGrantInvalid)
	}
	if err := ClaimError(rec.State, !now.Before(rec.ExpiresAt)); err != nil {
		return err
	}
	t := now
	rec.State = GrantStateClaimed
	rec.ClaimedAt = &t
	return nil
}

func (l *MemoryGrantLedger) Complete(_ context.Context, id contracts.ExecutionGrantID, resultStatus string, at time.Time) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	rec, ok := l.byID[id]
	if !ok {
		return contracts.ErrNotFound
	}
	if rec.State != GrantStateClaimed {
		return contracts.ErrConflict
	}
	t := at
	rec.CompletedAt = &t
	rec.ResultStatus = resultStatus
	if resultStatus == string(contracts.ExecutionStatusSucceeded) {
		rec.State = GrantStateCompleted
	} else {
		rec.State = GrantStateFailed
	}
	return nil
}

func (l *MemoryGrantLedger) Revoke(_ context.Context, id contracts.ExecutionGrantID, actor contracts.PrincipalID, reason string, at time.Time) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	rec, ok := l.byID[id]
	if !ok {
		return contracts.ErrNotFound
	}
	switch rec.State {
	case GrantStateRevoked:
		return nil
	case GrantStateIssued:
		t := at
		rec.State = GrantStateRevoked
		rec.RevokedAt = &t
		rec.RevokedBy = actor
		rec.RevokeReason = reason
		return nil
	}
	return contracts.ErrConflict
}

func (l *MemoryGrantLedger) Get(_ context.Context, id contracts.ExecutionGrantID) (GrantRecord, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	rec, ok := l.byID[id]
	if !ok {
		return GrantRecord{}, contracts.ErrNotFound
	}
	return *rec, nil
}

func (l *MemoryGrantLedger) Cleanup(_ context.Context, before time.Time) (int64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	var n int64
	for id, rec := range l.byID {
		if rec.ExpiresAt.Before(before) {
			delete(l.byNonce, rec.Nonce)
			delete(l.byID, id)
			n++
		}
	}
	return n, nil
}

var _ GrantLedger = (*MemoryGrantLedger)(nil)
