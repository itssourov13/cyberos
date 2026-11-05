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

type AssessmentRepo struct {
	db *DB
}

func NewAssessmentRepo(db *DB) *AssessmentRepo {
	return &AssessmentRepo{db: db}
}

func (r *AssessmentRepo) Create(ctx context.Context, rec ports.AssessmentRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO assessments (id, project_id, scope_id, principal_id, title, target, state, steps, step_results, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		rec.ID, rec.ProjectID, rec.ScopeID, rec.PrincipalID, rec.Title, rec.Target, rec.State,
		nullBytes(rec.StepsJSON), nullBytes(rec.ResultsJSON),
		rec.CreatedAt.UTC(), rec.UpdatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert assessment: %w", err)
	}
	return nil
}

func (r *AssessmentRepo) Update(ctx context.Context, rec ports.AssessmentRecord) error {
	rec.UpdatedAt = time.Now().UTC()
	res, err := r.db.ExecContext(ctx, `
		UPDATE assessments SET title=$2, target=$3, state=$4, steps=$5, step_results=$6, updated_at=$7
		WHERE id=$1`,
		rec.ID, rec.Title, rec.Target, rec.State, nullBytes(rec.StepsJSON), nullBytes(rec.ResultsJSON), rec.UpdatedAt,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return contracts.ErrNotFound
	}
	return nil
}

func (r *AssessmentRepo) GetByID(ctx context.Context, id string) (ports.AssessmentRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, scope_id, principal_id, title, target, state, steps, step_results, created_at, updated_at
		FROM assessments WHERE id=$1`, id)
	return scanAssessment(row)
}

// ClaimRun is a compare-and-set transition. Only one API replica can claim a
// planned assessment; a restarted or competing worker cannot replay it.
func (r *AssessmentRepo) ClaimRun(ctx context.Context, id string) (ports.AssessmentRecord, error) {
	if id == "" {
		return ports.AssessmentRecord{}, contracts.ErrInvalidArgument
	}
	rec, err := scanAssessment(r.db.QueryRowContext(ctx, `
		UPDATE assessments
		   SET state='RUNNING', updated_at=$2
		 WHERE id=$1 AND state='PLANNED'
		RETURNING id, project_id, scope_id, principal_id, title, target, state, steps, step_results, created_at, updated_at`,
		id, time.Now().UTC()))
	if err == nil {
		return rec, nil
	}
	if !errors.Is(err, contracts.ErrNotFound) {
		return ports.AssessmentRecord{}, err
	}
	if _, lookupErr := r.GetByID(ctx, id); errors.Is(lookupErr, contracts.ErrNotFound) {
		return ports.AssessmentRecord{}, contracts.ErrNotFound
	} else if lookupErr != nil {
		return ports.AssessmentRecord{}, lookupErr
	}
	return ports.AssessmentRecord{}, fmt.Errorf("%w: assessment is not in PLANNED state", contracts.ErrConflict)
}

func (r *AssessmentRepo) ListByProject(ctx context.Context, projectID contracts.ProjectID, limit, offset int) ([]ports.AssessmentRecord, error) {
	if projectID == "" {
		return nil, contracts.ErrInvalidArgument
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if offset < 0 {
		return nil, contracts.ErrInvalidArgument
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, scope_id, principal_id, title, target, state, steps, step_results, created_at, updated_at
		FROM assessments WHERE project_id=$1
		ORDER BY created_at DESC, id ASC LIMIT $2 OFFSET $3`, string(projectID), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ports.AssessmentRecord
	for rows.Next() {
		rec, err := scanAssessment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *AssessmentRepo) List(ctx context.Context, limit, offset int) ([]ports.AssessmentRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, scope_id, principal_id, title, target, state, steps, step_results, created_at, updated_at
		FROM assessments ORDER BY created_at DESC, id ASC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ports.AssessmentRecord
	for rows.Next() {
		rec, err := scanAssessment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func scanAssessment(s interface{ Scan(dest ...any) error }) (ports.AssessmentRecord, error) {
	var rec ports.AssessmentRecord
	var steps, results []byte
	var created, updated time.Time
	err := s.Scan(&rec.ID, &rec.ProjectID, &rec.ScopeID, &rec.PrincipalID, &rec.Title, &rec.Target, &rec.State,
		&steps, &results, &created, &updated)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return rec, contracts.ErrNotFound
		}
		return rec, err
	}
	rec.StepsJSON = steps
	rec.ResultsJSON = results
	rec.CreatedAt = created.UTC()
	rec.UpdatedAt = updated.UTC()
	return rec, nil
}

func nullBytes(b []byte) any {
	if len(b) == 0 {
		return []byte("[]")
	}
	return b
}
