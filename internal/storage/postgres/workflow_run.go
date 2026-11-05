package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/storage/ports"
)

// WorkflowRunRepo persists workflow_runs (Phase 10C).
type WorkflowRunRepo struct{ db *DB }

func NewWorkflowRunRepo(db *DB) *WorkflowRunRepo { return &WorkflowRunRepo{db: db} }

func (r *WorkflowRunRepo) Create(ctx context.Context, rec ports.WorkflowRunRecord) error {
	if err := validateWorkflowRunRecord(rec); err != nil {
		return err
	}
	finished, steps, stt := workflowRunValues(rec)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO workflow_runs (
			id, workflow_id, version, project_id, scope_id, principal_id, tenant_id,
			state, correlation_id, causation_id, error_message, started_at, finished_at,
			steps_json, step_to_task_json
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		rec.ID, rec.WorkflowID, rec.Version, string(rec.ProjectID), string(rec.ScopeID),
		string(rec.PrincipalID), string(rec.TenantID), rec.State, rec.CorrelationID, rec.CausationID,
		rec.Error, rec.StartedAt.UTC(), finished, steps, stt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return contracts.ErrConflict
		}
		return fmt.Errorf("insert workflow_run: %w", err)
	}
	return nil
}

func (r *WorkflowRunRepo) Upsert(ctx context.Context, rec ports.WorkflowRunRecord) error {
	if err := validateWorkflowRunRecord(rec); err != nil {
		return err
	}
	finished, steps, stt := workflowRunValues(rec)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO workflow_runs (
			id, workflow_id, version, project_id, scope_id, principal_id, tenant_id,
			state, correlation_id, causation_id, error_message, started_at, finished_at,
			steps_json, step_to_task_json
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		ON CONFLICT (id) DO UPDATE SET
			state=EXCLUDED.state, error_message=EXCLUDED.error_message,
			finished_at=EXCLUDED.finished_at, steps_json=EXCLUDED.steps_json,
			step_to_task_json=EXCLUDED.step_to_task_json`,
		rec.ID, rec.WorkflowID, rec.Version, string(rec.ProjectID), string(rec.ScopeID),
		string(rec.PrincipalID), string(rec.TenantID), rec.State, rec.CorrelationID, rec.CausationID,
		rec.Error, rec.StartedAt.UTC(), finished, steps, stt,
	)
	if err != nil {
		return fmt.Errorf("upsert workflow_run: %w", err)
	}
	return nil
}

func validateWorkflowRunRecord(rec ports.WorkflowRunRecord) error {
	if rec.ID == "" || rec.WorkflowID == "" || rec.ProjectID == "" {
		return fmt.Errorf("%w: workflow run id/workflow_id/project_id required", contracts.ErrInvalidArgument)
	}
	return nil
}

func workflowRunValues(rec ports.WorkflowRunRecord) (finished any, steps, stepToTask []byte) {
	if !rec.FinishedAt.IsZero() {
		finished = rec.FinishedAt.UTC()
	}
	steps = rec.StepsJSON
	if steps == nil {
		steps = []byte("{}")
	}
	stepToTask = rec.StepToTaskJSON
	if stepToTask == nil {
		stepToTask = []byte("{}")
	}
	return
}

func (r *WorkflowRunRepo) GetByID(ctx context.Context, id string) (ports.WorkflowRunRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, workflow_id, version, project_id, scope_id, principal_id, tenant_id,
			state, correlation_id, causation_id, error_message, started_at, finished_at,
			steps_json, step_to_task_json
		FROM workflow_runs WHERE id=$1`, id)
	return scanWorkflowRun(row)
}

func (r *WorkflowRunRepo) ListByProject(ctx context.Context, projectID contracts.ProjectID, limit, offset int) ([]ports.WorkflowRunRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, workflow_id, version, project_id, scope_id, principal_id, tenant_id,
			state, correlation_id, causation_id, error_message, started_at, finished_at,
			steps_json, step_to_task_json
		FROM workflow_runs WHERE project_id=$1 ORDER BY started_at DESC, id ASC LIMIT $2 OFFSET $3`,
		string(projectID), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ports.WorkflowRunRecord
	for rows.Next() {
		rec, err := scanWorkflowRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

type wfScannable interface {
	Scan(dest ...any) error
}

func scanWorkflowRun(row wfScannable) (ports.WorkflowRunRecord, error) {
	var rec ports.WorkflowRunRecord
	var finished sql.NullTime
	var steps, stt []byte
	err := row.Scan(
		&rec.ID, &rec.WorkflowID, &rec.Version, &rec.ProjectID, &rec.ScopeID,
		&rec.PrincipalID, &rec.TenantID, &rec.State, &rec.CorrelationID, &rec.CausationID,
		&rec.Error, &rec.StartedAt, &finished, &steps, &stt,
	)
	if err == sql.ErrNoRows {
		return ports.WorkflowRunRecord{}, contracts.ErrNotFound
	}
	if err != nil {
		return ports.WorkflowRunRecord{}, err
	}
	if finished.Valid {
		rec.FinishedAt = finished.Time.UTC()
	}
	rec.StepsJSON = steps
	rec.StepToTaskJSON = stt
	if rec.StartedAt.IsZero() {
		rec.StartedAt = time.Time{}
	}
	return rec, nil
}
