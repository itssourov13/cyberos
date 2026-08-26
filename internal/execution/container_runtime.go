package execution

import (
	"context"
	"fmt"
	"io"
	"time"

	"cyberos/internal/contracts"
)

// ContainerRunRequest is a constrained, host-configured run request.
// Untrusted callers must not populate privileged fields — ContainerHost builds this.
type ContainerRunRequest struct {
	Image          string
	Args           []string // argv only; never shell-string
	WorkDir        string
	Env            []string // non-secret only
	Timeout        time.Duration
	MemoryBytes    int64
	CPUQuota       int64 // relative units; runtime-specific mapping
	PidsLimit      int64
	MaxOutputBytes int64
	NetworkNone    bool
	ReadOnlyRoot   bool
	NoNewPrivs     bool
	User           string // non-root preferred, e.g. "65534:65534"
}

// ContainerRunResult is bounded capture of a container run.
type ContainerRunResult struct {
	ExitCode int
	Stdout   []byte
	Stderr   []byte
	Error    string
}

// ContainerRuntime launches a single hardened container.
// Implementations must never accept --privileged, host network, or docker.sock mounts.
type ContainerRuntime interface {
	Run(ctx context.Context, req ContainerRunRequest) (ContainerRunResult, error)
}

// ContainerHostConfig is trusted operator configuration (not request-controlled).
type ContainerHostConfig struct {
	Enabled        bool
	DefaultImage   string
	AllowedImages  []string
	RuntimeBinary  string // e.g. "docker" or "podman"; empty = fake-only / unavailable
	MemoryBytes    int64
	CPUQuota       int64
	PidsLimit      int64
	MaxOutputBytes int64
	Timeout        time.Duration
	User           string
	NetworkNone    bool
	ReadOnlyRoot   bool
	NoNewPrivs     bool
}

// DefaultContainerHostConfig returns fail-closed safe defaults.
func DefaultContainerHostConfig() ContainerHostConfig {
	return ContainerHostConfig{
		Enabled:        true,
		DefaultImage:   "cyberos/tool-runtime:stable",
		AllowedImages:  []string{"cyberos/tool-runtime:stable"},
		MemoryBytes:    512 << 20,
		CPUQuota:       100000, // 1 CPU equivalent for many runtimes
		PidsLimit:      128,
		MaxOutputBytes: 1 << 20,
		Timeout:        60 * time.Second,
		User:           "65534:65534",
		NetworkNone:    true,
		ReadOnlyRoot:   true,
		NoNewPrivs:     true,
	}
}

func (c ContainerHostConfig) imageAllowed(img string) bool {
	if img == "" {
		return false
	}
	for _, a := range c.AllowedImages {
		if a == img {
			return true
		}
	}
	return false
}

// Validate rejects unsafe operator config (privileged-style settings).
func (c ContainerHostConfig) Validate() error {
	if !c.Enabled {
		return fmt.Errorf("%w: container host disabled", contracts.ErrInvalidArgument)
	}
	if c.DefaultImage == "" || !c.imageAllowed(c.DefaultImage) {
		return fmt.Errorf("%w: default image not allowlisted", contracts.ErrInvalidArgument)
	}
	if c.MemoryBytes <= 0 || c.PidsLimit <= 0 || c.MaxOutputBytes <= 0 {
		return fmt.Errorf("%w: resource limits required", contracts.ErrInvalidArgument)
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("%w: timeout required", contracts.ErrInvalidArgument)
	}
	if !c.NetworkNone {
		// 17B default policy: network must be none unless later phases expand policy engine.
		return fmt.Errorf("%w: host/network modes other than none are not allowed in 17B", contracts.ErrInvalidArgument)
	}
	return nil
}

// ForbiddenMount reports paths that must never be mounted into containers.
func ForbiddenMount(path string) bool {
	switch path {
	case "/", "/etc", "/home", "/var", "/var/run", "/var/run/docker.sock",
		"/proc", "/sys", "/dev", "/root", "/usr", "/bin", "/sbin":
		return true
	}
	if len(path) >= len("/var/run/docker.sock") && path[:len("/var/run/docker.sock")] == "/var/run/docker.sock" {
		return true
	}
	return false
}

// LimitReader is used by runtimes to bound stdout/stderr.
func LimitReader(r io.Reader, n int64) io.Reader {
	return io.LimitReader(r, n)
}
