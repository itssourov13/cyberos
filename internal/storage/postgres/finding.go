package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/finding"
)

// FindingRepo persists findings (Phase 7A).
type FindingRepo struct {
	db *DB
}

// NewFindingRepo constructs a findings repository.
func NewFindingRepo(db *DB) *FindingRepo {
	return &FindingRepo{db: db}
}

// Upsert inserts or updates a finding by primary key.
func (r *FindingRepo) Upsert(ctx context.Context, f *finding.Finding) error {
	if f == nil {
		return contracts.ErrInvalidArgument
	}
	if err := f.Validate(); err != nil {
		return err
	}
	ev, _ := json.Marshal(f.EvidenceIDs)
	meta, _ := json.Marshal(f.Metadata)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO findings (
			id, project_id, scope_id, task_id, assessment_id, execution_id, principal_id,
			tool_id, capability_id, severity, confidence, status, title, description,
			target, location, fingerprint, evidence_ids, first_seen, last_seen, produced_at,
			correlation_id, metadata
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23
		)
		ON CONFLICT (id) DO UPDATE SET
			project_id=EXCLUDED.project_id, scope_id=EXCLUDED.scope_id, task_id=EXCLUDED.task_id,
			assessment_id=EXCLUDED.assessment_id, execution_id=EXCLUDED.execution_id,
			principal_id=EXCLUDED.principal_id, tool_id=EXCLUDED.tool_id, capability_id=EXCLUDED.capability_id,
			severity=EXCLUDED.severity, confidence=EXCLUDED.confidence, status=EXCLUDED.status,
			title=EXCLUDED.title, description=EXCLUDED.description, target=EXCLUDED.target,
			location=EXCLUDED.location, fingerprint=EXCLUDED.fingerprint, evidence_ids=EXCLUDED.evidence_ids,
			first_seen=EXCLUDED.first_seen, last_seen=EXCLUDED.last_seen, produced_at=EXCLUDED.produced_at,
			correlation_id=EXCLUDED.correlation_id, metadata=EXCLUDED.metadata`,
		string(f.ID), string(f.ProjectID), string(f.ScopeID), string(f.TaskID), f.AssessmentID, f.ExecutionID,
		string(f.PrincipalID), f.ToolID, string(f.CapabilityID), string(f.Severity), string(f.Confidence),
		string(f.Status), f.Title, f.Description, f.Target, f.Location, f.Fingerprint, ev,
		f.FirstSeen.UTC(), f.LastSeen.UTC(), f.ProducedAt.UTC(), string(f.CorrelationID), meta,
	)
	if err != nil {
		return fmt.Errorf("upsert finding: %w", err)
	}
	return nil
}

// GetByID loads a finding.
func (r *FindingRepo) GetByID(ctx context.Context, id contracts.FindingID) (*finding.Finding, error) {
	row := r.db.QueryRowContext(ctx, findingSelect+` WHERE id=$1`, string(id))
	return scanFinding(row)
}

// GetByFingerprint loads by project + fingerprint (dedup key).
func (r *FindingRepo) GetByFingerprint(ctx context.Context, projectID contracts.ProjectID, fingerprint string) (*finding.Finding, error) {
	row := r.db.QueryRowContext(ctx, findingSelect+` WHERE project_id=$1 AND fingerprint=$2 LIMIT 1`,
		string(projectID), fingerprint)
	return scanFinding(row)
}

// ListByProject lists findings for a project.
func (r *FindingRepo) ListByProject(ctx context.Context, projectID contracts.ProjectID, limit, offset int) ([]*finding.Finding, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, findingSelect+` WHERE project_id=$1 ORDER BY last_seen DESC LIMIT $2 OFFSET $3`,
		string(projectID), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFindings(rows)
}

// ListByAssessment lists findings for an assessment.
func (r *FindingRepo) ListByAssessment(ctx context.Context, assessmentID string, limit, offset int) ([]*finding.Finding, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, findingSelect+` WHERE assessment_id=$1 ORDER BY last_seen DESC LIMIT $2 OFFSET $3`,
		assessmentID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFindings(rows)
}

const findingSelect = `
SELECT id, project_id, scope_id, task_id, assessment_id, execution_id, principal_id,
	tool_id, capability_id, severity, confidence, status, title, description,
	target, location, fingerprint, evidence_ids, first_seen, last_seen, produced_at,
	correlation_id, metadata
FROM findings`

func scanFindings(rows *sql.Rows) ([]*finding.Finding, error) {
	var out []*finding.Finding
	for rows.Next() {
		f, err := scanFinding(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func scanFinding(s scannable) (*finding.Finding, error) {
	var (
		id, projectID, scopeID, taskID, assessmentID, executionID, principalID string
		toolID, capabilityID, severity, confidence, status, title, description string
		target, location, fingerprint, correlationID                           string
		evRaw, metaRaw                                                         []byte
		firstSeen, lastSeen, producedAt                                        time.Time
	)
	err := s.Scan(&id, &projectID, &scopeID, &taskID, &assessmentID, &executionID, &principalID,
		&toolID, &capabilityID, &severity, &confidence, &status, &title, &description,
		&target, &location, &fingerprint, &evRaw, &firstSeen, &lastSeen, &producedAt,
		&correlationID, &metaRaw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, contracts.ErrNotFound
		}
		return nil, err
	}
	var evIDs []contracts.EvidenceID
	_ = json.Unmarshal(evRaw, &evIDs)
	meta := map[string]string{}
	_ = json.Unmarshal(metaRaw, &meta)
	if meta == nil {
		meta = map[string]string{}
	}
	return &finding.Finding{
		ID: contracts.FindingID(id), ProjectID: contracts.ProjectID(projectID),
		ScopeID: contracts.ScopeID(scopeID), TaskID: contracts.TaskID(taskID),
		AssessmentID: assessmentID, ExecutionID: executionID,
		PrincipalID: contracts.PrincipalID(principalID), ToolID: toolID,
		CapabilityID: contracts.CapabilityID(capabilityID),
		Severity:     contracts.FindingSeverity(severity), Confidence: contracts.FindingConfidence(confidence),
		Status: contracts.FindingStatus(status), Title: title, Description: description,
		Target: target, Location: location, Fingerprint: fingerprint, EvidenceIDs: evIDs,
		FirstSeen: firstSeen.UTC(), LastSeen: lastSeen.UTC(), ProducedAt: producedAt.UTC(),
		CorrelationID: contracts.CorrelationID(correlationID), Metadata: meta,
	}, nil
}
