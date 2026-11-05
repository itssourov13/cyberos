package sdk

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cyberos/internal/contracts"
	domainproject "cyberos/internal/domain/project"
	"cyberos/internal/storage/ports"
)

type pagedProjectRepo struct {
	items []*domainproject.Project
}

func (r *pagedProjectRepo) Create(context.Context, *domainproject.Project) error { return nil }
func (r *pagedProjectRepo) GetByID(_ context.Context, id contracts.ProjectID) (*domainproject.Project, error) {
	for _, p := range r.items {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, contracts.ErrNotFound
}
func (r *pagedProjectRepo) Update(context.Context, *domainproject.Project) error { return nil }
func (r *pagedProjectRepo) List(_ context.Context, limit, offset int) ([]*domainproject.Project, error) {
	if offset >= len(r.items) {
		return []*domainproject.Project{}, nil
	}
	end := min(offset+limit, len(r.items))
	return r.items[offset:end], nil
}

var _ ports.ProjectRepository = (*pagedProjectRepo)(nil)

func TestRefreshFromReposLoadsEveryProjectAndRetainsTenantOwnership(t *testing.T) {
	const total = catalogPageSize + 17
	now := time.Now().UTC()
	repo := &pagedProjectRepo{items: make([]*domainproject.Project, 0, total)}
	for i := 0; i < total; i++ {
		repo.items = append(repo.items, &domainproject.Project{
			ID:       contracts.ProjectID(fmt.Sprintf("project-%04d", i)),
			TenantID: "tenant-a", Name: fmt.Sprintf("Project %d", i),
			Status: domainproject.StatusActive, CreatedAt: now,
		})
	}
	catalog := NewCatalog()
	if err := catalog.RefreshFromRepos(context.Background(), &ports.Repositories{Projects: repo}); err != nil {
		t.Fatal(err)
	}
	if got := len(catalog.ListProjects()); got != total {
		t.Fatalf("loaded %d projects, want %d", got, total)
	}
	if got := len(catalog.ListProjectsForTenant("tenant-a")); got != total {
		t.Fatalf("tenant ownership lost during catalog refresh: got %d projects, want %d", got, total)
	}
}
