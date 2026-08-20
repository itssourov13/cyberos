package secrets

import (
	"os"
	"strings"
	"testing"
)

func TestParseAndValidateRef(t *testing.T) {
	if err := ValidateRef("env:CYBEROS_API_TOKEN"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRef("plaintext-secret-value"); err == nil {
		t.Fatal("plaintext must fail")
	}
	if err := ValidateRef("vault:x"); err == nil {
		t.Fatal("unsupported scheme")
	}
}

func TestEnvResolver(t *testing.T) {
	_ = os.Setenv("CYBEROS_TEST_SECRET", "super-secret-value")
	defer os.Unsetenv("CYBEROS_TEST_SECRET")
	v, err := EnvResolver{}.Resolve("env:CYBEROS_TEST_SECRET")
	if err != nil || v != "super-secret-value" {
		t.Fatalf("resolve: %v %q", err, v)
	}
	_, err = EnvResolver{}.Resolve("env:DOES_NOT_EXIST_XYZ")
	if err == nil {
		t.Fatal("expected not found")
	}
}

func TestRedactString(t *testing.T) {
	s := RedactString("token=abc123 password=hunter2", "abc123")
	if strings.Contains(s, "abc123") || strings.Contains(s, "hunter2") {
		t.Fatalf("not redacted: %s", s)
	}
	if !strings.Contains(s, Redacted) {
		t.Fatal("expected placeholder")
	}
}

func TestSanitizeMap(t *testing.T) {
	m := SanitizeMap(map[string]any{
		"api_token": "env:CYBEROS_API_TOKEN",
		"password":  "should-hide",
		"note":      "ok",
	})
	if m["api_token"] != "env:CYBEROS_API_TOKEN" {
		t.Fatalf("ref should remain: %v", m["api_token"])
	}
	if m["password"] != Redacted {
		t.Fatalf("password: %v", m["password"])
	}
}

func TestAssertNoSecretInMetadata(t *testing.T) {
	if err := AssertNoSecretInMetadata(map[string]string{"token": "env:X"}); err != nil {
		t.Fatal(err)
	}
	if err := AssertNoSecretInMetadata(map[string]string{"token": "raw-secret-value-here-32chars!!"}); err == nil {
		t.Fatal("plaintext token must fail")
	}
}
