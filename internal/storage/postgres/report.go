package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/storage/ports"
)

// ReportRepo persists assessment reports.
type ReportRepo struct{ db *DB }

func NewReportRepo(db *DB) *ReportRepo { return &ReportRepo{db: db} }

func (r *ReportRepo) Upsert(ctx context.Context, rec ports.ReportRecord) error {
	if rec.ID == "" || rec.AssessmentID == "" {
		return contracts.ErrInvalidArgument
	}
	snap := rec.SnapshotJSON
	if snap == nil {
		snap = []byte("{}")
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO reports (
			id, tenant_id, project_id, assessment_id, generated_at, generated_by,
			schema_version, status, snapshot
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (id) DO UPDATE SET
			tenant_id=EXCLUDED.tenant_id, project_id=EXCLUDED.project_id,
			assessment_id=EXCLUDED.assessment_id, generated_at=EXCLUDED.generated_at,
			generated_by=EXCLUDED.generated_by, schema_version=EXCLUDED.schema_version,
			status=EXCLUDED.status, snapshot=EXCLUDED.snapshot`,
		rec.ID, string(rec.TenantID), string(rec.ProjectID), rec.AssessmentID,
		rec.GeneratedAt.UTC(), string(rec.GeneratedBy), rec.SchemaVersion, rec.Status, snap,
	)
	if err != nil {
		return fmt.Errorf("upsert report: %w", err)
	}
	return nil
}

func (r *ReportRepo) GetByID(ctx context.Context, id string) (ports.ReportRecord, error) {
	row := r.db.QueryRowContext(ctx, reportSelect+` WHERE id=$1`, id)
	return scanReport(row)
}

func (r *ReportRepo) List(ctx context.Context, limit, offset int) ([]ports.ReportRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, reportSelect+` ORDER BY generated_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReports(rows)
}

func (r *ReportRepo) ListByProject(ctx context.Context, projectID contracts.ProjectID, limit, offset int) ([]ports.ReportRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, reportSelect+` WHERE project_id=$1 ORDER BY generated_at DESC LIMIT $2 OFFSET $3`,
		string(projectID), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReports(rows)
}

const reportSelect = `
SELECT id, tenant_id, project_id, assessment_id, generated_at, generated_by, schema_version, status, snapshot
FROM reports`

func scanReports(rows *sql.Rows) ([]ports.ReportRecord, error) {
	var out []ports.ReportRecord
	for rows.Next() {
		r, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func scanReport(s scannable) (ports.ReportRecord, error) {
	var (
		id, tenantID, projectID, assessmentID, generatedBy, schema, status string
		generated                                                          time.Time
		snap                                                               []byte
	)
	err := s.Scan(&id, &tenantID, &projectID, &assessmentID, &generated, &generatedBy, &schema, &status, &snap)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ports.ReportRecord{}, contracts.ErrNotFound
		}
		return ports.ReportRecord{}, err
	}
	return ports.ReportRecord{
		ID: id, TenantID: contracts.TenantID(tenantID), ProjectID: contracts.ProjectID(projectID),
		AssessmentID: assessmentID, GeneratedAt: generated.UTC(), GeneratedBy: contracts.PrincipalID(generatedBy),
		SchemaVersion: schema, Status: status, SnapshotJSON: snap,
	}, nil
}
