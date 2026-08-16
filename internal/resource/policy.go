// Package resource validates and applies execution resource policies.
// Resource limits are not authorization.
package resource

import (
	"fmt"

	"cyberos/internal/contracts"
)

// ValidatePolicy ensures the policy is finite and within hard ceilings.
func ValidatePolicy(p contracts.ResourcePolicy) error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("%w: invalid resource policy", err)
	}
	return nil
}

// MergeDefaults fills zero fields from defaults without enabling network/fs.
func MergeDefaults(p contracts.ResourcePolicy) contracts.ResourcePolicy {
	d := contracts.DefaultResourcePolicy()
	if p.MaxDuration == 0 {
		p.MaxDuration = d.MaxDuration
	}
	if p.MaxMemoryBytes == 0 {
		p.MaxMemoryBytes = d.MaxMemoryBytes
	}
	if p.MaxOutputBytes == 0 {
		p.MaxOutputBytes = d.MaxOutputBytes
	}
	if p.MaxConcurrency == 0 {
		p.MaxConcurrency = d.MaxConcurrency
	}
	// Network and filesystem stay as requested but default false.
	return p
}

// EstimateOutputBytes is a coarse size estimate for map payloads.
func EstimateOutputBytes(v any) int64 {
	switch t := v.(type) {
	case nil:
		return 0
	case string:
		return int64(len(t))
	case []byte:
		return int64(len(t))
	case map[string]any:
		var n int64
		for k, val := range t {
			n += int64(len(k))
			n += EstimateOutputBytes(val)
		}
		return n
	default:
		return 64 // opaque small estimate
	}
}
