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

func (r *TaskRepo) Create(ctx context.Context, t *task.Task) error {
	if err := t.Validate(); err != nil {
		return err
	}
	meta, _ := json.Marshal(t.Metadata)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tasks (id, project_id, scope_id, principal_id, capability_id, type, state, title, description, created_at, updated_at, metadata)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		string(t.ID), string(t.ProjectID), string(t.ScopeID), string(t.PrincipalID), string(t.CapabilityID),
		string(t.Type), string(t.State), t.Title, t.Description,
		t.CreatedAt.UTC(), t.UpdatedAt.UTC(), meta,
	)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	return nil
}

func (r *TaskRepo) GetByID(ctx context.Context, id contracts.TaskID) (*task.Task, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, scope_id, principal_id, capability_id, type, state, title, description, created_at, updated_at, metadata
		FROM tasks WHERE id=$1`, string(id))
	return scanTask(row)
}

func (r *TaskRepo) Update(ctx context.Context, t *task.Task) error {
	if err := t.Validate(); err != nil {
		return err
	}
	t.UpdatedAt = time.Now().UTC()
	meta, _ := json.Marshal(t.Metadata)
	res, err := r.db.ExecContext(ctx, `
		UPDATE tasks SET state=$2, title=$3, description=$4, updated_at=$5, metadata=$6
		WHERE id=$1`,
		string(t.ID), string(t.State), t.Title, t.Description, t.UpdatedAt, meta,
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
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, scope_id, principal_id, capability_id, type, state, title, description, created_at, updated_at, metadata
		FROM tasks WHERE scope_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		string(scopeID), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*task.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func scanTask(s scannable) (*task.Task, error) {
	var (
		id, projectID, scopeID, principalID, capabilityID string
		typ, state, title, desc                           string
		created, updated                                  time.Time
		metaRaw                                           []byte
	)
	if err := s.Scan(&id, &projectID, &scopeID, &principalID, &capabilityID, &typ, &state, &title, &desc, &created, &updated, &metaRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, contracts.ErrNotFound
		}
		return nil, err
	}
	meta := map[string]string{}
	_ = json.Unmarshal(metaRaw, &meta)
	return &task.Task{
		ID:           contracts.TaskID(id),
		ProjectID:    contracts.ProjectID(projectID),
		ScopeID:      contracts.ScopeID(scopeID),
		PrincipalID:  contracts.PrincipalID(principalID),
		CapabilityID: contracts.CapabilityID(capabilityID),
		Type:         contracts.TaskType(typ),
		State:        contracts.TaskState(state),
		Title:        title,
		Description:  desc,
		CreatedAt:    created.UTC(),
		UpdatedAt:    updated.UTC(),
		Metadata:     meta,
	}, nil
}
