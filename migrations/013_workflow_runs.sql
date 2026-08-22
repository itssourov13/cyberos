-- Phase 10C — durable workflow runs (orchestration metadata only)
BEGIN;
CREATE TABLE IF NOT EXISTS workflow_runs (
    id               TEXT PRIMARY KEY,
    workflow_id      TEXT NOT NULL,
    version          TEXT NOT NULL DEFAULT '1.0.0',
    project_id       TEXT NOT NULL,
    scope_id         TEXT NOT NULL,
    principal_id     TEXT NOT NULL DEFAULT '',
    tenant_id        TEXT NOT NULL DEFAULT '',
    state            TEXT NOT NULL,
    correlation_id   TEXT NOT NULL DEFAULT '',
    causation_id     TEXT NOT NULL DEFAULT '',
    error_message    TEXT NOT NULL DEFAULT '',
    started_at       TIMESTAMPTZ NOT NULL,
    finished_at      TIMESTAMPTZ,
    steps_json       JSONB NOT NULL DEFAULT '{}'::jsonb,
    step_to_task_json JSONB NOT NULL DEFAULT '{}'::jsonb
);
CREATE INDEX IF NOT EXISTS idx_workflow_runs_project ON workflow_runs(project_id);
CREATE INDEX IF NOT EXISTS idx_workflow_runs_state ON workflow_runs(state);
INSERT INTO schema_migrations (version) VALUES ('013_workflow_runs') ON CONFLICT (version) DO NOTHING;
COMMIT;
