package sdk

import (
	"sync"
	"time"

	"cyberos/internal/contracts"
)

// ProjectView is a UI/API-facing project summary (no domain leakage).
type ProjectView struct {
	ID          contracts.ProjectID `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	CreatedAt   time.Time           `json:"created_at"`
}

// ScopeView is a UI/API-facing scope summary.
type ScopeView struct {
	ID        contracts.ScopeID     `json:"id"`
	ProjectID contracts.ProjectID   `json:"project_id"`
	Name      string                `json:"name"`
	Status    contracts.ScopeStatus `json:"status"`
}

// TaskView is a UI/API-facing task summary including schedule/execution status.
type TaskView struct {
	ID           contracts.TaskID       `json:"id"`
	ProjectID    contracts.ProjectID    `json:"project_id"`
	ScopeID      contracts.ScopeID      `json:"scope_id"`
	PrincipalID  contracts.PrincipalID  `json:"principal_id"`
	CapabilityID contracts.CapabilityID `json:"capability_id"`
	Title        string                 `json:"title"`
	State        contracts.TaskState    `json:"state"`
	Type         contracts.TaskType     `json:"type"`
	Scheduled    bool                   `json:"scheduled"`
	NotBefore    *time.Time             `json:"not_before,omitempty"`
}

// AssessmentView is a durable-facing assessment summary for API/UI.
type AssessmentView struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	ScopeID     string `json:"scope_id"`
	PrincipalID string `json:"principal_id"`
	Title       string `json:"title"`
	Target      string `json:"target,omitempty"`
	State       string `json:"state"`
	StepCount   int    `json:"step_count"`
}

// Catalog is an in-memory index for dashboard listings, synced from services.
// It is not a source of authorization.
type Catalog struct {
	mu          sync.RWMutex
	projects    map[contracts.ProjectID]ProjectView
	scopes      map[contracts.ScopeID]ScopeView
	tasks       map[contracts.TaskID]TaskView
	assessments map[string]AssessmentView
}

// NewCatalog constructs an empty catalog.
func NewCatalog() *Catalog {
	return &Catalog{
		projects:    make(map[contracts.ProjectID]ProjectView),
		scopes:      make(map[contracts.ScopeID]ScopeView),
		tasks:       make(map[contracts.TaskID]TaskView),
		assessments: make(map[string]AssessmentView),
	}
}

// UpsertProject stores a project view.
func (c *Catalog) UpsertProject(p ProjectView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.projects[p.ID] = p
}

// UpsertScope stores a scope view.
func (c *Catalog) UpsertScope(s ScopeView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.scopes[s.ID] = s
}

// UpsertTask stores a task view.
func (c *Catalog) UpsertTask(t TaskView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tasks[t.ID] = t
}

// ListProjects returns all projects.
func (c *Catalog) ListProjects() []ProjectView {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]ProjectView, 0, len(c.projects))
	for _, p := range c.projects {
		out = append(out, p)
	}
	return out
}

// ListScopes returns scopes, optionally filtered by project.
func (c *Catalog) ListScopes(projectID contracts.ProjectID) []ScopeView {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var out []ScopeView
	for _, s := range c.scopes {
		if projectID != "" && s.ProjectID != projectID {
			continue
		}
		out = append(out, s)
	}
	return out
}

// ListTasks returns tasks, optionally filtered by project or scope.
func (c *Catalog) ListTasks(projectID contracts.ProjectID, scopeID contracts.ScopeID) []TaskView {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var out []TaskView
	for _, t := range c.tasks {
		if projectID != "" && t.ProjectID != projectID {
			continue
		}
		if scopeID != "" && t.ScopeID != scopeID {
			continue
		}
		out = append(out, t)
	}
	return out
}

// GetTask returns a task view by ID.
func (c *Catalog) GetTask(id contracts.TaskID) (TaskView, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	t, ok := c.tasks[id]
	return t, ok
}

// SeedDemo loads a minimal demo catalog for local UI/API exploration.
func (c *Catalog) SeedDemo() {
	now := time.Now().UTC()
	c.UpsertProject(ProjectView{ID: "proj-1", Name: "Demo Project", Description: "Phase 3A foundation", CreatedAt: now})
	c.UpsertScope(ScopeView{ID: "scope-1", ProjectID: "proj-1", Name: "Main Scope", Status: contracts.ScopeStatusActive})
	c.UpsertTask(TaskView{
		ID: "task-1", ProjectID: "proj-1", ScopeID: "scope-1",
		PrincipalID: "user-1", CapabilityID: "recon.domain.enumerate",
		Title: "Demo enumeration", State: contracts.TaskStatePending, Type: contracts.TaskTypeRecon,
	})
}

// UpsertAssessment stores an assessment view.
func (c *Catalog) UpsertAssessment(a AssessmentView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.assessments[a.ID] = a
}

// ListAssessments returns assessment summaries.
func (c *Catalog) ListAssessments() []AssessmentView {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]AssessmentView, 0, len(c.assessments))
	for _, a := range c.assessments {
		out = append(out, a)
	}
	return out
}

// SyncTask upserts a task view from domain-like fields (durable path helper).
func (c *Catalog) SyncTask(v TaskView) { c.UpsertTask(v) }
