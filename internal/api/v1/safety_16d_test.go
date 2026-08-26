package v1_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"context"
	"testing"
	"time"

	"cyberos/internal/ai"
	v1 "cyberos/internal/api/v1"
	"cyberos/internal/ai/autonomy"
	"cyberos/internal/sdk"
)

func Test16D_RateLimitAI(t *testing.T) {
	client := sdk.NewInMemory()
	srv := v1.NewServer(client)
	srv.Limiter = v1.NewSafetyLimiter(v1.RateLimitConfig{AILimit: 2, Window: time.Minute, MaxBodyBytes: 1 << 20})
	body, _ := json.Marshal(ai.BrainRequest{ProjectID: "proj-a", Objective: "x"})
	var limited int
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/analyze", bytes.NewReader(body))
		rr := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rr, req)
		if rr.Code == http.StatusTooManyRequests {
			limited++
			if rr.Header().Get("Retry-After") == "" {
				t.Fatal("missing Retry-After")
			}
		}
	}
	if limited < 1 {
		t.Fatal("expected rate limit")
	}
}

func Test16D_OversizedBody(t *testing.T) {
	client := sdk.NewInMemory()
	srv := v1.NewServer(client)
	srv.Limiter = v1.NewSafetyLimiter(v1.RateLimitConfig{MaxBodyBytes: 64})
	big := bytes.Repeat([]byte("x"), 200)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/analyze", bytes.NewReader(big))
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusRequestEntityTooLarge {
		// MaxBytesReader causes decode error → 400
		if rr.Code == http.StatusOK {
			t.Fatal("oversized accepted")
		}
	}
}

func Test16D_ConcurrentRateLimit(t *testing.T) {
	client := sdk.NewInMemory()
	srv := v1.NewServer(client)
	srv.Limiter = v1.NewSafetyLimiter(v1.RateLimitConfig{AILimit: 3, Window: time.Minute})
	body, _ := json.Marshal(ai.BrainRequest{ProjectID: "p", Objective: "x"})
	var mu sync.Mutex
	okCount := 0
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/plan", bytes.NewReader(body))
			rr := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rr, req)
			if rr.Code != http.StatusTooManyRequests {
				mu.Lock()
				okCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if okCount > 3 {
		t.Fatalf("accepted %d > limit 3", okCount)
	}
}

func Test16D_AssistedApprovalRequired(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	run, err := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "ar-appr", Goal: "g", ProjectID: "proj-a", ScopeID: "s",
		Policy: autonomy.AutonomyPolicy{
			Version: autonomy.PolicyContractV1, Mode: autonomy.ModeAssisted, ProjectID: "proj-a", MaxSteps: 5,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.ApprovalState != autonomy.ApprovalPending {
		t.Fatal(run.ApprovalState)
	}
	adv, _ := ctrl.Advance(context.Background(), run.ID)
	if adv != nil && adv.MayAdvance {
		t.Fatal("ASSISTED without approval must not advance")
	}
	_, err = ctrl.Approve(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Approve ≠ Grant — still may not MayAdvance if MANUAL-like wait for ASSISTED mode in policy
	adv, _ = ctrl.Advance(context.Background(), run.ID)
	if adv != nil && adv.MayAdvance {
		t.Fatal("ASSISTED remains WAIT even after approval (policy)")
	}
	_, err = ctrl.Cancel(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ctrl.Approve(run.ID)
	if err == nil {
		t.Fatal("approve after cancel must fail")
	}
}

