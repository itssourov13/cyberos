package execution

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"cyberos/internal/contracts"
)

// CLIContainerRuntime invokes docker or podman with a fixed, hardened argv.
// Never uses shell interpolation. Requires RuntimeBinary from trusted config.
type CLIContainerRuntime struct {
	Binary string // "docker" or "podman" absolute or PATH name from operator config
}

func (r *CLIContainerRuntime) Run(ctx context.Context, req ContainerRunRequest) (ContainerRunResult, error) {
	if r == nil || r.Binary == "" {
		return ContainerRunResult{}, fmt.Errorf("%w: container runtime binary not configured", contracts.ErrInvalidArgument)
	}
	if req.Image == "" {
		return ContainerRunResult{}, fmt.Errorf("%w: image required", contracts.ErrInvalidArgument)
	}
	// Build argv — no shell.
	args := []string{"run", "--rm"}
	if req.NetworkNone {
		args = append(args, "--network", "none")
	}
	if req.ReadOnlyRoot {
		args = append(args, "--read-only")
	}
	if req.NoNewPrivs {
		args = append(args, "--security-opt", "no-new-privileges:true")
	}
	if req.User != "" {
		args = append(args, "--user", req.User)
	}
	if req.MemoryBytes > 0 {
		args = append(args, "--memory", strconv.FormatInt(req.MemoryBytes, 10))
	}
	if req.PidsLimit > 0 {
		args = append(args, "--pids-limit", strconv.FormatInt(req.PidsLimit, 10))
	}
	if req.CPUQuota > 0 {
		args = append(args, "--cpu-quota", strconv.FormatInt(req.CPUQuota, 10))
	}
	// Never privileged, never host network, never docker.sock
	args = append(args, req.Image)
	args = append(args, req.Args...)

	cmd := exec.CommandContext(ctx, r.Binary, args...)
	var stdout, stderr bytes.Buffer
	max := req.MaxOutputBytes
	if max <= 0 {
		max = 1 << 20
	}
	cmd.Stdout = &limitedWriter{buf: &stdout, n: max}
	cmd.Stderr = &limitedWriter{buf: &stderr, n: max}
	err := cmd.Run()
	res := ContainerRunResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			res.ExitCode = ee.ExitCode()
			return res, nil
		}
		res.Error = err.Error()
		return res, err
	}
	return res, nil
}

type limitedWriter struct {
	buf *bytes.Buffer
	n   int64
	w   int64
}

func (l *limitedWriter) Write(p []byte) (int, error) {
	if l.w >= l.n {
		return len(p), nil // drop excess
	}
	remain := l.n - l.w
	if int64(len(p)) > remain {
		p = p[:remain]
	}
	n, err := l.buf.Write(p)
	l.w += int64(n)
	return len(p), err // pretend full write to avoid pipe break
}

// BuildHardenedArgs is exported for tests asserting security flags.
func BuildHardenedArgs(req ContainerRunRequest) []string {
	r := &CLIContainerRuntime{Binary: "docker"}
	_ = r
	args := []string{"run", "--rm"}
	if req.NetworkNone {
		args = append(args, "--network", "none")
	}
	if req.ReadOnlyRoot {
		args = append(args, "--read-only")
	}
	if req.NoNewPrivs {
		args = append(args, "--security-opt", "no-new-privileges:true")
	}
	if req.User != "" {
		args = append(args, "--user", req.User)
	}
	if req.MemoryBytes > 0 {
		args = append(args, "--memory", strconv.FormatInt(req.MemoryBytes, 10))
	}
	if req.PidsLimit > 0 {
		args = append(args, "--pids-limit", strconv.FormatInt(req.PidsLimit, 10))
	}
	args = append(args, req.Image)
	args = append(args, req.Args...)
	return args
}

// RejectUnsafeMount is a helper for tests/config validation.
func RejectUnsafeMount(path string) error {
	if ForbiddenMount(path) {
		return fmt.Errorf("%w: forbidden mount %q", contracts.ErrInvalidArgument, path)
	}
	if strings.Contains(path, "docker.sock") {
		return fmt.Errorf("%w: docker.sock mount forbidden", contracts.ErrInvalidArgument)
	}
	return nil
}
