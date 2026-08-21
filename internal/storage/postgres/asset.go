package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/asset"
)

type AssetRepo struct {
	db *DB
}

func NewAssetRepo(db *DB) *AssetRepo {
	return &AssetRepo{db: db}
}

func (r *AssetRepo) Create(ctx context.Context, a *asset.Asset) error {
	if err := a.Validate(); err != nil {
		return err
	}
	meta, _ := json.Marshal(a.Metadata)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO assets (id, project_id, type, name, value, description, created_at, updated_at, metadata)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		string(a.ID), string(a.ProjectID), string(a.Type), a.Name, a.Value, a.Description,
		a.CreatedAt.UTC(), a.UpdatedAt.UTC(), meta,
	)
	if err != nil {
		return fmt.Errorf("insert asset: %w", err)
	}
	return nil
}

func (r *AssetRepo) Upsert(ctx context.Context, a *asset.Asset) error {
	if err := a.Validate(); err != nil {
		return err
	}
	meta, _ := json.Marshal(a.Metadata)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO assets (id, project_id, type, name, value, description, created_at, updated_at, metadata)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (id) DO UPDATE SET
			project_id=EXCLUDED.project_id, type=EXCLUDED.type, name=EXCLUDED.name, value=EXCLUDED.value,
			description=EXCLUDED.description, updated_at=EXCLUDED.updated_at, metadata=EXCLUDED.metadata`,
		string(a.ID), string(a.ProjectID), string(a.Type), a.Name, a.Value, a.Description,
		a.CreatedAt.UTC(), a.UpdatedAt.UTC(), meta,
	)
	if err != nil {
		return fmt.Errorf("upsert asset: %w", err)
	}
	return nil
}

func (r *AssetRepo) GetByID(ctx context.Context, id contracts.AssetID) (*asset.Asset, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, type, name, value, description, created_at, updated_at, metadata
		FROM assets WHERE id=$1`, string(id))
	return scanAsset(row)
}

func (r *AssetRepo) ListByProject(ctx context.Context, projectID contracts.ProjectID, limit, offset int) ([]*asset.Asset, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, type, name, value, description, created_at, updated_at, metadata
		FROM assets WHERE project_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		string(projectID), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*asset.Asset
	for rows.Next() {
		a, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func scanAsset(s scannable) (*asset.Asset, error) {
	var (
		id, projectID, typ, name, value, desc string
		created, updated                      time.Time
		metaRaw                               []byte
	)
	if err := s.Scan(&id, &projectID, &typ, &name, &value, &desc, &created, &updated, &metaRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, contracts.ErrNotFound
		}
		return nil, err
	}
	meta := map[string]string{}
	_ = json.Unmarshal(metaRaw, &meta)
	return &asset.Asset{
		ID:          contracts.AssetID(id),
		ProjectID:   contracts.ProjectID(projectID),
		Type:        contracts.AssetType(typ),
		Name:        name,
		Value:       value,
		Description: desc,
		CreatedAt:   created.UTC(),
		UpdatedAt:   updated.UTC(),
		Metadata:    meta,
	}, nil
}
