package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/storage/ports"
)

type WorkerRepo struct{ db *DB }

func NewWorkerRepo(db *DB) *WorkerRepo { return &WorkerRepo{db: db} }

func (r *WorkerRepo) Register(ctx context.Context, w ports.WorkerRecord) error {
	caps, _ := json.Marshal(w.Capabilities)
	if caps == nil {
		caps = []byte("{}")
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO execution_workers (
			id, project_id, tenant_id, host_id, status, credential_ref, cert_fingerprint,
			capabilities_json, created_at, updated_at, last_seen_at, expires_at, version
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		w.ID, string(w.ProjectID), string(w.TenantID), w.HostID, w.Status, w.CredentialRef, w.CertFingerprint,
		caps, w.CreatedAt.UTC(), w.UpdatedAt.UTC(), nullTimeWorker(w.LastSeenAt), nullTimeWorker(w.ExpiresAt), w.Version,
	)
	if err != nil {
		return fmt.Errorf("register worker: %w", err)
	}
	return nil
}

func (r *WorkerRepo) Get(ctx context.Context, projectID contracts.ProjectID, workerID string) (ports.WorkerRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, tenant_id, host_id, status, credential_ref, cert_fingerprint,
			capabilities_json, created_at, updated_at, last_seen_at, expires_at, version
		FROM execution_workers WHERE project_id=$1 AND id=$2`, string(projectID), workerID)
	return scanWorker(row)
}

func (r *WorkerRepo) List(ctx context.Context, projectID contracts.ProjectID) ([]ports.WorkerRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, tenant_id, host_id, status, credential_ref, cert_fingerprint,
			capabilities_json, created_at, updated_at, last_seen_at, expires_at, version
		FROM execution_workers WHERE project_id=$1`, string(projectID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ports.WorkerRecord
	for rows.Next() {
		w, err := scanWorker(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *WorkerRepo) UpdateStatus(ctx context.Context, projectID contracts.ProjectID, workerID, status string, version int64) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE execution_workers SET status=$1, version=version+1, updated_at=$2
		WHERE project_id=$3 AND id=$4 AND version=$5`,
		status, time.Now().UTC(), string(projectID), workerID, version)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return contracts.ErrConflict
	}
	return nil
}

func (r *WorkerRepo) TouchHeartbeat(ctx context.Context, projectID contracts.ProjectID, workerID string, at time.Time) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE execution_workers SET last_seen_at=$1, updated_at=$1 WHERE project_id=$2 AND id=$3`,
		at.UTC(), string(projectID), workerID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return contracts.ErrNotFound
	}
	return nil
}

func (r *WorkerRepo) Revoke(ctx context.Context, projectID contracts.ProjectID, workerID string) error {
	w, err := r.Get(ctx, projectID, workerID)
	if err != nil {
		return err
	}
	return r.UpdateStatus(ctx, projectID, workerID, "REVOKED", w.Version)
}

type workerScannable interface {
	Scan(dest ...any) error
}

func scanWorker(row workerScannable) (ports.WorkerRecord, error) {
	var w ports.WorkerRecord
	var proj, ten string
	var caps []byte
	var last, exp sql.NullTime
	err := row.Scan(&w.ID, &proj, &ten, &w.HostID, &w.Status, &w.CredentialRef, &w.CertFingerprint,
		&caps, &w.CreatedAt, &w.UpdatedAt, &last, &exp, &w.Version)
	if err == sql.ErrNoRows {
		return w, contracts.ErrNotFound
	}
	if err != nil {
		return w, err
	}
	w.ProjectID = contracts.ProjectID(proj)
	w.TenantID = contracts.TenantID(ten)
	_ = json.Unmarshal(caps, &w.Capabilities)
	if last.Valid {
		w.LastSeenAt = last.Time
	}
	if exp.Valid {
		w.ExpiresAt = exp.Time
	}
	return w, nil
}

func nullTimeWorker(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC()
}
