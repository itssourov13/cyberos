package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/storage/ports"
)

// AutonomyOutboxRepo persists autonomy_outbox (Phase 15C).
type AutonomyOutboxRepo struct{ db *DB }

func NewAutonomyOutboxRepo(db *DB) *AutonomyOutboxRepo { return &AutonomyOutboxRepo{db: db} }

func (r *AutonomyOutboxRepo) Enqueue(ctx context.Context, rec ports.AutonomyOutboxRecord) error {
	if rec.ID == "" || rec.ProjectID == "" || rec.IdempotencyKey == "" {
		return fmt.Errorf("%w: id, project_id, idempotency_key required", contracts.ErrInvalidArgument)
	}
	payload := rec.PayloadJSON
	if payload == nil {
		payload = []byte("{}")
	}
	status := rec.Status
	if status == "" {
		status = ports.OutboxPending
	}
	now := time.Now().UTC()
	avail := rec.AvailableAt
	if avail.IsZero() {
		avail = now
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO autonomy_outbox (
			id, project_id, tenant_id, scope_id, autonomy_run_id, workflow_run_id, execution_id,
			event_type, schema_version, idempotency_key, event_id, payload_json, status,
			attempts, available_at, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,0,$14,$15,$15)
		ON CONFLICT (project_id, idempotency_key) DO NOTHING`,
		rec.ID, string(rec.ProjectID), string(rec.TenantID), string(rec.ScopeID),
		rec.AutonomyRunID, rec.WorkflowRunID, rec.ExecutionID,
		rec.EventType, rec.SchemaVersion, rec.IdempotencyKey, rec.EventID, payload, status,
		avail, now,
	)
	if err != nil {
		return fmt.Errorf("enqueue autonomy_outbox: %w", err)
	}
	return nil
}

func (r *AutonomyOutboxRepo) ClaimBatch(ctx context.Context, workerID string, limit int, lease time.Duration) ([]ports.AutonomyOutboxRecord, error) {
	if limit <= 0 {
		limit = 10
	}
	now := time.Now().UTC()
	until := now.Add(lease)
	// Simple claim: select then update (good enough for Phase 15C; not full SKIP LOCKED)
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, tenant_id, scope_id, autonomy_run_id, workflow_run_id, execution_id,
			event_type, schema_version, idempotency_key, event_id, payload_json, status,
			attempts, available_at, locked_until, locked_by, last_error, created_at, updated_at
		FROM autonomy_outbox
		WHERE (status = 'PENDING' AND available_at <= $1)
		   OR (status = 'CLAIMED' AND (locked_until IS NULL OR locked_until < $1))
		ORDER BY autonomy_run_id, created_at
		LIMIT $2`, now, limit*3)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var candidates []ports.AutonomyOutboxRecord
	for rows.Next() {
		rec, err := scanOutbox(rows)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, rec)
	}
	seenRun := map[string]bool{}
	var out []ports.AutonomyOutboxRecord
	for _, rec := range candidates {
		if len(out) >= limit {
			break
		}
		rk := rec.AutonomyRunID
		if rk == "" {
			rk = rec.ID
		}
		if seenRun[rk] {
			continue
		}
		res, err := r.db.ExecContext(ctx, `
			UPDATE autonomy_outbox SET status='CLAIMED', locked_by=$1, locked_until=$2,
				attempts=attempts+1, updated_at=$2
			WHERE id=$3 AND (
				(status='PENDING' AND available_at <= $2)
				OR (status='CLAIMED' AND (locked_until IS NULL OR locked_until < $2))
			)`, workerID, until, rec.ID)
		if err != nil {
			continue
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			continue
		}
		seenRun[rk] = true
		rec.Status = ports.OutboxClaimed
		rec.LockedBy = workerID
		rec.LockedUntil = until
		rec.Attempts++
		out = append(out, rec)
	}
	return out, nil
}

func (r *AutonomyOutboxRepo) MarkDelivered(ctx context.Context, id string, workerID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE autonomy_outbox SET status='DELIVERED', locked_by='', locked_until=NULL, updated_at=$1
		WHERE id=$2 AND (locked_by=$3 OR locked_by='')`, time.Now().UTC(), id, workerID)
	return err
}

func (r *AutonomyOutboxRepo) MarkRetry(ctx context.Context, id string, workerID string, nextAttempt time.Time, reason string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE autonomy_outbox SET status='PENDING', available_at=$1, locked_by='', locked_until=NULL,
			last_error=$2, updated_at=$3 WHERE id=$4`, nextAttempt.UTC(), reason, time.Now().UTC(), id)
	return err
}

func (r *AutonomyOutboxRepo) MarkDeadLetter(ctx context.Context, id string, workerID string, reason string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE autonomy_outbox SET status='DEAD_LETTER', locked_by='', locked_until=NULL,
			last_error=$1, updated_at=$2 WHERE id=$3`, reason, time.Now().UTC(), id)
	return err
}

func (r *AutonomyOutboxRepo) Get(ctx context.Context, projectID contracts.ProjectID, id string) (ports.AutonomyOutboxRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, tenant_id, scope_id, autonomy_run_id, workflow_run_id, execution_id,
			event_type, schema_version, idempotency_key, event_id, payload_json, status,
			attempts, available_at, locked_until, locked_by, last_error, created_at, updated_at
		FROM autonomy_outbox WHERE id=$1 AND project_id=$2`, id, string(projectID))
	return scanOutbox(row)
}

type outboxScanner interface {
	Scan(dest ...any) error
}

func scanOutbox(row outboxScanner) (ports.AutonomyOutboxRecord, error) {
	var rec ports.AutonomyOutboxRecord
	var proj, ten, scope string
	var lockedUntil sql.NullTime
	var payload []byte
	err := row.Scan(
		&rec.ID, &proj, &ten, &scope, &rec.AutonomyRunID, &rec.WorkflowRunID, &rec.ExecutionID,
		&rec.EventType, &rec.SchemaVersion, &rec.IdempotencyKey, &rec.EventID, &payload, &rec.Status,
		&rec.Attempts, &rec.AvailableAt, &lockedUntil, &rec.LockedBy, &rec.LastError, &rec.CreatedAt, &rec.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return rec, contracts.ErrNotFound
	}
	if err != nil {
		return rec, err
	}
	rec.ProjectID = contracts.ProjectID(proj)
	rec.TenantID = contracts.TenantID(ten)
	rec.ScopeID = contracts.ScopeID(scope)
	rec.PayloadJSON = payload
	if lockedUntil.Valid {
		rec.LockedUntil = lockedUntil.Time
	}
	return rec, nil
}
