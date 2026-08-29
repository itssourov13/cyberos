package broker

import (
	"context"
	"testing"
	"time"

	"cyberos/internal/application/scope"
	"cyberos/internal/contracts"
	"cyberos/internal/controlplane"
	"cyberos/internal/domain/principal"
	domainscope "cyberos/internal/domain/scope"
	"cyberos/internal/execution"
	"cyberos/internal/identity"
)

type countingHost struct {
	id    execution.HostID
	calls int
}

func (h *countingHost) ID() execution.HostID { return h.id }
func (h *countingHost) Execute(_ context.Context, req contracts.ExecutionRequest) (contracts.ExecutionResult, error) {
	h.calls++
	return contracts.ExecutionResult{Status: contracts.ExecutionStatusSucceeded, Output: map[string]any{"ok": true}}, nil
}

func dig(n byte) string {
	b := make([]byte, 64)
	for i := range b {
		b[i] = 'a' + (n % 6)
	}
	return "sha256:" + string(b)
}

func setupPackCP(t *testing.T) (*Issuer, *controlplane.Service) {
	t.Helper()
	pr := &memP{}
	sr := &memS{}
	p, _ := principal.NewHuman("user-1", "alice")
	_ = pr.Create(context.Background(), p)
	sc, _ := domainscope.New("scope-1", "proj-1", "Main", contracts.ScopeStatusActive)
	_ = sr.Create(context.Background(), sc)
	cp := controlplane.NewService(
		identity.NewService(pr),
		scope.NewService(sr),
		controlplane.WithPolicyEvaluator(controlplane.PackTrustPolicyEvaluator{Inner: controlplane.DefaultPolicyEvaluator{}}),
	)
	return NewIssuer(5 * time.Minute), cp
}

func TestPackBind_Match_Allows(t *testing.T) {
	issuer, cp := setupPackCP(t)
	dA := dig(0)
	g, d, err := issuer.IssueAfterEvaluate(context.Background(), cp, controlplane.EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1",
		CapabilityID: "recon.domain.enumerate", TaskID: "task-1",
		RequireTrustedPack: true, EffectivePackTrust: "TRUSTED", PackContentDigest: dA, PackKeyID: "k1",
	})
	if err != nil {
		t.Fatal(err, d.Reason)
	}
	if g.PackContentDigest != dA {
		t.Fatal(g.PackContentDigest)
	}
	host := &countingHost{id: "local"}
	reg := execution.NewHostRegistry()
	_ = reg.Register(host)
	brk := NewWithHosts(reg)
	_, err = brk.Execute(contracts.ExecutionRequest{Grant: g.ExecutionGrant, PackContentDigest: dA})
	if err != nil {
		t.Fatal(err)
	}
	if host.calls != 1 {
		t.Fatal(host.calls)
	}
}

func TestPackBind_Swap_Denies_NoHost(t *testing.T) {
	issuer, cp := setupPackCP(t)
	dA, dB := dig(1), dig(2)
	g, _, err := issuer.IssueAfterEvaluate(context.Background(), cp, controlplane.EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1",
		CapabilityID: "recon.domain.enumerate", TaskID: "task-1",
		RequireTrustedPack: true, EffectivePackTrust: "TRUSTED", PackContentDigest: dA,
	})
	if err != nil {
		t.Fatal(err)
	}
	host := &countingHost{id: "local"}
	reg := execution.NewHostRegistry()
	_ = reg.Register(host)
	brk := NewWithHosts(reg)
	_, err = brk.Execute(contracts.ExecutionRequest{Grant: g.ExecutionGrant, PackContentDigest: dB})
	if err == nil {
		t.Fatal("pack swap must deny")
	}
	if host.calls != 0 {
		t.Fatal("Host.Execute must not be called")
	}
}

func TestPackBind_MissingRequestDigest_Denies(t *testing.T) {
	issuer, cp := setupPackCP(t)
	dA := dig(3)
	g, _, err := issuer.IssueAfterEvaluate(context.Background(), cp, controlplane.EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1",
		CapabilityID: "recon.domain.enumerate", TaskID: "task-1",
		RequireTrustedPack: true, EffectivePackTrust: "TRUSTED", PackContentDigest: dA,
	})
	if err != nil {
		t.Fatal(err)
	}
	host := &countingHost{id: "local"}
	reg := execution.NewHostRegistry()
	_ = reg.Register(host)
	brk := NewWithHosts(reg)
	_, err = brk.Execute(contracts.ExecutionRequest{Grant: g.ExecutionGrant})
	if err == nil {
		t.Fatal()
	}
	if host.calls != 0 {
		t.Fatal()
	}
}

func TestPackBind_TrustedWithoutDigest_NoGrant(t *testing.T) {
	issuer, cp := setupPackCP(t)
	_, _, err := issuer.IssueAfterEvaluate(context.Background(), cp, controlplane.EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1",
		CapabilityID: "recon.domain.enumerate", TaskID: "task-1",
		RequireTrustedPack: true, EffectivePackTrust: "TRUSTED",
	})
	if err == nil {
		t.Fatal("must not issue without digest")
	}
}

func TestPackBind_Legacy_NoPackDigest(t *testing.T) {
	issuer, brk, cp := setup(t)
	g, _, err := issuer.IssueAfterEvaluate(context.Background(), cp, controlplane.EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1",
		CapabilityID: "recon.domain.enumerate", TaskID: "task-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if g.PackContentDigest != "" {
		t.Fatal("legacy grant must not bind pack")
	}
	_, err = brk.Execute(contracts.ExecutionRequest{Grant: g.ExecutionGrant})
	if err != nil {
		t.Fatal(err)
	}
}
