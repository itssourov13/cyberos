package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/storage/ports"
)

// AutonomyOutboxRepo persists autonomy_outbox (Phase 15C / 16B atomic claim).
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

// ClaimBatch atomically claims up to limit rows using FOR UPDATE SKIP LOCKED.
// At most one event per autonomy_run_id is returned (per-run ordering preserved).
// Concurrent dispatchers cannot both successfully own the same row.
func (r *AutonomyOutboxRepo) ClaimBatch(ctx context.Context, workerID string, limit int, lease time.Duration) ([]ports.AutonomyOutboxRecord, error) {
	if limit <= 0 {
		limit = 10
	}
	if workerID == "" {
		workerID = "worker"
	}
	now := time.Now().UTC()
	until := now.Add(lease)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("outbox claim begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	seenRun := make(map[string]struct{})
	var out []ports.AutonomyOutboxRecord

	for len(out) < limit {
		// Atomic single-row claim: lock + update in one statement via CTE.
		// SKIP LOCKED skips rows already locked by other transactions.
		row := tx.QueryRowContext(ctx, `
			WITH candidate AS (
				SELECT id
				FROM autonomy_outbox
				WHERE status NOT IN ('DELIVERED', 'DEAD_LETTER')
				  AND (
					(status = 'PENDING' AND available_at <= $1)
					OR (status = 'CLAIMED' AND (locked_until IS NULL OR locked_until < $1))
				  )
				  AND NOT (COALESCE(NULLIF(autonomy_run_id, ''), id) = ANY($4::text[]))
				ORDER BY autonomy_run_id, created_at, id
				FOR UPDATE SKIP LOCKED
				LIMIT 1
			)
			UPDATE autonomy_outbox o
			SET status = 'CLAIMED',
			    locked_by = $2,
			    locked_until = $3,
			    attempts = attempts + 1,
			    updated_at = $1
			FROM candidate c
			WHERE o.id = c.id
			RETURNING o.id, o.project_id, o.tenant_id, o.scope_id, o.autonomy_run_id, o.workflow_run_id, o.execution_id,
				o.event_type, o.schema_version, o.idempotency_key, o.event_id, o.payload_json, o.status,
				o.attempts, o.available_at, o.locked_until, o.locked_by, o.last_error, o.created_at, o.updated_at`,
			now, workerID, until, runKeys(seenRun),
		)
		rec, err := scanOutbox(row)
		if err == sql.ErrNoRows || err == contracts.ErrNotFound {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("outbox claim: %w", err)
		}
		rk := rec.AutonomyRunID
		if rk == "" {
			rk = rec.ID
		}
		seenRun[rk] = struct{}{}
		out = append(out, rec)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("outbox claim commit: %w", err)
	}
	return out, nil
}

func runKeys(seen map[string]struct{}) []string {
	if len(seen) == 0 {
		return []string{}
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	return keys
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
