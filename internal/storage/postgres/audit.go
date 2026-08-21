package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cyberos/internal/audit"
	"cyberos/internal/contracts"
)

// AuditBackend persists audit.Entry rows append-only.
type AuditBackend struct {
	db *DB
}

// NewAuditBackend constructs a PostgreSQL audit backend.
func NewAuditBackend(db *DB) *AuditBackend {
	return &AuditBackend{db: db}
}

// Append inserts an audit entry (no updates/deletes).
func (b *AuditBackend) Append(e audit.Entry) error {
	meta, _ := json.Marshal(e.Metadata)
	var pid, ptype any
	if e.Principal != nil {
		pid = string(e.Principal.ID)
		ptype = string(e.Principal.Type)
	}
	_, err := b.db.ExecContext(context.Background(), `
		INSERT INTO audit_entries (
			id, action, ts, principal_id, principal_type, project_id, scope_id, task_id,
			decision_id, grant_id, correlation_id, causation_id, outcome, reason, metadata
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		string(e.ID), e.Action, e.Timestamp.UTC(), pid, ptype,
		nullStr(string(e.ProjectID)), nullStr(string(e.ScopeID)), nullStr(string(e.TaskID)),
		nullStr(string(e.DecisionID)), nullStr(string(e.GrantID)),
		string(e.CorrelationID), nullStr(string(e.CausationID)),
		e.Outcome, e.Reason, meta,
	)
	if err != nil {
		return fmt.Errorf("insert audit: %w", err)
	}
	return nil
}

// ListByCorrelation returns audit entries for a correlation id ordered by seq.
func (b *AuditBackend) ListByCorrelation(id contracts.CorrelationID) ([]audit.Entry, error) {
	rows, err := b.db.QueryContext(context.Background(), `
		SELECT id, action, ts, principal_id, principal_type, project_id, scope_id, task_id,
			decision_id, grant_id, correlation_id, causation_id, outcome, reason, metadata
		FROM audit_entries WHERE correlation_id=$1 ORDER BY seq ASC`, string(id))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAuditRows(rows)
}

// All returns all audit entries in order.
func (b *AuditBackend) All() ([]audit.Entry, error) {
	rows, err := b.db.QueryContext(context.Background(), `
		SELECT id, action, ts, principal_id, principal_type, project_id, scope_id, task_id,
			decision_id, grant_id, correlation_id, causation_id, outcome, reason, metadata
		FROM audit_entries ORDER BY seq ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAuditRows(rows)
}

func scanAuditRows(rows *sql.Rows) ([]audit.Entry, error) {
	var out []audit.Entry
	for rows.Next() {
		e, err := scanAudit(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func scanAudit(s interface{ Scan(dest ...any) error }) (audit.Entry, error) {
	var (
		id, action, corr, outcome, reason                                 string
		pid, ptype, projectID, scopeID, taskID, decisionID, grantID, caus sql.NullString
		ts                                                                time.Time
		meta                                                              []byte
	)
	if err := s.Scan(&id, &action, &ts, &pid, &ptype, &projectID, &scopeID, &taskID,
		&decisionID, &grantID, &corr, &caus, &outcome, &reason, &meta); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return audit.Entry{}, contracts.ErrNotFound
		}
		return audit.Entry{}, err
	}
	e := audit.Entry{
		ID: contracts.AuditID(id), Action: action, Timestamp: ts.UTC(),
		ProjectID: contracts.ProjectID(projectID.String), ScopeID: contracts.ScopeID(scopeID.String),
		TaskID: contracts.TaskID(taskID.String), DecisionID: contracts.DecisionID(decisionID.String),
		GrantID: contracts.ExecutionGrantID(grantID.String), CorrelationID: contracts.CorrelationID(corr),
		CausationID: contracts.CausationID(caus.String), Outcome: outcome, Reason: reason,
	}
	if pid.Valid && pid.String != "" {
		e.Principal = &contracts.PrincipalRef{ID: contracts.PrincipalID(pid.String), Type: contracts.PrincipalType(ptype.String)}
	}
	_ = json.Unmarshal(meta, &e.Metadata)
	return e, nil
}
