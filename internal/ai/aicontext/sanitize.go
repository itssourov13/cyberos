package aicontext

import (
	"regexp"
	"strings"

	"cyberos/internal/secrets"
)

var (
	reAPIKey   = regexp.MustCompile(`(?i)(api[_-]?key|access[_-]?key|secret[_-]?key)\s*[:=]\s*['"]?[A-Za-z0-9_\-]{8,}`)
	reBearer   = regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9\-._~+/]+=*`)
	rePassword = regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*\S+`)
	rePrivKey  = regexp.MustCompile(`(?is)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`)
	reAWS      = regexp.MustCompile(`(?i)(AKIA[0-9A-Z]{16})`)
	reJWT      = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`)
)

// RedactText applies conservative pattern redaction.
func RedactText(s string) string {
	if s == "" {
		return s
	}
	out := s
	out = rePrivKey.ReplaceAllString(out, secrets.Redacted)
	out = reAPIKey.ReplaceAllString(out, "${1}="+secrets.Redacted)
	out = reBearer.ReplaceAllString(out, "${1}"+secrets.Redacted)
	out = rePassword.ReplaceAllString(out, "${1}="+secrets.Redacted)
	out = reAWS.ReplaceAllString(out, secrets.Redacted)
	out = reJWT.ReplaceAllString(out, secrets.Redacted)
	return out
}

// RedactFields redacts sensitive keys and string values in a map (copy).
func RedactFields(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		lk := strings.ToLower(k)
		if isSensitiveKey(lk) {
			out[k] = secrets.Redacted
			continue
		}
		switch t := v.(type) {
		case string:
			out[k] = RedactText(t)
		default:
			out[k] = v
		}
	}
	return out
}

func isSensitiveKey(k string) bool {
	switch k {
	case "password", "passwd", "pwd", "secret", "token", "api_key", "apikey",
		"access_key", "private_key", "authorization", "auth", "credential", "credentials":
		return true
	default:
		return strings.Contains(k, "password") || strings.Contains(k, "secret") || strings.Contains(k, "token")
	}
}

// Truncate runes/bytes conservatively.
func Truncate(s string, max int) (string, bool) {
	if max <= 0 || len(s) <= max {
		return s, false
	}
	return s[:max] + "…", true
}
