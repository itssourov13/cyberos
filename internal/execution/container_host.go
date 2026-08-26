package execution

import (
	"context"
	"fmt"
	"time"

	"cyberos/internal/contracts"
)

const HostContainer HostID = "container"

// ContainerHost is an ExecutionHost that runs work inside a hardened container.
// It does not authorize: Broker must validate the Grant first.
type ContainerHost struct {
	Cfg     ContainerHostConfig
	Runtime ContainerRuntime
	// Optional underlying tool executor path is not used for container argv
	// construction in 17B; runtime receives a minimal fixed entrypoint probe
	// unless Args are supplied by trusted host config only.
}

// NewContainerHost builds a ContainerHost. Cfg must Validate().
func NewContainerHost(cfg ContainerHostConfig, rt ContainerRuntime) (*ContainerHost, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if rt == nil {
		return nil, fmt.Errorf("%w: container runtime required", contracts.ErrInvalidArgument)
	}
	return &ContainerHost{Cfg: cfg, Runtime: rt}, nil
}

func (h *ContainerHost) ID() HostID { return HostContainer }

// Execute runs a constrained container. Does not validate Grants (Broker does).
func (h *ContainerHost) Execute(ctx context.Context, req contracts.ExecutionRequest) (contracts.ExecutionResult, error) {
	start := time.Now().UTC()
	if h == nil || h.Runtime == nil {
		return failResult(start, "container host not configured"), fmt.Errorf("%w: container host not configured", contracts.ErrInvalidArgument)
	}
	if err := h.Cfg.Validate(); err != nil {
		return failResult(start, err.Error()), err
	}

	image := h.Cfg.DefaultImage
	// Optional image override only if allowlisted (never from free-form AI URLs).
	if req.Parameters != nil {
		if v, ok := req.Parameters["image"].(string); ok && v != "" {
			if !h.Cfg.imageAllowed(v) {
				return failResult(start, "image not allowlisted"), fmt.Errorf("%w: image not allowlisted", contracts.ErrInvalidArgument)
			}
			image = v
		}
		// Reject privilege / mount / network injection via parameters.
		for _, bad := range []string{"privileged", "docker_sock", "host_network", "mounts", "network_mode", "runtime_binary"} {
			if _, ok := req.Parameters[bad]; ok {
				return failResult(start, "unsafe parameter rejected"), fmt.Errorf("%w: parameter %q not allowed", contracts.ErrInvalidArgument, bad)
			}
		}
	}

	timeout := h.Cfg.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	runReq := ContainerRunRequest{
		Image:          image,
		Args:           []string{"true"}, // harmless default; real tools map in later phases
		Timeout:        timeout,
		MemoryBytes:    h.Cfg.MemoryBytes,
		CPUQuota:       h.Cfg.CPUQuota,
		PidsLimit:      h.Cfg.PidsLimit,
		MaxOutputBytes: h.Cfg.MaxOutputBytes,
		NetworkNone:    true,
		ReadOnlyRoot:   h.Cfg.ReadOnlyRoot,
		NoNewPrivs:     h.Cfg.NoNewPrivs,
		User:           h.Cfg.User,
	}

	res, err := h.Runtime.Run(runCtx, runReq)
	finished := time.Now().UTC()
	if err != nil {
		return contracts.ExecutionResult{
			Status: contracts.ExecutionStatusFailed, StartedAt: start, FinishedAt: finished,
			ErrorMessage: err.Error(), ExecutorID: "container", HostID: string(HostContainer),
		}, err
	}
	status := contracts.ExecutionStatusSucceeded
	if res.ExitCode != 0 {
		status = contracts.ExecutionStatusFailed
	}
	if runCtx.Err() == context.DeadlineExceeded {
		status = contracts.ExecutionStatusTimeout
	}
	out := map[string]any{
		"exit_code": res.ExitCode,
		"stdout":    string(res.Stdout),
		"stderr":    string(res.Stderr),
	}
	if res.Error != "" {
		out["runtime_error"] = res.Error
	}
	return contracts.ExecutionResult{
		Status: status, StartedAt: start, FinishedAt: finished,
		Output: out, ExecutorID: "container", HostID: string(HostContainer),
		CorrelationID: req.CorrelationID,
	}, nil
}

func failResult(start time.Time, msg string) contracts.ExecutionResult {
	now := time.Now().UTC()
	if start.IsZero() {
		start = now
	}
	return contracts.ExecutionResult{
		Status: contracts.ExecutionStatusRejected, StartedAt: start, FinishedAt: now,
		ErrorMessage: msg, ExecutorID: "container", HostID: string(HostContainer),
	}
}

var _ ExecutionHost = (*ContainerHost)(nil)
