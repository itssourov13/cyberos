-- Phase 16D — ASSISTED approval state (not authorization)
BEGIN;
ALTER TABLE autonomy_runs ADD COLUMN IF NOT EXISTS approval_state TEXT NOT NULL DEFAULT '';
INSERT INTO schema_migrations (version) VALUES ('017_autonomy_approval') ON CONFLICT (version) DO NOTHING;
COMMIT;
