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
		SELECT id, prefix, hash, principal_id, tenant_id, active, expires_at, created_at
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
		INSERT INTO api_keys (id, prefix, hash, principal_id, tenant_id, active, expires_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		key.ID, key.Prefix, key.Hash, string(key.PrincipalID), string(key.TenantID),
		key.Active, exp, key.CreatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert api key: %w", err)
	}
	return nil
}

func (s *APIKeyStore) CountActive(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM api_keys WHERE active=TRUE`).Scan(&n)
	return n, err
}

func (s *APIKeyStore) Revoke(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE api_keys SET active=FALSE WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return contracts.ErrNotFound
	}
	return nil
}

func scanAPIKey(s scannable) (auth.APIKey, error) {
	var (
		id, prefix, hash, principalID, tenantID string
		active                                  bool
		exp                                     sql.NullTime
		created                                 time.Time
	)
	err := s.Scan(&id, &prefix, &hash, &principalID, &tenantID, &active, &exp, &created)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return auth.APIKey{}, contracts.ErrNotFound
		}
		return auth.APIKey{}, err
	}
	k := auth.APIKey{
		ID: id, Prefix: prefix, Hash: hash,
		PrincipalID: contracts.PrincipalID(principalID), TenantID: contracts.TenantID(tenantID),
		Active: active, CreatedAt: created.UTC(),
	}
	if exp.Valid {
		t := exp.Time.UTC()
		k.ExpiresAt = &t
	}
	return k, nil
}
