-- Phase 5 — tenants + API keys + project tenant binding
BEGIN;

CREATE TABLE IF NOT EXISTS tenants (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'ACTIVE',
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    metadata    JSONB NOT NULL DEFAULT '{}'::jsonb
);

ALTER TABLE projects ADD COLUMN IF NOT EXISTS tenant_id TEXT;

CREATE TABLE IF NOT EXISTS api_keys (
    id            TEXT PRIMARY KEY,
    prefix        TEXT NOT NULL UNIQUE,
    hash          TEXT NOT NULL,
    principal_id  TEXT NOT NULL,
    tenant_id     TEXT NOT NULL DEFAULT '',
    active        BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys(prefix);
CREATE INDEX IF NOT EXISTS idx_projects_tenant ON projects(tenant_id);

INSERT INTO schema_migrations (version) VALUES ('006_auth_tenant')
ON CONFLICT (version) DO NOTHING;

COMMIT;
