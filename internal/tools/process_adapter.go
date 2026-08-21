package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/execution/process"
)

// ProcessToolAdapter runs an allowlisted external binary for a registered tool.
// It does not hard-code any specific security product; path comes from Descriptor.Binary
// and must pass process policy allowlist.
type ProcessToolAdapter struct {
	desc Descriptor
	proc *process.Runner
	// VersionArgs are used for health/version probes (e.g. ["--version"]).
	VersionArgs []string
	// BuildArgs constructs argv from capability + params. Must not introduce shell.
	BuildArgs func(capability contracts.CapabilityID, params map[string]any) ([]string, error)
	// Parse maps process output into NormalizedResult.
	Parse func(capability contracts.CapabilityID, params map[string]any, res process.Result) NormalizedResult
}

// NewProcessToolAdapter constructs a process-backed adapter.
func NewProcessToolAdapter(d Descriptor, proc *process.Runner) (*ProcessToolAdapter, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	if d.Binary == "" {
		return nil, fmt.Errorf("%w: process adapter requires descriptor.binary", contracts.ErrInvalidArgument)
	}
	if proc == nil {
		proc = process.NewRunner(process.DefaultPolicy())
	}
	a := &ProcessToolAdapter{desc: d, proc: proc, VersionArgs: []string{"--version"}}
	a.BuildArgs = defaultBuildArgs
	a.Parse = defaultParse(d.ID)
	return a, nil
}

func defaultBuildArgs(_ contracts.CapabilityID, params map[string]any) ([]string, error) {
	var args []string
	if raw, ok := params["args"].([]any); ok {
		for _, a := range raw {
			args = append(args, fmt.Sprint(a))
		}
	}
	if raw, ok := params["args"].([]string); ok {
		args = append(args, raw...)
	}
	// Reject shell-style invocation patterns in argv (never sh -c).
	for i, a := range args {
		if a == "-c" || a == "--command" || a == "/c" {
			return nil, fmt.Errorf("%w: shell command flags are not permitted in tool args", contracts.ErrInvalidArgument)
		}
		if i > 0 && (args[i-1] == "-c" || args[i-1] == "--command") {
			return nil, fmt.Errorf("%w: shell command string arguments are not permitted", contracts.ErrInvalidArgument)
		}
	}
	// Path is validated at Run with the real binary; validate args only here.
	if err := process.ValidateArgv("tool", args); err != nil {
		return nil, err
	}
	return args, nil
}

func defaultParse(toolID string) func(contracts.CapabilityID, map[string]any, process.Result) NormalizedResult {
	return func(cap contracts.CapabilityID, params map[string]any, res process.Result) NormalizedResult {
		return NormalizedResult{
			ToolID:     toolID,
			Capability: string(cap),
			Success:    res.ExitCode == 0 && !res.TimedOut,
			Summary:    fmt.Sprintf("exit=%d duration=%s", res.ExitCode, res.Duration),
			Metrics: map[string]any{
				"exit_code":   res.ExitCode,
				"duration_ms": res.Duration.Milliseconds(),
				"timed_out":   res.TimedOut,
			},
			Stdout: res.Stdout,
			Stderr: res.Stderr,
			Raw:    map[string]any{"params": params},
		}
	}
}

// Descriptor implements Adapter.
func (a *ProcessToolAdapter) Descriptor() Descriptor { return a.desc }

// Health checks binary presence and optional version probe under policy.
func (a *ProcessToolAdapter) Health(ctx context.Context) HealthReport {
	now := time.Now().UTC()
	path := a.desc.Binary
	if _, err := os.Stat(path); err != nil {
		// Try PATH lookup of basename only for health messaging; execution still needs allowlist path.
		if p, err2 := lookPath(filepath.Base(path)); err2 == nil {
			path = p
		} else {
			return HealthReport{Status: HealthUnhealthy, Message: "binary not found: " + a.desc.Binary, CheckedAt: now}
		}
	}
	if a.proc == nil || !a.proc.Policy.Enabled {
		return HealthReport{
			Status:    HealthDegraded,
			Message:   "binary present but process execution disabled by policy",
			CheckedAt: now,
		}
	}
	// Version probe
	res, err := a.proc.RunDetailed(ctx, process.Spec{Path: a.desc.Binary, Args: a.VersionArgs, Timeout: 3 * time.Second})
	if err != nil {
		return HealthReport{Status: HealthDegraded, Message: "version probe failed: " + err.Error(), CheckedAt: now}
	}
	msg := res.Stdout
	if msg == "" {
		msg = res.Stderr
	}
	if len(msg) > 120 {
		msg = msg[:120]
	}
	return HealthReport{Status: HealthHealthy, Message: "version: " + msg, CheckedAt: now}
}

func lookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// Run executes the allowlisted binary with validated argv.
func (a *ProcessToolAdapter) Run(ctx context.Context, capability contracts.CapabilityID, params map[string]any) (NormalizedResult, error) {
	if len(a.desc.Capabilities) > 0 {
		ok := false
		for _, c := range a.desc.Capabilities {
			if c == capability {
				ok = true
				break
			}
		}
		if !ok {
			return NormalizedResult{}, fmt.Errorf("%w: capability not declared", contracts.ErrCapabilityUnknown)
		}
	}
	args, err := a.BuildArgs(capability, params)
	if err != nil {
		return NormalizedResult{}, err
	}
	res, err := a.proc.RunDetailed(ctx, process.Spec{
		Path:    a.desc.Binary,
		Args:    args,
		Env:     stringMap(params["env"]),
		WorkDir: strParam(params["workdir"]),
		Timeout: durationParam(params["timeout"]),
	})
	nr := a.Parse(capability, params, res)
	if err != nil {
		nr.Success = false
		if nr.Summary == "" {
			nr.Summary = err.Error()
		}
		return nr, err
	}
	return nr, nil
}

func stringMap(v any) map[string]string {
	m, ok := v.(map[string]string)
	if ok {
		return m
	}
	if raw, ok := v.(map[string]any); ok {
		out := make(map[string]string, len(raw))
		for k, val := range raw {
			out[k] = fmt.Sprint(val)
		}
		return out
	}
	return nil
}

func strParam(v any) string {
	s, _ := v.(string)
	return s
}

func durationParam(v any) time.Duration {
	switch t := v.(type) {
	case time.Duration:
		return t
	case string:
		d, _ := time.ParseDuration(t)
		return d
	case float64:
		return time.Duration(t) * time.Second
	default:
		return 0
	}
}

var _ Adapter = (*ProcessToolAdapter)(nil)
