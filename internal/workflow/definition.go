// Package workflow provides declarative, versioned, capability-driven workflow
// definitions. Workflows orchestrate only — they never authorize or execute tools.
//
// Every tool execution remains:
//
//	Scope → Auth → CP → Decision → Grant → Broker → Sandbox → Executor
package workflow

import (
	"fmt"
	"time"

	"cyberos/internal/contracts"
)

// Definition is a versioned workflow/assessment plan.
type Definition struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description,omitempty"`
	Version     contracts.SchemaVersion `json:"version"`
	ProjectID   contracts.ProjectID     `json:"project_id"`
	ScopeID     contracts.ScopeID       `json:"scope_id"`
	Steps       []Step                  `json:"steps"`
}

// Step is one capability-targeted unit of planned work.
type Step struct {
	ID           string                 `json:"id"`
	CapabilityID contracts.CapabilityID `json:"capability_id"`
	Parameters   map[string]any         `json:"parameters,omitempty"`
	DependsOn    []string               `json:"depends_on,omitempty"` // step IDs
	Timeout      time.Duration          `json:"timeout,omitempty"`
	MaxAttempts  int                    `json:"max_attempts,omitempty"`
	Title        string                 `json:"title,omitempty"`
}

// Validate checks structural integrity (not authorization).
func (d Definition) Validate() error {
	if err := contracts.ValidateNonEmpty("workflow.id", d.ID); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("workflow.name", d.Name); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("workflow.version", string(d.Version)); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("workflow.project_id", string(d.ProjectID)); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("workflow.scope_id", string(d.ScopeID)); err != nil {
		return err
	}
	if len(d.Steps) == 0 {
		return fmt.Errorf("%w: workflow must have at least one step", contracts.ErrInvalidArgument)
	}
	seen := map[string]struct{}{}
	for i, s := range d.Steps {
		if err := contracts.ValidateNonEmpty(fmt.Sprintf("steps[%d].id", i), s.ID); err != nil {
			return err
		}
		if err := contracts.ValidateNonEmpty(fmt.Sprintf("steps[%d].capability_id", i), string(s.CapabilityID)); err != nil {
			return err
		}
		if _, ok := seen[s.ID]; ok {
			return fmt.Errorf("%w: duplicate step id %q", contracts.ErrInvalidArgument, s.ID)
		}
		seen[s.ID] = struct{}{}
		if s.MaxAttempts < 0 {
			return fmt.Errorf("%w: step %s max_attempts cannot be negative", contracts.ErrInvalidArgument, s.ID)
		}
	}
	// Dependency references
	for _, s := range d.Steps {
		for _, dep := range s.DependsOn {
			if _, ok := seen[dep]; !ok {
				return fmt.Errorf("%w: step %s depends on unknown step %q", contracts.ErrInvalidArgument, s.ID, dep)
			}
			if dep == s.ID {
				return fmt.Errorf("%w: step %s cannot depend on itself", contracts.ErrInvalidArgument, s.ID)
			}
		}
	}
	if err := detectCycle(d.Steps); err != nil {
		return err
	}
	return nil
}

func detectCycle(steps []Step) error {
	// Kahn topological sort
	indeg := map[string]int{}
	adj := map[string][]string{}
	for _, s := range steps {
		if _, ok := indeg[s.ID]; !ok {
			indeg[s.ID] = 0
		}
		for _, d := range s.DependsOn {
			adj[d] = append(adj[d], s.ID)
			indeg[s.ID]++
		}
	}
	var q []string
	for id, n := range indeg {
		if n == 0 {
			q = append(q, id)
		}
	}
	visited := 0
	for len(q) > 0 {
		n := q[0]
		q = q[1:]
		visited++
		for _, m := range adj[n] {
			indeg[m]--
			if indeg[m] == 0 {
				q = append(q, m)
			}
		}
	}
	if visited != len(steps) {
		return fmt.Errorf("%w: workflow dependency cycle detected", contracts.ErrInvalidArgument)
	}
	return nil
}
