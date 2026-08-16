package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/principal"
)

type PrincipalRepo struct {
	db *DB
}

func NewPrincipalRepo(db *DB) *PrincipalRepo {
	return &PrincipalRepo{db: db}
}

func (r *PrincipalRepo) Create(ctx context.Context, p *principal.Principal) error {
	if err := p.Validate(); err != nil {
		return err
	}
	meta, _ := json.Marshal(p.Metadata)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO principals (id, type, name, display_name, active, created_at, updated_at, metadata)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		string(p.ID), string(p.Type), p.Name, p.DisplayName, p.Active,
		p.CreatedAt.UTC(), p.UpdatedAt.UTC(), meta,
	)
	if err != nil {
		return fmt.Errorf("insert principal: %w", err)
	}
	return nil
}

func (r *PrincipalRepo) GetByID(ctx context.Context, id contracts.PrincipalID) (*principal.Principal, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, type, name, display_name, active, created_at, updated_at, metadata
		FROM principals WHERE id=$1`, string(id))
	return scanPrincipal(row)
}

func (r *PrincipalRepo) List(ctx context.Context, limit, offset int) ([]*principal.Principal, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, type, name, display_name, active, created_at, updated_at, metadata
		FROM principals ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*principal.Principal
	for rows.Next() {
		p, err := scanPrincipal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func scanPrincipal(s scannable) (*principal.Principal, error) {
	var (
		id, typ, name, display string
		active                 bool
		created, updated       time.Time
		metaRaw                []byte
	)
	if err := s.Scan(&id, &typ, &name, &display, &active, &created, &updated, &metaRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, contracts.ErrNotFound
		}
		return nil, err
	}
	meta := map[string]string{}
	_ = json.Unmarshal(metaRaw, &meta)
	return &principal.Principal{
		ID:          contracts.PrincipalID(id),
		Type:        contracts.PrincipalType(typ),
		Name:        name,
		DisplayName: display,
		Active:      active,
		CreatedAt:   created.UTC(),
		UpdatedAt:   updated.UTC(),
		Metadata:    meta,
	}, nil
}
