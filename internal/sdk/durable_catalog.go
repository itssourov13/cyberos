package sdk

import (
	"context"
	"encoding/json"

	"cyberos/internal/contracts"
	"cyberos/internal/storage/ports"
)

const catalogPageSize = 500

// RefreshFromRepos reloads catalog views from durable repositories.
// Persistence is not authorization. Every repository is paged to completion:
// a startup snapshot truncated at an arbitrary row count is not a safe source
// for project-ownership enforcement.
func (c *Catalog) RefreshFromRepos(ctx context.Context, repos *ports.Repositories) error {
	if c == nil || repos == nil {
		return contracts.ErrInvalidArgument
	}
	projects := make(map[contracts.ProjectID]ProjectView)
	scopes := make(map[contracts.ScopeID]ScopeView)
	tasks := make(map[contracts.TaskID]TaskView)
	assessments := make(map[string]AssessmentView)

	if repos.Projects != nil {
		for offset := 0; ; offset += catalogPageSize {
			list, err := repos.Projects.List(ctx, catalogPageSize, offset)
			if err != nil {
				return err
			}
			for _, p := range list {
				projects[p.ID] = ProjectView{
					ID: p.ID, Name: p.Name, Description: p.Description,
					CreatedAt: p.CreatedAt, TenantID: p.TenantID,
				}
			}
			if len(list) < catalogPageSize {
				break
			}
		}
	}
	if repos.Scopes != nil {
		for offset := 0; ; offset += catalogPageSize {
			list, err := repos.Scopes.ListAll(ctx, catalogPageSize, offset)
			if err != nil {
				return err
			}
			for _, sc := range list {
				scopes[sc.ID] = ScopeView{
					ID: sc.ID, ProjectID: sc.ProjectID, Name: sc.Name, Status: sc.Status,
				}
			}
			if len(list) < catalogPageSize {
				break
			}
		}
	}
	if repos.Tasks != nil {
		for offset := 0; ; offset += catalogPageSize {
			list, err := repos.Tasks.List(ctx, catalogPageSize, offset)
			if err != nil {
				return err
			}
			for _, task := range list {
				tv := TaskView{
					ID: task.ID, ProjectID: task.ProjectID, ScopeID: task.ScopeID,
					PrincipalID: task.PrincipalID, CapabilityID: task.CapabilityID,
					Title: task.Title, State: task.State, Type: task.Type,
				}
				if task.Schedule != nil {
					tv.Scheduled = true
					nb := task.Schedule.NotBefore
					tv.NotBefore = &nb
				}
				tasks[task.ID] = tv
			}
			if len(list) < catalogPageSize {
				break
			}
		}
	}
	if repos.Assessments != nil {
		for offset := 0; ; offset += catalogPageSize {
			list, err := repos.Assessments.List(ctx, catalogPageSize, offset)
			if err != nil {
				return err
			}
			for _, assessment := range list {
				steps := 0
				var arr []any
				if json.Unmarshal(assessment.StepsJSON, &arr) == nil {
					steps = len(arr)
				}
				assessments[assessment.ID] = AssessmentView{
					ID: assessment.ID, ProjectID: assessment.ProjectID, ScopeID: assessment.ScopeID,
					PrincipalID: assessment.PrincipalID, Title: assessment.Title, Target: assessment.Target,
					State: assessment.State, StepCount: steps,
				}
			}
			if len(list) < catalogPageSize {
				break
			}
		}
	}

	// Publish the new snapshot only after every repository read succeeds. An
	// incomplete load must never replace a previously valid ownership index.
	c.mu.Lock()
	c.projects = projects
	c.scopes = scopes
	c.tasks = tasks
	c.assessments = assessments
	c.mu.Unlock()
	return nil
}
