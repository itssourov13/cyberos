package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appscope "cyberos/internal/application/scope"
	"cyberos/internal/contracts"
	"cyberos/internal/controlplane"
	"cyberos/internal/domain/principal"
	domainscope "cyberos/internal/domain/scope"
	"cyberos/internal/identity"
	"cyberos/internal/sdk"
)

type memP struct {
	items map[contracts.PrincipalID]*principal.Principal
}

func (m *memP) Create(_ context.Context, p *principal.Principal) error {
	if m.items == nil {
		m.items = map[contracts.PrincipalID]*principal.Principal{}
	}
	cp := *p
	m.items[p.ID] = &cp
	return nil
}
func (m *memP) GetByID(_ context.Context, id contracts.PrincipalID) (*principal.Principal, error) {
	p, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *p
	return &cp, nil
}
func (m *memP) List(context.Context, int, int) ([]*principal.Principal, error) { return nil, nil }

type memS struct {
	items map[contracts.ScopeID]*domainscope.Scope
}

func (m *memS) Create(_ context.Context, s *domainscope.Scope) error {
	if m.items == nil {
		m.items = map[contracts.ScopeID]*domainscope.Scope{}
	}
	cp := *s
	m.items[s.ID] = &cp
	return nil
}
func (m *memS) GetByID(_ context.Context, id contracts.ScopeID) (*domainscope.Scope, error) {
	s, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *s
	return &cp, nil
}
func (m *memS) Update(context.Context, *domainscope.Scope) error { return nil }
func (m *memS) ListByProject(context.Context, contracts.ProjectID, int, int) ([]*domainscope.Scope, error) {
	return nil, nil
}
func (m *memS) ListAll(context.Context, int, int) ([]*domainscope.Scope, error) {
	return nil, nil
}

func testClient(t *testing.T) *sdk.Client {
	t.Helper()
	pr := &memP{}
	sr := &memS{}
	p, _ := principal.NewHuman("user-1", "alice")
	_ = pr.Create(context.Background(), p)
	sc, _ := domainscope.New("scope-1", "proj-1", "Main", contracts.ScopeStatusActive)
	_ = sr.Create(context.Background(), sc)
	idSvc := identity.NewService(pr)
	scSvc := appscope.NewService(sr)
	cp := controlplane.NewService(idSvc, scSvc)
	client := sdk.NewInMemory().WithIdentity(idSvc).WithScopes(scSvc).WithControlPlane(cp)
	_ = client.RegisterCapability(context.Background(), "cap-1", "Cap", contracts.CapabilityTypeUtility)
	// The canonical catalog is what the API validates scope/project refs against.
	client.Catalog.UpsertProject(sdk.ProjectView{ID: "proj-1", Name: "P1"})
	client.Catalog.UpsertScope(sdk.ScopeView{ID: "scope-1", ProjectID: "proj-1", Name: "Main", Status: contracts.ScopeStatusActive})
	return client
}

func TestHealthAndVersion(t *testing.T) {
	srv := NewServer(testClient(t))
	h := srv.Handler()

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("health: %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/version", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("version: %d", rr.Code)
	}
}

func TestEvaluateAuthorizedAndDenied(t *testing.T) {
	srv := NewServer(testClient(t))
	h := srv.Handler()

	body, _ := json.Marshal(map[string]string{
		"principal_id": "user-1", "project_id": "proj-1", "scope_id": "scope-1",
		"capability_id": "cap-1", "correlation_id": "test-1",
	})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/evaluate", bytes.NewReader(body)))
	if rr.Code != http.StatusOK {
		t.Fatalf("evaluate: %d %s", rr.Code, rr.Body.String())
	}
	var d contracts.Decision
	_ = json.Unmarshal(rr.Body.Bytes(), &d)
	if !d.Allowed() {
		t.Fatalf("expected allowed: %s", d.Reason)
	}

	// Missing capability → denied still 200 with decision body
	body2, _ := json.Marshal(map[string]string{
		"principal_id": "user-1", "project_id": "proj-1", "scope_id": "scope-1",
		"correlation_id": "test-2",
	})
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/evaluate", bytes.NewReader(body2)))
	_ = json.Unmarshal(rr.Body.Bytes(), &d)
	if d.Allowed() {
		t.Fatal("expected denied without capability")
	}
}

func TestGrantRequiresAuthorization(t *testing.T) {
	srv := NewServer(testClient(t))
	h := srv.Handler()
	body, _ := json.Marshal(map[string]string{
		"principal_id": "user-1", "project_id": "proj-1", "scope_id": "scope-1",
		"correlation_id": "g-1",
	})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/grants", bytes.NewReader(body)))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without capability, got %d", rr.Code)
	}
}

func TestCatalogAndDashboard(t *testing.T) {
	client := sdk.NewInMemory()
	client.Catalog.UpsertProject(sdk.ProjectView{ID: "proj-1", Name: "Demo"})
	client.Catalog.UpsertTask(sdk.TaskView{ID: "task-1", ProjectID: "proj-1", Title: "T", State: contracts.TaskStatePending})
	srv := NewServer(client)
	h := srv.Handler()

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("dashboard: %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("projects: %d %s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("tasks: %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/tasks/task-1", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("task by id: %d %s", rr.Code, rr.Body.String())
	}
}

