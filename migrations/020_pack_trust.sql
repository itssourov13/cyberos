-- Phase 19B: durable signer keys (public only) and project-scoped pack trust.

CREATE TABLE IF NOT EXISTS signer_keys (
    key_id      TEXT PRIMARY KEY,
    algorithm   TEXT NOT NULL DEFAULT 'ed25519',
    public_key  BYTEA NOT NULL,
    status      TEXT NOT NULL DEFAULT 'ACTIVE'
                CHECK (status IN ('ACTIVE', 'DISABLED', 'REVOKED')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    disabled_at TIMESTAMPTZ,
    revoked_at  TIMESTAMPTZ,
    CONSTRAINT signer_keys_algo_ed25519 CHECK (algorithm = 'ed25519'),
    CONSTRAINT signer_keys_pubkey_len CHECK (octet_length(public_key) = 32)
);

CREATE TABLE IF NOT EXISTS pack_trust (
    project_id           TEXT NOT NULL,
    pack_content_digest  TEXT NOT NULL,
    state                TEXT NOT NULL
                         CHECK (state IN ('TRUSTED', 'REVOKED')),
    reason               TEXT NOT NULL DEFAULT '',
    key_id               TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at           TIMESTAMPTZ,
    PRIMARY KEY (project_id, pack_content_digest)
);

CREATE INDEX IF NOT EXISTS idx_pack_trust_state ON pack_trust (project_id, state);
