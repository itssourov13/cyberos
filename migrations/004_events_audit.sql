-- Phase 4B — durable events + audit
BEGIN;

CREATE TABLE IF NOT EXISTS events (
    id              TEXT PRIMARY KEY,
    event_type      TEXT NOT NULL,
    schema_version  TEXT NOT NULL DEFAULT '1',
    ts              TIMESTAMPTZ NOT NULL,
    principal_id    TEXT,
    principal_type  TEXT,
    project_id      TEXT,
    scope_id        TEXT,
    task_id         TEXT,
    correlation_id  TEXT NOT NULL,
    causation_id    TEXT,
    payload         JSONB NOT NULL DEFAULT '{}'::jsonb,
    seq             BIGSERIAL
);

CREATE INDEX IF NOT EXISTS idx_events_correlation ON events(correlation_id, seq);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type, seq);
CREATE INDEX IF NOT EXISTS idx_events_ts ON events(ts, seq);

CREATE TABLE IF NOT EXISTS audit_entries (
    id              TEXT PRIMARY KEY,
    action          TEXT NOT NULL,
    ts              TIMESTAMPTZ NOT NULL,
    principal_id    TEXT,
    principal_type  TEXT,
    project_id      TEXT,
    scope_id        TEXT,
    task_id         TEXT,
    decision_id     TEXT,
    grant_id        TEXT,
    correlation_id  TEXT NOT NULL DEFAULT '',
    causation_id    TEXT,
    outcome         TEXT NOT NULL DEFAULT '',
    reason          TEXT NOT NULL DEFAULT '',
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    seq             BIGSERIAL
);

CREATE INDEX IF NOT EXISTS idx_audit_correlation ON audit_entries(correlation_id, seq);
CREATE INDEX IF NOT EXISTS idx_audit_ts ON audit_entries(ts, seq);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_entries(action, seq);

INSERT INTO schema_migrations (version) VALUES ('004_events_audit')
ON CONFLICT (version) DO NOTHING;

COMMIT;
