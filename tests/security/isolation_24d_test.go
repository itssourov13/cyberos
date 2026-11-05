package security_test

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	v1 "cyberos/internal/api/v1"
	"cyberos/internal/auth"
	"cyberos/internal/contracts"
	"cyberos/internal/sdk"
)

type tenantEnv struct {
	h      http.Handler
	keyA   string // tenant-a
	keyB   string // tenant-b
	keyOp  string // operator (no tenant)
	client *sdk.Client
}

func newTenantEnv(t *testing.T) *tenantEnv {
	t.Helper()
	client := sdk.NewInMemory()
	client.Catalog.UpsertProject(sdk.ProjectView{ID: "proj-a", Name: "A", TenantID: "tenant-a"})
	client.Catalog.UpsertProject(sdk.ProjectView{ID: "proj-b", Name: "B", TenantID: "tenant-b"})
	client.Catalog.UpsertProject(sdk.ProjectView{ID: "proj-orphan", Name: "O"})
	client.Catalog.UpsertScope(sdk.ScopeView{ID: "scope-b", ProjectID: "proj-b", Name: "SB"})
	client.Catalog.UpsertTask(sdk.TaskView{ID: "task-b", ProjectID: "proj-b", ScopeID: "scope-b", Title: "tb"})

	a := auth.NewAPIKeyAuthenticator(auth.NewMemoryKeyStore())
	ka, _, _ := a.Issue(context.Background(), "user-a", "tenant-a", time.Hour)
	kb, _, _ := a.Issue(context.Background(), "user-b", "tenant-b", time.Hour)
	ko, _, _ := a.Issue(context.Background(), "ops", "", time.Hour)
	srv := v1.NewServer(client)
	srv.Auth = auth.NewMiddleware(a)
	return &tenantEnv{h: srv.Handler(), keyA: ka, keyB: kb, keyOp: ko, client: client}
}

func (e *tenantEnv) get(key, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+key)
	rr := httptest.NewRecorder()
	e.h.ServeHTTP(rr, req)
	return rr
}

func TestIsolation_TenantCannotDeclareForeignProject(t *testing.T) {
	e := newTenantEnv(t)
	for _, path := range []string{
		"/api/v1/scopes?project_id=proj-b",
		"/api/v1/tasks?project_id=proj-b",
		"/api/v1/tasks/task-b?project_id=proj-b",
		"/api/v1/findings?project_id=proj-b",
	} {
		if rr := e.get(e.keyA, path); rr.Code != http.StatusForbidden {
			t.Fatalf("%s: tenant-a reading tenant-b project must be 403, got %d", path, rr.Code)
		}
	}
	// Unassigned/unknown projects are denied for tenant callers (fail closed).
	for _, p := range []string{"proj-orphan", "proj-nope"} {
		if rr := e.get(e.keyA, "/api/v1/scopes?project_id="+p); rr.Code != http.StatusForbidden {
			t.Fatalf("project %s must be denied, got %d", p, rr.Code)
		}
	}
}

func TestIsolation_TaskByIDRequiresMatchingProject(t *testing.T) {
	e := newTenantEnv(t)
	// tenant-a declares its own project but asks for tenant-b's task id.
	if rr := e.get(e.keyA, "/api/v1/tasks/task-b?project_id=proj-a"); rr.Code != http.StatusForbidden {
		t.Fatalf("ID-guessing across projects must be 403, got %d", rr.Code)
	}
	if rr := e.get(e.keyB, "/api/v1/tasks/task-b?project_id=proj-b"); rr.Code != http.StatusOK {
		t.Fatalf("owner read: %d", rr.Code)
	}
}

func TestIsolation_ProjectListAndDashboardAreTenantScoped(t *testing.T) {
	e := newTenantEnv(t)
	rr := e.get(e.keyA, "/api/v1/projects")
	if rr.Code != http.StatusOK || strings.Contains(rr.Body.String(), "proj-b") || strings.Contains(rr.Body.String(), "proj-orphan") {
		t.Fatalf("tenant-a project list leaked: %d %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "proj-a") {
		t.Fatalf("own project missing: %s", rr.Body.String())
	}
	d := e.get(e.keyA, "/api/v1/dashboard")
	if !strings.Contains(d.Body.String(), `"projects":1`) || !strings.Contains(d.Body.String(), `"tasks":0`) {
		t.Fatalf("dashboard counts must be tenant-scoped: %s", d.Body.String())
	}
	if op := e.get(e.keyOp, "/api/v1/projects"); !strings.Contains(op.Body.String(), "proj-b") {
		t.Fatalf("operator sees all projects: %s", op.Body.String())
	}
}

func TestIsolation_CapabilityRegistrationIsOperatorOnly(t *testing.T) {
	e := newTenantEnv(t)
	post := func(key string) int {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/capabilities",
			strings.NewReader(`{"id":"x.cap","name":"X","type":"UTILITY"}`))
		req.Header.Set("Authorization", "Bearer "+key)
		rr := httptest.NewRecorder()
		e.h.ServeHTTP(rr, req)
		return rr.Code
	}
	if c := post(e.keyA); c != http.StatusForbidden {
		t.Fatalf("tenant caller must not register capabilities: %d", c)
	}
	if c := post(e.keyOp); c == http.StatusForbidden || c == http.StatusUnauthorized {
		t.Fatalf("operator should be allowed past authorization: %d", c)
	}
}

func TestIsolation_SSEFiltersForeignTenantEvents(t *testing.T) {
	e := newTenantEnv(t)
	// Hub must be attached for the stream to exist.
	vsrv := v1.NewServer(e.client).WithEventHub(e.client.Hub)
	a := auth.NewAPIKeyAuthenticator(auth.NewMemoryKeyStore())
	ka, _, _ := a.Issue(context.Background(), "user-a", "tenant-a", time.Hour)
	vsrv.Auth = auth.NewMiddleware(a)
	hs := httptest.NewServer(vsrv.Handler())
	defer hs.Close()

	req, _ := http.NewRequest(http.MethodGet, hs.URL+"/api/v1/events/stream?project_id=proj-a", nil)
	req.Header.Set("Authorization", "Bearer "+ka)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stream status %d", resp.StatusCode)
	}
	rd := bufio.NewReader(resp.Body)
	_, _ = rd.ReadString('\n') // ": connected"

	mk := func(id, project string) contracts.EventEnvelope {
		return contracts.EventEnvelope{ID: contracts.EventID(id), Type: "test.event", SchemaVersion: "1",
			Timestamp: time.Now().UTC(), ProjectID: contracts.ProjectID(project)}
	}
	e.client.Hub.Publish(mk("ev-b", "proj-b"))
	e.client.Hub.Publish(mk("ev-sys", ""))
	e.client.Hub.Publish(mk("ev-a", "proj-a"))

	var got strings.Builder
	for {
		line, err := rd.ReadString('\n')
		got.WriteString(line)
		if strings.Contains(line, "ev-a") || err != nil {
			break
		}
	}
	out := got.String()
	if strings.Contains(out, "ev-b") || strings.Contains(out, "ev-sys") {
		t.Fatalf("SSE leaked foreign/system events to tenant-a: %s", out)
	}
	if !strings.Contains(out, "ev-a") {
		t.Fatalf("own event not delivered: %s", out)
	}
}
