// Package runtime provides language-labeled ToolExecutor adapters (Go/Python/Rust)
// that share the process boundary and never hard-code security tools into the core.
package runtime

import (
	"context"
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/execution/process"
)

// Executor is a ToolExecutor for a given language runtime.
// With the default fail-closed process policy it never spawns host processes.
type Executor struct {
	manifest contracts.ToolManifest
	proc     *process.Runner
}

// New builds a runtime executor. proc may be nil (treated as disabled process policy).
func New(m contracts.ToolManifest, proc *process.Runner) (*Executor, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	if proc == nil {
		proc = process.NewRunner(process.DefaultPolicy())
	}
	return &Executor{manifest: m, proc: proc}, nil
}

// Manifest implements contracts.ToolExecutor.
func (e *Executor) Manifest() contracts.ToolManifest { return e.manifest }

// Execute implements contracts.Executor.
// Without an explicit allowlisted process request in parameters, returns a safe
// structured result describing the runtime (no host command execution).
func (e *Executor) Execute(req contracts.ExecutionRequest) (contracts.ExecutionResult, error) {
	start := time.Now().UTC()
	if err := req.Validate(); err != nil {
		return reject(start, e.manifest.ExecutorID, err.Error()), err
	}

	// Optional controlled process: parameters["process"] = { "path": "...", "args": [] }
	if raw, ok := req.Parameters["process"].(map[string]any); ok {
		spec := process.Spec{
			Runtime: e.manifest.Runtime,
			Path:    str(raw["path"]),
			Timeout: e.proc.Policy.DefaultTimeout,
		}
		if args, ok := raw["args"].([]any); ok {
			for _, a := range args {
				spec.Args = append(spec.Args, fmt.Sprint(a))
			}
		}
		stdout, stderr, err := e.proc.Run(context.Background(), spec)
		finish := time.Now().UTC()
		status := contracts.ExecutionStatusSucceeded
		msg := ""
		if err != nil {
			status = contracts.ExecutionStatusFailed
			msg = err.Error()
			if e.proc.Policy.Enabled == false {
				status = contracts.ExecutionStatusRejected
			}
		}
		return contracts.ExecutionResult{
			Status: status, StartedAt: start, FinishedAt: finish,
			Output: map[string]any{
				"runtime":     string(e.manifest.Runtime),
				"executor_id": e.manifest.ExecutorID,
				"stdout":      stdout,
				"stderr":      stderr,
				"process":     true,
			},
			ErrorMessage:  msg,
			CorrelationID: req.CorrelationID,
			ExecutorID:    e.manifest.ExecutorID,
		}, err
	}

	// Default safe path: no process spawn — adapter metadata only.
	finish := time.Now().UTC()
	return contracts.ExecutionResult{
		Status:     contracts.ExecutionStatusSucceeded,
		StartedAt:  start,
		FinishedAt: finish,
		Output: map[string]any{
			"runtime":     string(e.manifest.Runtime),
			"executor_id": e.manifest.ExecutorID,
			"tool_id":     e.manifest.ID,
			"capability":  string(req.Grant.CapabilityID),
			"message":     "runtime adapter invoked without external process",
			"parameters":  req.Parameters,
		},
		CorrelationID: req.CorrelationID,
		ExecutorID:    e.manifest.ExecutorID,
	}, nil
}

func str(v any) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func reject(start time.Time, id, msg string) contracts.ExecutionResult {
	now := time.Now().UTC()
	return contracts.ExecutionResult{
		Status: contracts.ExecutionStatusRejected, StartedAt: start, FinishedAt: now,
		ErrorMessage: msg, ExecutorID: id,
	}
}

// NewGo / NewPython / NewRust helpers for common runtime labels.
func NewGo(id string, caps []contracts.CapabilityID, proc *process.Runner) (*Executor, error) {
	return New(contracts.ToolManifest{
		ID: id, Name: id, Version: "1", Runtime: contracts.PluginRuntimeGo,
		Capabilities: caps, ExecutorID: id,
	}, proc)
}

func NewPython(id string, caps []contracts.CapabilityID, proc *process.Runner) (*Executor, error) {
	return New(contracts.ToolManifest{
		ID: id, Name: id, Version: "1", Runtime: contracts.PluginRuntimePython,
		Capabilities: caps, ExecutorID: id,
	}, proc)
}

func NewRust(id string, caps []contracts.CapabilityID, proc *process.Runner) (*Executor, error) {
	return New(contracts.ToolManifest{
		ID: id, Name: id, Version: "1", Runtime: contracts.PluginRuntimeRust,
		Capabilities: caps, ExecutorID: id,
	}, proc)
}

var _ contracts.ToolExecutor = (*Executor)(nil)
