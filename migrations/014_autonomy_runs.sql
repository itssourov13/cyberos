-- Phase 14A — durable AutonomyRun (eligibility state only; no grants)
BEGIN;
CREATE TABLE IF NOT EXISTS autonomy_runs (
    id                 TEXT PRIMARY KEY,
    project_id         TEXT NOT NULL,
    tenant_id          TEXT NOT NULL DEFAULT '',
    scope_id           TEXT NOT NULL DEFAULT '',
    principal_id       TEXT NOT NULL DEFAULT '',
    correlation_id     TEXT NOT NULL DEFAULT '',
    goal               TEXT NOT NULL DEFAULT '',
    mode               TEXT NOT NULL DEFAULT '',
    policy_version     TEXT NOT NULL DEFAULT '1',
    policy_json        JSONB NOT NULL DEFAULT '{}'::jsonb,
    state              TEXT NOT NULL,
    iteration          INT NOT NULL DEFAULT 0,
    steps_consumed     INT NOT NULL DEFAULT 0,
    analysis_id        TEXT NOT NULL DEFAULT '',
    plan_id            TEXT NOT NULL DEFAULT '',
    parent_plan_id     TEXT NOT NULL DEFAULT '',
    workflow_def_id    TEXT NOT NULL DEFAULT '',
    workflow_run_id    TEXT NOT NULL DEFAULT '',
    last_decision      TEXT NOT NULL DEFAULT '',
    stop_reason        TEXT NOT NULL DEFAULT '',
    version            BIGINT NOT NULL DEFAULT 1,
    started_at         TIMESTAMPTZ NOT NULL,
    updated_at         TIMESTAMPTZ NOT NULL,
    finished_at        TIMESTAMPTZ,
    cancelled_at       TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_autonomy_runs_project ON autonomy_runs(project_id);
CREATE INDEX IF NOT EXISTS idx_autonomy_runs_state ON autonomy_runs(state);
CREATE INDEX IF NOT EXISTS idx_autonomy_runs_project_state ON autonomy_runs(project_id, state);
INSERT INTO schema_migrations (version) VALUES ('014_autonomy_runs') ON CONFLICT (version) DO NOTHING;
COMMIT;
