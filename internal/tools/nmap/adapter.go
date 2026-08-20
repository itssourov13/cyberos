// Package nmap is the first real security-tool adapter for CYBEROS.
//
// It is NOT part of the core: nmap is registered via tools.Manager and executed
// only through Broker → Sandbox → Executor under a valid grant.
// Capability discovery never authorizes execution.
package nmap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/execution/process"
	"cyberos/internal/tools"
)

const (
	ToolID       = "tool.nmap"
	ExecutorID   = "tool.nmap"
	CapabilityID = "recon.network.portscan"
	DefaultBin   = "nmap"
)

// Adapter is the Nmap security-tool integration.
type Adapter struct {
	desc tools.Descriptor
	proc *process.Runner
	bin  string
}

// Options configures binary path and process policy.
type Options struct {
	Binary string // default: nmap (resolved via LookPath for health only)
	Proc   *process.Runner
}

// New constructs an Nmap adapter. Process execution remains fail-closed unless
// Policy.Enabled and Binary is allowlisted.
func New(opts Options) *Adapter {
	bin := opts.Binary
	if bin == "" {
		bin = DefaultBin
	}
	proc := opts.Proc
	if proc == nil {
		// Fail-closed by default; operators enable allowlist for /usr/bin/nmap.
		proc = process.NewRunner(process.DefaultPolicy())
	}
	return &Adapter{
		bin:  bin,
		proc: proc,
		desc: tools.Descriptor{
			ID:           ToolID,
			Name:         "Nmap",
			Version:      "1",
			Runtime:      contracts.PluginRuntimeGo,
			Capabilities: []contracts.CapabilityID{CapabilityID},
			ExecutorID:   ExecutorID,
			Description:  "Network port scanner (external process; allowlist required)",
			Binary:       bin,
		},
	}
}

// Descriptor implements tools.Adapter.
func (a *Adapter) Descriptor() tools.Descriptor { return a.desc }

// Health checks binary presence and version probe under process policy.
func (a *Adapter) Health(ctx context.Context) tools.HealthReport {
	now := time.Now().UTC()
	path, err := resolveBinary(a.bin)
	if err != nil {
		return tools.HealthReport{
			Status: tools.HealthUnhealthy, Message: "nmap binary not found: " + a.bin, CheckedAt: now,
		}
	}
	if a.proc == nil || !a.proc.Policy.Enabled {
		return tools.HealthReport{
			Status:    tools.HealthDegraded,
			Message:   "binary present at " + path + " but process execution disabled by policy",
			CheckedAt: now,
		}
	}
	// Version probe: nmap --version
	res, err := a.proc.RunDetailed(ctx, process.Spec{
		Path: path, Args: []string{"--version"}, Timeout: 5 * time.Second,
	})
	if err != nil {
		return tools.HealthReport{
			Status: tools.HealthDegraded, Message: "version probe failed: " + err.Error(), CheckedAt: now,
		}
	}
	msg := firstLine(res.Stdout)
	if msg == "" {
		msg = firstLine(res.Stderr)
	}
	if len(msg) > 160 {
		msg = msg[:160]
	}
	return tools.HealthReport{Status: tools.HealthHealthy, Message: msg, CheckedAt: now}
}

// Run executes a constrained nmap argv derived from params (no shell).
// Expected params:
//   - target (string, required): host or CIDR label for scan
//   - ports (string, optional): e.g. "22,80,443" or "1-1024"
//   - mode (string, optional): "list" (default, -sL) or "syn" (-sS requires privileges)
//
// Default mode is host discovery list scan (-sL) which is non-invasive for tests.
func (a *Adapter) Run(ctx context.Context, capability contracts.CapabilityID, params map[string]any) (tools.NormalizedResult, error) {
	if capability != CapabilityID {
		return tools.NormalizedResult{}, fmt.Errorf("%w: capability %s not provided by nmap", contracts.ErrCapabilityUnknown, capability)
	}
	path, err := resolveBinary(a.bin)
	if err != nil {
		return tools.NormalizedResult{
			ToolID: ToolID, Capability: string(capability), Success: false, Summary: "nmap unavailable",
		}, fmt.Errorf("%w: nmap binary not available", contracts.ErrForbidden)
	}
	args, target, err := buildArgv(params)
	if err != nil {
		return tools.NormalizedResult{}, err
	}
	res, err := a.proc.RunDetailed(ctx, process.Spec{
		Path: path, Args: args, Timeout: durationOr(params, 60*time.Second),
	})
	nr := tools.NormalizedResult{
		ToolID: ToolID, Capability: string(capability),
		Success: res.ExitCode == 0 && !res.TimedOut,
		Summary: fmt.Sprintf("nmap exit=%d target=%s", res.ExitCode, target),
		Metrics: map[string]any{
			"exit_code":   res.ExitCode,
			"duration_ms": res.Duration.Milliseconds(),
			"timed_out":   res.TimedOut,
			"binary":      path,
		},
		Stdout: res.Stdout,
		Stderr: res.Stderr,
		Raw:    map[string]any{"target": target, "args": args},
	}
	if res.ExitCode == 0 && res.Stdout != "" {
		nr.Findings = []tools.FindingHint{{
			Title:    "Nmap scan completed",
			Severity: "INFO",
			Detail:   "Target: " + target,
		}}
	}
	if err != nil {
		nr.Success = false
		if nr.Summary == "" {
			nr.Summary = err.Error()
		}
		return nr, err
	}
	return nr, nil
}

func buildArgv(params map[string]any) ([]string, string, error) {
	target, _ := params["target"].(string)
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, "", fmt.Errorf("%w: target required", contracts.ErrInvalidArgument)
	}
	// Reject shell metacharacters in target (argv still separate, defense in depth)
	if strings.ContainsAny(target, " \t\n|&;<>$`\\\"'") {
		return nil, "", fmt.Errorf("%w: target contains unsafe characters", contracts.ErrInvalidArgument)
	}
	mode, _ := params["mode"].(string)
	if mode == "" {
		mode = "list"
	}
	var args []string
	switch mode {
	case "list":
		// Non-invasive list scan
		args = []string{"-sL", "--", target}
	case "connect":
		ports, _ := params["ports"].(string)
		ports = strings.TrimSpace(ports)
		if ports == "" {
			ports = "80,443"
		}
		if strings.ContainsAny(ports, " \t\n|&;<>$`\\\"'") {
			return nil, "", fmt.Errorf("%w: ports contains unsafe characters", contracts.ErrInvalidArgument)
		}
		args = []string{"-sT", "-Pn", "-p", ports, "--", target}
	default:
		return nil, "", fmt.Errorf("%w: unsupported nmap mode %q", contracts.ErrInvalidArgument, mode)
	}
	return args, target, nil
}

func resolveBinary(bin string) (string, error) {
	if filepath.IsAbs(bin) {
		if _, err := os.Stat(bin); err != nil {
			return "", err
		}
		return bin, nil
	}
	return exec.LookPath(bin)
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

func durationOr(params map[string]any, def time.Duration) time.Duration {
	switch v := params["timeout"].(type) {
	case time.Duration:
		return v
	case string:
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

var _ tools.Adapter = (*Adapter)(nil)
