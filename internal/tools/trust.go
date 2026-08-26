package tools

import (
	"fmt"
	"sync"
	"time"

	"cyberos/internal/contracts"
)

// PackTrustState is explicit trust evaluation outcome (Phase 18C).
// Trust ≠ authorization.
type PackTrustState string

const (
	TrustUntrusted PackTrustState = "UNTRUSTED"
	TrustVerified  PackTrustState = "VERIFIED" // crypto OK, not policy-trusted
	TrustTrusted   PackTrustState = "TRUSTED"  // explicit policy
	TrustRevoked   PackTrustState = "REVOKED"  // terminal deny
)

// SignerKeyRevoked is terminal key status (extends 18B ACTIVE/DISABLED).
const SignerKeyRevoked SignerKeyStatus = "REVOKED"

// VerifyTrustRequired requires verified signature + explicit trust policy.
const VerifyTrustRequired VerificationMode = "TRUST_REQUIRED"

// TrustDecision is the result of trust evaluation (not a Grant).
type TrustDecision struct {
	State    PackTrustState
	Identity PackIdentity
	KeyID    string
	Digest   string
	Reason   string
}

// PackRevocation is digest-level deny (project-scoped).
type PackRevocation struct {
	ProjectID contracts.ProjectID
	Digest    string
	Reason    string
	RevokedAt time.Time
	RevokedBy string
}

// TrustPolicy is explicit allow/deny configuration.
// TrustedSignerInheritance is OFF by default: valid signature alone → VERIFIED.
type TrustPolicy struct {
	// TrustedDigests: exact content digests allowed as TRUSTED (strongest).
	TrustedDigests map[string]struct{}
	// TrustedKeyIDs only apply if AllowSignerInheritance is true.
	TrustedKeyIDs map[string]struct{}
	// AllowSignerInheritance: when true, ACTIVE trusted key can elevate to TRUSTED.
	// Default false — prevents compromised signer from auto-trusting all packs.
	AllowSignerInheritance bool
}

// NewTrustPolicy constructs empty policy (fail closed: nothing trusted).
func NewTrustPolicy() *TrustPolicy {
	return &TrustPolicy{
		TrustedDigests: make(map[string]struct{}),
		TrustedKeyIDs:  make(map[string]struct{}),
	}
}

// TrustStore holds project-scoped pack revocations and digests marked trusted.
type TrustStore struct {
	mu            sync.RWMutex
	revokedDigest map[string]PackRevocation // project|digest
	trustedDigest map[string]struct{}       // project|digest
}

func NewTrustStore() *TrustStore {
	return &TrustStore{
		revokedDigest: make(map[string]PackRevocation),
		trustedDigest: make(map[string]struct{}),
	}
}

func trustKey(project contracts.ProjectID, digest string) string {
	return string(project) + "|" + digest
}

