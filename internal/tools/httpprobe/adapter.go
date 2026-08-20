// Package httpprobe is a Go-runtime web-pack adapter using allowlisted curl.
package httpprobe

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
	ToolID       = "tool.http.probe"
	ExecutorID   = "tool.http.probe"
	CapabilityID = "web.http.probe"
	DefaultBin   = "curl"
)

// Adapter runs constrained curl HEAD/GET against a validated URL host.
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

// New constructs the HTTP probe adapter.
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
			ID: ToolID, Name: "HTTP Probe", Version: "1",
			Runtime: contracts.PluginRuntimeGo, Capabilities: []contracts.CapabilityID{CapabilityID},
			ExecutorID: ExecutorID, Description: "HTTP HEAD probe via allowlisted curl",
			Binary: bin,
		},
	}
}

func (a *Adapter) Descriptor() tools.Descriptor { return a.desc }

func (a *Adapter) Health(ctx context.Context) tools.HealthReport {
	now := time.Now().UTC()
	path, err := exec.LookPath(a.bin)
	if err != nil {
		return tools.HealthReport{Status: tools.HealthUnhealthy, Message: "curl not found", CheckedAt: now}
	}
	if a.proc == nil || !a.proc.Policy.Enabled {
		return tools.HealthReport{Status: tools.HealthDegraded, Message: "curl present but process disabled", CheckedAt: now}
	}
	res, err := a.proc.RunDetailed(ctx, process.Spec{Path: path, Args: []string{"--version"}, Timeout: 3 * time.Second})
	msg := firstLine(res.Stdout + res.Stderr)
	if err != nil {
		return tools.HealthReport{Status: tools.HealthDegraded, Message: err.Error(), CheckedAt: now}
	}
	return tools.HealthReport{Status: tools.HealthHealthy, Message: msg, CheckedAt: now}
}

func (a *Adapter) Run(ctx context.Context, capability contracts.CapabilityID, params map[string]any) (tools.NormalizedResult, error) {
	if capability != CapabilityID {
		return tools.NormalizedResult{}, fmt.Errorf("%w: capability", contracts.ErrCapabilityUnknown)
	}
	url, _ := params["url"].(string)
	url = strings.TrimSpace(url)
	if url == "" {
		return tools.NormalizedResult{}, fmt.Errorf("%w: url required", contracts.ErrInvalidArgument)
	}
	if strings.ContainsAny(url, " \t\n|&;<>$`\\\"'") {
		return tools.NormalizedResult{}, fmt.Errorf("%w: url contains unsafe characters", contracts.ErrInvalidArgument)
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return tools.NormalizedResult{}, fmt.Errorf("%w: url must be http(s)", contracts.ErrInvalidArgument)
	}
	path, err := exec.LookPath(a.bin)
	if err != nil {
		return tools.NormalizedResult{ToolID: ToolID, Success: false, Summary: "curl missing"}, fmt.Errorf("%w: curl missing", contracts.ErrForbidden)
	}
	// Fixed safe argv: HEAD only, no redirects follow, max time
	args := []string{"-sS", "-I", "-m", "10", "--", url}
	res, err := a.proc.RunDetailed(ctx, process.Spec{Path: path, Args: args, Timeout: 15 * time.Second})
	nr := tools.NormalizedResult{
		ToolID: ToolID, Capability: string(capability),
		Success: res.ExitCode == 0 && !res.TimedOut,
		Summary: fmt.Sprintf("http probe exit=%d", res.ExitCode),
		Metrics: map[string]any{"exit_code": res.ExitCode, "duration_ms": res.Duration.Milliseconds()},
		Stdout:  res.Stdout, Stderr: res.Stderr,
		Raw: map[string]any{"url": url},
	}
	if err != nil {
		nr.Success = false
		return nr, err
	}
	return nr, nil
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

var _ tools.Adapter = (*Adapter)(nil)
