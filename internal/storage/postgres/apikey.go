package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"cyberos/internal/auth"
	"cyberos/internal/contracts"
)

// APIKeyStore is a PostgreSQL KeyStore implementation (Phase 7B).
type APIKeyStore struct{ db *DB }

func NewAPIKeyStore(db *DB) *APIKeyStore { return &APIKeyStore{db: db} }

func (s *APIKeyStore) LookupByPrefix(ctx context.Context, prefix string) (auth.APIKey, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, prefix, hash, principal_id, tenant_id, active, expires_at, created_at, role
		FROM api_keys WHERE prefix=$1`, prefix)
	return scanAPIKey(row)
}

func (s *APIKeyStore) Create(ctx context.Context, key auth.APIKey) error {
	if key.ID == "" || key.Prefix == "" || key.Hash == "" {
		return contracts.ErrInvalidArgument
	}
	var exp any
	if key.ExpiresAt != nil {
		exp = key.ExpiresAt.UTC()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO api_keys (id, prefix, hash, principal_id, tenant_id, active, expires_at, created_at, role)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		key.ID, key.Prefix, key.Hash, string(key.PrincipalID), string(key.TenantID),
		key.Active, exp, key.CreatedAt.UTC(), roleOrDefault(key),
	)
	if err != nil {
		return fmt.Errorf("insert api key: %w", err)
	}
	return nil
}

func (s *APIKeyStore) CountActive(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM api_keys WHERE active=TRUE AND (expires_at IS NULL OR expires_at > NOW())`).Scan(&n)
	return n, err
}

func (s *APIKeyStore) CountActiveOperators(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM api_keys WHERE active=TRUE AND COALESCE(tenant_id,'')='' AND role='operator' AND (expires_at IS NULL OR expires_at > NOW())`).Scan(&n)
	return n, err
}

func (s *APIKeyStore) Revoke(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin API key revocation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, `UPDATE api_keys SET active=FALSE WHERE id=$1 AND active=TRUE`, id)
	if err != nil {
		return fmt.Errorf("revoke API key: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM api_keys WHERE id=$1)`, id).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return contracts.ErrNotFound
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM auth_sessions WHERE api_key_id=$1`, id); err != nil {
		return fmt.Errorf("invalidate API key sessions: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit API key revocation: %w", err)
	}
	return nil
}

func scanAPIKey(s scannable) (auth.APIKey, error) {
	var (
		id, prefix, hash, principalID, tenantID, role string
		active                                        bool
		exp                                           sql.NullTime
		created                                       time.Time
	)
	err := s.Scan(&id, &prefix, &hash, &principalID, &tenantID, &active, &exp, &created, &role)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return auth.APIKey{}, contracts.ErrNotFound
		}
		return auth.APIKey{}, err
	}
	parsedRole := auth.ParseRole(role)
	if role == "" {
		// Compatibility for pre-role rows only; migration 024 backfills all rows.
		if tenantID == "" {
			parsedRole = auth.RoleOperator
		} else {
			parsedRole = auth.RoleAnalyst
		}
	} else if parsedRole == "" {
		return auth.APIKey{}, fmt.Errorf("stored API key %q has invalid role", id)
	}
	if tenantID == "" && parsedRole != auth.RoleOperator {
		return auth.APIKey{}, fmt.Errorf("tenant-less API key %q must have operator role", id)
	}
	if tenantID != "" && parsedRole == auth.RoleOperator {
		return auth.APIKey{}, fmt.Errorf("tenant-scoped API key %q cannot have operator role", id)
	}
	k := auth.APIKey{
		ID: id, Prefix: prefix, Hash: hash,
		PrincipalID: contracts.PrincipalID(principalID), TenantID: contracts.TenantID(tenantID),
		Role: parsedRole, Active: active, CreatedAt: created.UTC(),
	}
	if exp.Valid {
		t := exp.Time.UTC()
		k.ExpiresAt = &t
	}
	return k, nil
}

// roleOrDefault stores the key's role; legacy callers that left it empty get
// analyst (tenant) or operator (tenant-less), matching RoleFromContext.
func roleOrDefault(k auth.APIKey) string {
	if k.Role != "" {
		return string(k.Role)
	}
	if k.TenantID == "" {
		return string(auth.RoleOperator)
	}
	return string(auth.RoleAnalyst)
}
