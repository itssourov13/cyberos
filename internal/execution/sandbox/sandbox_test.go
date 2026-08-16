package sandbox

import (
	"strings"
	"testing"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/execution/mock"
)

func sampleReq() contracts.ExecutionRequest {
	now := time.Now().UTC()
	return contracts.ExecutionRequest{
		Grant: contracts.ExecutionGrant{
			ID: "g1", Principal: contracts.PrincipalRef{ID: "u", Type: contracts.PrincipalTypeHuman},
			ProjectID: "p", ScopeID: "s", TaskID: "t", CapabilityID: "c",
			Status:   contracts.ExecutionGrantStatusIssued,
			IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute),
		},
	}
}

func TestSandboxAllowsMock(t *testing.T) {
	s := New(mock.New())
	res, err := s.Execute(sampleReq())
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != contracts.ExecutionStatusSucceeded {
		t.Fatalf("status=%s", res.Status)
	}
	if !strings.HasPrefix(res.ExecutorID, "sandbox") {
		t.Errorf("executor=%s", res.ExecutorID)
	}
}

func TestRejectNetworkPolicy(t *testing.T) {
	s := New(mock.New(), WithPolicy(contracts.ResourcePolicy{
		MaxDuration:  time.Second,
		AllowNetwork: true,
	}))
	_, err := s.Execute(sampleReq())
	if err == nil {
		t.Fatal("network must be rejected")
	}
}

func TestRejectFilesystemPolicy(t *testing.T) {
	s := New(mock.New(), WithPolicy(contracts.ResourcePolicy{
		MaxDuration:     time.Second,
		AllowFilesystem: true,
	}))
	_, err := s.Execute(sampleReq())
	if err == nil {
		t.Fatal("filesystem must be rejected")
	}
}

func TestOutputSizeLimit(t *testing.T) {
	inner := mock.New()
	s := New(inner, WithPolicy(contracts.ResourcePolicy{
		MaxDuration:    time.Minute,
		MaxOutputBytes: 10, // tiny
	}))
	res, err := s.Execute(sampleReq())
	// mock output includes several keys — should exceed 10 bytes
	if err == nil {
		t.Fatalf("expected output limit error, got success: %+v", res)
	}
	if res.Status != contracts.ExecutionStatusRejected {
		t.Errorf("status=%s", res.Status)
	}
}

func TestConcurrencyLimit(t *testing.T) {
	block := make(chan struct{})
	inner := &blockingExecutor{block: block}
	s := New(inner, WithPolicy(contracts.ResourcePolicy{
		MaxDuration:    time.Minute,
		MaxConcurrency: 1,
	}))
	done := make(chan error, 2)
	go func() {
		_, err := s.Execute(sampleReq())
		done <- err
	}()
	// wait until first is in flight
	time.Sleep(20 * time.Millisecond)
	_, err := s.Execute(sampleReq())
	if err == nil {
		t.Fatal("second concurrent exec should fail")
	}
	close(block)
	<-done
}

type blockingExecutor struct {
	block chan struct{}
}

func (b *blockingExecutor) Execute(req contracts.ExecutionRequest) (contracts.ExecutionResult, error) {
	<-b.block
	now := time.Now().UTC()
	return contracts.ExecutionResult{
		Status: contracts.ExecutionStatusSucceeded, StartedAt: now, FinishedAt: now, ExecutorID: "block",
	}, nil
}
