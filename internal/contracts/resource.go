package contracts

import "time"

// ResourcePolicy defines safe execution resource limits.
// It is NOT authorization — grants and Control Plane remain mandatory.
type ResourcePolicy struct {
	// MaxDuration is the wall-clock limit for a single execution attempt.
	MaxDuration time.Duration `json:"max_duration,omitempty"`
	// MaxMemoryBytes soft upper bound for reported/allowed memory usage.
	MaxMemoryBytes int64 `json:"max_memory_bytes,omitempty"`
	// MaxOutputBytes limits result payload size.
	MaxOutputBytes int64 `json:"max_output_bytes,omitempty"`
	// MaxConcurrency is a local concurrency hint (0 = unspecified).
	MaxConcurrency int `json:"max_concurrency,omitempty"`
	// AllowNetwork is always false in Phase 2C for the default sandbox.
	AllowNetwork bool `json:"allow_network,omitempty"`
	// AllowFilesystem is always false in Phase 2C for the default sandbox.
	AllowFilesystem bool `json:"allow_filesystem,omitempty"`
}

// DefaultResourcePolicy returns conservative defaults for Phase 2C.
func DefaultResourcePolicy() ResourcePolicy {
	return ResourcePolicy{
		MaxDuration:     30 * time.Second,
		MaxMemoryBytes:  256 * 1024 * 1024, // 256 MiB
		MaxOutputBytes:  1 * 1024 * 1024,   // 1 MiB
		MaxConcurrency:  1,
		AllowNetwork:    false,
		AllowFilesystem: false,
	}
}

// Validate checks that limits are finite and non-negative.
func (p ResourcePolicy) Validate() error {
	if p.MaxDuration < 0 {
		return ErrInvalidArgument
	}
	if p.MaxMemoryBytes < 0 || p.MaxOutputBytes < 0 {
		return ErrInvalidArgument
	}
	if p.MaxConcurrency < 0 {
		return ErrInvalidArgument
	}
	// Hard ceilings to prevent "unlimited" disguised as huge numbers.
	const maxDur = 24 * time.Hour
	const maxMem = int64(16) * 1024 * 1024 * 1024 // 16 GiB absolute ceiling
	const maxOut = int64(64) * 1024 * 1024        // 64 MiB absolute ceiling
	if p.MaxDuration > maxDur {
		return ErrInvalidArgument
	}
	if p.MaxMemoryBytes > maxMem || p.MaxOutputBytes > maxOut {
		return ErrInvalidArgument
	}
	if p.MaxConcurrency > 1024 {
		return ErrInvalidArgument
	}
	return nil
}

// IsRestrictive reports whether the policy has at least one positive limit.
func (p ResourcePolicy) IsRestrictive() bool {
	return p.MaxDuration > 0 || p.MaxMemoryBytes > 0 || p.MaxOutputBytes > 0 || p.MaxConcurrency > 0
}
