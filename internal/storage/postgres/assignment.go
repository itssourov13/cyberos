package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"cyberos/internal/contracts"
	"cyberos/internal/storage/ports"
)

type AssignmentRepo struct{ db *DB }

func NewAssignmentRepo(db *DB) *AssignmentRepo { return &AssignmentRepo{db: db} }

func (r *AssignmentRepo) Create(ctx context.Context, a ports.AssignmentRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO execution_assignments (
			id, execution_id, workflow_run_id, project_id, tenant_id, host_id, worker_id,
			status, attempt, max_attempts, lease_until, created_at, dispatched_at, started_at,
			completed_at, cancelled_at, result_hash, result_status, error_message, version
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`,
		a.ID, a.ExecutionID, a.WorkflowRunID, string(a.ProjectID), string(a.TenantID), a.HostID, a.WorkerID,
		a.Status, a.Attempt, a.MaxAttempts, nullTimeWorker(a.LeaseUntil), a.CreatedAt.UTC(),
		nullTimeWorker(a.DispatchedAt), nullTimeWorker(a.StartedAt), nullTimeWorker(a.CompletedAt),
		nullTimeWorker(a.CancelledAt), a.ResultHash, a.ResultStatus, a.ErrorMessage, a.Version,
	)
	if err != nil {
		return fmt.Errorf("create assignment: %w", err)
	}
	return nil
}

func (r *AssignmentRepo) Get(ctx context.Context, projectID contracts.ProjectID, id string) (ports.AssignmentRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, execution_id, workflow_run_id, project_id, tenant_id, host_id, worker_id,
			status, attempt, max_attempts, lease_until, created_at, dispatched_at, started_at,
			completed_at, cancelled_at, result_hash, result_status, error_message, version
		FROM execution_assignments WHERE project_id=$1 AND id=$2`, string(projectID), id)
	return scanAssignment(row)
}

func (r *AssignmentRepo) GetByExecutionID(ctx context.Context, projectID contracts.ProjectID, executionID string) (ports.AssignmentRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, execution_id, workflow_run_id, project_id, tenant_id, host_id, worker_id,
			status, attempt, max_attempts, lease_until, created_at, dispatched_at, started_at,
			completed_at, cancelled_at, result_hash, result_status, error_message, version
		FROM execution_assignments WHERE project_id=$1 AND execution_id=$2`, string(projectID), executionID)
	return scanAssignment(row)
}

func (r *AssignmentRepo) Update(ctx context.Context, a ports.AssignmentRecord) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE execution_assignments SET
			status=$1, attempt=$2, lease_until=$3, dispatched_at=$4, started_at=$5,
			completed_at=$6, cancelled_at=$7, result_hash=$8, result_status=$9, error_message=$10,
			version=version+1
		WHERE project_id=$11 AND id=$12 AND version=$13`,
		a.Status, a.Attempt, nullTimeWorker(a.LeaseUntil), nullTimeWorker(a.DispatchedAt),
		nullTimeWorker(a.StartedAt), nullTimeWorker(a.CompletedAt), nullTimeWorker(a.CancelledAt),
		a.ResultHash, a.ResultStatus, a.ErrorMessage,
		string(a.ProjectID), a.ID, a.Version,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return contracts.ErrConflict
	}
	return nil
}

type assignmentScannable interface {
	Scan(dest ...any) error
}

func scanAssignment(row assignmentScannable) (ports.AssignmentRecord, error) {
	var a ports.AssignmentRecord
	var proj, ten string
	var lease, disp, start, comp, canc sql.NullTime
	err := row.Scan(&a.ID, &a.ExecutionID, &a.WorkflowRunID, &proj, &ten, &a.HostID, &a.WorkerID,
		&a.Status, &a.Attempt, &a.MaxAttempts, &lease, &a.CreatedAt, &disp, &start, &comp, &canc,
		&a.ResultHash, &a.ResultStatus, &a.ErrorMessage, &a.Version)
	if err == sql.ErrNoRows {
		return a, contracts.ErrNotFound
	}
	if err != nil {
		return a, err
	}
	a.ProjectID = contracts.ProjectID(proj)
	a.TenantID = contracts.TenantID(ten)
	if lease.Valid {
		a.LeaseUntil = lease.Time
	}
	if disp.Valid {
		a.DispatchedAt = disp.Time
	}
	if start.Valid {
		a.StartedAt = start.Time
	}
	if comp.Valid {
		a.CompletedAt = comp.Time
	}
	if canc.Valid {
		a.CancelledAt = canc.Time
	}
	return a, nil
}
