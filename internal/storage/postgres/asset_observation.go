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

// AssetObservationRepo persists asset_observations (migration 008).
type AssetObservationRepo struct{ db *DB }

func NewAssetObservationRepo(db *DB) *AssetObservationRepo {
	return &AssetObservationRepo{db: db}
}

func (r *AssetObservationRepo) Upsert(ctx context.Context, o *asset.Observation) error {
	if o == nil || o.ID == "" {
		return contracts.ErrInvalidArgument
	}
	meta, _ := json.Marshal(o.Metadata)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO asset_observations (
			id, asset_id, project_id, scope_id, task_id, execution_id, tool_id, capability_id,
			kind, summary, detail, endpoint_id, observed_at, metadata
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT (id) DO UPDATE SET
			asset_id=EXCLUDED.asset_id, project_id=EXCLUDED.project_id, scope_id=EXCLUDED.scope_id,
			task_id=EXCLUDED.task_id, execution_id=EXCLUDED.execution_id, tool_id=EXCLUDED.tool_id,
			capability_id=EXCLUDED.capability_id, kind=EXCLUDED.kind, summary=EXCLUDED.summary,
			detail=EXCLUDED.detail, endpoint_id=EXCLUDED.endpoint_id, observed_at=EXCLUDED.observed_at,
			metadata=EXCLUDED.metadata`,
		o.ID, string(o.AssetID), string(o.ProjectID), string(o.ScopeID), string(o.TaskID),
		o.ExecutionID, o.ToolID, string(o.CapabilityID), string(o.Kind), o.Summary, o.Detail,
		o.EndpointID, o.ObservedAt.UTC(), meta,
	)
	if err != nil {
		return fmt.Errorf("upsert asset observation: %w", err)
	}
	return nil
}

func (r *AssetObservationRepo) GetByID(ctx context.Context, id string) (*asset.Observation, error) {
	row := r.db.QueryRowContext(ctx, obsSelect+` WHERE id=$1`, id)
	return scanObservation(row)
}

func (r *AssetObservationRepo) ListByAsset(ctx context.Context, assetID contracts.AssetID, limit, offset int) ([]*asset.Observation, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, obsSelect+` WHERE asset_id=$1 ORDER BY observed_at DESC LIMIT $2 OFFSET $3`,
		string(assetID), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanObservations(rows)
}

func (r *AssetObservationRepo) ListByProject(ctx context.Context, projectID contracts.ProjectID, limit, offset int) ([]*asset.Observation, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, obsSelect+` WHERE project_id=$1 ORDER BY observed_at DESC LIMIT $2 OFFSET $3`,
		string(projectID), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanObservations(rows)
}

const obsSelect = `
SELECT id, asset_id, project_id, scope_id, task_id, execution_id, tool_id, capability_id,
	kind, summary, detail, endpoint_id, observed_at, metadata
FROM asset_observations`

func scanObservations(rows *sql.Rows) ([]*asset.Observation, error) {
	var out []*asset.Observation
	for rows.Next() {
		o, err := scanObservation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func scanObservation(s scannable) (*asset.Observation, error) {
	var (
		id, assetID, projectID, scopeID, taskID, execID, toolID, capID string
		kind, summary, detail, endpointID                              string
		observed                                                       time.Time
		metaRaw                                                        []byte
	)
	err := s.Scan(&id, &assetID, &projectID, &scopeID, &taskID, &execID, &toolID, &capID,
		&kind, &summary, &detail, &endpointID, &observed, &metaRaw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, contracts.ErrNotFound
		}
		return nil, err
	}
	meta := map[string]string{}
	_ = json.Unmarshal(metaRaw, &meta)
	return &asset.Observation{
		ID: id, AssetID: contracts.AssetID(assetID), ProjectID: contracts.ProjectID(projectID),
		ScopeID: contracts.ScopeID(scopeID), TaskID: contracts.TaskID(taskID), ExecutionID: execID,
		ToolID: toolID, CapabilityID: contracts.CapabilityID(capID),
		Kind: contracts.ObservationKind(kind), Summary: summary, Detail: detail, EndpointID: endpointID,
		ObservedAt: observed.UTC(), Metadata: meta,
	}, nil
}
