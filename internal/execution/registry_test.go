package execution

import (
	"testing"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/execution/mock"
	"cyberos/internal/execution/runtime"
)

func TestRegistryRegisterAndRouter(t *testing.T) {
	reg := NewRegistry()
	ex, err := runtime.NewGo("go-runtime", []contracts.CapabilityID{"cap.1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(ex); err != nil {
		t.Fatal(err)
	}
	// also register mock as default alternative
	type mockTE struct{ *mock.Executor }
	// use runtime only
	rt := &Router{Reg: reg}
	now := time.Now().UTC()
	res, err := rt.Execute(contracts.ExecutionRequest{
		Grant: contracts.ExecutionGrant{
			ID: "g", Principal: contracts.PrincipalRef{ID: "u", Type: contracts.PrincipalTypeHuman},
			ProjectID: "p", ScopeID: "s", TaskID: "t", CapabilityID: "cap.1",
			Status:   contracts.ExecutionGrantStatusIssued,
			IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute),
		},
		Parameters: map[string]any{"hello": "world"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Output["runtime"] != "go" {
		t.Fatalf("%v", res.Output)
	}
}

func TestProcessDisabledByDefault(t *testing.T) {
	ex, _ := runtime.NewPython("py", nil, nil)
	now := time.Now().UTC()
	res, err := ex.Execute(contracts.ExecutionRequest{
		Grant: contracts.ExecutionGrant{
			ID: "g", Principal: contracts.PrincipalRef{ID: "u", Type: contracts.PrincipalTypeHuman},
			ProjectID: "p", ScopeID: "s", TaskID: "t", CapabilityID: "c",
			Status:   contracts.ExecutionGrantStatusIssued,
			IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute),
		},
		Parameters: map[string]any{
			"process": map[string]any{"path": "/bin/true", "args": []any{}},
		},
	})
	if err == nil {
		t.Fatal("process must be disabled")
	}
	if res.Status != contracts.ExecutionStatusRejected && res.Status != contracts.ExecutionStatusFailed {
		t.Fatalf("status=%s", res.Status)
	}
}
