// Package pythonprobe is a minimal Python-runtime adapter proving multi-language
// execution through Broker → Sandbox → process allowlist (not a core dependency).
package pythonprobe

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/execution/process"
	"cyberos/internal/tools"
)

const (
	ToolID       = "tool.python.probe"
	ExecutorID   = "tool.python.probe"
	CapabilityID = "runtime.python.probe"
	DefaultBin   = "python3"
)

// Adapter runs a constrained python3 invocation (version / fixed -c only).
type Adapter struct {
	desc tools.Descriptor
	proc *process.Runner
	bin  string
}

// Options configures binary and process runner.
type Options struct {
	Binary string
	Proc   *process.Runner
}

// New constructs the Python probe adapter.
func New(opts Options) *Adapter {
	bin := opts.Binary
	if bin == "" {
		bin = DefaultBin
	}
	proc := opts.Proc
	if proc == nil {
		proc = process.NewRunner(process.DefaultPolicy())
	}
	return &Adapter{
		bin:  bin,
		proc: proc,
		desc: tools.Descriptor{
			ID: ToolID, Name: "Python Runtime Probe", Version: "1",
			Runtime: contracts.PluginRuntimePython, Capabilities: []contracts.CapabilityID{CapabilityID},
			ExecutorID: ExecutorID, Description: "Proves Python runtime via allowlisted python3 process",
			Binary: bin,
		},
	}
}

func (a *Adapter) Descriptor() tools.Descriptor { return a.desc }

func (a *Adapter) Health(ctx context.Context) tools.HealthReport {
	now := time.Now().UTC()
	path, err := exec.LookPath(a.bin)
	if err != nil {
		return tools.HealthReport{Status: tools.HealthUnhealthy, Message: "python3 not found", CheckedAt: now}
	}
	if a.proc == nil || !a.proc.Policy.Enabled {
		return tools.HealthReport{Status: tools.HealthDegraded, Message: "python present at " + path + " but process disabled", CheckedAt: now}
	}
	res, err := a.proc.RunDetailed(ctx, process.Spec{Path: path, Args: []string{"--version"}, Timeout: 3 * time.Second})
	msg := strings.TrimSpace(res.Stdout + " " + res.Stderr)
	if err != nil {
		return tools.HealthReport{Status: tools.HealthDegraded, Message: err.Error(), CheckedAt: now}
	}
	return tools.HealthReport{Status: tools.HealthHealthy, Message: msg, CheckedAt: now}
}

func (a *Adapter) Run(ctx context.Context, capability contracts.CapabilityID, params map[string]any) (tools.NormalizedResult, error) {
	if capability != CapabilityID {
		return tools.NormalizedResult{}, fmt.Errorf("%w: capability", contracts.ErrCapabilityUnknown)
	}
	path, err := exec.LookPath(a.bin)
	if err != nil {
		return tools.NormalizedResult{ToolID: ToolID, Success: false, Summary: "python missing"}, fmt.Errorf("%w: python missing", contracts.ErrForbidden)
	}
	// Fixed argv only — no user-supplied code execution.
	args := []string{"-c", "print('cyberos-python-probe-ok')"}
	res, err := a.proc.RunDetailed(ctx, process.Spec{Path: path, Args: args, Timeout: 5 * time.Second})
	nr := tools.NormalizedResult{
		ToolID: ToolID, Capability: string(capability),
		Success: res.ExitCode == 0 && !res.TimedOut,
		Summary: fmt.Sprintf("python probe exit=%d", res.ExitCode),
		Metrics: map[string]any{"exit_code": res.ExitCode, "duration_ms": res.Duration.Milliseconds(), "runtime": "python"},
		Stdout:  res.Stdout, Stderr: res.Stderr,
	}
	if err != nil {
		nr.Success = false
		return nr, err
	}
	return nr, nil
}

var _ tools.Adapter = (*Adapter)(nil)
