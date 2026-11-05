package security_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	v1 "cyberos/internal/api/v1"
	"cyberos/internal/auth"
	"cyberos/internal/execution"
	"cyberos/internal/sdk"
	"cyberos/internal/storage/memory"
)

func TestAssignmentsRead_TenantIsolationAndRedaction(t *testing.T) {
	client := sdk.NewInMemory()
	client.Catalog.UpsertProject(sdk.ProjectView{ID: "proj-a", Name: "A", TenantID: "tenant-a"})
	client.Catalog.UpsertProject(sdk.ProjectView{ID: "proj-b", Name: "B", TenantID: "tenant-b"})

	svc := execution.NewAssignmentService(memory.NewAssignmentRepo())
	ctx := context.Background()
	for _, a := range []execution.ExecutionAssignment{
		{ID: "asg-a1", ExecutionID: "ex-1", ProjectID: "proj-a", TenantID: "tenant-a", HostID: "host-1", WorkerID: "w1", GrantID: "grant-1"},
		{ID: "asg-a2", ExecutionID: "ex-2", ProjectID: "proj-a", TenantID: "tenant-a", HostID: "host-1", WorkerID: "w1", GrantID: "grant-2"},
		{ID: "asg-b1", ExecutionID: "ex-3", ProjectID: "proj-b", TenantID: "tenant-b", HostID: "host-2", WorkerID: "w2", GrantID: "grant-3"},
	} {
		if _, err := svc.Create(ctx, a); err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Millisecond)
	}

	a := auth.NewAPIKeyAuthenticator(auth.NewMemoryKeyStore())
	viewerA, _, _ := a.IssueWithRole(ctx, "viewer-a", "tenant-a", auth.RoleViewer, time.Hour)
	operator, _, _ := a.IssueWithRole(ctx, "root", "", auth.RoleOperator, time.Hour)

	srv := v1.NewServer(client)
	srv.Auth = auth.NewMiddleware(a)
	srv.Assignments = svc
	h := srv.Handler()

	get := func(path, key string) (int, []map[string]any, string) {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if key != "" {
			req.Header.Set("Authorization", "Bearer "+key)
		}
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		var body struct {
			Assignments []map[string]any `json:"assignments"`
		}
		_ = json.Unmarshal(rr.Body.Bytes(), &body)
		return rr.Code, body.Assignments, rr.Body.String()
	}

	if code, _, _ := get("/api/v1/projects/proj-a/assignments", ""); code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated must be 401, got %d", code)
	}
	// A viewer holds read, so it can list its own project, newest first.
	code, list, raw := get("/api/v1/projects/proj-a/assignments", viewerA)
	if code != http.StatusOK || len(list) != 2 || list[0]["assignment_id"] != "asg-a2" {
		t.Fatalf("want 2 newest-first, got %d %s", code, raw)
	}
	if strings.Contains(strings.ToLower(raw), "secret") || strings.Contains(raw, "tenant-b") {
		t.Fatalf("unexpected content in feed: %s", raw)
	}
	// Foreign project is refused.
	if code, _, _ := get("/api/v1/projects/proj-b/assignments", viewerA); code != http.StatusForbidden {
		t.Fatalf("foreign project must be 403, got %d", code)
	}
	// Operator can read any project.
	if code, list, _ := get("/api/v1/projects/proj-b/assignments", operator); code != http.StatusOK || len(list) != 1 {
		t.Fatalf("operator want 1, got %d (%d)", len(list), code)
	}
	// Unconfigured service fails closed with 501, not an empty success.
	srv.Assignments = nil
	if code, _, _ := get("/api/v1/projects/proj-a/assignments", viewerA); code != http.StatusNotImplemented {
		t.Fatalf("nil service must be 501, got %d", code)
	}
}
