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
	"cyberos/internal/audit"
	"cyberos/internal/auth"
	"cyberos/internal/contracts"
	"cyberos/internal/sdk"
)

func TestAuditFeed_RoleIsolationAndRedaction(t *testing.T) {
	client := sdk.NewInMemory()
	client.Catalog.UpsertProject(sdk.ProjectView{ID: "proj-a", Name: "A", TenantID: "tenant-a"})
	client.Catalog.UpsertProject(sdk.ProjectView{ID: "proj-b", Name: "B", TenantID: "tenant-b"})
	svc := audit.NewService(nil)
	client.WithAudit(svc)

	actor := contracts.PrincipalRef{ID: "op-1", Type: contracts.PrincipalTypeHuman}
	secretMeta := map[string]string{"internal_note": "must-not-leak"}
	for _, e := range []struct{ proj, corr string }{
		{"proj-a", "a1"}, {"proj-a", "a2"}, {"proj-b", "b1"}, {"", "platform1"},
	} {
		if _, err := svc.RecordTrustMutation("PACK_TRUSTED", actor, contracts.ProjectID(e.proj), "sha256:x", "", "test", e.corr, copyMeta(secretMeta)); err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Millisecond)
	}

	a := auth.NewAPIKeyAuthenticator(auth.NewMemoryKeyStore())
	ctx := context.Background()
	viewerA, _, _ := a.IssueWithRole(ctx, "viewer-a", "tenant-a", auth.RoleViewer, time.Hour)
	analystA, _, _ := a.IssueWithRole(ctx, "analyst-a", "tenant-a", auth.RoleAnalyst, time.Hour)
	operator, _, _ := a.IssueWithRole(ctx, "root", "", auth.RoleOperator, time.Hour)

	srv := v1.NewServer(client)
	srv.Auth = auth.NewMiddleware(a)
	h := srv.Handler()

	get := func(path, key string) (int, []map[string]any, string) {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+key)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		var body struct {
			Entries []map[string]any `json:"entries"`
		}
		_ = json.Unmarshal(rr.Body.Bytes(), &body)
		return rr.Code, body.Entries, rr.Body.String()
	}

	// Unauthenticated: rejected by the middleware.
	if rr := httptest.NewRecorder(); func() bool {
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/audit?project_id=proj-a", nil))
		return rr.Code == http.StatusUnauthorized
	}() == false {
		t.Fatalf("unauthenticated audit read must be 401, got %d", rr.Code)
	}

	// A viewer lacks audit.read.
	if code, _, _ := get("/api/v1/audit?project_id=proj-a", viewerA); code != http.StatusForbidden {
		t.Fatalf("viewer must be 403, got %d", code)
	}

	// An analyst sees only its own project's entries, newest first, no metadata.
	code, entries, raw := get("/api/v1/audit?project_id=proj-a", analystA)
	if code != http.StatusOK || len(entries) != 2 {
		t.Fatalf("analyst want 2 entries, got %d (%d): %s", len(entries), code, raw)
	}
	if entries[0]["correlation_id"] != "a2" || entries[1]["correlation_id"] != "a1" {
		t.Fatalf("not newest-first: %v", entries)
	}
	if strings.Contains(raw, "must-not-leak") || strings.Contains(raw, "sha256:x") {
		t.Fatalf("metadata leaked into audit feed: %s", raw)
	}
	for _, e := range entries {
		if e["project_id"] != "proj-a" {
			t.Fatalf("foreign entry leaked: %v", e)
		}
	}

	// Another tenant's project and a missing project are both refused.
	if code, _, _ := get("/api/v1/audit?project_id=proj-b", analystA); code != http.StatusForbidden {
		t.Fatalf("foreign project must be 403, got %d", code)
	}
	if code, _, _ := get("/api/v1/audit", analystA); code != http.StatusForbidden {
		t.Fatalf("tenant caller without project must be 403, got %d", code)
	}

	// The operator may read everything, including platform-level entries.
	if code, entries, _ := get("/api/v1/audit", operator); code != http.StatusOK || len(entries) != 4 {
		t.Fatalf("operator want 4 entries, got %d (%d)", len(entries), code)
	}

	// limit is honoured and validated.
	if _, entries, _ := get("/api/v1/audit?limit=1", operator); len(entries) != 1 {
		t.Fatalf("limit=1 want 1 entry, got %d", len(entries))
	}
	if code, _, _ := get("/api/v1/audit?limit=abc", operator); code != http.StatusBadRequest {
		t.Fatalf("bad limit must be 400, got %d", code)
	}
}

func copyMeta(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
