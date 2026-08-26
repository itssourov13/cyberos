package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"cyberos/internal/contracts"
)

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// DigestPrefix is the algorithm prefix for pack content digests (Phase 18A).
const DigestPrefix = "sha256:"

// PackIdentity is content-bound pack identity (Phase 18A).
// Digest proves content integrity only — not signer trust or authorization.
type PackIdentity struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	Digest    string `json:"digest"` // sha256:<hex>
}

// Validate checks identity fields and digest format.
func (id PackIdentity) Validate() error {
	if err := contracts.ValidateNonEmpty("pack_identity.name", id.Name); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("pack_identity.version", id.Version); err != nil {
		return err
	}
	return ValidateDigestFormat(id.Digest)
}

// ValidateDigestFormat accepts empty (legacy unsigned) or sha256:<64 lowercase hex>.
func ValidateDigestFormat(d string) error {
	if d == "" {
		return nil // legacy undigested packs allowed until signed trust lands
	}
	if !strings.HasPrefix(d, DigestPrefix) {
		return fmt.Errorf("%w: digest must start with %q", contracts.ErrInvalidArgument, DigestPrefix)
	}
	hexPart := strings.TrimPrefix(d, DigestPrefix)
	if len(hexPart) != 64 {
		return fmt.Errorf("%w: digest hex must be 64 characters", contracts.ErrInvalidArgument)
	}
	for _, c := range hexPart {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return fmt.Errorf("%w: digest must be lowercase hex", contracts.ErrInvalidArgument)
		}
	}
	return nil
}

// CanonicalBytes returns deterministic JSON bytes for a PackFile.
// Encoding is independent of map insertion order and whitespace of source JSON.
func CanonicalBytes(f PackFile) ([]byte, error) {
	if err := f.Validate(); err != nil {
		return nil, err
	}
	// ContentDigest and Signature are verification metadata, not content.
	// Excluding them avoids circular digests and matches SIGNED_BYTES design.
	f.Pack.ContentDigest = ""
	f.Signature = nil
	// Normalize defaults so empty format_version matches PackFormatV1.
	if f.FormatVersion == "" {
		f.FormatVersion = PackFormatV1
	}
	// Sort tools by ID for stable ordering.
	tools := make([]PackTool, len(f.Tools))
	copy(tools, f.Tools)
	sort.Slice(tools, func(i, j int) bool { return tools[i].ID < tools[j].ID })
	f.Tools = tools
	if len(f.Pack.ToolIDs) > 1 {
		ids := append([]string(nil), f.Pack.ToolIDs...)
		sort.Strings(ids)
		f.Pack.ToolIDs = ids
	}
	if len(f.Pack.CapabilityIDs) > 1 {
		caps := make([]string, len(f.Pack.CapabilityIDs))
		for i, c := range f.Pack.CapabilityIDs {
			caps[i] = string(c)
		}
		sort.Strings(caps)
		out := make([]contracts.CapabilityID, len(caps))
		for i, c := range caps {
			out[i] = contracts.CapabilityID(c)
		}
		f.Pack.CapabilityIDs = out
	}
	for i := range f.Tools {
		if len(f.Tools[i].Capabilities) > 1 {
			caps := make([]string, len(f.Tools[i].Capabilities))
			for j, c := range f.Tools[i].Capabilities {
				caps[j] = string(c)
			}
			sort.Strings(caps)
			out := make([]contracts.CapabilityID, len(caps))
			for j, c := range caps {
				out[j] = contracts.CapabilityID(c)
			}
			f.Tools[i].Capabilities = out
		}
	}
	// Encode via map[string]any with recursive key sorting for determinism.
	raw, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return marshalCanonical(v)
}

func marshalCanonical(v any) ([]byte, error) {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var b strings.Builder
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			kb, err := json.Marshal(k)
			if err != nil {
				return nil, err
			}
			b.Write(kb)
			b.WriteByte(':')
			vb, err := marshalCanonical(t[k])
			if err != nil {
				return nil, err
			}
			b.Write(vb)
		}
		b.WriteByte('}')
		return []byte(b.String()), nil
	case []any:
		var b strings.Builder
		b.WriteByte('[')
		for i, el := range t {
			if i > 0 {
				b.WriteByte(',')
			}
			eb, err := marshalCanonical(el)
			if err != nil {
				return nil, err
			}
			b.Write(eb)
		}
		b.WriteByte(']')
		return []byte(b.String()), nil
	default:
		return json.Marshal(t)
	}
}

// Digest computes sha256:<hex> over CanonicalBytes(f).
func Digest(f PackFile) (string, error) {
	b, err := CanonicalBytes(f)
	if err != nil {
		return "", err
	}
	return DigestPrefix + sha256Hex(b), nil
}

// Identity returns content-bound PackIdentity for f.
func Identity(f PackFile) (PackIdentity, error) {
	if err := f.Validate(); err != nil {
		return PackIdentity{}, err
	}
	d, err := Digest(f)
	if err != nil {
		return PackIdentity{}, err
	}
	return PackIdentity{
		Namespace: "", // reserved; packs use Pack.ID as primary id today
		Name:      f.Pack.Name,
		Version:   string(f.Pack.Version),
		Digest:    d,
	}, nil
}

// VerifyDigest fails closed if expected is non-empty and does not match computed digest.
func VerifyDigest(f PackFile, expected string) error {
	if err := ValidateDigestFormat(expected); err != nil {
		return err
	}
	if expected == "" {
		return nil // legacy undigested
	}
	got, err := Digest(f)
	if err != nil {
		return err
	}
	if got != expected {
		return fmt.Errorf("%w: pack digest mismatch (expected %s, got %s)", contracts.ErrInvalidArgument, expected, got)
	}
	return nil
}
