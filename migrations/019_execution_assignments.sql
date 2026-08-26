-- Phase 17D — Durable remote execution assignments (no Grants/secrets)
BEGIN;
CREATE TABLE IF NOT EXISTS execution_assignments (
    id TEXT NOT NULL,
    execution_id TEXT NOT NULL,
    workflow_run_id TEXT NOT NULL DEFAULT '',
    project_id TEXT NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT '',
    host_id TEXT NOT NULL,
    worker_id TEXT NOT NULL,
    status TEXT NOT NULL,
    attempt INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 1,
    lease_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    dispatched_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    result_hash TEXT NOT NULL DEFAULT '',
    result_status TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    version BIGINT NOT NULL DEFAULT 1,
    PRIMARY KEY (project_id, id),
    UNIQUE (project_id, execution_id)
);
CREATE INDEX IF NOT EXISTS idx_exec_assign_status ON execution_assignments (project_id, status, lease_until);
CREATE INDEX IF NOT EXISTS idx_exec_assign_worker ON execution_assignments (project_id, worker_id);
INSERT INTO schema_migrations (version) VALUES ('019_execution_assignments') ON CONFLICT (version) DO NOTHING;
COMMIT;
