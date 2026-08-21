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

// AssetEndpointRepo persists asset_endpoints (migration 008).
type AssetEndpointRepo struct{ db *DB }

func NewAssetEndpointRepo(db *DB) *AssetEndpointRepo { return &AssetEndpointRepo{db: db} }

func (r *AssetEndpointRepo) Upsert(ctx context.Context, ep *asset.ServiceEndpoint) error {
	if ep == nil {
		return contracts.ErrInvalidArgument
	}
	if err := ep.Validate(); err != nil {
		return err
	}
	meta, _ := json.Marshal(ep.Metadata)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO asset_endpoints (
			id, asset_id, project_id, protocol, port, path, banner, first_seen, last_seen, metadata
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (id) DO UPDATE SET
			asset_id=EXCLUDED.asset_id, project_id=EXCLUDED.project_id, protocol=EXCLUDED.protocol,
			port=EXCLUDED.port, path=EXCLUDED.path, banner=EXCLUDED.banner,
			first_seen=EXCLUDED.first_seen, last_seen=EXCLUDED.last_seen, metadata=EXCLUDED.metadata`,
		ep.ID, string(ep.AssetID), string(ep.ProjectID), ep.Protocol, ep.Port, ep.Path, ep.Banner,
		ep.FirstSeen.UTC(), ep.LastSeen.UTC(), meta,
	)
	if err != nil {
		return fmt.Errorf("upsert asset endpoint: %w", err)
	}
	return nil
}

func (r *AssetEndpointRepo) GetByID(ctx context.Context, id string) (*asset.ServiceEndpoint, error) {
	row := r.db.QueryRowContext(ctx, epSelect+` WHERE id=$1`, id)
	return scanEndpoint(row)
}

func (r *AssetEndpointRepo) ListByAsset(ctx context.Context, assetID contracts.AssetID, limit, offset int) ([]*asset.ServiceEndpoint, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, epSelect+` WHERE asset_id=$1 ORDER BY last_seen DESC LIMIT $2 OFFSET $3`,
		string(assetID), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEndpoints(rows)
}

func (r *AssetEndpointRepo) ListByProject(ctx context.Context, projectID contracts.ProjectID, limit, offset int) ([]*asset.ServiceEndpoint, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, epSelect+` WHERE project_id=$1 ORDER BY last_seen DESC LIMIT $2 OFFSET $3`,
		string(projectID), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEndpoints(rows)
}

const epSelect = `
SELECT id, asset_id, project_id, protocol, port, path, banner, first_seen, last_seen, metadata
FROM asset_endpoints`

func scanEndpoints(rows *sql.Rows) ([]*asset.ServiceEndpoint, error) {
	var out []*asset.ServiceEndpoint
	for rows.Next() {
		ep, err := scanEndpoint(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ep)
	}
	return out, rows.Err()
}

func scanEndpoint(s scannable) (*asset.ServiceEndpoint, error) {
	var (
		id, assetID, projectID, protocol, path, banner string
		port                                           int
		first, last                                    time.Time
		metaRaw                                        []byte
	)
	err := s.Scan(&id, &assetID, &projectID, &protocol, &port, &path, &banner, &first, &last, &metaRaw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, contracts.ErrNotFound
		}
		return nil, err
	}
	meta := map[string]string{}
	_ = json.Unmarshal(metaRaw, &meta)
	return &asset.ServiceEndpoint{
		ID: id, AssetID: contracts.AssetID(assetID), ProjectID: contracts.ProjectID(projectID),
		Protocol: protocol, Port: port, Path: path, Banner: banner,
		FirstSeen: first.UTC(), LastSeen: last.UTC(), Metadata: meta,
	}, nil
}
