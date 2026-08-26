package tools

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"cyberos/internal/contracts"
)

// Signature domain separation prefix (Phase 18B).
const PackSigDomain = "CYBEROS-PACK-SIG-V1\n"

// PackSigAlgorithm is the only supported pack signature algorithm.
const PackSigAlgorithm = "ed25519"

// PackSignature is verification metadata attached to a PackFile (no private keys).
type PackSignature struct {
	Algorithm string `json:"algorithm"` // "ed25519"
	KeyID     string `json:"key_id"`
	Signature string `json:"signature"` // base64.StdEncoding of 64-byte Ed25519 sig
}

// SignerKeyStatus controls whether a key may verify signatures.
type SignerKeyStatus string

const (
	SignerKeyActive   SignerKeyStatus = "ACTIVE"
	SignerKeyDisabled SignerKeyStatus = "DISABLED"
)

// SignerKey is a trusted verification public key (never private material).
type SignerKey struct {
	KeyID     string
	Algorithm string
	PublicKey ed25519.PublicKey
	Status    SignerKeyStatus
	CreatedAt time.Time
}

// SignerKeyRegistry is the trusted key store. Packs cannot register themselves.
type SignerKeyRegistry struct {
	mu   sync.RWMutex
	keys map[string]SignerKey
}

// NewSignerKeyRegistry constructs an empty trusted key registry.
func NewSignerKeyRegistry() *SignerKeyRegistry {
	return &SignerKeyRegistry{keys: make(map[string]SignerKey)}
}

// Register adds an ACTIVE public key. Duplicate KeyID fails closed (no replace).
func (r *SignerKeyRegistry) Register(key SignerKey) error {
	if r == nil {
		return fmt.Errorf("%w: signer registry not configured", contracts.ErrInvalidArgument)
	}
	if err := contracts.ValidateNonEmpty("key_id", key.KeyID); err != nil {
		return err
	}
	if key.Algorithm == "" {
		key.Algorithm = PackSigAlgorithm
	}
	if key.Algorithm != PackSigAlgorithm {
		return fmt.Errorf("%w: unsupported algorithm %q", contracts.ErrInvalidArgument, key.Algorithm)
	}
	if len(key.PublicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("%w: invalid public key size", contracts.ErrInvalidArgument)
	}
	if key.Status == "" {
		key.Status = SignerKeyActive
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.keys[key.KeyID]; exists {
		return fmt.Errorf("%w: key_id already registered", contracts.ErrConflict)
	}
	if key.CreatedAt.IsZero() {
		key.CreatedAt = time.Now().UTC()
	}
	r.keys[key.KeyID] = key
	return nil
}

// Get resolves a KeyID.
func (r *SignerKeyRegistry) Get(keyID string) (SignerKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	k, ok := r.keys[keyID]
	if !ok {
		return SignerKey{}, contracts.ErrNotFound
	}
	return k, nil
}

// Disable marks a key DISABLED (verification fails thereafter).
func (r *SignerKeyRegistry) Disable(keyID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k, ok := r.keys[keyID]
	if !ok {
		return contracts.ErrNotFound
	}
	k.Status = SignerKeyDisabled
	r.keys[keyID] = k
	return nil
}

// signedMessage returns domain-separated bytes over Phase 18A canonical content.
func signedMessage(canonical []byte) []byte {
	msg := make([]byte, 0, len(PackSigDomain)+len(canonical))
	msg = append(msg, []byte(PackSigDomain)...)
	msg = append(msg, canonical...)
	return msg
}

// SignPack signs Phase 18A canonical pack content with an Ed25519 private key.
// Does not persist the private key. Sets ContentDigest and Signature on a copy.
func SignPack(f PackFile, keyID string, priv ed25519.PrivateKey) (PackFile, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return PackFile{}, fmt.Errorf("%w: invalid private key", contracts.ErrInvalidArgument)
	}
	if err := contracts.ValidateNonEmpty("key_id", keyID); err != nil {
		return PackFile{}, err
	}
	// Clear verification metadata before canonicalization (no circular digest).
	f.Pack.ContentDigest = ""
	f.Signature = nil
	canon, err := CanonicalBytes(f)
	if err != nil {
		return PackFile{}, err
	}
	digest := DigestPrefix + sha256Hex(canon)
	sig := ed25519.Sign(priv, signedMessage(canon))
	f.Pack.ContentDigest = digest
	f.Signature = &PackSignature{
		Algorithm: PackSigAlgorithm,
		KeyID:     keyID,
		Signature: base64.StdEncoding.EncodeToString(sig),
	}
	return f, nil
}

// VerificationMode controls pack integrity checks at load time.
type VerificationMode string

