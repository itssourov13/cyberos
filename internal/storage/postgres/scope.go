package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/scope"
)

type ScopeRepo struct {
	db *DB
}

func NewScopeRepo(db *DB) *ScopeRepo {
	return &ScopeRepo{db: db}
}

func (r *ScopeRepo) Create(ctx context.Context, s *scope.Scope) error {
	if err := s.Validate(); err != nil {
		return err
	}
	meta, _ := json.Marshal(s.Metadata)
	constraints, _ := json.Marshal(s.Constraints)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO scopes (id, project_id, name, status, starts_at, expires_at, constraints, created_at, updated_at, metadata)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		string(s.ID), string(s.ProjectID), s.Name, string(s.Status),
		nullTime(s.StartsAt), nullTime(s.ExpiresAt), constraints,
		s.CreatedAt.UTC(), s.UpdatedAt.UTC(), meta,
	)
	if err != nil {
		return fmt.Errorf("insert scope: %w", err)
	}
	return nil
}

func (r *ScopeRepo) GetByID(ctx context.Context, id contracts.ScopeID) (*scope.Scope, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, name, status, starts_at, expires_at, constraints, created_at, updated_at, metadata
		FROM scopes WHERE id=$1`, string(id))
	return scanScope(row)
}

func (r *ScopeRepo) Update(ctx context.Context, s *scope.Scope) error {
	if err := s.Validate(); err != nil {
		return err
	}
	s.UpdatedAt = time.Now().UTC()
	meta, _ := json.Marshal(s.Metadata)
	constraints, _ := json.Marshal(s.Constraints)
	res, err := r.db.ExecContext(ctx, `
		UPDATE scopes SET name=$2, status=$3, starts_at=$4, expires_at=$5, constraints=$6, updated_at=$7, metadata=$8
		WHERE id=$1`,
		string(s.ID), s.Name, string(s.Status), nullTime(s.StartsAt), nullTime(s.ExpiresAt),
		constraints, s.UpdatedAt, meta,
	)
	if err != nil {
		return fmt.Errorf("update scope: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return contracts.ErrNotFound
	}
	return nil
}

func (r *ScopeRepo) ListByProject(ctx context.Context, projectID contracts.ProjectID, limit, offset int) ([]*scope.Scope, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, name, status, starts_at, expires_at, constraints, created_at, updated_at, metadata
		FROM scopes WHERE project_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		string(projectID), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*scope.Scope
	for rows.Next() {
		s, err := scanScope(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func nullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC()
}

func scanScope(s scannable) (*scope.Scope, error) {
	var (
		id, projectID, name, status string
		starts, expires             sql.NullTime
		constraintsRaw, metaRaw     []byte
		created, updated            time.Time
	)
	if err := s.Scan(&id, &projectID, &name, &status, &starts, &expires, &constraintsRaw, &created, &updated, &metaRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, contracts.ErrNotFound
		}
		return nil, err
	}
	meta := map[string]string{}
	_ = json.Unmarshal(metaRaw, &meta)
	var constraints contracts.ScopeConstraints
	_ = json.Unmarshal(constraintsRaw, &constraints)

	sc := &scope.Scope{
		ID:          contracts.ScopeID(id),
		ProjectID:   contracts.ProjectID(projectID),
		Name:        name,
		Status:      contracts.ScopeStatus(status),
		Constraints: constraints,
		CreatedAt:   created.UTC(),
		UpdatedAt:   updated.UTC(),
		Metadata:    meta,
	}
	if starts.Valid {
		t := starts.Time.UTC()
		sc.StartsAt = &t
	}
	if expires.Valid {
		t := expires.Time.UTC()
		sc.ExpiresAt = &t
	}
	return sc, nil
}
