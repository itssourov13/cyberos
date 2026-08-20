package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/task"
)

type TaskRepo struct {
	db *DB
}

func NewTaskRepo(db *DB) *TaskRepo {
	return &TaskRepo{db: db}
}

type scheduleDTO struct {
	NotBefore time.Time  `json:"not_before"`
	Priority  int        `json:"priority"`
	QueuedAt  *time.Time `json:"queued_at,omitempty"`
}

type retryDTO struct {
	MaxAttempts int        `json:"max_attempts"`
	BackoffNS   int64      `json:"backoff_ns"`
	Attempts    int        `json:"attempts"`
	LastError   string     `json:"last_error"`
	NextRetry   *time.Time `json:"next_retry,omitempty"`
}

const taskSelect = `
		SELECT id, project_id, scope_id, principal_id, capability_id, type, state, title, description,
			created_at, updated_at, metadata, depends_on, schedule, retry, timeout_ns, deadline
		FROM tasks`

func (r *TaskRepo) Create(ctx context.Context, t *task.Task) error {
	if err := t.Validate(); err != nil {
		return err
	}
	meta, _ := json.Marshal(t.Metadata)
	deps, _ := json.Marshal(t.DependsOn)
	var sched any
	if t.Schedule != nil {
		b, _ := json.Marshal(scheduleDTO{NotBefore: t.Schedule.NotBefore.UTC(), Priority: t.Schedule.Priority, QueuedAt: t.Schedule.QueuedAt})
		sched = b
	}
	var retry any
	if t.Retry != nil {
		b, _ := json.Marshal(retryDTO{
			MaxAttempts: t.Retry.Policy.MaxAttempts, BackoffNS: int64(t.Retry.Policy.Backoff),
			Attempts: t.Retry.Attempts, LastError: t.Retry.LastError, NextRetry: t.Retry.NextRetry,
		})
		retry = b
	}
	var deadline any
	if t.Deadline != nil {
		deadline = t.Deadline.UTC()
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tasks (
			id, project_id, scope_id, principal_id, capability_id, type, state, title, description,
			created_at, updated_at, metadata, depends_on, schedule, retry, timeout_ns, deadline
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		string(t.ID), string(t.ProjectID), string(t.ScopeID), string(t.PrincipalID), string(t.CapabilityID),
		string(t.Type), string(t.State), t.Title, t.Description,
		t.CreatedAt.UTC(), t.UpdatedAt.UTC(), meta, deps, sched, retry, int64(t.Timeout), deadline,
	)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	return nil
}

func (r *TaskRepo) GetByID(ctx context.Context, id contracts.TaskID) (*task.Task, error) {
	row := r.db.QueryRowContext(ctx, taskSelect+` WHERE id=$1`, string(id))
	return scanTaskRow(row)
}

func (r *TaskRepo) Update(ctx context.Context, t *task.Task) error {
	if err := t.Validate(); err != nil {
		return err
	}
	t.UpdatedAt = time.Now().UTC()
	meta, _ := json.Marshal(t.Metadata)
	deps, _ := json.Marshal(t.DependsOn)
	var sched any
	if t.Schedule != nil {
		b, _ := json.Marshal(scheduleDTO{NotBefore: t.Schedule.NotBefore.UTC(), Priority: t.Schedule.Priority, QueuedAt: t.Schedule.QueuedAt})
		sched = b
	}
	var retry any
	if t.Retry != nil {
		b, _ := json.Marshal(retryDTO{
			MaxAttempts: t.Retry.Policy.MaxAttempts, BackoffNS: int64(t.Retry.Policy.Backoff),
			Attempts: t.Retry.Attempts, LastError: t.Retry.LastError, NextRetry: t.Retry.NextRetry,
		})
		retry = b
	}
	var deadline any
	if t.Deadline != nil {
		deadline = t.Deadline.UTC()
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE tasks SET state=$2, title=$3, description=$4, updated_at=$5, metadata=$6,
			depends_on=$7, schedule=$8, retry=$9, timeout_ns=$10, deadline=$11
		WHERE id=$1`,
		string(t.ID), string(t.State), t.Title, t.Description, t.UpdatedAt, meta,
		deps, sched, retry, int64(t.Timeout), deadline,
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return contracts.ErrNotFound
	}
	return nil
}

func (r *TaskRepo) ListByScope(ctx context.Context, scopeID contracts.ScopeID, limit, offset int) ([]*task.Task, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, taskSelect+` WHERE scope_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		string(scopeID), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTaskRows(rows)
}

func (r *TaskRepo) ListByState(ctx context.Context, state contracts.TaskState, limit, offset int) ([]*task.Task, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, taskSelect+` WHERE state=$1 ORDER BY updated_at ASC LIMIT $2 OFFSET $3`,
		string(state), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTaskRows(rows)
}

func (r *TaskRepo) ListDueScheduled(ctx context.Context, now time.Time, limit int) ([]*task.Task, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, taskSelect+`
		WHERE state=$1 AND schedule IS NOT NULL
		  AND (schedule->>'not_before')::timestamptz <= $2
		ORDER BY COALESCE((schedule->>'priority')::int, 0) DESC, updated_at ASC
		LIMIT $3`,
		string(contracts.TaskStateScheduled), now.UTC(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTaskRows(rows)
}

func scanTaskRows(rows *sql.Rows) ([]*task.Task, error) {
	var out []*task.Task
	for rows.Next() {
		t, err := scanTaskRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func scanTaskRow(s scannable) (*task.Task, error) {
	var (
		id, projectID, scopeID, principalID, capabilityID string
		typ, state, title, desc                           string
		created, updated                                  time.Time
		metaRaw, depsRaw, schedRaw, retryRaw              []byte
		timeoutNS                                         int64
		deadline                                          sql.NullTime
	)
	if err := s.Scan(&id, &projectID, &scopeID, &principalID, &capabilityID, &typ, &state, &title, &desc,
		&created, &updated, &metaRaw, &depsRaw, &schedRaw, &retryRaw, &timeoutNS, &deadline); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, contracts.ErrNotFound
		}
		return nil, err
	}
	meta := map[string]string{}
	_ = json.Unmarshal(metaRaw, &meta)
	var deps []contracts.TaskID
	_ = json.Unmarshal(depsRaw, &deps)
	t := &task.Task{
		ID: contracts.TaskID(id), ProjectID: contracts.ProjectID(projectID),
		ScopeID: contracts.ScopeID(scopeID), PrincipalID: contracts.PrincipalID(principalID),
		CapabilityID: contracts.CapabilityID(capabilityID), Type: contracts.TaskType(typ),
		State: contracts.TaskState(state), Title: title, Description: desc,
		CreatedAt: created.UTC(), UpdatedAt: updated.UTC(), Metadata: meta, DependsOn: deps,
		Timeout: time.Duration(timeoutNS),
	}
	if len(schedRaw) > 0 && string(schedRaw) != "null" {
		var d scheduleDTO
		if json.Unmarshal(schedRaw, &d) == nil && !d.NotBefore.IsZero() {
			t.Schedule = &task.ScheduleSpec{NotBefore: d.NotBefore.UTC(), Priority: d.Priority, QueuedAt: d.QueuedAt}
		}
	}
	if len(retryRaw) > 0 && string(retryRaw) != "null" {
		var d retryDTO
		if json.Unmarshal(retryRaw, &d) == nil && d.MaxAttempts >= 1 {
			t.Retry = &task.RetryState{
				Policy:   task.RetryPolicy{MaxAttempts: d.MaxAttempts, Backoff: time.Duration(d.BackoffNS)},
				Attempts: d.Attempts, LastError: d.LastError, NextRetry: d.NextRetry,
			}
		}
	}
	if deadline.Valid {
		d := deadline.Time.UTC()
		t.Deadline = &d
	}
	return t, nil
}

func (r *TaskRepo) List(ctx context.Context, limit, offset int) ([]*task.Task, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, taskSelect+` ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTaskRows(rows)
}

// TryClaimAtomically claims PENDING tasks with free or expired leases using
// UPDATE ... WHERE ... RETURNING for cross-worker exclusivity.
func (r *TaskRepo) TryClaimAtomically(ctx context.Context, worker string, leaseUntil, now time.Time, limit int) ([]*task.Task, error) {
	if limit <= 0 {
		limit = 1
	}
	// Select candidate IDs under row lock, then update.
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(ctx, `
		SELECT id FROM tasks
		WHERE state = $1
		  AND (lease_until IS NULL OR lease_until < $2)
		ORDER BY updated_at ASC
		LIMIT $3
		FOR UPDATE SKIP LOCKED`,
		string(contracts.TaskStatePending), now.UTC(), limit)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var out []*task.Task
	for _, id := range ids {
		_, err := tx.ExecContext(ctx, `
			UPDATE tasks SET worker_id=$2, lease_until=$3, updated_at=$4
			WHERE id=$1 AND state=$5 AND (lease_until IS NULL OR lease_until < $4)`,
			id, worker, leaseUntil.UTC(), now.UTC(), string(contracts.TaskStatePending))
		if err != nil {
			return nil, err
		}
		row := tx.QueryRowContext(ctx, taskSelect+` WHERE id=$1`, id)
		tk, err := scanTaskRow(row)
		if err != nil {
			return nil, err
		}
		if tk.Metadata == nil {
			tk.Metadata = map[string]string{}
		}
		tk.Metadata["runner_worker"] = worker
		tk.Metadata["runner_lease_until"] = leaseUntil.UTC().Format(time.RFC3339)
		out = append(out, tk)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

// ClearExpiredLeases clears worker_id/lease_until on expired leases.
func (r *TaskRepo) ClearExpiredLeases(ctx context.Context, now time.Time) (int, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE tasks SET worker_id=NULL, lease_until=NULL, updated_at=$1
		WHERE lease_until IS NOT NULL AND lease_until < $1`, now.UTC())
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}
