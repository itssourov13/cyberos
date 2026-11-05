package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"

	"cyberos/internal/broker"
	"cyberos/internal/contracts"
)

// GrantLedger is the PostgreSQL broker.GrantLedger. Every state change is a
// single conditional statement, so claim/revoke are atomic across processes.
type GrantLedger struct{ db *DB }

// NewGrantLedger constructs the durable grant ledger.
func NewGrantLedger(db *DB) *GrantLedger { return &GrantLedger{db: db} }

var _ broker.GrantLedger = (*GrantLedger)(nil)

func (l *GrantLedger) Record(ctx context.Context, r broker.GrantRecord) error {
	if r.GrantID == "" || r.Nonce == "" {
		return fmt.Errorf("%w: grant id and nonce required", contracts.ErrInvalidArgument)
	}
	_, err := l.db.ExecContext(ctx, `
		INSERT INTO execution_grant_ledger (
			grant_id, nonce, principal_id, project_id, scope_id, task_id, capability_id,
			correlation_id, issued_at, expires_at, state
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'ISSUED')`,
		string(r.GrantID), r.Nonce, string(r.PrincipalID), string(r.ProjectID), string(r.ScopeID),
		string(r.TaskID), string(r.CapabilityID), string(r.CorrelationID),
		r.IssuedAt.UTC(), r.ExpiresAt.UTC())
	if err != nil {
		var pe *pq.Error
		if errors.As(err, &pe) && pe.Code == "23505" {
			return contracts.ErrConflict
		}
		return fmt.Errorf("record grant: %w", err)
	}
	return nil
}

// Claim atomically moves ISSUED -> CLAIMED. Exactly one concurrent caller wins.
func (l *GrantLedger) Claim(ctx context.Context, id contracts.ExecutionGrantID, nonce string, now time.Time) error {
	res, err := l.db.ExecContext(ctx, `
		UPDATE execution_grant_ledger
		   SET state='CLAIMED', claimed_at=$3
		 WHERE grant_id=$1 AND nonce=$2 AND state='ISSUED' AND expires_at > $3`,
		string(id), nonce, now.UTC())
	if err != nil {
		return fmt.Errorf("claim grant: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 1 {
		return nil
	}
	// Lost the race or not claimable: classify for a precise error.
	var state, storedNonce string
	var expires time.Time
	err = l.db.QueryRowContext(ctx,
		`SELECT state, expires_at, nonce FROM execution_grant_ledger WHERE grant_id=$1`, string(id)).
		Scan(&state, &expires, &storedNonce)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && storedNonce != nonce) {
		return fmt.Errorf("%w: grant was not issued by this control plane", contracts.ErrGrantInvalid)
	}
	if err != nil {
		return fmt.Errorf("claim grant: %w", err)
	}
	if cerr := broker.ClaimError(broker.GrantState(state), !now.Before(expires)); cerr != nil {
		return cerr
	}
	return fmt.Errorf("%w: grant not claimable", contracts.ErrGrantInvalid)
}

func (l *GrantLedger) Complete(ctx context.Context, id contracts.ExecutionGrantID, resultStatus string, at time.Time) error {
	next := "FAILED"
	if resultStatus == string(contracts.ExecutionStatusSucceeded) {
		next = "COMPLETED"
	}
	res, err := l.db.ExecContext(ctx, `
		UPDATE execution_grant_ledger
		   SET state=$2, completed_at=$3, result_status=$4
		 WHERE grant_id=$1 AND state='CLAIMED'`,
		string(id), next, at.UTC(), resultStatus)
	if err != nil {
		return fmt.Errorf("complete grant: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 1 {
		return nil
	}
	var one int
	err = l.db.QueryRowContext(ctx, `SELECT 1 FROM execution_grant_ledger WHERE grant_id=$1`, string(id)).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return contracts.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("complete grant: %w", err)
	}
	return contracts.ErrConflict
}

func (l *GrantLedger) Revoke(ctx context.Context, id contracts.ExecutionGrantID, actor contracts.PrincipalID, reason string, at time.Time) error {
	res, err := l.db.ExecContext(ctx, `
		UPDATE execution_grant_ledger
		   SET state='REVOKED', revoked_at=$2, revoked_by=$3, revoke_reason=$4
		 WHERE grant_id=$1 AND state='ISSUED'`,
		string(id), at.UTC(), string(actor), reason)
	if err != nil {
		return fmt.Errorf("revoke grant: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 1 {
		return nil
	}
	var state string
	err = l.db.QueryRowContext(ctx, `SELECT state FROM execution_grant_ledger WHERE grant_id=$1`, string(id)).Scan(&state)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return contracts.ErrNotFound
	case err != nil:
		return fmt.Errorf("revoke grant: %w", err)
	case state == string(broker.GrantStateRevoked):
		return nil // idempotent
	}
	return contracts.ErrConflict
}

func (l *GrantLedger) Get(ctx context.Context, id contracts.ExecutionGrantID) (broker.GrantRecord, error) {
	var (
		r                                broker.GrantRecord
		gid, pid, proj, sc, task, cp, co string
		state, revBy                     string
		claimed, completed, revoked      sql.NullTime
	)
	err := l.db.QueryRowContext(ctx, `
		SELECT grant_id, nonce, principal_id, project_id, scope_id, task_id, capability_id, correlation_id,
		       issued_at, expires_at, state, claimed_at, completed_at, result_status,
		       revoked_at, revoked_by, revoke_reason
		  FROM execution_grant_ledger WHERE grant_id=$1`, string(id)).
		Scan(&gid, &r.Nonce, &pid, &proj, &sc, &task, &cp, &co,
			&r.IssuedAt, &r.ExpiresAt, &state, &claimed, &completed, &r.ResultStatus,
			&revoked, &revBy, &r.RevokeReason)
	if errors.Is(err, sql.ErrNoRows) {
		return broker.GrantRecord{}, contracts.ErrNotFound
	}
	if err != nil {
		return broker.GrantRecord{}, fmt.Errorf("get grant: %w", err)
	}
	r.GrantID = contracts.ExecutionGrantID(gid)
	r.PrincipalID = contracts.PrincipalID(pid)
	r.ProjectID = contracts.ProjectID(proj)
	r.ScopeID = contracts.ScopeID(sc)
	r.TaskID = contracts.TaskID(task)
	r.CapabilityID = contracts.CapabilityID(cp)
	r.CorrelationID = contracts.CorrelationID(co)
	r.RevokedBy = contracts.PrincipalID(revBy)
	r.State = broker.GrantState(state)
	if claimed.Valid {
		t := claimed.Time
		r.ClaimedAt = &t
	}
	if completed.Valid {
		t := completed.Time
		r.CompletedAt = &t
	}
	if revoked.Valid {
		t := revoked.Time
		r.RevokedAt = &t
	}
	return r, nil
}

func (l *GrantLedger) Cleanup(ctx context.Context, expiredBefore time.Time) (int64, error) {
	res, err := l.db.ExecContext(ctx, `DELETE FROM execution_grant_ledger
		WHERE expires_at < $1 AND state IN ('COMPLETED','FAILED','REVOKED')`, expiredBefore.UTC())
	if err != nil {
		return 0, fmt.Errorf("cleanup grants: %w", err)
	}
	return res.RowsAffected()
}
