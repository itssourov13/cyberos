package tools

import (
	"errors"
	"strings"
	"testing"

	"cyberos/internal/audit"
	"cyberos/internal/contracts"
)

type failingAuditStore struct{}

func (failingAuditStore) Append(audit.Entry) error { return errors.New("audit database unavailable") }
func (failingAuditStore) ListByCorrelation(contracts.CorrelationID) ([]audit.Entry, error) {
	return nil, errors.New("audit database unavailable")
}
func (failingAuditStore) All() ([]audit.Entry, error) {
	return nil, errors.New("audit database unavailable")
}

func TestTrustAdminReturnsAuditPersistenceFailure(t *testing.T) {
	digest := DigestPrefix + strings.Repeat("a", 64)
	store := NewTrustStore()
	svc := audit.NewService(nil).WithDurable(failingAuditStore{})
	admin := &TrustAdmin{Store: store, Audit: svc}
	operator := contracts.PrincipalRef{ID: "platform-service", Type: contracts.PrincipalTypeService}
	err := admin.TrustPack(operator, "project-1", digest, "corr-trust-1", "approved")
	if err == nil || !strings.Contains(err.Error(), "audit persistence failed") {
		t.Fatalf("expected explicit audit failure after mutation, got %v", err)
	}
	if !store.IsTrusted("project-1", digest) {
		t.Fatal("test precondition: trust mutation should already be committed")
	}
}
