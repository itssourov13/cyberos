package broker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/execution/mock"
)

func ledgerFixture(t *testing.T) (*Issuer, *Broker, *MemoryGrantLedger, *GrantSigner) {
	t.Helper()
	signer, err := NewEphemeralGrantSigner()
	if err != nil {
		t.Fatal(err)
	}
	l := NewMemoryGrantLedger()
	iss := NewIssuer(time.Minute)
	iss.Signer, iss.Ledger = signer, l
	brk := New(mock.New())
	brk.Signer, brk.Ledger = signer, l
	return iss, brk, l, signer
}

func issueOne(t *testing.T, iss *Issuer) contracts.ExecutionGrant {
	t.Helper()
	p := contracts.PrincipalRef{ID: "p1", Type: contracts.PrincipalTypeHuman}
	g, err := iss.Issue(context.Background(), IssueRequest{
		Decision: contracts.Decision{
			ID: "dec-1", Timestamp: time.Now().UTC(), Outcome: contracts.DecisionAllowed,
			Principal: p, ProjectID: "proj", ScopeID: "scope", TaskID: "task", CapabilityID: "cap", Reason: "ok",
		},
		Principal: p, ProjectID: "proj", ScopeID: "scope", TaskID: "task", CapabilityID: "cap",
	})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	return g.ExecutionGrant
}

func TestLedger_IssuanceIsRecordedAndExecutesOnce(t *testing.T) {
	iss, brk, l, _ := ledgerFixture(t)
	g := issueOne(t, iss)

	rec, err := l.Get(context.Background(), g.ID)
	if err != nil || rec.State != GrantStateIssued || rec.PrincipalID != "p1" || rec.Nonce != g.Nonce {
		t.Fatalf("issued grant must be registered: %+v %v", rec, err)
	}
	if _, err := brk.Execute(execReq(g)); err != nil {
		t.Fatalf("legitimate grant: %v", err)
	}
	rec, _ = l.Get(context.Background(), g.ID)
	if rec.State != GrantStateCompleted && rec.State != GrantStateFailed {
		t.Fatalf("execution outcome must be recorded, state=%s", rec.State)
	}
	if _, err := brk.Execute(execReq(g)); !errors.Is(err, contracts.ErrGrantInvalid) {
		t.Fatalf("replay must be rejected, got %v", err)
	}
}

func TestLedger_ValidlySignedButUnregisteredGrantRejected(t *testing.T) {
	_, brk, _, signer := ledgerFixture(t)
	// Signature is genuine (e.g. ledger lost/rolled back) but the ledger never saw it.
	now := time.Now().UTC()
	g := contracts.ExecutionGrant{
		ID: "grant-ghost", Principal: contracts.PrincipalRef{ID: "p1", Type: contracts.PrincipalTypeHuman},
		ProjectID: "proj", ScopeID: "scope", TaskID: "task", CapabilityID: "cap",
		Status: contracts.ExecutionGrantStatusIssued, IssuedAt: now, ExpiresAt: now.Add(time.Minute),
		Nonce: "nonce-ghost", CorrelationID: "c",
	}
	sig, err := signer.Sign(g)
	if err != nil {
		t.Fatal(err)
	}
	g.Signature = sig
	if _, err := brk.Execute(execReq(g)); !errors.Is(err, contracts.ErrGrantInvalid) {
		t.Fatalf("unissued grant must be rejected, got %v", err)
	}
}

func TestLedger_RevokedGrantRejectedAcrossInstances(t *testing.T) {
	iss, brkA, l, signer := ledgerFixture(t)
	// A second Broker instance sharing only the ledger and the key.
	brkB := New(mock.New())
	brkB.Signer, brkB.Ledger = signer, l

	g := issueOne(t, iss)
	if err := brkA.RevokeGrantDurable(context.Background(), g.ID, "admin", "test"); err != nil {
		t.Fatal(err)
	}
	if _, err := brkB.Execute(execReq(g)); !errors.Is(err, contracts.ErrGrantInvalid) {
		t.Fatalf("revocation must be visible to other instances, got %v", err)
	}
	// Revocation after use is a conflict, not a silent success.
	g2 := issueOne(t, iss)
	if _, err := brkA.Execute(execReq(g2)); err != nil {
		t.Fatal(err)
	}
	if err := brkA.RevokeGrantDurable(context.Background(), g2.ID, "admin", "late"); !errors.Is(err, contracts.ErrConflict) {
		t.Fatalf("revoking a used grant must conflict, got %v", err)
	}
}

func TestLedger_ExpiredGrantRejected(t *testing.T) {
	iss, brk, _, _ := ledgerFixture(t)
	g := issueOne(t, iss)
	brk.Now = func() time.Time { return g.ExpiresAt.Add(time.Second) }
	if _, err := brk.Execute(execReq(g)); !errors.Is(err, contracts.ErrGrantExpired) && !errors.Is(err, contracts.ErrGrantInvalid) {
		t.Fatalf("expired grant must be rejected, got %v", err)
	}
}

func TestLedger_ConcurrentClaimHasExactlyOneWinner(t *testing.T) {
	iss, brk, _, _ := ledgerFixture(t)
	g := issueOne(t, iss)
	var wins int32
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := brk.Execute(execReq(g)); err == nil {
				atomic.AddInt32(&wins, 1)
			}
		}()
	}
	wg.Wait()
	if wins != 1 {
		t.Fatalf("exactly one concurrent execution may win, got %d", wins)
	}
}

type failingLedger struct{ GrantLedger }

func (failingLedger) Record(context.Context, GrantRecord) error { return errors.New("db down") }

func TestIssuer_FailsClosedWhenLedgerCannotRecord(t *testing.T) {
	iss, _, _, _ := ledgerFixture(t)
	iss.Ledger = failingLedger{}
	p := contracts.PrincipalRef{ID: "p1", Type: contracts.PrincipalTypeHuman}
	_, err := iss.Issue(context.Background(), IssueRequest{
		Decision: contracts.Decision{
			ID: "d", Timestamp: time.Now().UTC(), Outcome: contracts.DecisionAllowed,
			Principal: p, ProjectID: "proj", ScopeID: "scope", TaskID: "task", CapabilityID: "cap", Reason: "ok",
		},
		Principal: p, ProjectID: "proj", ScopeID: "scope", TaskID: "task", CapabilityID: "cap",
	})
	if err == nil {
		t.Fatal("issuance must fail when the grant cannot be registered")
	}
}
