// Package auth provides authentication only — never authorization.
// Successful authentication binds an identity to a PrincipalRef; it does not
// issue grants or bypass Control Plane evaluation.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"cyberos/internal/contracts"
)

// Context keys for authenticated identity.
type ctxKey int

const (
	keyPrincipal ctxKey = iota + 1
	keyTenant
	keyAPIKeyID
)

// Identity is the result of successful authentication.
type Identity struct {
	Principal contracts.PrincipalRef
	TenantID  contracts.TenantID // may be empty for legacy/single-tenant
	APIKeyID  string
}

// Authenticator verifies credentials and returns an Identity.
// Implementations must not perform authorization.
type Authenticator interface {
	Authenticate(ctx context.Context, credential string) (Identity, error)
}

// APIKey is a durable credential bound to a principal and optional tenant.
type APIKey struct {
	ID          string
	Prefix      string // public prefix for lookup
	Hash        string // sha256 hex of secret
	PrincipalID contracts.PrincipalID
	TenantID    contracts.TenantID
	Active      bool
	ExpiresAt   *time.Time
	CreatedAt   time.Time
}

// KeyStore stores API keys (memory or future DB).
type KeyStore interface {
	LookupByPrefix(ctx context.Context, prefix string) (APIKey, error)
	Create(ctx context.Context, key APIKey) error
	Revoke(ctx context.Context, id string) error
	// CountActive returns the number of active (non-revoked) keys.
	CountActive(ctx context.Context) (int, error)
}

// MemoryKeyStore is an in-process API key store.
type MemoryKeyStore struct {
	mu   sync.RWMutex
	byID map[string]APIKey
	byPx map[string]string // prefix → id
}

// NewMemoryKeyStore constructs an empty key store.
func NewMemoryKeyStore() *MemoryKeyStore {
	return &MemoryKeyStore{byID: map[string]APIKey{}, byPx: map[string]string{}}
}

func (s *MemoryKeyStore) LookupByPrefix(_ context.Context, prefix string) (APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.byPx[prefix]
	if !ok {
		return APIKey{}, contracts.ErrNotFound
	}
	return s.byID[id], nil
}

func (s *MemoryKeyStore) Create(_ context.Context, key APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byID[key.ID]; ok {
		return contracts.ErrConflict
	}
	s.byID[key.ID] = key
	s.byPx[key.Prefix] = key.ID
	return nil
}

func (s *MemoryKeyStore) CountActive(_ context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, k := range s.byID {
		if k.Active {
			n++
		}
	}
	return n, nil
}

func (s *MemoryKeyStore) Revoke(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k, ok := s.byID[id]
	if !ok {
		return contracts.ErrNotFound
	}
	k.Active = false
	s.byID[id] = k
	return nil
}

// APIKeyAuthenticator validates "Bearer cos_…" or raw API keys.
type APIKeyAuthenticator struct {
	Store KeyStore
	Now   func() time.Time
}

// NewAPIKeyAuthenticator constructs an API-key authenticator.
func NewAPIKeyAuthenticator(store KeyStore) *APIKeyAuthenticator {
	return &APIKeyAuthenticator{Store: store, Now: func() time.Time { return time.Now().UTC() }}
}

// Issue creates a new API key and returns the plaintext secret once.
func (a *APIKeyAuthenticator) Issue(ctx context.Context, principalID contracts.PrincipalID, tenantID contracts.TenantID, ttl time.Duration) (plaintext string, key APIKey, err error) {
	if principalID == "" {
		return "", APIKey{}, fmt.Errorf("%w: principal required", contracts.ErrInvalidArgument)
	}
	secret, err := randomHex(24)
	if err != nil {
		return "", APIKey{}, err
	}
	prefix := "cos_" + secret[:8]
	full := prefix + secret[8:]
	sum := sha256.Sum256([]byte(full))
	key = APIKey{
		ID: "key-" + prefix, Prefix: prefix, Hash: hex.EncodeToString(sum[:]),
		PrincipalID: principalID, TenantID: tenantID, Active: true, CreatedAt: a.Now(),
	}
	if ttl > 0 {
		exp := a.Now().Add(ttl)
		key.ExpiresAt = &exp
	}
	if err := a.Store.Create(ctx, key); err != nil {
		return "", APIKey{}, err
	}
	return full, key, nil
}

