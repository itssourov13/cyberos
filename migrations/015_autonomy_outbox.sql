-- Phase 15C — durable autonomy event outbox (delivery only; not authorization)
BEGIN;
CREATE TABLE IF NOT EXISTS autonomy_outbox (
    id               TEXT PRIMARY KEY,
    project_id       TEXT NOT NULL,
    tenant_id        TEXT NOT NULL DEFAULT '',
    scope_id         TEXT NOT NULL DEFAULT '',
    autonomy_run_id  TEXT NOT NULL DEFAULT '',
    workflow_run_id  TEXT NOT NULL DEFAULT '',
    execution_id     TEXT NOT NULL DEFAULT '',
    event_type       TEXT NOT NULL,
    schema_version   TEXT NOT NULL DEFAULT '1',
    idempotency_key  TEXT NOT NULL,
    event_id         TEXT NOT NULL DEFAULT '',
    payload_json     JSONB NOT NULL DEFAULT '{}'::jsonb,
    status           TEXT NOT NULL DEFAULT 'PENDING',
    attempts         INT NOT NULL DEFAULT 0,
    available_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_until     TIMESTAMPTZ,
    locked_by        TEXT NOT NULL DEFAULT '',
    last_error       TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, idempotency_key)
);
CREATE INDEX IF NOT EXISTS idx_autonomy_outbox_claim
    ON autonomy_outbox (status, available_at)
    WHERE status IN ('PENDING', 'CLAIMED');
CREATE INDEX IF NOT EXISTS idx_autonomy_outbox_project ON autonomy_outbox (project_id);
CREATE INDEX IF NOT EXISTS idx_autonomy_outbox_run ON autonomy_outbox (autonomy_run_id);
INSERT INTO schema_migrations (version) VALUES ('015_autonomy_outbox') ON CONFLICT (version) DO NOTHING;
COMMIT;
