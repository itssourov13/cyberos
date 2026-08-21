-- Phase 9D — durable execution lifecycle records
BEGIN;
CREATE TABLE IF NOT EXISTS execution_records (
    id              TEXT PRIMARY KEY,
    state           TEXT NOT NULL,
    task_id         TEXT NOT NULL,
    grant_id        TEXT NOT NULL,
    principal_id    TEXT NOT NULL,
    principal_type  TEXT NOT NULL DEFAULT '',
    project_id      TEXT NOT NULL,
    scope_id        TEXT NOT NULL,
    capability_id   TEXT NOT NULL,
    attempt         INT NOT NULL DEFAULT 1,
    started_at      TIMESTAMPTZ NOT NULL,
    finished_at     TIMESTAMPTZ,
    duration_ms     BIGINT NOT NULL DEFAULT 0,
    outcome         TEXT NOT NULL DEFAULT '',
    failure_reason  TEXT NOT NULL DEFAULT '',
    executor_id     TEXT NOT NULL DEFAULT '',
    correlation_id  TEXT NOT NULL DEFAULT '',
    causation_id    TEXT NOT NULL DEFAULT '',
    idempotency_key TEXT NOT NULL DEFAULT '',
    tool_id         TEXT NOT NULL DEFAULT '',
    pack_version    TEXT NOT NULL DEFAULT '',
    tenant_id       TEXT NOT NULL DEFAULT '',
    metrics         JSONB NOT NULL DEFAULT '{}'::jsonb
);
CREATE INDEX IF NOT EXISTS idx_exec_records_task ON execution_records(task_id);
CREATE INDEX IF NOT EXISTS idx_exec_records_project ON execution_records(project_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_exec_records_idem ON execution_records(idempotency_key) WHERE idempotency_key <> '';
INSERT INTO schema_migrations (version) VALUES ('012_execution_records') ON CONFLICT (version) DO NOTHING;
COMMIT;
