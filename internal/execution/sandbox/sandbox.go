// Package sandbox enforces the executor boundary: resource limits and
// host-isolation flags before delegating to an inner Executor.
//
// Phase 2C keeps the inner executor as the safe mock. No unrestricted
// host, network, or filesystem access is granted.
package sandbox

import (
	"fmt"
	"sync"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/resource"
)

// Executor wraps an inner contracts.Executor with resource policy enforcement.
type Executor struct {
	inner  contracts.Executor
	policy contracts.ResourcePolicy

	mu       sync.Mutex
	inFlight int
	// Now overrides clock for tests.
	Now func() time.Time
}

// Option configures the sandbox.
type Option func(*Executor)

// WithPolicy sets the resource policy (merged with defaults).
func WithPolicy(p contracts.ResourcePolicy) Option {
	return func(e *Executor) { e.policy = p }
}

// New wraps inner with sandbox enforcement.
func New(inner contracts.Executor, opts ...Option) *Executor {
	e := &Executor{
		inner:  inner,
		policy: contracts.DefaultResourcePolicy(),
		Now:    func() time.Time { return time.Now().UTC() },
	}
	for _, o := range opts {
		o(e)
	}
	e.policy = resource.MergeDefaults(e.policy)
	return e
}

// Execute validates grant presence is left to the broker; this layer enforces
// resource policy and isolation flags, then delegates.
func (e *Executor) Execute(req contracts.ExecutionRequest) (contracts.ExecutionResult, error) {
	if e == nil || e.inner == nil {
		return reject("sandbox not configured"), fmt.Errorf("%w: sandbox not configured", contracts.ErrInvalidArgument)
	}
	if err := resource.ValidatePolicy(e.policy); err != nil {
		return reject(err.Error()), err
	}

	// Isolation: Phase 2C default forbids network and filesystem.
	if e.policy.AllowNetwork {
		return reject("network access not permitted in Phase 2C sandbox"),
			fmt.Errorf("%w: network not permitted", contracts.ErrForbidden)
	}
	if e.policy.AllowFilesystem {
		return reject("filesystem access not permitted in Phase 2C sandbox"),
			fmt.Errorf("%w: filesystem not permitted", contracts.ErrForbidden)
	}

	// Concurrency limit
	if e.policy.MaxConcurrency > 0 {
		e.mu.Lock()
		if e.inFlight >= e.policy.MaxConcurrency {
			e.mu.Unlock()
			return reject("concurrency limit exceeded"),
				fmt.Errorf("%w: concurrency limit exceeded", contracts.ErrInvalidArgument)
		}
		e.inFlight++
		e.mu.Unlock()
		defer func() {
			e.mu.Lock()
			e.inFlight--
			e.mu.Unlock()
		}()
	}

	start := e.Now()
	res, err := e.inner.Execute(req)
	finish := e.Now()

	// Duration enforcement (post-check for mock; real tools would use context cancel)
	if e.policy.MaxDuration > 0 && finish.Sub(start) > e.policy.MaxDuration {
		res.Status = contracts.ExecutionStatusTimeout
		res.ErrorMessage = "execution exceeded max_duration"
		res.FinishedAt = finish
		return res, fmt.Errorf("%w: max_duration exceeded", contracts.ErrInvalidArgument)
	}

	// Output size limit
	if e.policy.MaxOutputBytes > 0 {
		size := resource.EstimateOutputBytes(res.Output)
		if size > e.policy.MaxOutputBytes {
			res.Status = contracts.ExecutionStatusRejected
			res.ErrorMessage = fmt.Sprintf("output size %d exceeds max_output_bytes %d", size, e.policy.MaxOutputBytes)
			res.Output = nil
			return res, fmt.Errorf("%w: output too large", contracts.ErrInvalidArgument)
		}
	}

	if res.ExecutorID == "" {
		res.ExecutorID = "sandbox"
	} else {
		res.ExecutorID = "sandbox/" + res.ExecutorID
	}
	return res, err
}

func reject(msg string) contracts.ExecutionResult {
	now := time.Now().UTC()
	return contracts.ExecutionResult{
		Status:       contracts.ExecutionStatusRejected,
		StartedAt:    now,
		FinishedAt:   now,
		ErrorMessage: msg,
		ExecutorID:   "sandbox",
	}
}

// Policy returns the effective policy.
func (e *Executor) Policy() contracts.ResourcePolicy {
	return e.policy
}

var _ contracts.Executor = (*Executor)(nil)
