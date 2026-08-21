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

// RiskEvalRepo persists riskprio evaluations (not Control Plane risk).
type RiskEvalRepo struct {
	db *DB
}

// NewRiskEvalRepo constructs a risk evaluation repository.
func NewRiskEvalRepo(db *DB) *RiskEvalRepo {
	return &RiskEvalRepo{db: db}
}

// Upsert inserts or updates a risk evaluation.
func (r *RiskEvalRepo) Upsert(ctx context.Context, rec ports.RiskEvaluationRecord) error {
	if rec.ID == "" || rec.FindingID == "" {
		return contracts.ErrInvalidArgument
	}
	factors, _ := json.Marshal(rec.Factors)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO risk_evaluations (
			id, project_id, assessment_id, finding_id, asset_id, severity, confidence,
			asset_criticality, exposure, score, priority, factors, model_version, calculated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT (id) DO UPDATE SET
			project_id=EXCLUDED.project_id, assessment_id=EXCLUDED.assessment_id,
			finding_id=EXCLUDED.finding_id, asset_id=EXCLUDED.asset_id, severity=EXCLUDED.severity,
			confidence=EXCLUDED.confidence, asset_criticality=EXCLUDED.asset_criticality,
			exposure=EXCLUDED.exposure, score=EXCLUDED.score, priority=EXCLUDED.priority,
			factors=EXCLUDED.factors, model_version=EXCLUDED.model_version,
			calculated_at=EXCLUDED.calculated_at`,
		rec.ID, string(rec.ProjectID), rec.AssessmentID, string(rec.FindingID), string(rec.AssetID),
		string(rec.Severity), string(rec.Confidence), string(rec.AssetCriticality), string(rec.Exposure),
		rec.Score, string(rec.Priority), factors, rec.ModelVersion, rec.CalculatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("upsert risk evaluation: %w", err)
	}
	return nil
}

// GetByID loads by risk id.
func (r *RiskEvalRepo) GetByID(ctx context.Context, id string) (ports.RiskEvaluationRecord, error) {
	row := r.db.QueryRowContext(ctx, riskSelect+` WHERE id=$1`, id)
	return scanRisk(row)
}

// GetByFinding loads latest evaluation for a finding (by calculated_at).
func (r *RiskEvalRepo) GetByFinding(ctx context.Context, findingID contracts.FindingID) (ports.RiskEvaluationRecord, error) {
	row := r.db.QueryRowContext(ctx, riskSelect+` WHERE finding_id=$1 ORDER BY calculated_at DESC LIMIT 1`, string(findingID))
	return scanRisk(row)
}

// ListByProject lists evaluations for a project.
func (r *RiskEvalRepo) ListByProject(ctx context.Context, projectID contracts.ProjectID, limit, offset int) ([]ports.RiskEvaluationRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, riskSelect+` WHERE project_id=$1 ORDER BY calculated_at DESC LIMIT $2 OFFSET $3`,
		string(projectID), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRisks(rows)
}

// ListByAssessment lists evaluations for an assessment.
func (r *RiskEvalRepo) ListByAssessment(ctx context.Context, assessmentID string, limit, offset int) ([]ports.RiskEvaluationRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, riskSelect+` WHERE assessment_id=$1 ORDER BY calculated_at DESC LIMIT $2 OFFSET $3`,
		assessmentID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRisks(rows)
}

const riskSelect = `
SELECT id, project_id, assessment_id, finding_id, asset_id, severity, confidence,
	asset_criticality, exposure, score, priority, factors, model_version, calculated_at
FROM risk_evaluations`

func scanRisks(rows *sql.Rows) ([]ports.RiskEvaluationRecord, error) {
	var out []ports.RiskEvaluationRecord
	for rows.Next() {
		r, err := scanRisk(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func scanRisk(s scannable) (ports.RiskEvaluationRecord, error) {
	var (
		id, projectID, assessmentID, findingID, assetID       string
		severity, confidence, crit, exposure, priority, model string
		score                                                 int
		factorsRaw                                            []byte
		calculated                                            time.Time
	)
	err := s.Scan(&id, &projectID, &assessmentID, &findingID, &assetID, &severity, &confidence,
		&crit, &exposure, &score, &priority, &factorsRaw, &model, &calculated)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ports.RiskEvaluationRecord{}, contracts.ErrNotFound
		}
		return ports.RiskEvaluationRecord{}, err
	}
	factors := map[string]string{}
	_ = json.Unmarshal(factorsRaw, &factors)
	return ports.RiskEvaluationRecord{
		ID: id, ProjectID: contracts.ProjectID(projectID), AssessmentID: assessmentID,
		FindingID: contracts.FindingID(findingID), AssetID: contracts.AssetID(assetID),
		Severity: contracts.FindingSeverity(severity), Confidence: contracts.FindingConfidence(confidence),
		AssetCriticality: contracts.AssetCriticality(crit), Exposure: contracts.ExposureContext(exposure),
		Score: score, Priority: contracts.RiskPriority(priority), Factors: factors,
		ModelVersion: model, CalculatedAt: calculated.UTC(),
	}, nil
}
