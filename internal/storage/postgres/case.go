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
			title=EXCLUDED.title, description=EXCLUDED.description,
			priority=EXCLUDED.priority, finding_ids=EXCLUDED.finding_ids, asset_ids=EXCLUDED.asset_ids,
			updated_at=EXCLUDED.updated_at, metadata=EXCLUDED.metadata`,
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


// Insert creates a case; an existing id yields contracts.ErrConflict.
func (r *CaseRepo) Insert(ctx context.Context, c ports.CaseRecord) error {
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
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO security_cases (
			id, tenant_id, project_id, assessment_id, title, description, status, priority,
			finding_ids, asset_ids, created_by, created_at, updated_at, resolved_at, metadata
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		ON CONFLICT (id) DO NOTHING`,
		c.ID, string(c.TenantID), string(c.ProjectID), c.AssessmentID, c.Title, c.Description,
		string(c.Status), string(c.Priority), fids, aids, string(c.CreatedBy),
		c.CreatedAt.UTC(), c.UpdatedAt.UTC(), resolved, meta,
	)
	if err != nil {
		return fmt.Errorf("insert case: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return contracts.ErrConflict
	}
	return nil
}

// Transition changes status and appends history atomically (row-locked).
func (r *CaseRepo) Transition(ctx context.Context, id string, from, to contracts.CaseStatus, ch ports.CaseStatusChange) (ports.CaseRecord, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ports.CaseRecord{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var current string
	var tenantID, projectID string
	err = tx.QueryRowContext(ctx, `SELECT status, tenant_id, project_id FROM security_cases WHERE id=$1 FOR UPDATE`, id).
		Scan(&current, &tenantID, &projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ports.CaseRecord{}, contracts.ErrNotFound
		}
		return ports.CaseRecord{}, err
	}
	if contracts.CaseStatus(current) != from {
		return ports.CaseRecord{}, contracts.ErrConflict
	}
	at := ch.At.UTC()
	var resolved any
	switch to {
	case contracts.CaseStatusResolved, contracts.CaseStatusAccepted:
		resolved = at
	}
	if _, err = tx.ExecContext(ctx,
		`UPDATE security_cases SET status=$2, updated_at=$3, resolved_at=$4 WHERE id=$1`,
		id, string(to), at, resolved); err != nil {
		return ports.CaseRecord{}, err
	}
	if _, err = tx.ExecContext(ctx, `
		INSERT INTO case_status_history (case_id, tenant_id, project_id, from_status, to_status, actor, reason, changed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		id, tenantID, projectID, string(from), string(to), string(ch.Actor), ch.Reason, at); err != nil {
		return ports.CaseRecord{}, err
	}
	if err = tx.Commit(); err != nil {
		return ports.CaseRecord{}, err
	}
	return r.GetByID(ctx, id)
}

// StatusHistory lists status changes oldest first.
func (r *CaseRepo) StatusHistory(ctx context.Context, caseID string) ([]ports.CaseStatusChange, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT case_id, tenant_id, project_id, from_status, to_status, actor, reason, changed_at
		FROM case_status_history WHERE case_id=$1 ORDER BY id ASC`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ports.CaseStatusChange
	for rows.Next() {
		var cid, tid, pid, from, to, actor, reason string
		var at time.Time
		if err := rows.Scan(&cid, &tid, &pid, &from, &to, &actor, &reason, &at); err != nil {
			return nil, err
		}
		out = append(out, ports.CaseStatusChange{
			CaseID: cid, TenantID: contracts.TenantID(tid), ProjectID: contracts.ProjectID(pid),
			From: contracts.CaseStatus(from), To: contracts.CaseStatus(to),
			Actor: contracts.PrincipalID(actor), Reason: reason, At: at.UTC(),
		})
	}
	return out, rows.Err()
}
