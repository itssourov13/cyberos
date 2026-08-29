package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/tools"
)

// PackTrustRepo is the PostgreSQL TrustStorePort (Phase 19B).
type PackTrustRepo struct{ db *DB }

func NewPackTrustRepo(db *DB) *PackTrustRepo { return &PackTrustRepo{db: db} }

func (r *PackTrustRepo) TrustPack(project contracts.ProjectID, digest string) error {
	return r.TrustPackCtx(context.Background(), project, digest)
}

func (r *PackTrustRepo) TrustPackCtx(ctx context.Context, project contracts.ProjectID, digest string) error {
	if err := tools.ValidateDigestFormat(digest); err != nil || digest == "" {
		return fmt.Errorf("%w: digest required", contracts.ErrInvalidArgument)
	}
	var state string
	err := r.db.QueryRowContext(ctx,
		`SELECT state FROM pack_trust WHERE project_id=$1 AND pack_content_digest=$2`,
		string(project), digest).Scan(&state)
	if err == nil && state == "REVOKED" {
		return fmt.Errorf("%w: pack digest revoked", contracts.ErrConflict)
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO pack_trust (project_id, pack_content_digest, state, updated_at)
		VALUES ($1,$2,'TRUSTED',NOW())
		ON CONFLICT (project_id, pack_content_digest) DO UPDATE
		SET state='TRUSTED', updated_at=NOW(), revoked_at=NULL
		WHERE pack_trust.state <> 'REVOKED'`,
		string(project), digest)
	if err != nil {
		return fmt.Errorf("trust pack: %w", err)
	}
	// Re-check: if still revoked, conflict
	err = r.db.QueryRowContext(ctx,
		`SELECT state FROM pack_trust WHERE project_id=$1 AND pack_content_digest=$2`,
		string(project), digest).Scan(&state)
	if err == sql.ErrNoRows {
		return fmt.Errorf("%w: trust pack failed", contracts.ErrConflict)
	}
	if state == "REVOKED" {
		return fmt.Errorf("%w: pack digest revoked", contracts.ErrConflict)
	}
	return nil
}

func (r *PackTrustRepo) RevokePack(project contracts.ProjectID, digest, reason, by string) error {
	return r.RevokePackCtx(context.Background(), project, digest, reason, by)
}

func (r *PackTrustRepo) RevokePackCtx(ctx context.Context, project contracts.ProjectID, digest, reason, by string) error {
	if err := tools.ValidateDigestFormat(digest); err != nil || digest == "" {
		return fmt.Errorf("%w: digest required for revocation", contracts.ErrInvalidArgument)
	}
	if reason == "" {
		reason = "revoked"
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO pack_trust (project_id, pack_content_digest, state, reason, revoked_at, updated_at)
		VALUES ($1,$2,'REVOKED',$3,NOW(),NOW())
		ON CONFLICT (project_id, pack_content_digest) DO UPDATE
		SET state='REVOKED', reason=$3, revoked_at=NOW(), updated_at=NOW()`,
		string(project), digest, reason)
	if err != nil {
		return fmt.Errorf("revoke pack: %w", err)
	}
	return nil
}

func (r *PackTrustRepo) IsTrusted(project contracts.ProjectID, digest string) bool {
	ok, err := r.IsTrustedCtx(context.Background(), project, digest)
	return err == nil && ok
}

func (r *PackTrustRepo) IsTrustedCtx(ctx context.Context, project contracts.ProjectID, digest string) (bool, error) {
	var state string
	err := r.db.QueryRowContext(ctx,
		`SELECT state FROM pack_trust WHERE project_id=$1 AND pack_content_digest=$2`,
		string(project), digest).Scan(&state)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return state == "TRUSTED", nil
}

func (r *PackTrustRepo) IsRevoked(project contracts.ProjectID, digest string) bool {
	ok, err := r.IsRevokedCtx(context.Background(), project, digest)
	return err == nil && ok
}

func (r *PackTrustRepo) IsRevokedCtx(ctx context.Context, project contracts.ProjectID, digest string) (bool, error) {
	var state string
	err := r.db.QueryRowContext(ctx,
		`SELECT state FROM pack_trust WHERE project_id=$1 AND pack_content_digest=$2`,
		string(project), digest).Scan(&state)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return state == "REVOKED", nil
}

// SignerKeyRepo is the PostgreSQL SignerKeyRegistryPort (Phase 19B).
type SignerKeyRepo struct{ db *DB }

func NewSignerKeyRepo(db *DB) *SignerKeyRepo { return &SignerKeyRepo{db: db} }

func (r *SignerKeyRepo) Register(key tools.SignerKey) error {
	return r.RegisterCtx(context.Background(), key)
}

func (r *SignerKeyRepo) RegisterCtx(ctx context.Context, key tools.SignerKey) error {
	if err := contracts.ValidateNonEmpty("key_id", key.KeyID); err != nil {
		return err
	}
	if key.Algorithm == "" {
		key.Algorithm = tools.PackSigAlgorithm
	}
	if key.Algorithm != tools.PackSigAlgorithm {
		return fmt.Errorf("%w: unsupported algorithm", contracts.ErrInvalidArgument)
	}
	if len(key.PublicKey) != 32 {
		return fmt.Errorf("%w: invalid public key size", contracts.ErrInvalidArgument)
	}
	status := string(key.Status)
	if status == "" {
		status = string(tools.SignerKeyActive)
	}
	created := key.CreatedAt
	if created.IsZero() {
		created = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO signer_keys (key_id, algorithm, public_key, status, created_at)
		VALUES ($1,$2,$3,$4,$5)`,
		key.KeyID, key.Algorithm, []byte(key.PublicKey), status, created)
	if err != nil {
		return fmt.Errorf("%w: key_id already registered or insert failed: %v", contracts.ErrConflict, err)
	}
	return nil
}

func (r *SignerKeyRepo) Get(keyID string) (tools.SignerKey, error) {
	return r.GetCtx(context.Background(), keyID)
}

func (r *SignerKeyRepo) GetCtx(ctx context.Context, keyID string) (tools.SignerKey, error) {
	var k tools.SignerKey
	var pk []byte
	var status string
	err := r.db.QueryRowContext(ctx,
		`SELECT key_id, algorithm, public_key, status, created_at FROM signer_keys WHERE key_id=$1`,
		keyID).Scan(&k.KeyID, &k.Algorithm, &pk, &status, &k.CreatedAt)
	if err == sql.ErrNoRows {
		return tools.SignerKey{}, contracts.ErrNotFound
	}
	if err != nil {
		return tools.SignerKey{}, err
	}
	k.PublicKey = pk
	k.Status = tools.SignerKeyStatus(status)
	return k, nil
}

func (r *SignerKeyRepo) Disable(keyID string) error {
	res, err := r.db.ExecContext(context.Background(),
		`UPDATE signer_keys SET status='DISABLED', disabled_at=NOW() WHERE key_id=$1 AND status='ACTIVE'`, keyID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		// may already be disabled/revoked or missing
		_, err := r.Get(keyID)
		return err
	}
	return nil
}

func (r *SignerKeyRepo) Revoke(keyID string) error {
	res, err := r.db.ExecContext(context.Background(),
		`UPDATE signer_keys SET status='REVOKED', revoked_at=NOW() WHERE key_id=$1`, keyID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return contracts.ErrNotFound
	}
	return nil
}
