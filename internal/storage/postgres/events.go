package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/events"
)

// EventBackend persists EventEnvelope rows append-only.
type EventBackend struct {
	db *DB
}

// NewEventBackend constructs a PostgreSQL event backend.
func NewEventBackend(db *DB) *EventBackend {
	return &EventBackend{db: db}
}

func (b *EventBackend) Append(env contracts.EventEnvelope) error {
	payload, _ := json.Marshal(env.Payload)
	var pid, ptype any
	if env.Principal != nil {
		pid = string(env.Principal.ID)
		ptype = string(env.Principal.Type)
	}
	_, err := b.db.ExecContext(context.Background(), `
		INSERT INTO events (
			id, event_type, schema_version, ts, principal_id, principal_type,
			project_id, scope_id, task_id, correlation_id, causation_id, payload
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		string(env.ID), string(env.Type), string(env.SchemaVersion), env.Timestamp.UTC(),
		pid, ptype, nullStr(string(env.ProjectID)), nullStr(string(env.ScopeID)), nullStr(string(env.TaskID)),
		string(env.CorrelationID), nullStr(string(env.CausationID)), payload,
	)
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}
	return nil
}

func (b *EventBackend) Get(id contracts.EventID) (contracts.EventEnvelope, error) {
	row := b.db.QueryRowContext(context.Background(), `
		SELECT id, event_type, schema_version, ts, principal_id, principal_type,
			project_id, scope_id, task_id, correlation_id, causation_id, payload
		FROM events WHERE id=$1`, string(id))
	return scanEvent(row)
}

func (b *EventBackend) All() ([]contracts.EventEnvelope, error) {
	rows, err := b.db.QueryContext(context.Background(), `
		SELECT id, event_type, schema_version, ts, principal_id, principal_type,
			project_id, scope_id, task_id, correlation_id, causation_id, payload
		FROM events ORDER BY seq ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvents(rows)
}

func (b *EventBackend) ByCorrelation(id contracts.CorrelationID) ([]contracts.EventEnvelope, error) {
	rows, err := b.db.QueryContext(context.Background(), `
		SELECT id, event_type, schema_version, ts, principal_id, principal_type,
			project_id, scope_id, task_id, correlation_id, causation_id, payload
		FROM events WHERE correlation_id=$1 ORDER BY seq ASC`, string(id))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvents(rows)
}

func (b *EventBackend) ByType(t contracts.EventType) ([]contracts.EventEnvelope, error) {
	rows, err := b.db.QueryContext(context.Background(), `
		SELECT id, event_type, schema_version, ts, principal_id, principal_type,
			project_id, scope_id, task_id, correlation_id, causation_id, payload
		FROM events WHERE event_type=$1 ORDER BY seq ASC`, string(t))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvents(rows)
}

func (b *EventBackend) Len() (int, error) {
	var n int
	err := b.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM events`).Scan(&n)
	return n, err
}

func scanEvents(rows *sql.Rows) ([]contracts.EventEnvelope, error) {
	var out []contracts.EventEnvelope
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func scanEvent(s interface{ Scan(dest ...any) error }) (contracts.EventEnvelope, error) {
	var (
		id, typ, ver, corr string
		pid, ptype, projectID, scopeID, taskID, caus sql.NullString
		ts time.Time
		payload []byte
	)
	if err := s.Scan(&id, &typ, &ver, &ts, &pid, &ptype, &projectID, &scopeID, &taskID, &corr, &caus, &payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contracts.EventEnvelope{}, contracts.ErrNotFound
		}
		return contracts.EventEnvelope{}, err
	}
	env := contracts.EventEnvelope{
		ID: contracts.EventID(id), Type: contracts.EventType(typ), SchemaVersion: contracts.SchemaVersion(ver),
		Timestamp: ts.UTC(), CorrelationID: contracts.CorrelationID(corr),
		ProjectID: contracts.ProjectID(projectID.String), ScopeID: contracts.ScopeID(scopeID.String),
		TaskID: contracts.TaskID(taskID.String), CausationID: contracts.CausationID(caus.String),
	}
	if pid.Valid && pid.String != "" {
		env.Principal = &contracts.PrincipalRef{ID: contracts.PrincipalID(pid.String), Type: contracts.PrincipalType(ptype.String)}
	}
	_ = json.Unmarshal(payload, &env.Payload)
	return env, nil
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

var _ events.Backend = (*EventBackend)(nil)
