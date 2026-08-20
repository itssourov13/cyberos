package sdk

import (
	"context"
	"testing"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/project"
	"cyberos/internal/domain/scope"
	"cyberos/internal/domain/task"
	"cyberos/internal/storage/memory"
	"cyberos/internal/storage/ports"
)

func TestCatalogRefreshFromMemoryRepos(t *testing.T) {
	repos := memory.NewRepos()
	ctx := context.Background()
	p := &project.Project{ID: "p1", Name: "Project One", Status: project.StatusActive, CreatedAt: time.Now().UTC()}
	_ = repos.Projects.Create(ctx, p)
	sc, _ := scope.New("s1", "p1", "Scope", contracts.ScopeStatusActive)
	_ = repos.Scopes.Create(ctx, sc)
	tk, _ := task.New("t1", "p1", "s1", "u1", "c1", contracts.TaskTypeCustom, "Task")
	_ = repos.Tasks.Create(ctx, tk)
	_ = repos.Assessments.Create(ctx, ports.AssessmentRecord{
		ID: "a1", ProjectID: "p1", ScopeID: "s1", PrincipalID: "u1",
		Title: "A", State: "CREATED", StepsJSON: []byte("[]"), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	})

	cat := NewCatalog()
	if err := cat.RefreshFromRepos(ctx, repos.AsPorts()); err != nil {
		t.Fatal(err)
	}
	if len(cat.ListProjects()) != 1 {
		t.Fatalf("projects=%d", len(cat.ListProjects()))
	}
	if len(cat.ListScopes("")) != 1 {
		t.Fatalf("scopes=%d", len(cat.ListScopes("")))
	}
	if len(cat.ListTasks("", "")) != 1 {
		t.Fatalf("tasks=%d", len(cat.ListTasks("", "")))
	}
	if len(cat.ListAssessments()) != 1 {
		t.Fatalf("assessments=%d", len(cat.ListAssessments()))
	}
}

func TestNewInMemoryHasEmptyCatalog(t *testing.T) {
	c := NewInMemory()
	if len(c.Catalog.ListProjects()) != 0 {
		t.Fatal("demo seed must not run by default")
	}
}
