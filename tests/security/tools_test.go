package security_test

import (
	"testing"
	"time"

	"cyberos/internal/broker"
	"cyberos/internal/contracts"
	"cyberos/internal/execution"
	"cyberos/internal/execution/sandbox"
	"cyberos/internal/tools"
)

func TestSecurity_ToolAdapterRequiresGrant(t *testing.T) {
	mgr := tools.NewManager(nil)
	_ = mgr.Register(tools.NewStubScanner(nil))
	reg := execution.NewRegistry()
	_ = mgr.RegisterAllExecutors(reg)
	brk := broker.New(sandbox.New(&execution.Router{Reg: reg}))
	_, err := brk.Execute(contracts.ExecutionRequest{})
	if err == nil {
		t.Fatal("no grant ⇒ no execution")
	}
}

func TestSecurity_UnknownToolNotRegistered(t *testing.T) {
	mgr := tools.NewManager(nil)
	_ = mgr.Register(tools.NewStubScanner(nil))
	if _, err := mgr.Get("nuclei"); err == nil {
		t.Fatal("fail closed on unknown tools")
	}
}

func TestSecurity_DiscoveryNotAuthorization(t *testing.T) {
	mgr := tools.NewManager(nil)
	_ = mgr.Register(tools.NewStubScanner(nil))
	list := mgr.List()
	if len(list) != 1 {
		t.Fatal("expected stub registered")
	}
	// Listing does not produce execution results.
	now := time.Now().UTC()
	ex, _ := mgr.ExecutorFor("stub.security.scanner")
	// Direct executor still needs valid grant structure
	_, err := ex.Execute(contracts.ExecutionRequest{})
	if err == nil {
		t.Fatal("empty grant rejected at executor")
	}
	_ = now
}
