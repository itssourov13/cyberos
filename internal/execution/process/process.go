// Package process defines the external-process adapter boundary.
//
// By default, process spawning is DISABLED. This prevents unrestricted shell
// and host command execution. Future tool adapters may enable a strict allowlist
// without changing the Tool Broker or core security chain.
package process

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"cyberos/internal/contracts"
)

// Spec describes a controlled external process invocation.
type Spec struct {
	Runtime        contracts.PluginRuntime
	Path           string
	Args           []string
	Env            map[string]string
	WorkDir        string
	Timeout        time.Duration
	MaxOutputBytes int64
}

// Policy controls whether process execution is permitted at all.
type Policy struct {
	Enabled bool
	// AllowedBins is a strict basename or absolute path allowlist.
	AllowedBins []string
	// AllowedDirs restricts WorkDir when non-empty.
	AllowedDirs []string
	// AllowShell permits shell binaries (default false — fail closed).
	AllowShell     bool
	DefaultTimeout time.Duration
	MaxOutputBytes int64
}

// DefaultPolicy returns a fail-closed policy (no process execution).
func DefaultPolicy() Policy {
	return Policy{
		Enabled:        false,
		AllowShell:     false,
		DefaultTimeout: 10 * time.Second,
		MaxOutputBytes: 256 * 1024,
	}
}

// Result captures process outcomes for normalization upstream.
const HardMaxOutputBytes int64 = 4 * 1024 * 1024

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
	TimedOut bool
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

// Run validates policy and starts a process when permitted.
func (r *Runner) Run(ctx context.Context, spec Spec) (stdout, stderr string, err error) {
	res, err := r.RunDetailed(ctx, spec)
	return res.Stdout, res.Stderr, err
}

// RunDetailed returns structured process results.
func (r *Runner) RunDetailed(ctx context.Context, spec Spec) (Result, error) {
	start := time.Now()
	if r == nil {
		return Result{}, fmt.Errorf("%w: process runner nil", contracts.ErrInvalidArgument)
	}
	if !r.Policy.Enabled {
		return Result{Duration: time.Since(start)}, fmt.Errorf("%w: external process execution is disabled", contracts.ErrForbidden)
	}
	if err := ValidateArgv(spec.Path, spec.Args); err != nil {
		return Result{Duration: time.Since(start)}, err
	}
	if IsShellBinary(spec.Path) && !r.Policy.AllowShell {
		return Result{Duration: time.Since(start)}, fmt.Errorf("%w: shell/interpreter execution is disabled", contracts.ErrForbidden)
	}
	if err := r.allowBin(spec.Path); err != nil {
		return Result{Duration: time.Since(start)}, err
	}
	if spec.WorkDir != "" {
		if err := r.allowDir(spec.WorkDir); err != nil {
			return Result{Duration: time.Since(start)}, err
		}
	}

	timeout := spec.Timeout
	if timeout <= 0 {
		timeout = r.Policy.DefaultTimeout
	}
	maxOut := spec.MaxOutputBytes
	if maxOut <= 0 {
		maxOut = r.Policy.MaxOutputBytes
	}
	if maxOut <= 0 || maxOut > HardMaxOutputBytes {
		maxOut = HardMaxOutputBytes
	}

	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, spec.Path, spec.Args...)
	if spec.WorkDir != "" {
		cmd.Dir = spec.WorkDir
	}
	// Explicit env only — never inherit full host environment by default.
	if env := SanitizeEnv(spec.Env); len(env) > 0 {
		cmd.Env = env
	} else {
		cmd.Env = []string{} // empty env when none provided
	}

	stdoutBuf := &limitedBuffer{limit: maxOut}
	stderrBuf := &limitedBuffer{limit: maxOut}
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf

	runErr := cmd.Run()
	dur := time.Since(start)
	res := Result{
		Stdout:   stdoutBuf.Result(),
		Stderr:   stderrBuf.Result(),
		Duration: dur,
		ExitCode: 0,
	}
	if cctx.Err() == context.DeadlineExceeded {
		res.TimedOut = true
		res.ExitCode = -1
		return res, fmt.Errorf("%w: process timeout", contracts.ErrInvalidArgument)
	}
	if runErr != nil {
		if ee, ok := runErr.(*exec.ExitError); ok {
			res.ExitCode = ee.ExitCode()
		} else {
			res.ExitCode = -1
		}
		return res, runErr
	}
	return res, nil
}

func (r *Runner) allowBin(path string) error {
	if path == "" {
		return fmt.Errorf("%w: empty process path", contracts.ErrInvalidArgument)
	}
	base := filepath.Base(path)
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

// limitedBuffer bounds process output while the child is running. The standard
// bytes.Buffer path retained the complete stdout/stderr in memory and truncated
// only afterwards, allowing a noisy tool to exhaust the API process heap.
type limitedBuffer struct {
	buf       bytes.Buffer
	limit     int64
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	original := len(p)
	remaining := b.limit - int64(b.buf.Len())
	if remaining <= 0 {
		if original > 0 {
			b.truncated = true
		}
		return original, nil
	}
	if int64(len(p)) > remaining {
		p = p[:int(remaining)]
		b.truncated = true
	}
	_, err := b.buf.Write(p)
	return original, err
}

func (b *limitedBuffer) Result() string {
	value := b.buf.String()
	if b.truncated {
		return value + "…(truncated)"
	}
	return value
}
