//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cyberos/internal/broker"
	"cyberos/internal/contracts"
	"cyberos/internal/storage/postgres"
)

func TestGrantLedger_Durability(t *testing.T) {
	db := setupDB(t)
	l := postgres.NewGrantLedger(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	mk := func(id, nonce string) broker.GrantRecord {
		return broker.GrantRecord{
			GrantID: contracts.ExecutionGrantID(id), Nonce: nonce, PrincipalID: "u1",
			ProjectID: "p1", ScopeID: "s1", TaskID: "t1", CapabilityID: "cap",
			IssuedAt: now, ExpiresAt: now.Add(time.Minute),
		}
	}

	// Unknown grant and wrong nonce are not issued by this control plane.
	if err := l.Claim(ctx, contracts.ExecutionGrantID("grant-unknown-"+t.Name()), "x", now); !errors.Is(err, contracts.ErrGrantInvalid) {
		t.Fatalf("unknown grant must be rejected, got %v", err)
	}
	id, nonce := "grant-ledger-"+time.Now().Format("150405.000000"), "n-"+time.Now().Format("150405.000000")
	if err := l.Record(ctx, mk(id, nonce)); err != nil {
		t.Fatal(err)
	}
	if err := l.Record(ctx, mk(id, nonce+"b")); !errors.Is(err, contracts.ErrConflict) {
		t.Fatalf("duplicate grant id must conflict, got %v", err)
	}
	if err := l.Claim(ctx, contracts.ExecutionGrantID(id), "wrong", now); !errors.Is(err, contracts.ErrGrantInvalid) {
		t.Fatalf("wrong nonce must be rejected, got %v", err)
	}

	// 12 concurrent claims: exactly one winner.
	var wins int32
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if l.Claim(ctx, contracts.ExecutionGrantID(id), nonce, now) == nil {
				atomic.AddInt32(&wins, 1)
			}
		}()
	}
	wg.Wait()
	if wins != 1 {
		t.Fatalf("exactly one claim may win, got %d", wins)
	}
	if err := l.Complete(ctx, contracts.ExecutionGrantID(id), string(contracts.ExecutionStatusSucceeded), now); err != nil {
		t.Fatal(err)
	}
	if rec, err := l.Get(ctx, contracts.ExecutionGrantID(id)); err != nil || rec.State != broker.GrantStateCompleted {
		t.Fatalf("state: %+v %v", rec, err)
	}
	if err := l.Revoke(ctx, contracts.ExecutionGrantID(id), "admin", "late", now); !errors.Is(err, contracts.ErrConflict) {
		t.Fatalf("revoking a used grant must conflict, got %v", err)
	}

	// Revoked and expired grants cannot be claimed.
	id2, n2 := id+"-r", nonce+"-r"
	_ = l.Record(ctx, mk(id2, n2))
	if err := l.Revoke(ctx, contracts.ExecutionGrantID(id2), "admin", "test", now); err != nil {
		t.Fatal(err)
	}
	if err := l.Claim(ctx, contracts.ExecutionGrantID(id2), n2, now); !errors.Is(err, contracts.ErrGrantInvalid) {
		t.Fatalf("revoked grant must be rejected, got %v", err)
	}
	id3, n3 := id+"-e", nonce+"-e"
	_ = l.Record(ctx, mk(id3, n3))
	if err := l.Claim(ctx, contracts.ExecutionGrantID(id3), n3, now.Add(2*time.Minute)); !errors.Is(err, contracts.ErrGrantExpired) {
		t.Fatalf("expired grant must be rejected, got %v", err)
	}
}
