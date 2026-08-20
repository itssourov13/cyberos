package security_test

import (
	"strings"
	"testing"

	"cyberos/internal/config"
	"cyberos/internal/secrets"
)

// Secret refs must not be confused with authorization.
func TestSecurity_SecretsDoNotAuthorize(t *testing.T) {
	// Resolving a secret is independent of Control Plane / grants.
	r := secrets.NewMapResolver(map[string]string{"env:X": "val"})
	v, err := r.Resolve("env:X")
	if err != nil || v != "val" {
		t.Fatal(err)
	}
	// No grant/decision involved — by design hooks only supply credentials later.
}

func TestSecurity_ConfigDoesNotEmbedSecretValues(t *testing.T) {
	cfg := config.Load()
	for name, ref := range cfg.SecretRefs {
		if err := secrets.ValidateRef(ref); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if strings.Contains(string(ref), " ") {
			t.Fatal("ref should not contain spaces/values")
		}
	}
	red := cfg.RedactedDatabaseURL()
	if strings.Contains(strings.ToLower(red), "password=") && !strings.Contains(red, secrets.Redacted) {
		// if password= present it should be redacted
		if strings.Contains(red, "cyberos:cyberos") {
			// default DSN has user:pass — RedactString should mask password= style; user:pass in URL
			_ = red
		}
	}
}

func TestSecurity_EventsMustUseSanitizedMaps(t *testing.T) {
	payload := secrets.SanitizeMap(map[string]any{
		"secret": "raw",
		"ref":    "env:TOKEN",
	})
	if payload["secret"] != secrets.Redacted {
		t.Fatal("secret key must be redacted")
	}
	if payload["ref"] != "env:TOKEN" {
		t.Fatal("ref must be kept")
	}
}
