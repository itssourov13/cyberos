package workflow

import (
	"context"
	"fmt"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/task"
	"cyberos/internal/taskengine"
)

// MaterializeResult is the outcome of translating a workflow into tasks.
type MaterializeResult struct {
	WorkflowID string
	Version    contracts.SchemaVersion
	Tasks      []*task.Task
	// StepToTask maps workflow step ID → task ID
	StepToTask map[string]contracts.TaskID
	// Resolutions maps step ID → resolved tool (discovery only)
	Resolutions map[string]ResolvedTool
}

// Materializer turns validated workflows into task-engine tasks.
// It does not evaluate CP, issue grants, or call the Broker.
type Materializer struct {
	Engine   *taskengine.Engine
	Resolver *CapabilityResolver
}

// NewMaterializer constructs a materializer.
func NewMaterializer(eng *taskengine.Engine, res *CapabilityResolver) *Materializer {
	return &Materializer{Engine: eng, Resolver: res}
}

// Materialize validates the definition, resolves capabilities, and creates tasks
// with dependencies. Fail-closed on unknown capabilities or validation errors.
func (m *Materializer) Materialize(ctx context.Context, def Definition, principalID contracts.PrincipalID) (*MaterializeResult, error) {
	if m == nil || m.Engine == nil || m.Resolver == nil {
		return nil, fmt.Errorf("%w: workflow materializer not configured", contracts.ErrInvalidArgument)
	}
	if err := def.Validate(); err != nil {
		return nil, err
	}
	if principalID == "" {
		return nil, fmt.Errorf("%w: principal required", contracts.ErrInvalidArgument)
	}

	out := &MaterializeResult{
		WorkflowID:  def.ID,
		Version:     def.Version,
		StepToTask:  map[string]contracts.TaskID{},
		Resolutions: map[string]ResolvedTool{},
	}

	// Resolve all capabilities first (fail closed before creating any task)
	for _, step := range def.Steps {
		rt, err := m.Resolver.ResolvePrimary(step.CapabilityID)
		if err != nil {
			return nil, fmt.Errorf("step %s: %w", step.ID, err)
		}
		out.Resolutions[step.ID] = rt
	}

	// Create tasks in definition order
	for _, step := range def.Steps {
		title := step.Title
		if title == "" {
			title = fmt.Sprintf("%s:%s", def.ID, step.ID)
		}
		taskID := contracts.TaskID(fmt.Sprintf("wf-%s-%s", def.ID, step.ID))
		tk, err := m.Engine.Create(ctx, taskengine.CreateRequest{
			ID:           taskID,
			ProjectID:    def.ProjectID,
			ScopeID:      def.ScopeID,
			PrincipalID:  principalID,
			CapabilityID: step.CapabilityID,
			Type:         contracts.TaskTypeScan,
			Title:        title,
			Description:  fmt.Sprintf("workflow=%s version=%s step=%s tool=%s", def.ID, def.Version, step.ID, out.Resolutions[step.ID].ToolID),
		})
		if err != nil {
			return nil, fmt.Errorf("create task for step %s: %w", step.ID, err)
		}
		if step.MaxAttempts > 0 {
			if _, err := m.Engine.ApplyRetryPolicy(ctx, tk.ID, step.MaxAttempts, 0); err != nil {
				return nil, err
			}
		}
		if step.Timeout > 0 {
			if _, err := m.Engine.SetTimeoutDeadline(ctx, tk.ID, step.Timeout, nil); err != nil {
				return nil, err
			}
		}
		out.Tasks = append(out.Tasks, tk)
		out.StepToTask[step.ID] = tk.ID
	}

	// Wire dependencies (step deps → task deps)
	for _, step := range def.Steps {
		if len(step.DependsOn) == 0 {
			continue
		}
		var deps []contracts.TaskID
		for _, sid := range step.DependsOn {
			deps = append(deps, out.StepToTask[sid])
		}
		if _, err := m.Engine.SetDependencies(ctx, out.StepToTask[step.ID], deps); err != nil {
			return nil, fmt.Errorf("dependencies for step %s: %w", step.ID, err)
		}
	}

	return out, nil
}
