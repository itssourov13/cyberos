// Package secrets provides secure secret references and resolution hooks.
//
// Secret values must not be stored in task state, events, audit records, or logs.
// Resolution never authorizes execution — Control Plane → Grant remain mandatory.
package secrets

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"cyberos/internal/contracts"
)

// Scheme identifies how a secret is referenced.
type Scheme string

const (
	SchemeEnv  Scheme = "env"  // environment variable
	SchemeFile Scheme = "file" // path to file (future adapters may use vault:// etc.)
)

// Ref is a secret reference — never a secret value.
// Example: "env:CYBEROS_API_TOKEN"
type Ref string

// ParseRef parses "scheme:name" into components.
func ParseRef(r Ref) (Scheme, string, error) {
	s := string(r)
	if s == "" {
		return "", "", fmt.Errorf("%w: empty secret ref", contracts.ErrInvalidArgument)
	}
	// Reject values that look like embedded secrets (no scheme).
	if !strings.Contains(s, ":") {
		return "", "", fmt.Errorf("%w: secret ref must be scheme:name", contracts.ErrInvalidArgument)
	}
	i := strings.IndexByte(s, ':')
	scheme := Scheme(strings.ToLower(s[:i]))
	name := s[i+1:]
	if name == "" {
		return "", "", fmt.Errorf("%w: secret name empty", contracts.ErrInvalidArgument)
	}
	switch scheme {
	case SchemeEnv, SchemeFile:
		return scheme, name, nil
	default:
		return "", "", fmt.Errorf("%w: unsupported secret scheme %q", contracts.ErrInvalidArgument, scheme)
	}
}

// ValidateRef checks structural validity without resolving.
func ValidateRef(r Ref) error {
	_, _, err := ParseRef(r)
	return err
}

// Resolver resolves secret references to values in memory only.
// Callers must not persist the returned value into events, audit, or task metadata.
type Resolver interface {
	Resolve(ref Ref) (string, error)
}

// EnvResolver resolves env: references from the process environment.
type EnvResolver struct{}

// Resolve implements Resolver.
func (EnvResolver) Resolve(ref Ref) (string, error) {
	scheme, name, err := ParseRef(ref)
	if err != nil {
		return "", err
	}
	if scheme != SchemeEnv {
		return "", fmt.Errorf("%w: EnvResolver only supports env:", contracts.ErrInvalidArgument)
	}
	v, ok := os.LookupEnv(name)
	if !ok || v == "" {
		return "", fmt.Errorf("%w: secret %s not found", contracts.ErrNotFound, name)
	}
	return v, nil
}

// MapResolver is a test/in-memory resolver (values must not be logged).
type MapResolver struct {
	mu   sync.RWMutex
	data map[string]string // key = full ref string or env name
}

// NewMapResolver constructs a map-backed resolver.
func NewMapResolver(m map[string]string) *MapResolver {
	cp := make(map[string]string, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return &MapResolver{data: cp}
}

// Resolve implements Resolver.
func (m *MapResolver) Resolve(ref Ref) (string, error) {
	if err := ValidateRef(ref); err != nil {
		return "", err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if v, ok := m.data[string(ref)]; ok {
		return v, nil
	}
	_, name, _ := ParseRef(ref)
	if v, ok := m.data[name]; ok {
		return v, nil
	}
	return "", fmt.Errorf("%w: secret not found", contracts.ErrNotFound)
}

// Chain tries resolvers in order.
type Chain []Resolver

// Resolve implements Resolver.
func (c Chain) Resolve(ref Ref) (string, error) {
	var last error
	for _, r := range c {
		if r == nil {
			continue
		}
		v, err := r.Resolve(ref)
		if err == nil {
			return v, nil
		}
		last = err
	}
	if last != nil {
		return "", last
	}
	return "", fmt.Errorf("%w: no resolver", contracts.ErrInvalidArgument)
}

// Redacted is the placeholder used in logs/events.
const Redacted = "[REDACTED]"

// RedactString replaces occurrences of secret values and common patterns.
func RedactString(s string, secretValues ...string) string {
	out := s
	for _, v := range secretValues {
		if v == "" {
			continue
		}
		out = strings.ReplaceAll(out, v, Redacted)
	}
	// Heuristic: password=..., token=..., secret=...
	out = redactKeyValue(out, "password")
	out = redactKeyValue(out, "token")
	out = redactKeyValue(out, "secret")
	out = redactKeyValue(out, "api_key")
	out = redactKeyValue(out, "authorization")
	return out
}

func redactKeyValue(s, key string) string {
	// case-insensitive simple scan for key=
	lower := strings.ToLower(s)
	k := strings.ToLower(key) + "="
	var b strings.Builder
	i := 0
	for {
		idx := strings.Index(lower[i:], k)
		if idx < 0 {
			b.WriteString(s[i:])
			break
		}
		idx += i
		b.WriteString(s[i:idx])
		b.WriteString(key)
		b.WriteString("=")
		b.WriteString(Redacted)
		// skip value until space or end
		j := idx + len(k)
		for j < len(s) && s[j] != ' ' && s[j] != '&' && s[j] != '"' && s[j] != '\'' {
			j++
		}
		i = j
	}
	return b.String()
}

// SanitizeMap returns a copy of m with secret refs preserved but values removed,
// and known sensitive keys redacted.
func SanitizeMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		if s, ok := v.(string); ok && (strings.HasPrefix(s, "env:") || strings.HasPrefix(s, "file:")) {
			out[k] = s
			continue
		}
		lk := strings.ToLower(k)
		if strings.Contains(lk, "password") || strings.Contains(lk, "secret") ||
			strings.Contains(lk, "token") || strings.Contains(lk, "api_key") {
			out[k] = Redacted
			continue
		}
		switch t := v.(type) {
		case string:
			if looksLikeSecretValue(t) {
				out[k] = Redacted
			} else {
				out[k] = t
			}
		default:
			out[k] = v
		}
	}
	return out
}

func looksLikeSecretValue(s string) bool {
	// High-entropy-ish long tokens without scheme
	if strings.Contains(s, ":") && (strings.HasPrefix(s, "env:") || strings.HasPrefix(s, "file:")) {
		return false
	}
	if len(s) >= 32 && !strings.Contains(s, " ") {
		return true
	}
	return false
}

// AssertNoSecretInMetadata rejects metadata that appears to embed raw secrets.
func AssertNoSecretInMetadata(meta map[string]string) error {
	for k, v := range meta {
		lk := strings.ToLower(k)
		if strings.Contains(lk, "password") || strings.Contains(lk, "secret") || strings.Contains(lk, "token") {
			if !strings.HasPrefix(v, "env:") && !strings.HasPrefix(v, "file:") {
				return fmt.Errorf("%w: metadata %q must use secret ref, not plaintext", contracts.ErrInvalidArgument, k)
			}
		}
		if looksLikeSecretValue(v) && !strings.HasPrefix(v, "env:") && !strings.HasPrefix(v, "file:") {
			return fmt.Errorf("%w: metadata value looks like embedded secret", contracts.ErrInvalidArgument)
		}
	}
	return nil
}
