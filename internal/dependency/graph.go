// Package dependency validates task dependency graphs.
// Dependencies gate readiness only — they do not authorize execution.
package dependency

import (
	"fmt"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/task"
)

// Lookup resolves tasks by ID.
type Lookup interface {
	GetByID(id contracts.TaskID) (*task.Task, error)
}

// Graph performs dependency validation and satisfaction checks.
type Graph struct {
	lookup Lookup
}

// New constructs a dependency graph helper.
func New(lookup Lookup) *Graph {
	return &Graph{lookup: lookup}
}

// ValidateEdges checks that all dependencies exist, are not self, and introduce no cycles.
func (g *Graph) ValidateEdges(taskID contracts.TaskID, dependsOn []contracts.TaskID) error {
	if g == nil || g.lookup == nil {
		return fmt.Errorf("%w: dependency graph not configured", contracts.ErrInvalidArgument)
	}
	if taskID == "" {
		return fmt.Errorf("%w: task id required", contracts.ErrInvalidArgument)
	}
	for _, dep := range dependsOn {
		if dep == "" {
			return fmt.Errorf("%w: empty dependency id", contracts.ErrInvalidArgument)
		}
		if dep == taskID {
			return fmt.Errorf("%w: self-dependency forbidden", contracts.ErrInvalidArgument)
		}
		if _, err := g.lookup.GetByID(dep); err != nil {
			return fmt.Errorf("%w: dependency %s: %v", contracts.ErrNotFound, dep, err)
		}
	}
	// Cycle detection on the would-be graph: edges from taskID → each dep mean
	// "taskID depends on dep". A cycle exists if dep can reach taskID.
	for _, dep := range dependsOn {
		if g.reaches(dep, taskID, map[contracts.TaskID]bool{}) {
			return fmt.Errorf("%w: circular dependency involving %s and %s",
				contracts.ErrInvalidArgument, taskID, dep)
		}
	}
	return nil
}

// reaches reports whether from can reach target following DependsOn edges.
func (g *Graph) reaches(from, target contracts.TaskID, visiting map[contracts.TaskID]bool) bool {
	if from == target {
		return true
	}
	if visiting[from] {
		return false
	}
	visiting[from] = true
	t, err := g.lookup.GetByID(from)
	if err != nil || t == nil {
		return false
	}
	for _, dep := range t.DependsOn {
		if g.reaches(dep, target, visiting) {
			return true
		}
	}
	return false
}

// Satisfied reports whether all dependencies of t are in SUCCEEDED state.
func (g *Graph) Satisfied(t *task.Task) (bool, []contracts.TaskID, error) {
	if g == nil || g.lookup == nil {
		return false, nil, fmt.Errorf("%w: dependency graph not configured", contracts.ErrInvalidArgument)
	}
	if t == nil {
		return false, nil, contracts.ErrInvalidArgument
	}
	var unsatisfied []contracts.TaskID
	for _, depID := range t.DependsOn {
		dep, err := g.lookup.GetByID(depID)
		if err != nil {
			return false, nil, fmt.Errorf("%w: dependency %s: %v", contracts.ErrNotFound, depID, err)
		}
		if dep.State != contracts.TaskStateSucceeded {
			unsatisfied = append(unsatisfied, depID)
		}
	}
	return len(unsatisfied) == 0, unsatisfied, nil
}

// AssertRunnable returns an error if dependencies are not satisfied.
// This does not authorize execution.
func (g *Graph) AssertRunnable(t *task.Task) error {
	ok, pending, err := g.Satisfied(t)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: dependencies not satisfied: %v", contracts.ErrInvalidStateTransition, pending)
	}
	return nil
}
