package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/capability"
)

type CapabilityRepo struct {
	db *DB
}

func NewCapabilityRepo(db *DB) *CapabilityRepo {
	return &CapabilityRepo{db: db}
}

func (r *CapabilityRepo) Create(ctx context.Context, c *capability.Capability) error {
	if err := c.Validate(); err != nil {
		return err
	}
	meta, _ := json.Marshal(c.Metadata)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO capabilities (id, name, type, description, version, active, created_at, updated_at, metadata)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		string(c.ID), c.Name, string(c.Type), c.Description, string(c.Version), c.Active,
		c.CreatedAt.UTC(), c.UpdatedAt.UTC(), meta,
	)
	if err != nil {
		return fmt.Errorf("insert capability: %w", err)
	}
	return nil
}

func (r *CapabilityRepo) GetByID(ctx context.Context, id contracts.CapabilityID) (*capability.Capability, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, type, description, version, active, created_at, updated_at, metadata
		FROM capabilities WHERE id=$1`, string(id))
	return scanCapability(row)
}

func (r *CapabilityRepo) List(ctx context.Context, limit, offset int) ([]*capability.Capability, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, type, description, version, active, created_at, updated_at, metadata
		FROM capabilities ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*capability.Capability
	for rows.Next() {
		c, err := scanCapability(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func scanCapability(s scannable) (*capability.Capability, error) {
	var (
		id, name, typ, desc, version string
		active                       bool
		created, updated             time.Time
		metaRaw                      []byte
	)
	if err := s.Scan(&id, &name, &typ, &desc, &version, &active, &created, &updated, &metaRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, contracts.ErrNotFound
		}
		return nil, err
	}
	meta := map[string]string{}
	_ = json.Unmarshal(metaRaw, &meta)
	return &capability.Capability{
		ID:          contracts.CapabilityID(id),
		Name:        name,
		Type:        contracts.CapabilityType(typ),
		Description: desc,
		Version:     contracts.SchemaVersion(version),
		Active:      active,
		CreatedAt:   created.UTC(),
		UpdatedAt:   updated.UTC(),
		Metadata:    meta,
	}, nil
}