// Authenticate verifies an API key credential.
func (a *APIKeyAuthenticator) Authenticate(ctx context.Context, credential string) (Identity, error) {
	if a == nil || a.Store == nil {
		return Identity{}, fmt.Errorf("%w: authenticator not configured", contracts.ErrInvalidArgument)
	}
	cred := strings.TrimSpace(credential)
	cred = strings.TrimPrefix(cred, "Bearer ")
	cred = strings.TrimSpace(cred)
	if cred == "" || !strings.HasPrefix(cred, "cos_") || len(cred) < 12 {
		return Identity{}, fmt.Errorf("%w: invalid credentials", contracts.ErrForbidden)
	}
	prefix := cred[:12] // cos_ + 8 chars
	key, err := a.Store.LookupByPrefix(ctx, prefix)
	if err != nil {
		return Identity{}, fmt.Errorf("%w: invalid credentials", contracts.ErrForbidden)
	}
	if !key.Active {
		return Identity{}, fmt.Errorf("%w: credentials revoked", contracts.ErrForbidden)
	}
	now := a.Now()
	if key.ExpiresAt != nil && !key.ExpiresAt.After(now) {
		return Identity{}, fmt.Errorf("%w: credentials expired", contracts.ErrForbidden)
	}
	sum := sha256.Sum256([]byte(cred))
	if hex.EncodeToString(sum[:]) != key.Hash {
		return Identity{}, fmt.Errorf("%w: invalid credentials", contracts.ErrForbidden)
	}
	return Identity{
		Principal: contracts.PrincipalRef{ID: key.PrincipalID, Type: contracts.PrincipalTypeHuman},
		TenantID:  key.TenantID,
		APIKeyID:  key.ID,
	}, nil
}

// EnsureBootstrap installs an env-provided key or reuses existing active keys.
// Returns plaintext only when a new key is created; empty string means "already configured".
func (a *APIKeyAuthenticator) EnsureBootstrap(ctx context.Context, bootstrapPlain string, tenantID contracts.TenantID) (plaintext string, created bool, err error) {
	if a == nil || a.Store == nil {
		return "", false, fmt.Errorf("%w: authenticator not configured", contracts.ErrInvalidArgument)
	}
	if bootstrapPlain != "" {
		bootstrapPlain = strings.TrimSpace(bootstrapPlain)
		// If already valid, do not re-issue.
		if _, aerr := a.Authenticate(ctx, bootstrapPlain); aerr == nil {
			return "", false, nil
		}
		if !strings.HasPrefix(bootstrapPlain, "cos_") || len(bootstrapPlain) < 12 {
			return "", false, fmt.Errorf("%w: CYBEROS_BOOTSTRAP_API_KEY must be a cos_… API key", contracts.ErrInvalidArgument)
		}
		prefix := bootstrapPlain[:12]
		sum := sha256.Sum256([]byte(bootstrapPlain))
		key := APIKey{
			ID: "key-bootstrap-env", Prefix: prefix, Hash: hex.EncodeToString(sum[:]),
			PrincipalID: "bootstrap", TenantID: tenantID, Active: true, CreatedAt: a.Now(),
		}
		if err := a.Store.Create(ctx, key); err != nil {
			// Conflict: try authenticate again after concurrent create
			if _, aerr := a.Authenticate(ctx, bootstrapPlain); aerr == nil {
				return "", false, nil
			}
			return "", false, err
		}
		return "", false, nil // env key installed; do not print
	}
	n, err := a.Store.CountActive(ctx)
	if err != nil {
		return "", false, err
	}
	if n > 0 {
		return "", false, nil
	}
	plain, _, err := a.Issue(ctx, "bootstrap", tenantID, 0)
	if err != nil {
		return "", false, err
	}
	return plain, true, nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// WithIdentity stores authenticated identity on the context.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	ctx = context.WithValue(ctx, keyPrincipal, id.Principal)
	ctx = context.WithValue(ctx, keyTenant, id.TenantID)
	ctx = context.WithValue(ctx, keyAPIKeyID, id.APIKeyID)
	return ctx
}

// PrincipalFromContext returns the authenticated principal if present.
func PrincipalFromContext(ctx context.Context) (contracts.PrincipalRef, bool) {
	p, ok := ctx.Value(keyPrincipal).(contracts.PrincipalRef)
	return p, ok && p.ID != ""
}

// TenantFromContext returns the authenticated tenant if present.
func TenantFromContext(ctx context.Context) (contracts.TenantID, bool) {
	t, ok := ctx.Value(keyTenant).(contracts.TenantID)
	return t, ok && t != ""
}

// OIDCBoundary is a placeholder for future OIDC adapters.
// It is intentionally non-functional until an IdP is configured.
type OIDCBoundary struct {
	Enabled bool
}

// Authenticate rejects until OIDC is configured (fail closed).
func (o *OIDCBoundary) Authenticate(context.Context, string) (Identity, error) {
	return Identity{}, fmt.Errorf("%w: OIDC not configured", contracts.ErrForbidden)
}
