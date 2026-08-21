package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/provenance"
)

// ProvenanceRepo persists provenance records (Phase 8A).
type ProvenanceRepo struct{ db *DB }

func NewProvenanceRepo(db *DB) *ProvenanceRepo { return &ProvenanceRepo{db: db} }

func (r *ProvenanceRepo) Upsert(ctx context.Context, p *provenance.Provenance) error {
	if p == nil {
		return contracts.ErrInvalidArgument
	}
	if err := p.Validate(); err != nil {
		return err
	}
	parents, _ := json.Marshal(p.ParentIDs)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO provenance (
			subject_type, subject_id, principal_id, principal_type, project_id, scope_id,
			task_id, capability_id, grant_id, executor_id, produced_at, correlation_id, parent_ids
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (subject_type, subject_id) DO UPDATE SET
			principal_id=EXCLUDED.principal_id, principal_type=EXCLUDED.principal_type,
			project_id=EXCLUDED.project_id, scope_id=EXCLUDED.scope_id, task_id=EXCLUDED.task_id,
			capability_id=EXCLUDED.capability_id, grant_id=EXCLUDED.grant_id, executor_id=EXCLUDED.executor_id,
			produced_at=EXCLUDED.produced_at, correlation_id=EXCLUDED.correlation_id, parent_ids=EXCLUDED.parent_ids`,
		p.SubjectType, p.SubjectID, string(p.Principal.ID), string(p.Principal.Type),
		string(p.ProjectID), string(p.ScopeID), string(p.TaskID), string(p.CapabilityID),
		string(p.GrantID), p.ExecutorID, p.ProducedAt.UTC(), string(p.CorrelationID), parents,
	)
	if err != nil {
		return fmt.Errorf("upsert provenance: %w", err)
	}
	return nil
}

func (r *ProvenanceRepo) Get(ctx context.Context, subjectType, subjectID string) (*provenance.Provenance, error) {
	row := r.db.QueryRowContext(ctx, provenanceSelect+` WHERE subject_type=$1 AND subject_id=$2`, subjectType, subjectID)
	return scanProvenance(row)
}

func (r *ProvenanceRepo) ListByProject(ctx context.Context, projectID contracts.ProjectID, limit, offset int) ([]*provenance.Provenance, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, provenanceSelect+` WHERE project_id=$1 ORDER BY produced_at DESC LIMIT $2 OFFSET $3`,
		string(projectID), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*provenance.Provenance
	for rows.Next() {
		p, err := scanProvenance(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

const provenanceSelect = `
SELECT subject_type, subject_id, principal_id, principal_type, project_id, scope_id,
	task_id, capability_id, grant_id, executor_id, produced_at, correlation_id, parent_ids
FROM provenance`

func scanProvenance(s scannable) (*provenance.Provenance, error) {
	var (
		subjectType, subjectID, principalID, principalType string
		projectID, scopeID, taskID, capID, grantID, execID string
		corr                                               string
		parentsRaw                                         []byte
		produced                                           time.Time
	)
	err := s.Scan(&subjectType, &subjectID, &principalID, &principalType, &projectID, &scopeID,
		&taskID, &capID, &grantID, &execID, &produced, &corr, &parentsRaw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, contracts.ErrNotFound
		}
		return nil, err
	}
	var parents []string
	_ = json.Unmarshal(parentsRaw, &parents)
	return &provenance.Provenance{
		Provenance: contracts.Provenance{
			SubjectType: subjectType, SubjectID: subjectID,
			Principal: contracts.PrincipalRef{
				ID: contracts.PrincipalID(principalID), Type: contracts.PrincipalType(principalType),
			},
			ProjectID: contracts.ProjectID(projectID), ScopeID: contracts.ScopeID(scopeID),
			TaskID: contracts.TaskID(taskID), CapabilityID: contracts.CapabilityID(capID),
			GrantID: contracts.ExecutionGrantID(grantID), ExecutorID: execID,
			ProducedAt: produced.UTC(), CorrelationID: contracts.CorrelationID(corr), ParentIDs: parents,
		},
	}, nil
}