// RevokePack marks a content digest revoked for a project (idempotent).
func (s *TrustStore) RevokePack(project contracts.ProjectID, digest, reason, by string) error {
	if err := ValidateDigestFormat(digest); err != nil || digest == "" {
		return fmt.Errorf("%w: digest required for revocation", contracts.ErrInvalidArgument)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	k := trustKey(project, digest)
	s.revokedDigest[k] = PackRevocation{
		ProjectID: project, Digest: digest, Reason: reason,
		RevokedAt: time.Now().UTC(), RevokedBy: by,
	}
	delete(s.trustedDigest, k)
	return nil
}

// TrustPack explicitly trusts a digest under a project (idempotent).
func (s *TrustStore) TrustPack(project contracts.ProjectID, digest string) error {
	if err := ValidateDigestFormat(digest); err != nil || digest == "" {
		return fmt.Errorf("%w: digest required", contracts.ErrInvalidArgument)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	k := trustKey(project, digest)
	if _, rev := s.revokedDigest[k]; rev {
		return fmt.Errorf("%w: pack digest revoked", contracts.ErrConflict)
	}
	s.trustedDigest[k] = struct{}{}
	return nil
}

func (s *TrustStore) IsRevoked(project contracts.ProjectID, digest string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.revokedDigest[trustKey(project, digest)]
	return ok
}

func (s *TrustStore) IsTrusted(project contracts.ProjectID, digest string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.trustedDigest[trustKey(project, digest)]
	return ok
}

// Revoke marks a signer REVOKED (terminal for verification trust).
func (r *SignerKeyRegistry) Revoke(keyID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k, ok := r.keys[keyID]
	if !ok {
		return contracts.ErrNotFound
	}
	k.Status = SignerKeyRevoked
	r.keys[keyID] = k
	return nil
}

// TrustEvaluator combines crypto verification + policy (not authorization).
type TrustEvaluator struct {
	Keys   *SignerKeyRegistry
	Store  *TrustStore
	Policy *TrustPolicy
}

// Evaluate returns trust state for a pack under a project.
// project isolation: trust/revoke records are project-scoped.
func (e *TrustEvaluator) Evaluate(project contracts.ProjectID, f PackFile) TrustDecision {
	dec := TrustDecision{State: TrustUntrusted, Reason: "UNTRUSTED"}
	if e == nil {
		return dec
	}
	// Cryptographic verification first
	vres := VerifyPackSignature(f, e.Keys)
	if !vres.Valid {
		dec.Reason = vres.Reason
		if vres.Reason == "KEY_DISABLED" {
			// Treat disabled similarly; revoked keys also fail VerifyPackSignature via status
			dec.Reason = vres.Reason
		}
		// Check if key is revoked for clearer reason
		if f.Signature != nil && e.Keys != nil {
			if k, err := e.Keys.Get(f.Signature.KeyID); err == nil && k.Status == SignerKeyRevoked {
				dec.State = TrustRevoked
				dec.Reason = "KEY_REVOKED"
				dec.KeyID = f.Signature.KeyID
				return dec
			}
		}
		return dec
	}
	dec.Digest = vres.Digest
	dec.Identity = vres.Identity
	dec.KeyID = vres.KeyID

	// Pack digest revocation dominates
	if e.Store != nil && e.Store.IsRevoked(project, vres.Digest) {
		dec.State = TrustRevoked
		dec.Reason = "PACK_REVOKED"
		return dec
	}
	// Signer revoked (double-check status after verify — verify only allows ACTIVE)
	if e.Keys != nil {
		if k, err := e.Keys.Get(vres.KeyID); err == nil && k.Status == SignerKeyRevoked {
			dec.State = TrustRevoked
			dec.Reason = "KEY_REVOKED"
			return dec
		}
	}

	// Explicit pack digest trust
	if e.Store != nil && e.Store.IsTrusted(project, vres.Digest) {
		dec.State = TrustTrusted
		dec.Reason = "EXPLICIT_PACK_TRUST"
		return dec
	}
	// Optional signer inheritance (default OFF)
	pol := e.Policy
	if pol == nil {
		pol = NewTrustPolicy()
	}
	if pol.AllowSignerInheritance {
		if _, ok := pol.TrustedKeyIDs[vres.KeyID]; ok {
			dec.State = TrustTrusted
			dec.Reason = "TRUSTED_SIGNER"
			return dec
		}
	}
	if _, ok := pol.TrustedDigests[vres.Digest]; ok {
		dec.State = TrustTrusted
		dec.Reason = "POLICY_DIGEST"
		return dec
	}

	dec.State = TrustVerified
	dec.Reason = "VERIFIED_NOT_TRUSTED"
	return dec
}

// EvaluateMode runs verification/trust for a loader mode.
func (e *TrustEvaluator) EvaluateMode(project contracts.ProjectID, f PackFile, mode VerificationMode) TrustDecision {
	if mode == VerifyTrustRequired {
		d := e.Evaluate(project, f)
		if d.State != TrustTrusted {
			if d.State != TrustRevoked {
				d.State = TrustUntrusted
				if d.Reason == "VERIFIED_NOT_TRUSTED" {
					d.Reason = "NOT_TRUSTED"
				}
			}
		}
		return d
	}
	// Non-trust modes: map crypto result only
	vres := VerifyPack(f, e.Keys, mode)
	if !vres.Valid {
		return TrustDecision{State: TrustUntrusted, Reason: vres.Reason}
	}
	if vres.Signed {
		return TrustDecision{
			State: TrustVerified, Digest: vres.Digest, Identity: vres.Identity,
			KeyID: vres.KeyID, Reason: "VERIFIED_NOT_TRUSTED",
		}
	}
	return TrustDecision{State: TrustUntrusted, Reason: "LEGACY_UNSIGNED", Digest: vres.Digest}
}
