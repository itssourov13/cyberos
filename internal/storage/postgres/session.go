package postgres

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"cyberos/internal/auth"
	"cyberos/internal/contracts"
)

// SessionStore is a PostgreSQL-backed, multi-instance session repository.
// Only a SHA-256 digest of the opaque browser token is persisted. Lookup joins
// the API-key table on every request, so revocation/expiry/role changes take
// effect for already-established sessions immediately.
type SessionStore struct {
	db   *DB
	ttl  time.Duration
	idle time.Duration
}

func NewSessionStore(db *DB, ttl, idle time.Duration) *SessionStore {
	if ttl <= 0 {
		ttl = 12 * time.Hour
	}
	if idle <= 0 {
		idle = 30 * time.Minute
	}
	return &SessionStore{db: db, ttl: ttl, idle: idle}
}

func (s *SessionStore) Create(ctx context.Context, id auth.Identity) (string, auth.Session, error) {
	if s == nil || s.db == nil || id.APIKeyID == "" || id.Principal.ID == "" {
		return "", auth.Session{}, fmt.Errorf("%w: durable session requires a persisted API-key identity", contracts.ErrInvalidArgument)
	}
	token, err := randomSessionToken(32)
	if err != nil {
		return "", auth.Session{}, err
	}
	csrf, err := randomSessionToken(24)
	if err != nil {
		return "", auth.Session{}, err
	}
	now := time.Now().UTC()
	sess := auth.Session{
		Identity: id, CSRFToken: csrf, CreatedAt: now,
		ExpiresAt: now.Add(s.ttl), IdleUntil: now.Add(s.idle),
	}
	if sess.IdleUntil.After(sess.ExpiresAt) {
		sess.IdleUntil = sess.ExpiresAt
	}
	digest := sessionDigest(token)
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO auth_sessions (token_hash, api_key_id, principal_type, csrf_token, created_at, expires_at, idle_until)
		SELECT $1, k.id, $2, $3, $4, $5, $6
		  FROM api_keys k
		 WHERE k.id=$7 AND k.principal_id=$8 AND k.tenant_id=$9
		   AND k.active=TRUE AND (k.expires_at IS NULL OR k.expires_at > $4)`,
		digest, string(id.Principal.Type), csrf, now, sess.ExpiresAt, sess.IdleUntil,
		id.APIKeyID, string(id.Principal.ID), string(id.TenantID))
	if err != nil {
		return "", auth.Session{}, fmt.Errorf("persist session: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return "", auth.Session{}, fmt.Errorf("persist session result: %w", err)
	}
	if n != 1 {
		return "", auth.Session{}, fmt.Errorf("%w: API key was revoked or changed during login", contracts.ErrForbidden)
	}
	return token, sess, nil
}

func (s *SessionStore) LookupContext(ctx context.Context, token string) (auth.Session, bool, error) {
	if s == nil || s.db == nil || token == "" || len(token) > 128 {
		return auth.Session{}, false, nil
	}
	now := time.Now().UTC()
	newIdleUntil := now.Add(s.idle)
	var sess auth.Session
	var principalID, tenantID, role, keyID, principalType string
	err := s.db.QueryRowContext(ctx, `
		UPDATE auth_sessions s
		   SET idle_until=LEAST(s.expires_at, $3)
		  FROM api_keys k
		 WHERE s.token_hash=$1 AND s.api_key_id=k.id
		   AND s.expires_at > $2 AND s.idle_until > $2
		   AND k.active=TRUE AND (k.expires_at IS NULL OR k.expires_at > $2)
		RETURNING s.csrf_token, s.created_at, s.expires_at, s.idle_until,
		          s.principal_type, k.principal_id, k.tenant_id, k.role, k.id`,
		sessionDigest(token), now, newIdleUntil).Scan(
		&sess.CSRFToken, &sess.CreatedAt, &sess.ExpiresAt, &sess.IdleUntil,
		&principalType, &principalID, &tenantID, &role, &keyID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Opportunistically remove a stale token, but do not mask lookup errors.
			_, _ = s.db.ExecContext(ctx, `DELETE FROM auth_sessions WHERE token_hash=$1 AND (expires_at <= $2 OR idle_until <= $2)`, sessionDigest(token), now)
			return auth.Session{}, false, nil
		}
		return auth.Session{}, false, fmt.Errorf("lookup session: %w", err)
	}
	if !auth.ValidRole(role) ||
		(tenantID == "" && auth.Role(role) != auth.RoleOperator) ||
		(tenantID != "" && auth.Role(role) == auth.RoleOperator) {
		return auth.Session{}, false, fmt.Errorf("stored API key has invalid tenant/role binding")
	}
	sess.Identity = auth.Identity{
		Principal: contracts.PrincipalRef{ID: contracts.PrincipalID(principalID), Type: contracts.PrincipalType(principalType)},
		TenantID:  contracts.TenantID(tenantID), APIKeyID: keyID, Role: auth.Role(role),
	}
	return sess, true, nil
}

func (s *SessionStore) DeleteContext(ctx context.Context, token string) error {
	if s == nil || s.db == nil || token == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM auth_sessions WHERE token_hash=$1`, sessionDigest(token))
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (s *SessionStore) DeleteByAPIKey(ctx context.Context, keyID string) error {
	if s == nil || s.db == nil || keyID == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM auth_sessions WHERE api_key_id=$1`, keyID)
	if err != nil {
		return fmt.Errorf("delete API-key sessions: %w", err)
	}
	return nil
}

func (s *SessionStore) CleanupExpired(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM auth_sessions WHERE expires_at <= $1 OR idle_until <= $1`, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func randomSessionToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func sessionDigest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

var _ auth.SessionRepository = (*SessionStore)(nil)
