-- Phase 22B: Assignment ↔ Grant / PackContentDigest binding (no private keys).
BEGIN;
ALTER TABLE execution_assignments
  ADD COLUMN IF NOT EXISTS grant_id TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS pack_content_digest TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_exec_assign_grant ON execution_assignments (project_id, grant_id);
INSERT INTO schema_migrations (version) VALUES ('021_assignment_binding') ON CONFLICT (version) DO NOTHING;
COMMIT;
