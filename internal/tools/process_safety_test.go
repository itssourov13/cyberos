package tools

import (
	"context"
	"testing"
	"time"

	"cyberos/internal/broker"
	"cyberos/internal/contracts"
	"cyberos/internal/execution/mock"
	"cyberos/internal/execution/process"
	"cyberos/internal/execution/sandbox"
)

func TestDefaultBuildArgsRejectsShellFlags(t *testing.T) {
	_, err := defaultBuildArgs("cap", map[string]any{"args": []string{"-c", "id"}})
	if err == nil {
		t.Fatal("expected reject -c")
	}
	_, err = defaultBuildArgs("cap", map[string]any{"args": []string{"a&&b"}})
	if err == nil {
		t.Fatal("expected compound reject")
	}
	args, err := defaultBuildArgs("cap", map[string]any{"args": []string{"-v", "target.com"}})
	if err != nil || len(args) != 2 {
		t.Fatal(err, args)
	}
}

func TestProcessDisabledByDefault_Adapter(t *testing.T) {
	a, err := NewProcessToolAdapter(Descriptor{
		ID: "tool.t", Name: "t", Version: "1.0.0", Runtime: contracts.PluginRuntimeGo,
		ExecutorID: "tool.t", Binary: "/bin/true", Capabilities: []contracts.CapabilityID{"cap.t"},
	}, process.NewRunner(process.DefaultPolicy()))
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.Run(context.Background(), "cap.t", map[string]any{})
	if err == nil {
		t.Fatal("disabled policy must block")
	}
}

func TestAllowlistedBrokerStillRequiresGrant(t *testing.T) {
	brk := broker.New(sandbox.New(mock.New()))
	if _, err := brk.Execute(contracts.ExecutionRequest{}); err == nil {
		t.Fatal("empty grant must fail")
	}
	now := time.Now().UTC()
	res, err := brk.Execute(contracts.ExecutionRequest{Grant: contracts.ExecutionGrant{
		ID: "g1", TaskID: "t1", ProjectID: "p1", ScopeID: "s1", CapabilityID: "cap.mock",
		Principal: contracts.PrincipalRef{ID: "u1", Type: contracts.PrincipalTypeHuman},
		Status:    contracts.ExecutionGrantStatusIssued, IssuedAt: now, ExpiresAt: now.Add(time.Hour),
		Nonce: "n-9a-1",
	}})
	if err != nil {
		t.Fatal(err)
	}
	_ = res
}

func TestNonAllowlistedDenied(t *testing.T) {
	r := process.NewRunner(process.Policy{Enabled: true, AllowedBins: []string{"/bin/true"}})
	_, err := r.RunDetailed(context.Background(), process.Spec{Path: "/bin/false"})
	if err == nil {
		t.Fatal("non-allowlisted must deny")
	}
}

func TestShellDeniedEvenIfAllowlisted(t *testing.T) {
	r := process.NewRunner(process.Policy{
		Enabled: true, AllowedBins: []string{"bash", "/bin/bash"}, AllowShell: false,
	})
	_, err := r.RunDetailed(context.Background(), process.Spec{Path: "/bin/bash", Args: []string{"-c", "echo x"}})
	if err == nil {
		t.Fatal("shell must be denied")
	}
}
