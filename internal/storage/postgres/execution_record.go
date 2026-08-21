package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cyberos/internal/contracts"
)

// ExecutionRecordRepo persists execution_records (Phase 9D).
type ExecutionRecordRepo struct{ db *DB }

func NewExecutionRecordRepo(db *DB) *ExecutionRecordRepo {
	return &ExecutionRecordRepo{db: db}
}

func (r *ExecutionRecordRepo) Upsert(ctx context.Context, rec contracts.ExecutionRecord) error {
	if err := rec.Validate(); err != nil {
		return err
	}
	metrics, _ := json.Marshal(rec.Metrics)
	var finished any
	if !rec.FinishedAt.IsZero() {
		finished = rec.FinishedAt.UTC()
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO execution_records (
			id, state, task_id, grant_id, principal_id, principal_type, project_id, scope_id,
			capability_id, attempt, started_at, finished_at, duration_ms, outcome, failure_reason,
			executor_id, correlation_id, causation_id, idempotency_key, tool_id, pack_version, tenant_id, metrics
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)
		ON CONFLICT (id) DO UPDATE SET
			state=EXCLUDED.state, finished_at=EXCLUDED.finished_at, duration_ms=EXCLUDED.duration_ms,
			outcome=EXCLUDED.outcome, failure_reason=EXCLUDED.failure_reason, executor_id=EXCLUDED.executor_id,
			metrics=EXCLUDED.metrics, attempt=EXCLUDED.attempt`,
		string(rec.ID), string(rec.State), string(rec.TaskID), string(rec.GrantID),
		string(rec.Principal.ID), string(rec.Principal.Type),
		string(rec.ProjectID), string(rec.ScopeID), string(rec.CapabilityID), rec.Attempt,
		rec.StartedAt.UTC(), finished, rec.Duration.Milliseconds(), string(rec.Outcome), rec.FailureReason,
		rec.ExecutorID, string(rec.CorrelationID), rec.CausationID, rec.IdempotencyKey,
		rec.ToolID, rec.PackVersion, string(rec.TenantID), metrics,
	)
	if err != nil {
		return fmt.Errorf("upsert execution_record: %w", err)
	}
	return nil
}

func (r *ExecutionRecordRepo) GetByID(ctx context.Context, id contracts.ExecutionID) (contracts.ExecutionRecord, error) {
	row := r.db.QueryRowContext(ctx, execSelect+` WHERE id=$1`, string(id))
	return scanExecRecord(row)
}

func (r *ExecutionRecordRepo) GetByIdempotencyKey(ctx context.Context, key string) (contracts.ExecutionRecord, error) {
	if key == "" {
		return contracts.ExecutionRecord{}, contracts.ErrNotFound
	}
	row := r.db.QueryRowContext(ctx, execSelect+` WHERE idempotency_key=$1`, key)
	return scanExecRecord(row)
}

func (r *ExecutionRecordRepo) ListByTask(ctx context.Context, taskID contracts.TaskID, limit, offset int) ([]contracts.ExecutionRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, execSelect+` WHERE task_id=$1 ORDER BY started_at DESC LIMIT $2 OFFSET $3`,
		string(taskID), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExecRecords(rows)
}

func (r *ExecutionRecordRepo) ListByProject(ctx context.Context, projectID contracts.ProjectID, limit, offset int) ([]contracts.ExecutionRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, execSelect+` WHERE project_id=$1 ORDER BY started_at DESC LIMIT $2 OFFSET $3`,
		string(projectID), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExecRecords(rows)
}

const execSelect = `
SELECT id, state, task_id, grant_id, principal_id, principal_type, project_id, scope_id,
	capability_id, attempt, started_at, finished_at, duration_ms, outcome, failure_reason,
	executor_id, correlation_id, causation_id, idempotency_key, tool_id, pack_version, tenant_id, metrics
FROM execution_records`

func scanExecRecords(rows *sql.Rows) ([]contracts.ExecutionRecord, error) {
	var out []contracts.ExecutionRecord
	for rows.Next() {
		r, err := scanExecRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func scanExecRecord(s scannable) (contracts.ExecutionRecord, error) {
	var (
		id, state, taskID, grantID, principalID, principalType string
		projectID, scopeID, capID, outcome, fail, execID       string
		corr, caus, idem, toolID, packVer, tenant              string
		attempt                                                int
		durMs                                                  int64
		started                                                time.Time
		finished                                               sql.NullTime
		metricsRaw                                             []byte
	)
	err := s.Scan(&id, &state, &taskID, &grantID, &principalID, &principalType, &projectID, &scopeID,
		&capID, &attempt, &started, &finished, &durMs, &outcome, &fail,
		&execID, &corr, &caus, &idem, &toolID, &packVer, &tenant, &metricsRaw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contracts.ExecutionRecord{}, contracts.ErrNotFound
		}
		return contracts.ExecutionRecord{}, err
	}
	metrics := map[string]any{}
	_ = json.Unmarshal(metricsRaw, &metrics)
	rec := contracts.ExecutionRecord{
		ID: contracts.ExecutionID(id), State: contracts.ExecutionLifecycleState(state),
		TaskID: contracts.TaskID(taskID), GrantID: contracts.ExecutionGrantID(grantID),
		Principal: contracts.PrincipalRef{ID: contracts.PrincipalID(principalID), Type: contracts.PrincipalType(principalType)},
		ProjectID: contracts.ProjectID(projectID), ScopeID: contracts.ScopeID(scopeID),
		CapabilityID: contracts.CapabilityID(capID), Attempt: attempt,
		StartedAt: started.UTC(), Duration: time.Duration(durMs) * time.Millisecond,
		Outcome: contracts.ExecutionStatus(outcome), FailureReason: fail, ExecutorID: execID,
		CorrelationID: contracts.CorrelationID(corr), CausationID: caus, IdempotencyKey: idem,
		ToolID: toolID, PackVersion: packVer, TenantID: contracts.TenantID(tenant), Metrics: metrics,
	}
	if finished.Valid {
		rec.FinishedAt = finished.Time.UTC()
	}
	return rec, nil
}
