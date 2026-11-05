package security_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	v1 "cyberos/internal/api/v1"
	"cyberos/internal/auth"
	"cyberos/internal/contracts"
	"cyberos/internal/sdk"
)

func TestRoles_IssuanceRulesAndPersistenceOfRole(t *testing.T) {
	a := auth.NewAPIKeyAuthenticator(auth.NewMemoryKeyStore())
	ctx := context.Background()

	if _, _, err := a.IssueWithRole(ctx, "u", "tenant-a", auth.RoleOperator, time.Hour); !errors.Is(err, contracts.ErrInvalidArgument) {
		t.Fatalf("tenant key must not be issuable as operator: %v", err)
	}
	if _, _, err := a.IssueWithRole(ctx, "u", "", auth.RoleViewer, time.Hour); !errors.Is(err, contracts.ErrInvalidArgument) {
		t.Fatalf("tenant-less key must be operator only: %v", err)
	}
	if _, _, err := a.IssueWithRole(ctx, "u", "tenant-a", auth.Role("root"), time.Hour); !errors.Is(err, contracts.ErrInvalidArgument) {
		t.Fatalf("unknown role must be rejected: %v", err)
	}
	for _, role := range []auth.Role{auth.RoleViewer, auth.RoleAnalyst, auth.RoleAdmin} {
		plain, _, err := a.IssueWithRole(ctx, "u", "tenant-a", role, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		id, err := a.Authenticate(ctx, plain)
		if err != nil || id.Role != role {
			t.Fatalf("role %s not carried through authentication: %+v %v", role, id, err)
		}
	}
}

func TestRoles_ViewerCannotWriteAnalystCan(t *testing.T) {
	client := sdk.NewInMemory()
	client.Catalog.UpsertProject(sdk.ProjectView{ID: "proj-a", Name: "A", TenantID: "tenant-a"})
	a := auth.NewAPIKeyAuthenticator(auth.NewMemoryKeyStore())
	viewer, _, _ := a.IssueWithRole(context.Background(), "viewer-1", "tenant-a", auth.RoleViewer, time.Hour)
	analyst, _, _ := a.IssueWithRole(context.Background(), "analyst-1", "tenant-a", auth.RoleAnalyst, time.Hour)
	srv := v1.NewServer(client)
	srv.Auth = auth.NewMiddleware(a)
	h := srv.Handler()

	do := func(method, path, key, body string) int {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+key)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		return rr.Code
	}
	if c := do(http.MethodGet, "/api/v1/scopes?project_id=proj-a", viewer, ""); c != http.StatusOK {
		t.Fatalf("viewer read must succeed, got %d", c)
	}
	evalBody := func(p string) string {
		return `{"principal_id":"` + p + `","project_id":"proj-a","scope_id":"s","capability_id":"c"}`
	}
	if c := do(http.MethodPost, "/api/v1/evaluate", viewer, evalBody("viewer-1")); c != http.StatusForbidden {
		t.Fatalf("viewer evaluate must be 403, got %d", c)
	}
	if c := do(http.MethodPost, "/api/v1/grants", viewer, evalBody("viewer-1")); c != http.StatusForbidden {
		t.Fatalf("viewer grant must be 403, got %d", c)
	}
	// Analyst passes the permission gate (the unknown scope is then a 400, not a 403/401).
	if c := do(http.MethodPost, "/api/v1/evaluate", analyst, evalBody("analyst-1")); c == http.StatusForbidden || c == http.StatusUnauthorized {
		t.Fatalf("analyst must pass the permission gate, got %d", c)
	}
}
