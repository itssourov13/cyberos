package provider_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cyberos/internal/ai"
	"cyberos/internal/ai/provider"
	"cyberos/internal/secrets"
)

func TestOpenAICompatPlanValid(t *testing.T) {
	planJSON := `{
		"id":"p1","contract_version":"1","goal":"scan","project_id":"proj","scope_id":"scope",
		"steps":[{"id":"s1","capability_id":"recon.network.portscan","parameters":{"target":"example.com"}}]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("auth")
		}
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "test-key") {
			t.Error("key in body")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": planJSON}}},
		})
	}))
	defer srv.Close()

	cfg := provider.Config{
		ID: "oa", Type: provider.TypeOpenAICompatible, BaseURL: srv.URL, Model: "test-model",
		APIKeyRef: "env:TEST_KEY", Timeout: 5 * time.Second,
	}
	b, err := provider.NewOpenAICompat(cfg, secrets.NewMapResolver(map[string]string{"env:TEST_KEY": "test-key"}))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := b.Plan(context.Background(), ai.BrainRequest{
		Operation: ai.OpPlan, Objective: "scan", ProjectID: "proj", ScopeID: "scope",
	})
	if err != nil || resp.Plan == nil {
		t.Fatal(err, resp)
	}
	if resp.Plan.Steps[0].CapabilityID != "recon.network.portscan" {
		t.Fatal(resp.Plan)
	}
}

func TestMalformedPlanRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": `{"id":"p","goal":"x","steps":[{"id":"s","capability_id":"nmap"}]}`}}},
		})
	}))
	defer srv.Close()
	b, _ := provider.NewOpenAICompat(provider.Config{
		ID: "oa", Type: provider.TypeOpenAICompatible, BaseURL: srv.URL, Model: "m",
	}, nil)
	_, err := b.Plan(context.Background(), ai.BrainRequest{Operation: ai.OpPlan, Objective: "x", ProjectID: "p", ScopeID: "s"})
	if err == nil {
		t.Fatal("nmap capability must fail validation")
	}
}

func TestResponseTooLarge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(make([]byte, 2000))
	}))
	defer srv.Close()
	b, _ := provider.NewOpenAICompat(provider.Config{
		ID: "oa", Type: provider.TypeOpenAICompatible, BaseURL: srv.URL, Model: "m", MaxResponseBytes: 100,
	}, nil)
	_, err := b.Analyze(context.Background(), ai.BrainRequest{Operation: ai.OpAnalyze, Objective: "x"})
	if err == nil {
		t.Fatal("size")
	}
}

func TestRegistryMock(t *testing.T) {
	reg := &provider.Registry{}
	b, err := reg.New(provider.Config{ID: "m", Type: provider.TypeMock})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := b.Plan(context.Background(), ai.BrainRequest{Operation: ai.OpPlan, Objective: "o", ProjectID: "p", ScopeID: "s"})
	if err != nil || resp.Plan == nil {
		t.Fatal(err, resp)
	}
}

func TestUnknownProvider(t *testing.T) {
	reg := &provider.Registry{}
	_, err := reg.New(provider.Config{ID: "x", Type: "unknown"})
	if err == nil {
		t.Fatal("unknown")
	}
}

func TestConfigRequiresBaseURL(t *testing.T) {
	err := provider.Config{ID: "x", Type: provider.TypeOpenAICompatible, Model: "m"}.Validate()
	if err == nil {
		t.Fatal("base_url")
	}
}

func TestProviderMetaNoSecret(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"content":"{\"id\":\"p1\",\"contract_version\":\"1\",\"goal\":\"g\",\"project_id\":\"proj\",\"scope_id\":\"s\",\"steps\":[{\"id\":\"1\",\"capability_id\":\"recon.network.portscan\"}]}"}}]}`))
	}))
	defer srv.Close()
	b, err := provider.NewOpenAICompat(provider.Config{
		ID: "oa", Type: provider.TypeOpenAICompatible, BaseURL: srv.URL, Model: "m",
		APIKeyRef: "env:TEST_KEY", Timeout: 5 * time.Second,
	}, secrets.NewMapResolver(map[string]string{"env:TEST_KEY": "sk-super-secret-key"}))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := b.Plan(context.Background(), ai.BrainRequest{
		Operation: ai.OpPlan, ProjectID: "proj", ScopeID: "s", Objective: "scan",
	})
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range resp.ProviderMeta {
		if strings.Contains(v, "sk-super") || strings.Contains(k, "key") {
			t.Fatal(resp.ProviderMeta)
		}
	}
	if resp.ProviderMeta["provider_type"] == "" {
		t.Fatal("expected provider_type in meta")
	}
}

func TestRedirectStripsAuthorization(t *testing.T) {
	var secondAuth string
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secondAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"choices":[{"message":{"content":"{}"}}]}`))
	}))
	defer final.Close()
	redir := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL+"/chat/completions", http.StatusFound)
	}))
	defer redir.Close()
	b, err := provider.NewOpenAICompat(provider.Config{
		ID: "oa", Type: provider.TypeOpenAICompatible, BaseURL: redir.URL, Model: "m",
		APIKeyRef: "env:K", Timeout: 5 * time.Second,
	}, secrets.NewMapResolver(map[string]string{"env:K": "secret-token"}))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = b.Analyze(context.Background(), ai.BrainRequest{
		Operation: ai.OpAnalyze, ProjectID: "p", Objective: "x",
	})
	if secondAuth != "" {
		t.Fatalf("Authorization leaked across host redirect: %q", secondAuth)
	}
}
