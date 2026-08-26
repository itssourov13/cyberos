package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/storage/ports"
)

// AutonomyRunRepo persists autonomy_runs (Phase 14A / 16C / 16D).
type AutonomyRunRepo struct{ db *DB }

func NewAutonomyRunRepo(db *DB) *AutonomyRunRepo { return &AutonomyRunRepo{db: db} }

func (r *AutonomyRunRepo) Create(ctx context.Context, rec ports.AutonomyRunRecord) error {
	if rec.ID == "" || rec.ProjectID == "" {
		return fmt.Errorf("%w: id and project_id required", contracts.ErrInvalidArgument)
	}
	if rec.Version == 0 {
		rec.Version = 1
	}
	policy := rec.PolicyJSON
	if policy == nil {
		policy = []byte("{}")
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO autonomy_runs (
			id, project_id, tenant_id, scope_id, principal_id, correlation_id, goal, mode,
			policy_version, policy_json, state, iteration, steps_consumed,
			llm_calls_used, no_progress_count, last_progress_key, next_allowed_at, approval_state,
			analysis_id, plan_id, parent_plan_id, workflow_def_id, workflow_run_id,
			last_decision, stop_reason, version, started_at, updated_at, finished_at, cancelled_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30
		)`,
		rec.ID, string(rec.ProjectID), string(rec.TenantID), string(rec.ScopeID), string(rec.PrincipalID),
		string(rec.CorrelationID), rec.Goal, rec.Mode, rec.PolicyVersion, policy, rec.State,
		rec.Iteration, rec.StepsConsumed, rec.LLMCallsUsed, rec.NoProgressCount, rec.LastProgressKey,
		nullTimeAutonomy(rec.NextAllowedAt), rec.ApprovalState,
		rec.AnalysisID, rec.PlanID, rec.ParentPlanID, rec.WorkflowDefID, rec.WorkflowRunID,
		rec.LastDecision, rec.StopReason, rec.Version,
		rec.StartedAt.UTC(), rec.UpdatedAt.UTC(), nullTimeAutonomy(rec.FinishedAt), nullTimeAutonomy(rec.CancelledAt),
	)
	if err != nil {
		return fmt.Errorf("create autonomy_run: %w", err)
	}
	return nil
}

func (r *AutonomyRunRepo) Get(ctx context.Context, projectID contracts.ProjectID, runID string) (ports.AutonomyRunRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, tenant_id, scope_id, principal_id, correlation_id, goal, mode,
			policy_version, policy_json, state, iteration, steps_consumed,
			llm_calls_used, no_progress_count, last_progress_key, next_allowed_at, approval_state,
			analysis_id, plan_id, parent_plan_id, workflow_def_id, workflow_run_id,
			last_decision, stop_reason, version, started_at, updated_at, finished_at, cancelled_at
		FROM autonomy_runs WHERE id=$1 AND project_id=$2`, runID, string(projectID))
	return scanAutonomyRun(row)
}

func (r *AutonomyRunRepo) Update(ctx context.Context, rec ports.AutonomyRunRecord) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE autonomy_runs SET
			state=$1, iteration=$2, steps_consumed=$3, llm_calls_used=$4, no_progress_count=$5,
			last_progress_key=$6, next_allowed_at=$7, approval_state=$8, analysis_id=$9, plan_id=$10, parent_plan_id=$11,
			workflow_def_id=$12, workflow_run_id=$13, last_decision=$14, stop_reason=$15,
			version=version+1, updated_at=$16, finished_at=$17, cancelled_at=$18
		WHERE id=$19 AND project_id=$20 AND version=$21`,
		rec.State, rec.Iteration, rec.StepsConsumed, rec.LLMCallsUsed, rec.NoProgressCount,
		rec.LastProgressKey, nullTimeAutonomy(rec.NextAllowedAt), rec.ApprovalState,
		rec.AnalysisID, rec.PlanID, rec.ParentPlanID, rec.WorkflowDefID, rec.WorkflowRunID,
		rec.LastDecision, rec.StopReason,
		time.Now().UTC(), nullTimeAutonomy(rec.FinishedAt), nullTimeAutonomy(rec.CancelledAt),
		rec.ID, string(rec.ProjectID), rec.Version,
	)
	if err != nil {
		return fmt.Errorf("update autonomy_run: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%w: autonomy run version conflict or not found", contracts.ErrConflict)
	}
	return nil
}

func (r *AutonomyRunRepo) Cancel(ctx context.Context, projectID contracts.ProjectID, runID string) error {
	now := time.Now().UTC()
	res, err := r.db.ExecContext(ctx, `
		UPDATE autonomy_runs SET state='CANCELLED', stop_reason='operator_cancelled',
			cancelled_at=$1, finished_at=$1, updated_at=$1, version=version+1
		WHERE id=$2 AND project_id=$3 AND state NOT IN ('CANCELLED','SUCCEEDED','FAILED')`,
		now, runID, string(projectID),
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		_, err2 := r.Get(ctx, projectID, runID)
		return err2
	}
	return nil
}

func (r *AutonomyRunRepo) ListResumable(ctx context.Context, projectID contracts.ProjectID) ([]ports.AutonomyRunRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, tenant_id, scope_id, principal_id, correlation_id, goal, mode,
			policy_version, policy_json, state, iteration, steps_consumed,
			llm_calls_used, no_progress_count, last_progress_key, next_allowed_at, approval_state,
			analysis_id, plan_id, parent_plan_id, workflow_def_id, workflow_run_id,
			last_decision, stop_reason, version, started_at, updated_at, finished_at, cancelled_at
		FROM autonomy_runs WHERE project_id=$1 AND state IN ('RUNNING','WAITING','PENDING')`, string(projectID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ports.AutonomyRunRecord
	for rows.Next() {
		rec, err := scanAutonomyRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

type autonomyScannable interface {
	Scan(dest ...any) error
}

func scanAutonomyRun(row autonomyScannable) (ports.AutonomyRunRecord, error) {
	var rec ports.AutonomyRunRecord
	var proj, ten, scope, prin, corr string
	var finished, cancelled, nextAllowed sql.NullTime
	var policy []byte
	err := row.Scan(
		&rec.ID, &proj, &ten, &scope, &prin, &corr, &rec.Goal, &rec.Mode,
		&rec.PolicyVersion, &policy, &rec.State, &rec.Iteration, &rec.StepsConsumed,
		&rec.LLMCallsUsed, &rec.NoProgressCount, &rec.LastProgressKey, &nextAllowed, &rec.ApprovalState,
		&rec.AnalysisID, &rec.PlanID, &rec.ParentPlanID, &rec.WorkflowDefID, &rec.WorkflowRunID,
		&rec.LastDecision, &rec.StopReason, &rec.Version, &rec.StartedAt, &rec.UpdatedAt, &finished, &cancelled,
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
	rec.PrincipalID = contracts.PrincipalID(prin)
	rec.CorrelationID = contracts.CorrelationID(corr)
	rec.PolicyJSON = policy
	if finished.Valid {
		rec.FinishedAt = finished.Time
	}
	if cancelled.Valid {
		rec.CancelledAt = cancelled.Time
	}
	if nextAllowed.Valid {
		rec.NextAllowedAt = nextAllowed.Time
	}
	return rec, nil
}

func nullTimeAutonomy(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC()
}
