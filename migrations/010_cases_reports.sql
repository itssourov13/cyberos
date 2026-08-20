BEGIN;
CREATE TABLE IF NOT EXISTS security_cases (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL DEFAULT '',
    project_id TEXT NOT NULL,
    assessment_id TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    priority TEXT NOT NULL DEFAULT 'MEDIUM',
    finding_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    asset_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);
CREATE TABLE IF NOT EXISTS reports (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL DEFAULT '',
    project_id TEXT NOT NULL,
    assessment_id TEXT NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL,
    generated_by TEXT NOT NULL DEFAULT '',
    schema_version TEXT NOT NULL,
    status TEXT NOT NULL,
    snapshot JSONB NOT NULL DEFAULT '{}'::jsonb
);
INSERT INTO schema_migrations (version) VALUES ('010_cases_reports') ON CONFLICT (version) DO NOTHING;
COMMIT;
