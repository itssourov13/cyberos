package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/project"
)

type ProjectRepo struct {
	db *DB
}

func NewProjectRepo(db *DB) *ProjectRepo {
	return &ProjectRepo{db: db}
}

func (r *ProjectRepo) Create(ctx context.Context, p *project.Project) error {
	if err := p.Validate(); err != nil {
		return err
	}
	meta, _ := json.Marshal(p.Metadata)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO projects (id, name, description, status, created_at, updated_at, metadata, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		string(p.ID), p.Name, p.Description, string(p.Status),
		p.CreatedAt.UTC(), p.UpdatedAt.UTC(), meta, nullStr(string(p.TenantID)),
	)
	if err != nil {
		return fmt.Errorf("insert project: %w", err)
	}
	return nil
}

func (r *ProjectRepo) GetByID(ctx context.Context, id contracts.ProjectID) (*project.Project, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, description, status, created_at, updated_at, metadata, tenant_id
		FROM projects WHERE id = $1`, string(id))
	return scanProject(row)
}

func (r *ProjectRepo) Update(ctx context.Context, p *project.Project) error {
	if err := p.Validate(); err != nil {
		return err
	}
	p.UpdatedAt = time.Now().UTC()
	meta, _ := json.Marshal(p.Metadata)
	res, err := r.db.ExecContext(ctx, `
		UPDATE projects SET name=$2, description=$3, status=$4, updated_at=$5, metadata=$6
		WHERE id=$1 AND COALESCE(tenant_id,'')=$7`,
		string(p.ID), p.Name, p.Description, string(p.Status), p.UpdatedAt, meta, string(p.TenantID),
	)
	if err != nil {
		return fmt.Errorf("update project: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return contracts.ErrNotFound
	}
	return nil
}

func (r *ProjectRepo) List(ctx context.Context, limit, offset int) ([]*project.Project, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, description, status, created_at, updated_at, metadata, tenant_id
		FROM projects ORDER BY created_at DESC, id ASC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*project.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

type scannable interface {
	Scan(dest ...any) error
}

func scanProject(s scannable) (*project.Project, error) {
	var (
		id, name, desc, status string
		tenantID               sql.NullString
		created, updated       time.Time
		metaRaw                []byte
	)
	if err := s.Scan(&id, &name, &desc, &status, &created, &updated, &metaRaw, &tenantID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, contracts.ErrNotFound
		}
		return nil, err
	}
	meta := map[string]string{}
	_ = json.Unmarshal(metaRaw, &meta)
	p := &project.Project{
		ID:          contracts.ProjectID(id),
		TenantID:    contracts.TenantID(tenantID.String),
		Name:        name,
		Description: desc,
		Status:      project.Status(status),
		CreatedAt:   created.UTC(),
		UpdatedAt:   updated.UTC(),
		Metadata:    meta,
	}
	return p, nil
}
