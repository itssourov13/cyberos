package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/evidence"
)

// EvidenceRepo persists evidence (Phase 8A).
type EvidenceRepo struct{ db *DB }

func NewEvidenceRepo(db *DB) *EvidenceRepo { return &EvidenceRepo{db: db} }

func (r *EvidenceRepo) Upsert(ctx context.Context, e *evidence.Evidence) error {
	if e == nil {
		return contracts.ErrInvalidArgument
	}
	if err := e.Validate(); err != nil {
		return err
	}
	payload, _ := json.Marshal(e.Payload)
	meta, _ := json.Marshal(e.Metadata)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO evidence (
			id, task_id, grant_id, capability_id, principal_id, principal_type,
			project_id, scope_id, content_type, digest, payload, produced_at, correlation_id, metadata
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT (id) DO UPDATE SET
			task_id=EXCLUDED.task_id, grant_id=EXCLUDED.grant_id, capability_id=EXCLUDED.capability_id,
			principal_id=EXCLUDED.principal_id, principal_type=EXCLUDED.principal_type,
			project_id=EXCLUDED.project_id, scope_id=EXCLUDED.scope_id, content_type=EXCLUDED.content_type,
			digest=EXCLUDED.digest, payload=EXCLUDED.payload, produced_at=EXCLUDED.produced_at,
			correlation_id=EXCLUDED.correlation_id, metadata=EXCLUDED.metadata`,
		string(e.ID), string(e.TaskID), string(e.GrantID), string(e.CapabilityID),
		string(e.Principal.ID), string(e.Principal.Type),
		string(e.ProjectID), string(e.ScopeID), e.ContentType, e.Digest, payload,
		e.ProducedAt.UTC(), string(e.CorrelationID), meta,
	)
	if err != nil {
		return fmt.Errorf("upsert evidence: %w", err)
	}
	return nil
}

func (r *EvidenceRepo) GetByID(ctx context.Context, id contracts.EvidenceID) (*evidence.Evidence, error) {
	row := r.db.QueryRowContext(ctx, evidenceSelect+` WHERE id=$1`, string(id))
	return scanEvidence(row)
}

func (r *EvidenceRepo) ListByTask(ctx context.Context, taskID contracts.TaskID, limit, offset int) ([]*evidence.Evidence, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, evidenceSelect+` WHERE task_id=$1 ORDER BY produced_at DESC LIMIT $2 OFFSET $3`,
		string(taskID), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvidences(rows)
}

func (r *EvidenceRepo) ListByProject(ctx context.Context, projectID contracts.ProjectID, limit, offset int) ([]*evidence.Evidence, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, evidenceSelect+` WHERE project_id=$1 ORDER BY produced_at DESC LIMIT $2 OFFSET $3`,
		string(projectID), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvidences(rows)
}

const evidenceSelect = `
SELECT id, task_id, grant_id, capability_id, principal_id, principal_type,
	project_id, scope_id, content_type, digest, payload, produced_at, correlation_id, metadata
FROM evidence`

func scanEvidences(rows *sql.Rows) ([]*evidence.Evidence, error) {
	var out []*evidence.Evidence
	for rows.Next() {
		e, err := scanEvidence(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func scanEvidence(s scannable) (*evidence.Evidence, error) {
	var (
		id, taskID, grantID, capID, principalID, principalType string
		projectID, scopeID, contentType, digest, corr          string
		payloadRaw, metaRaw                                    []byte
		produced                                               time.Time
	)
	err := s.Scan(&id, &taskID, &grantID, &capID, &principalID, &principalType,
		&projectID, &scopeID, &contentType, &digest, &payloadRaw, &produced, &corr, &metaRaw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, contracts.ErrNotFound
		}
		return nil, err
	}
	payload := map[string]any{}
	_ = json.Unmarshal(payloadRaw, &payload)
	meta := map[string]string{}
	_ = json.Unmarshal(metaRaw, &meta)
	return &evidence.Evidence{
		ID: contracts.EvidenceID(id), TaskID: contracts.TaskID(taskID),
		GrantID: contracts.ExecutionGrantID(grantID), CapabilityID: contracts.CapabilityID(capID),
		Principal: contracts.PrincipalRef{ID: contracts.PrincipalID(principalID), Type: contracts.PrincipalType(principalType)},
		ProjectID: contracts.ProjectID(projectID), ScopeID: contracts.ScopeID(scopeID),
		ContentType: contentType, Digest: digest, Payload: payload,
		ProducedAt: produced.UTC(), CorrelationID: contracts.CorrelationID(corr), Metadata: meta,
	}, nil
}
