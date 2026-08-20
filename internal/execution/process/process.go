// Package process defines the external-process adapter boundary.
//
// By default, process spawning is DISABLED. This prevents unrestricted shell
// and host command execution. Future tool adapters may enable a strict allowlist
// without changing the Tool Broker or core security chain.
package process

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"cyberos/internal/contracts"
)

// Spec describes a controlled external process invocation.
type Spec struct {
	// Runtime labels the language runtime (go/python/rust) for metadata only.
	Runtime contracts.PluginRuntime
	// Path is the absolute or resolved binary path (must be allowlisted).
	Path string
	// Args are non-shell arguments (never passed through a shell).
	Args []string
	// Env is an explicit environment map (no implicit host secret leakage).
	Env map[string]string
	// WorkDir if set must also be allowlisted by the policy.
	WorkDir string
	// Timeout bounds process lifetime.
	Timeout time.Duration
	// MaxOutputBytes truncates captured stdout/stderr.
	MaxOutputBytes int64
}

// Policy controls whether process execution is permitted at all.
type Policy struct {
	Enabled bool
	// AllowedBins is a strict basename or path allowlist.
	AllowedBins []string
	// AllowedDirs restricts WorkDir when non-empty.
	AllowedDirs    []string
	DefaultTimeout time.Duration
	MaxOutputBytes int64
}

// DefaultPolicy returns a fail-closed policy (no process execution).
func DefaultPolicy() Policy {
	return Policy{
		Enabled:        false,
		DefaultTimeout: 10 * time.Second,
		MaxOutputBytes: 256 * 1024,
	}
}

// Runner executes Spec under Policy without using a shell.
type Runner struct {
	Policy Policy
}

// NewRunner constructs a process runner.
func NewRunner(p Policy) *Runner {
	if p.DefaultTimeout <= 0 {
		p.DefaultTimeout = 10 * time.Second
	}
	if p.MaxOutputBytes <= 0 {
		p.MaxOutputBytes = 256 * 1024
	}
	return &Runner{Policy: p}
}

// Run validates policy and optionally starts a process.
// When policy is disabled, returns a rejected result without spawning.
func (r *Runner) Run(ctx context.Context, spec Spec) (stdout, stderr string, err error) {
	if r == nil {
		return "", "", fmt.Errorf("%w: process runner nil", contracts.ErrInvalidArgument)
	}
	if !r.Policy.Enabled {
		return "", "", fmt.Errorf("%w: external process execution is disabled", contracts.ErrForbidden)
	}
	if err := r.allowBin(spec.Path); err != nil {
		return "", "", err
	}
	if spec.WorkDir != "" {
		if err := r.allowDir(spec.WorkDir); err != nil {
			return "", "", err
		}
	}
	timeout := spec.Timeout
	if timeout <= 0 {
		timeout = r.Policy.DefaultTimeout
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, spec.Path, spec.Args...)
	if spec.WorkDir != "" {
		cmd.Dir = spec.WorkDir
	}
	if len(spec.Env) > 0 {
		env := make([]string, 0, len(spec.Env))
		for k, v := range spec.Env {
			env = append(env, k+"="+v)
		}
		cmd.Env = env // explicit only — does not inherit full host env
	}
	out, err := cmd.Output()
	// On error, try to include stderr from ExitError
	var errb []byte
	if ee, ok := err.(*exec.ExitError); ok {
		errb = ee.Stderr
	}
	stdout = truncate(string(out), r.Policy.MaxOutputBytes)
	stderr = truncate(string(errb), r.Policy.MaxOutputBytes)
	if cctx.Err() == context.DeadlineExceeded {
		return stdout, stderr, fmt.Errorf("%w: process timeout", contracts.ErrInvalidArgument)
	}
	return stdout, stderr, err
}

func (r *Runner) allowBin(path string) error {
	if path == "" {
		return fmt.Errorf("%w: empty process path", contracts.ErrInvalidArgument)
	}
	base := path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		base = path[i+1:]
	}
	for _, a := range r.Policy.AllowedBins {
		if a == path || a == base {
			return nil
		}
	}
	return fmt.Errorf("%w: binary not allowlisted: %s", contracts.ErrForbidden, path)
}

func (r *Runner) allowDir(dir string) error {
	if len(r.Policy.AllowedDirs) == 0 {
		return fmt.Errorf("%w: workdir not allowlisted", contracts.ErrForbidden)
	}
	for _, d := range r.Policy.AllowedDirs {
		if d == dir || strings.HasPrefix(dir, strings.TrimRight(d, "/")+"/") {
			return nil
		}
	}
	return fmt.Errorf("%w: workdir not allowlisted", contracts.ErrForbidden)
}

func truncate(s string, max int64) string {
	if max <= 0 || int64(len(s)) <= max {
		return s
	}
	return s[:max] + "…(truncated)"
}
