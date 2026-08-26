package autonomy_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"cyberos/internal/ai"
	"cyberos/internal/ai/autonomy"
	"cyberos/internal/contracts"
	"cyberos/internal/storage/memory"
)

func Test16C_ReserveLLMCallBudget(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	ctrl.Repo = memory.NewAutonomyRunRepo()
	run, err := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "ar-llm", Goal: "g", ProjectID: "proj-a", ScopeID: "s",
		Policy: autonomy.AutonomyPolicy{
			Version: autonomy.PolicyContractV1, Mode: autonomy.ModeControlledAutonomous,
			ProjectID: "proj-a", MaxSteps: 10, MaxLLMCalls: 2, MaxRuntime: time.Hour,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ok, _, err := ctrl.ReserveLLMCall(run.ID)
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	ok, _, err = ctrl.ReserveLLMCall(run.ID)
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	ok, blocked, err := ctrl.ReserveLLMCall(run.ID)
	if err != nil || ok {
		t.Fatal("expected block", ok, err)
	}
	if blocked.State != autonomy.RunPolicyBlocked {
		t.Fatal(blocked.State)
	}
}

func Test16C_ConcurrentReserve(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	ctrl.Repo = memory.NewAutonomyRunRepo()
	_, err := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "ar-conc-llm", Goal: "g", ProjectID: "proj-a", ScopeID: "s",
		Policy: autonomy.AutonomyPolicy{
			Version: autonomy.PolicyContractV1, Mode: autonomy.ModeControlledAutonomous,
			ProjectID: "proj-a", MaxSteps: 20, MaxLLMCalls: 1, MaxRuntime: time.Hour,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	success := 0
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, _, _ := ctrl.ReserveLLMCall("ar-conc-llm")
			if ok {
				mu.Lock()
				success++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if success != 1 {
		t.Fatalf("expected exactly 1 success, got %d", success)
	}
}

func Test16C_CooldownAndRestart(t *testing.T) {
	repo := memory.NewAutonomyRunRepo()
	ctrl := autonomy.NewController(&ai.MockBrain{})
	ctrl.Repo = repo
	_, err := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "ar-cd", Goal: "g", ProjectID: "proj-a", ScopeID: "s",
		Policy: autonomy.AutonomyPolicy{
			Version: autonomy.PolicyContractV1, Mode: autonomy.ModeControlledAutonomous,
			ProjectID: "proj-a", MaxSteps: 10, MaxLLMCalls: 5, Cooldown: time.Hour, MaxRuntime: time.Hour,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ok, _, _ := ctrl.ReserveLLMCall("ar-cd")
	if !ok {
		t.Fatal("first must succeed")
	}
	ok, _, _ = ctrl.ReserveLLMCall("ar-cd")
	if ok {
		t.Fatal("cooldown must block")
	}
	ctrl2 := autonomy.NewController(&ai.MockBrain{})
	ctrl2.Repo = repo
	restored, err := ctrl2.LoadFromRepo(context.Background(), "proj-a", "ar-cd")
	if err != nil {
		t.Fatal(err)
	}
	if restored.LLMCallsUsed != 1 {
		t.Fatal(restored.LLMCallsUsed)
	}
	if restored.NextAllowedAt.IsZero() {
		t.Fatal("cooldown must persist")
	}
	ok, _, _ = ctrl2.ReserveLLMCall("ar-cd")
	if ok {
		t.Fatal("cooldown after restart must block")
	}
}

func Test16C_NoProgress(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	_, err := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "ar-np", Goal: "g", ProjectID: "proj-a", ScopeID: "s",
		Policy: autonomy.AutonomyPolicy{
			Version: autonomy.PolicyContractV1, Mode: autonomy.ModeControlledAutonomous,
			ProjectID: "proj-a", MaxSteps: 10, MaxLLMCalls: 20, MaxNoProgress: 2, MaxRuntime: time.Hour,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = ctrl.NoteProgress("ar-np", "plan-a")
	_ = ctrl.NoteProgress("ar-np", "plan-a") // same key → increment
	_ = ctrl.NoteProgress("ar-np", "plan-a")
	ok, blocked, _ := ctrl.ReserveLLMCall("ar-np")
	if ok {
		t.Fatal("no-progress must block")
	}
	if blocked.StopReason != "no_progress_exceeded" {
		t.Fatal(blocked.StopReason)
	}
	_ = contracts.ProjectID("proj-a")
}
