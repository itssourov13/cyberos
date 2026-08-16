// Package mock provides a safe mock executor for Phase 1.
// It never invokes shell, network scanners, or real security tools.
package mock

import (
	"fmt"
	"time"

	"cyberos/internal/contracts"
)

// Executor is a safe mock implementation of contracts.Executor.
// It only returns controlled, synthetic output.
type Executor struct {
	// ID identifies this executor in results (default "mock").
	ID string
}

// New returns a mock executor.
func New() *Executor {
	return &Executor{ID: "mock"}
}

// Execute satisfies contracts.Executor.
// It does not perform any real work, shell execution, or tool invocation.
func (e *Executor) Execute(req contracts.ExecutionRequest) (contracts.ExecutionResult, error) {
	if e == nil {
		return contracts.ExecutionResult{}, fmt.Errorf("%w: mock executor nil", contracts.ErrInvalidArgument)
	}
	if err := req.Validate(); err != nil {
		return contracts.ExecutionResult{
			Status:       contracts.ExecutionStatusRejected,
			StartedAt:    time.Now().UTC(),
			FinishedAt:   time.Now().UTC(),
			ErrorMessage: err.Error(),
			ExecutorID:   e.executorID(),
		}, err
	}

	start := time.Now().UTC()
	// Synthetic, safe output only.
	out := map[string]any{
		"mock":       true,
		"capability": string(req.Grant.CapabilityID),
		"task_id":    string(req.Grant.TaskID),
		"message":    "safe mock execution completed",
		"parameters": req.Parameters,
	}
	finish := time.Now().UTC()
	return contracts.ExecutionResult{
		Status:        contracts.ExecutionStatusSucceeded,
		StartedAt:     start,
		FinishedAt:    finish,
		Output:        out,
		CorrelationID: req.CorrelationID,
		ExecutorID:    e.executorID(),
	}, nil
}

func (e *Executor) executorID() string {
	if e.ID == "" {
		return "mock"
	}
	return e.ID
}

// Ensure interface compliance at compile time.
var _ contracts.Executor = (*Executor)(nil)
