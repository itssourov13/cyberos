package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/storage/ports"
)

// CaseRepo persists security cases.
type CaseRepo struct{ db *DB }

func NewCaseRepo(db *DB) *CaseRepo { return &CaseRepo{db: db} }

func (r *CaseRepo) Upsert(ctx context.Context, c ports.CaseRecord) error {
	if c.ID == "" || c.ProjectID == "" {
		return contracts.ErrInvalidArgument
	}
	fids, _ := json.Marshal(c.FindingIDs)
	aids, _ := json.Marshal(c.AssetIDs)
	meta, _ := json.Marshal(c.Metadata)
	var resolved any
	if c.ResolvedAt != nil {
		resolved = c.ResolvedAt.UTC()
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO security_cases (
			id, tenant_id, project_id, assessment_id, title, description, status, priority,
			finding_ids, asset_ids, created_by, created_at, updated_at, resolved_at, metadata
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		ON CONFLICT (id) DO UPDATE SET
			tenant_id=EXCLUDED.tenant_id, project_id=EXCLUDED.project_id, assessment_id=EXCLUDED.assessment_id,
			title=EXCLUDED.title, description=EXCLUDED.description, status=EXCLUDED.status,
			priority=EXCLUDED.priority, finding_ids=EXCLUDED.finding_ids, asset_ids=EXCLUDED.asset_ids,
			created_by=EXCLUDED.created_by, updated_at=EXCLUDED.updated_at, resolved_at=EXCLUDED.resolved_at,
			metadata=EXCLUDED.metadata`,
		c.ID, string(c.TenantID), string(c.ProjectID), c.AssessmentID, c.Title, c.Description,
		string(c.Status), string(c.Priority), fids, aids, string(c.CreatedBy),
		c.CreatedAt.UTC(), c.UpdatedAt.UTC(), resolved, meta,
	)
	if err != nil {
		return fmt.Errorf("upsert case: %w", err)
	}
	return nil
}

func (r *CaseRepo) GetByID(ctx context.Context, id string) (ports.CaseRecord, error) {
	row := r.db.QueryRowContext(ctx, caseSelect+` WHERE id=$1`, id)
	return scanCase(row)
}

func (r *CaseRepo) ListByProject(ctx context.Context, projectID contracts.ProjectID, limit, offset int) ([]ports.CaseRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, caseSelect+` WHERE project_id=$1 ORDER BY updated_at DESC LIMIT $2 OFFSET $3`,
		string(projectID), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCases(rows)
}

func (r *CaseRepo) ListByAssessment(ctx context.Context, assessmentID string, limit, offset int) ([]ports.CaseRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, caseSelect+` WHERE assessment_id=$1 ORDER BY updated_at DESC LIMIT $2 OFFSET $3`,
		assessmentID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCases(rows)
}

func (r *CaseRepo) ListByTenant(ctx context.Context, tenantID contracts.TenantID, limit, offset int) ([]ports.CaseRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, caseSelect+` WHERE tenant_id=$1 ORDER BY updated_at DESC LIMIT $2 OFFSET $3`,
		string(tenantID), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCases(rows)
}

const caseSelect = `
SELECT id, tenant_id, project_id, assessment_id, title, description, status, priority,
	finding_ids, asset_ids, created_by, created_at, updated_at, resolved_at, metadata
FROM security_cases`

func scanCases(rows *sql.Rows) ([]ports.CaseRecord, error) {
	var out []ports.CaseRecord
	for rows.Next() {
		c, err := scanCase(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func scanCase(s scannable) (ports.CaseRecord, error) {
	var (
		id, tenantID, projectID, assessmentID, title, description, status, priority, createdBy string
		fRaw, aRaw, metaRaw                                                                    []byte
		created, updated                                                                       time.Time
		resolved                                                                               sql.NullTime
	)
	err := s.Scan(&id, &tenantID, &projectID, &assessmentID, &title, &description, &status, &priority,
		&fRaw, &aRaw, &createdBy, &created, &updated, &resolved, &metaRaw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ports.CaseRecord{}, contracts.ErrNotFound
		}
		return ports.CaseRecord{}, err
	}
	var fids []contracts.FindingID
	var aids []contracts.AssetID
	_ = json.Unmarshal(fRaw, &fids)
	_ = json.Unmarshal(aRaw, &aids)
	meta := map[string]string{}
	_ = json.Unmarshal(metaRaw, &meta)
	rec := ports.CaseRecord{
		ID: id, TenantID: contracts.TenantID(tenantID), ProjectID: contracts.ProjectID(projectID),
		AssessmentID: assessmentID, Title: title, Description: description,
		Status: contracts.CaseStatus(status), Priority: contracts.RiskPriority(priority),
		FindingIDs: fids, AssetIDs: aids, CreatedBy: contracts.PrincipalID(createdBy),
		CreatedAt: created.UTC(), UpdatedAt: updated.UTC(), Metadata: meta,
	}
	if resolved.Valid {
		t := resolved.Time.UTC()
		rec.ResolvedAt = &t
	}
	return rec, nil
}
