package sdk

import (
	"context"
	"encoding/json"

	"cyberos/internal/contracts"
	"cyberos/internal/storage/ports"
)

// RefreshFromRepos reloads catalog views from durable repositories.
// Persistence is not authorization.
func (c *Catalog) RefreshFromRepos(ctx context.Context, repos *ports.Repositories) error {
	if c == nil || repos == nil {
		return contracts.ErrInvalidArgument
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.projects = make(map[contracts.ProjectID]ProjectView)
	c.scopes = make(map[contracts.ScopeID]ScopeView)
	c.tasks = make(map[contracts.TaskID]TaskView)
	c.assessments = make(map[string]AssessmentView)

	if repos.Projects != nil {
		list, err := repos.Projects.List(ctx, 500, 0)
		if err != nil {
			return err
		}
		for _, p := range list {
			c.projects[p.ID] = ProjectView{
				ID: p.ID, Name: p.Name, Description: p.Description, CreatedAt: p.CreatedAt, TenantID: p.TenantID,
			}
		}
	}
	if repos.Scopes != nil {
		list, err := repos.Scopes.ListAll(ctx, 500, 0)
		if err != nil {
			return err
		}
		for _, s := range list {
			c.scopes[s.ID] = ScopeView{
				ID: s.ID, ProjectID: s.ProjectID, Name: s.Name, Status: s.Status,
			}
		}
	}
	if repos.Tasks != nil {
		list, err := repos.Tasks.List(ctx, 500, 0)
		if err != nil {
			return err
		}
		for _, t := range list {
			tv := TaskView{
				ID: t.ID, ProjectID: t.ProjectID, ScopeID: t.ScopeID,
				PrincipalID: t.PrincipalID, CapabilityID: t.CapabilityID,
				Title: t.Title, State: t.State, Type: t.Type,
			}
			if t.Schedule != nil {
				tv.Scheduled = true
				nb := t.Schedule.NotBefore
				tv.NotBefore = &nb
			}
			c.tasks[t.ID] = tv
		}
	}
	if repos.Assessments != nil {
		list, err := repos.Assessments.List(ctx, 500, 0)
		if err != nil {
			return err
		}
		for _, a := range list {
			steps := 0
			var arr []any
			if json.Unmarshal(a.StepsJSON, &arr) == nil {
				steps = len(arr)
			}
			c.assessments[a.ID] = AssessmentView{
				ID: a.ID, ProjectID: a.ProjectID, ScopeID: a.ScopeID,
				PrincipalID: a.PrincipalID, Title: a.Title, Target: a.Target,
				State: a.State, StepCount: steps,
			}
		}
	}
	return nil
}
