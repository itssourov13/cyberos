package broker

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"

	"cyberos/internal/contracts"
)

// MinGrantKeyBytes is the minimum accepted signing key length.
const MinGrantKeyBytes = 32

// GrantSigner proves that an ExecutionGrant was issued by this deployment's
// Control Plane issuer. The signature covers every authoritative field of the
// grant (identity, scope chain, validity window, nonce, constraints, pack
// binding) so a client cannot manufacture or alter a grant. The mutable Status
// field is deliberately excluded: lifecycle state is server-side (see
// Broker.RevokeGrant), never taken from the client.
//
// The key never leaves the server process and is never sent to the browser.
type GrantSigner struct {
	key []byte

	mu      sync.RWMutex
	revoked map[contracts.ExecutionGrantID]struct{}
}

// NewGrantSigner constructs a signer from a secret of at least MinGrantKeyBytes.
func NewGrantSigner(key []byte) (*GrantSigner, error) {
	if len(key) < MinGrantKeyBytes {
		return nil, fmt.Errorf("%w: grant signing key must be at least %d bytes", contracts.ErrInvalidArgument, MinGrantKeyBytes)
	}
	k := make([]byte, len(key))
	copy(k, key)
	return &GrantSigner{key: k, revoked: map[contracts.ExecutionGrantID]struct{}{}}, nil
}

// NewEphemeralGrantSigner generates a random per-process key. Grants do not
// survive restarts, which is the fail-closed default when no key is configured.
func NewEphemeralGrantSigner() (*GrantSigner, error) {
	k := make([]byte, MinGrantKeyBytes)
	if _, err := rand.Read(k); err != nil {
		return nil, err
	}
	return NewGrantSigner(k)
}

// canonical is the signed projection of a grant. Field order is fixed by the
// struct definition; map keys inside Constraints are sorted by encoding/json.
type canonical struct {
	V                 int                                 `json:"v"`
	ID                contracts.ExecutionGrantID          `json:"id"`
	PrincipalID       contracts.PrincipalID               `json:"principal_id"`
	PrincipalType     contracts.PrincipalType             `json:"principal_type"`
	ProjectID         contracts.ProjectID                 `json:"project_id"`
	ScopeID           contracts.ScopeID                   `json:"scope_id"`
	TaskID            contracts.TaskID                    `json:"task_id"`
	CapabilityID      contracts.CapabilityID              `json:"capability_id"`
	IssuedAt          int64                               `json:"issued_at_unix_nano"`
	ExpiresAt         int64                               `json:"expires_at_unix_nano"`
	Constraints       contracts.ExecutionGrantConstraints `json:"constraints"`
	CorrelationID     contracts.CorrelationID             `json:"correlation_id"`
	Nonce             string                              `json:"nonce"`
	PackContentDigest string                              `json:"pack_content_digest"`
	PackKeyID         string                              `json:"pack_key_id"`
}

func (s *GrantSigner) mac(g contracts.ExecutionGrant) ([]byte, error) {
	b, err := json.Marshal(canonical{
		V: 1, ID: g.ID, PrincipalID: g.Principal.ID, PrincipalType: g.Principal.Type,
		ProjectID: g.ProjectID, ScopeID: g.ScopeID, TaskID: g.TaskID, CapabilityID: g.CapabilityID,
		IssuedAt: g.IssuedAt.UnixNano(), ExpiresAt: g.ExpiresAt.UnixNano(),
		Constraints: g.Constraints, CorrelationID: g.CorrelationID, Nonce: g.Nonce,
		PackContentDigest: g.PackContentDigest, PackKeyID: g.PackKeyID,
	})
	if err != nil {
		return nil, err
	}
	h := hmac.New(sha256.New, s.key)
	h.Write(b)
	return h.Sum(nil), nil
}

// Sign returns the hex signature for a grant.
func (s *GrantSigner) Sign(g contracts.ExecutionGrant) (string, error) {
	if s == nil {
		return "", fmt.Errorf("%w: grant signer not configured", contracts.ErrInvalidArgument)
	}
	m, err := s.mac(g)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(m), nil
}

// Verify checks that the grant carries a valid issuer signature and has not
// been revoked. It is constant-time with respect to the signature.
func (s *GrantSigner) Verify(g contracts.ExecutionGrant) error {
	if s == nil {
		return fmt.Errorf("%w: grant signer not configured", contracts.ErrGrantInvalid)
	}
	if g.Signature == "" {
		return fmt.Errorf("%w: grant is not signed by the issuer", contracts.ErrGrantInvalid)
	}
	got, err := hex.DecodeString(g.Signature)
	if err != nil {
		return fmt.Errorf("%w: malformed grant signature", contracts.ErrGrantInvalid)
	}
	want, err := s.mac(g)
	if err != nil {
		return err
	}
	if !hmac.Equal(got, want) {
		return fmt.Errorf("%w: grant signature invalid (not issued by control plane)", contracts.ErrGrantInvalid)
	}
	s.mu.RLock()
	_, revoked := s.revoked[g.ID]
	s.mu.RUnlock()
	if revoked {
		return fmt.Errorf("%w: grant revoked", contracts.ErrGrantInvalid)
	}
	return nil
}

// Revoke marks an issued grant unusable. Revocation state is process-local
// until a durable registry exists (documented dependency); grants are
// short-lived, so a restart bounds the exposure window to the grant TTL.
func (s *GrantSigner) Revoke(id contracts.ExecutionGrantID) {
	if s == nil || id == "" {
		return
	}
	s.mu.Lock()
	s.revoked[id] = struct{}{}
	s.mu.Unlock()
}
