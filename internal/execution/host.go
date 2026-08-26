package execution

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	"cyberos/internal/contracts"
)

// HostID is a logical, configuration-controlled execution host identifier.
// It is NOT a URL, socket path, shell command, or network endpoint.
type HostID string

const (
	// HostLocal is the default in-process host (Phase 17A).
	HostLocal HostID = "local"
)

var hostIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

// ValidateHostID rejects URLs, schemes, paths, and other unsafe identities.
func ValidateHostID(id HostID) error {
	if id == "" {
		return fmt.Errorf("%w: empty host id", contracts.ErrInvalidArgument)
	}
	s := string(id)
	if !hostIDPattern.MatchString(s) {
		return fmt.Errorf("%w: invalid host id %q (logical id only; not a URL/endpoint)", contracts.ErrInvalidArgument, id)
	}
	// Explicitly reject scheme-like prefixes even if regex were loosened later.
	for _, bad := range []string{"http", "https", "ssh", "docker", "unix", "tcp", "file"} {
		if s == bad || len(s) > len(bad) && s[:len(bad)+1] == bad+":" {
			return fmt.Errorf("%w: host id must not be a network/runtime endpoint", contracts.ErrInvalidArgument)
		}
	}
	return nil
}

// ExecutionHost routes an already-authorized execution request into a
// concrete execution environment. It is NOT an authorization boundary:
// Grants are validated by the Broker before any host is invoked.
type ExecutionHost interface {
	ID() HostID
	Execute(ctx context.Context, req contracts.ExecutionRequest) (contracts.ExecutionResult, error)
}

// HostRegistry maps allowlisted HostIDs to ExecutionHost implementations.
// Unknown hosts fail closed. Default is HostLocal when registered.
type HostRegistry struct {
	mu       sync.RWMutex
	hosts    map[HostID]ExecutionHost
	defaultH HostID
}

// NewHostRegistry constructs an empty allowlist registry.
func NewHostRegistry() *HostRegistry {
	return &HostRegistry{hosts: make(map[HostID]ExecutionHost), defaultH: HostLocal}
}

// Register adds an allowlisted host. Duplicate IDs conflict.
func (r *HostRegistry) Register(h ExecutionHost) error {
	if r == nil || h == nil {
		return fmt.Errorf("%w: host required", contracts.ErrInvalidArgument)
	}
	id := h.ID()
	if err := ValidateHostID(id); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.hosts[id]; exists {
		return contracts.ErrConflict
	}
	r.hosts[id] = h
	if r.defaultH == "" {
		r.defaultH = id
	}
	return nil
}

// SetDefault selects the default host for requests without HostID.
func (r *HostRegistry) SetDefault(id HostID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.hosts[id]; !ok {
		return contracts.ErrNotFound
	}
	r.defaultH = id
	return nil
}

// Resolve returns the host for the request. Empty HostID → default (local).
// Unknown or invalid HostID → error (fail closed).
func (r *HostRegistry) Resolve(req contracts.ExecutionRequest) (ExecutionHost, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: host registry not configured", contracts.ErrInvalidArgument)
	}
	id := HostID(req.HostID)
	if id == "" {
		r.mu.RLock()
		defer r.mu.RUnlock()
		h, ok := r.hosts[r.defaultH]
		if !ok {
			return nil, fmt.Errorf("%w: default host %q not registered", contracts.ErrNotFound, r.defaultH)
		}
		return h, nil
	}
	if err := ValidateHostID(id); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.hosts[id]
	if !ok {
		return nil, fmt.Errorf("%w: unknown execution host %q", contracts.ErrNotFound, id)
	}
	return h, nil
}
