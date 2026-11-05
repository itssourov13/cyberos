BEGIN;
-- Persist the permission level of each API key (Phase 24D). Existing tenant keys
-- keep their effective behaviour (analyst); tenant-less keys remain operators.
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'analyst';
UPDATE api_keys SET role='operator' WHERE tenant_id='' AND role='analyst';
INSERT INTO schema_migrations (version) VALUES ('024_api_key_role') ON CONFLICT (version) DO NOTHING;
COMMIT;
