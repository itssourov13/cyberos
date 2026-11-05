package security_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apiv1 "cyberos/internal/api/v1"
	"cyberos/internal/auth"
	"cyberos/internal/cases"
	"cyberos/internal/contracts"
	"cyberos/internal/findings"
	"cyberos/internal/sdk"
)

func TestAPI_CrossProjectFindingGetDenied(t *testing.T) {
	client := sdk.NewInMemory()
	client.Catalog.UpsertProject(sdk.ProjectView{ID: "proj-a", Name: "proj-a", TenantID: "tenant-b"})
	if _, err := client.Findings.Create(findings.CreateRequest{
		ID: "f-iso-1", TaskID: "t1", ProjectID: "proj-a", ScopeID: "scope-a",
		Severity: contracts.FindingSeverityHigh, Title: "secret",
	}); err != nil {
		t.Fatal(err)
	}

	store := auth.NewMemoryKeyStore()
	a := auth.NewAPIKeyAuthenticator(store)
	plain, _, err := a.Issue(context.Background(), "user-b", "tenant-b", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	srv := apiv1.NewServer(client)
	srv.Auth = auth.NewMiddleware(a)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/findings/f-iso-1?project_id=proj-b", nil)
	req.Header.Set("Authorization", "Bearer "+plain)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}

	reqOK := httptest.NewRequest(http.MethodGet, "/api/v1/findings/f-iso-1?project_id=proj-a", nil)
	reqOK.Header.Set("Authorization", "Bearer "+plain)
	rrOK := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rrOK, reqOK)
	if rrOK.Code != http.StatusOK {
		t.Fatalf("expected 200 with matching project, got %d body=%s", rrOK.Code, rrOK.Body.String())
	}
}

func TestAPI_CrossTenantCaseGetDenied(t *testing.T) {
	client := sdk.NewInMemory()
	if _, err := client.Cases.Create(cases.CreateRequest{
		ID: "case-iso", ProjectID: "proj-a", Title: "T", TenantID: "tenant-a",
	}); err != nil {
		t.Fatal(err)
	}
	store := auth.NewMemoryKeyStore()
	a := auth.NewAPIKeyAuthenticator(store)
	plain, _, err := a.Issue(context.Background(), "user-b", "tenant-b", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	srv := apiv1.NewServer(client)
	srv.Auth = auth.NewMiddleware(a)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases/case-iso?project_id=proj-a", nil)
	req.Header.Set("Authorization", "Bearer "+plain)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 cross-tenant, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAPI_ListRequiresProjectForTenant(t *testing.T) {
	client := sdk.NewInMemory()
	client.Catalog.UpsertProject(sdk.ProjectView{ID: "proj-a", Name: "proj-a", TenantID: "tenant-a"})
	store := auth.NewMemoryKeyStore()
	a := auth.NewAPIKeyAuthenticator(store)
	plain, _, err := a.Issue(context.Background(), "user-a", "tenant-a", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	srv := apiv1.NewServer(client)
	srv.Auth = auth.NewMiddleware(a)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/findings", nil)
	req.Header.Set("Authorization", "Bearer "+plain)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without project_id, got %d body=%s", rr.Code, rr.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/findings?project_id=proj-a", nil)
	req2.Header.Set("Authorization", "Bearer "+plain)
	rr2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200 with project_id, got %d", rr2.Code)
	}
}

func TestAPI_UnauthenticatedLegacyEmptyList(t *testing.T) {
	client := sdk.NewInMemory()
	srv := apiv1.NewServer(client)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/findings", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 legacy, got %d", rr.Code)
	}
}
