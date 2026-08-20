package security_test

import (
	"testing"
	"time"

	"cyberos/internal/broker"
	"cyberos/internal/contracts"
	"cyberos/internal/execution"
	"cyberos/internal/execution/runtime"
	"cyberos/internal/execution/sandbox"
)

// Router behind sandbox still requires valid grant via broker.
func TestSecurity_RuntimeBehindBrokerRequiresGrant(t *testing.T) {
	reg := execution.NewRegistry()
	ex, _ := runtime.NewRust("rust-runtime", nil, nil)
	_ = reg.Register(ex)
	brk := broker.New(sandbox.New(&execution.Router{Reg: reg}))
	_, err := brk.Execute(contracts.ExecutionRequest{})
	if err == nil {
		t.Fatal("empty grant must fail")
	}
}

func TestSecurity_NoUnrestrictedProcess(t *testing.T) {
	ex, _ := runtime.NewGo("go-runtime", nil, nil)
	now := time.Now().UTC()
	_, err := ex.Execute(contracts.ExecutionRequest{
		Grant: contracts.ExecutionGrant{
			ID: "g", Principal: contracts.PrincipalRef{ID: "u", Type: contracts.PrincipalTypeHuman},
			ProjectID: "p", ScopeID: "s", TaskID: "t", CapabilityID: "c",
			Status:   contracts.ExecutionGrantStatusIssued,
			IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute),
		},
		Parameters: map[string]any{"process": map[string]any{"path": "/bin/sh", "args": []any{"-c", "id"}}},
	})
	if err == nil {
		t.Fatal("shell must be blocked by default process policy")
	}
}
