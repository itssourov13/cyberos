package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	appscope "cyberos/internal/application/scope"
	"cyberos/internal/contracts"
	"cyberos/internal/controlplane"
	"cyberos/internal/domain/principal"
	domainscope "cyberos/internal/domain/scope"
	"cyberos/internal/identity"
	"cyberos/internal/sdk"
)

type memP struct {
	items map[contracts.PrincipalID]*principal.Principal
}

func (m *memP) Create(_ context.Context, p *principal.Principal) error {
	if m.items == nil {
		m.items = map[contracts.PrincipalID]*principal.Principal{}
	}
	cp := *p
	m.items[p.ID] = &cp
	return nil
}
func (m *memP) GetByID(_ context.Context, id contracts.PrincipalID) (*principal.Principal, error) {
	p, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *p
	return &cp, nil
}
func (m *memP) List(context.Context, int, int) ([]*principal.Principal, error) { return nil, nil }

type memS struct {
	items map[contracts.ScopeID]*domainscope.Scope
}

func (m *memS) Create(_ context.Context, s *domainscope.Scope) error {
	if m.items == nil {
		m.items = map[contracts.ScopeID]*domainscope.Scope{}
	}
	cp := *s
	m.items[s.ID] = &cp
	return nil
}
func (m *memS) GetByID(_ context.Context, id contracts.ScopeID) (*domainscope.Scope, error) {
	s, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *s
	return &cp, nil
}
func (m *memS) Update(context.Context, *domainscope.Scope) error { return nil }
func (m *memS) ListByProject(context.Context, contracts.ProjectID, int, int) ([]*domainscope.Scope, error) {
	return nil, nil
}
func (m *memS) ListAll(context.Context, int, int) ([]*domainscope.Scope, error) {
	return nil, nil
}

func TestCLIVersionAndHealth(t *testing.T) {
	app := New(sdk.NewInMemory())
	if code := app.Run([]string{"version"}); code != 0 {
		t.Fatalf("version exit %d", code)
	}
	if code := app.Run([]string{"health"}); code != 0 {
		t.Fatalf("health exit %d", code)
	}
}

func TestCLICapabilitiesRegisterList(t *testing.T) {
	app := New(sdk.NewInMemory())
	if code := app.Run([]string{"capabilities", "register", "c1", "Cap", "UTILITY"}); code != 0 {
		t.Fatalf("register exit %d", code)
	}
	// capture list output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	code := app.Run([]string{"capabilities", "list"})
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	if code != 0 {
		t.Fatalf("list exit %d", code)
	}
	if !bytes.Contains(buf.Bytes(), []byte("c1")) {
		t.Errorf("list output missing c1: %s", buf.String())
	}
}

func TestCLIEvaluateUsesControlPlane(t *testing.T) {
	pr := &memP{}
	sr := &memS{}
	p, _ := principal.NewHuman("user-1", "a")
	_ = pr.Create(context.Background(), p)
	sc, _ := domainscope.New("scope-1", "proj-1", "M", contracts.ScopeStatusActive)
	_ = sr.Create(context.Background(), sc)
	client := sdk.NewInMemory().
		WithIdentity(identity.NewService(pr)).
		WithScopes(appscope.NewService(sr)).
		WithControlPlane(controlplane.NewService(identity.NewService(pr), appscope.NewService(sr)))
	_ = client.RegisterCapability(context.Background(), "cap-1", "C", contracts.CapabilityTypeUtility)

	app := New(client)
	code := app.Run([]string{
		"evaluate",
		"--principal", "user-1",
		"--project", "proj-1",
		"--scope", "scope-1",
		"--capability", "cap-1",
	})
	if code != 0 {
		t.Fatalf("evaluate exit %d", code)
	}
}
