BEGIN;
-- Durable server-side browser sessions. The raw cookie token is never stored;
-- API-key joins at lookup enforce live revocation and role changes.
CREATE TABLE IF NOT EXISTS auth_sessions (
    token_hash      TEXT PRIMARY KEY,
    api_key_id      TEXT NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    principal_type  TEXT NOT NULL DEFAULT 'human',
    csrf_token      TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    idle_until      TIMESTAMPTZ NOT NULL,
    CHECK (idle_until <= expires_at)
);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_api_key ON auth_sessions(api_key_id);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_expiry ON auth_sessions(expires_at, idle_until);
INSERT INTO schema_migrations (version) VALUES ('025_durable_auth_sessions')
ON CONFLICT (version) DO NOTHING;
COMMIT;
