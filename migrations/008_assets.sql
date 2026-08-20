-- Phase 6E — assets inventory (optional durable wire-up)
BEGIN;
CREATE TABLE IF NOT EXISTS assets (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    value TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);
CREATE INDEX IF NOT EXISTS idx_assets_project_value ON assets(project_id, value);
CREATE TABLE IF NOT EXISTS asset_endpoints (
    id TEXT PRIMARY KEY,
    asset_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    protocol TEXT NOT NULL DEFAULT '',
    port INT NOT NULL DEFAULT 0,
    path TEXT NOT NULL DEFAULT '',
    banner TEXT NOT NULL DEFAULT '',
    first_seen TIMESTAMPTZ NOT NULL,
    last_seen TIMESTAMPTZ NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);
CREATE TABLE IF NOT EXISTS asset_observations (
    id TEXT PRIMARY KEY,
    asset_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    scope_id TEXT NOT NULL DEFAULT '',
    task_id TEXT NOT NULL DEFAULT '',
    execution_id TEXT NOT NULL DEFAULT '',
    tool_id TEXT NOT NULL DEFAULT '',
    capability_id TEXT NOT NULL DEFAULT '',
    kind TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    detail TEXT NOT NULL DEFAULT '',
    endpoint_id TEXT NOT NULL DEFAULT '',
    observed_at TIMESTAMPTZ NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);
INSERT INTO schema_migrations (version) VALUES ('008_assets') ON CONFLICT (version) DO NOTHING;
COMMIT;
