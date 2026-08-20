package findings

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// Fingerprint computes a deterministic finding fingerprint from stable attributes.
// Timestamps and random IDs are intentionally excluded.
func Fingerprint(projectID, target, capability, title, location string) string {
	parts := []string{
		strings.ToLower(strings.TrimSpace(projectID)),
		strings.ToLower(strings.TrimSpace(target)),
		strings.ToLower(strings.TrimSpace(capability)),
		strings.ToLower(strings.TrimSpace(title)),
		strings.ToLower(strings.TrimSpace(location)),
	}
	h := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(h[:16])
}