const (
	VerifyLegacy         VerificationMode = "LEGACY"          // undigested/unsigned ok
	VerifyDigestOnly     VerificationMode = "DIGEST_ONLY"     // content_digest required
	VerifySignedRequired VerificationMode = "SIGNED_REQUIRED" // signature + active key
)

// PackVerificationResult is structured verification output (not authorization).
type PackVerificationResult struct {
	Valid     bool
	Digest    string
	Identity  PackIdentity
	Signed    bool
	KeyID     string
	Algorithm string
	Reason    string
}

// VerifyPackSignature verifies Ed25519 signature against the trusted registry.
// Signature ≠ trust ≠ authorization.
func VerifyPackSignature(f PackFile, reg *SignerKeyRegistry) PackVerificationResult {
	out := PackVerificationResult{}
	if err := f.Validate(); err != nil {
		out.Reason = "INVALID_PACK"
		return out
	}
	sig := f.Signature
	if sig == nil || sig.Signature == "" {
		out.Reason = "SIGNATURE_MISSING"
		return out
	}
	if sig.Algorithm != "" && sig.Algorithm != PackSigAlgorithm {
		out.Reason = "UNSUPPORTED_ALGORITHM"
		return out
	}
	if reg == nil {
		out.Reason = "UNKNOWN_KEY"
		return out
	}
	key, err := reg.Get(sig.KeyID)
	if err != nil {
		out.Reason = "UNKNOWN_KEY"
		return out
	}
	if key.Status != SignerKeyActive {
		out.Reason = "KEY_DISABLED"
		return out
	}
	raw, err := base64.StdEncoding.DecodeString(sig.Signature)
	if err != nil || len(raw) != ed25519.SignatureSize {
		out.Reason = "INVALID_SIGNATURE"
		return out
	}
	// Recompute canonical over content without signature/digest fields.
	tmp := f
	tmp.Pack.ContentDigest = ""
	tmp.Signature = nil
	canon, err := CanonicalBytes(tmp)
	if err != nil {
		out.Reason = "INVALID_PACK"
		return out
	}
	if !ed25519.Verify(key.PublicKey, signedMessage(canon), raw) {
		out.Reason = "INVALID_SIGNATURE"
		return out
	}
	digest := DigestPrefix + sha256Hex(canon)
	if f.Pack.ContentDigest != "" && f.Pack.ContentDigest != digest {
		out.Reason = "DIGEST_MISMATCH"
		return out
	}
	id := PackIdentity{Name: f.Pack.Name, Version: string(f.Pack.Version), Digest: digest}
	out.Valid = true
	out.Digest = digest
	out.Identity = id
	out.Signed = true
	out.KeyID = sig.KeyID
	out.Algorithm = PackSigAlgorithm
	return out
}

// VerifyPack applies the requested verification mode.
func VerifyPack(f PackFile, reg *SignerKeyRegistry, mode VerificationMode) PackVerificationResult {
	if mode == "" {
		mode = VerifyLegacy
	}
	switch mode {
	case VerifyLegacy:
		if f.Signature != nil && f.Signature.Signature != "" {
			return VerifyPackSignature(f, reg)
		}
		if f.Pack.ContentDigest != "" {
			tmp := f
			tmp.Pack.ContentDigest = ""
			tmp.Signature = nil
			if err := VerifyDigest(tmp, f.Pack.ContentDigest); err != nil {
				return PackVerificationResult{Reason: "DIGEST_MISMATCH"}
			}
			d := f.Pack.ContentDigest
			return PackVerificationResult{
				Valid: true, Digest: d,
				Identity: PackIdentity{Name: f.Pack.Name, Version: string(f.Pack.Version), Digest: d},
			}
		}
		return PackVerificationResult{Valid: true, Reason: "LEGACY_UNSIGNED"}
	case VerifyDigestOnly:
		tmp := f
		tmp.Pack.ContentDigest = ""
		tmp.Signature = nil
		if f.Pack.ContentDigest == "" {
			return PackVerificationResult{Reason: "DIGEST_MISMATCH"}
		}
		if err := VerifyDigest(tmp, f.Pack.ContentDigest); err != nil {
			return PackVerificationResult{Reason: "DIGEST_MISMATCH"}
		}
		return PackVerificationResult{
			Valid: true, Digest: f.Pack.ContentDigest,
			Identity: PackIdentity{Name: f.Pack.Name, Version: string(f.Pack.Version), Digest: f.Pack.ContentDigest},
		}
	case VerifySignedRequired:
		return VerifyPackSignature(f, reg)
	default:
		return PackVerificationResult{Reason: "UNSUPPORTED_ALGORITHM"}
	}
}
