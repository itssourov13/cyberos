package taskengine

import (
	"context"
	"errors"
	"testing"
	"time"

	appscope "cyberos/internal/application/scope"
	"cyberos/internal/capability"
	"cyberos/internal/contracts"
	domaincap "cyberos/internal/domain/capability"
	"cyberos/internal/domain/principal"
	domainscope "cyberos/internal/domain/scope"
	"cyberos/internal/domain/task"
	"cyberos/internal/identity"
)

// --- minimal in-memory task repo ---

type memTaskRepo struct {
	items map[contracts.TaskID]*task.Task
}

func (m *memTaskRepo) Create(_ context.Context, t *task.Task) error {
	if m.items == nil {
		m.items = map[contracts.TaskID]*task.Task{}
	}
	if _, ok := m.items[t.ID]; ok {
		return contracts.ErrConflict
	}
	cp := *t
	m.items[t.ID] = &cp
	return nil
}
func (m *memTaskRepo) GetByID(_ context.Context, id contracts.TaskID) (*task.Task, error) {
	t, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *t
	return &cp, nil
}
func (m *memTaskRepo) Update(_ context.Context, t *task.Task) error {
	if _, ok := m.items[t.ID]; !ok {
		return contracts.ErrNotFound
	}
	cp := *t
	m.items[t.ID] = &cp
	return nil
}
func (m *memTaskRepo) ListByScope(_ context.Context, scopeID contracts.ScopeID, _, _ int) ([]*task.Task, error) {
	var out []*task.Task
	for _, t := range m.items {
		if t.ScopeID == scopeID {
			cp := *t
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (m *memTaskRepo) List(context.Context, int, int) ([]*task.Task, error) { return nil, nil }
func (m *memTaskRepo) ListByState(context.Context, contracts.TaskState, int, int) ([]*task.Task, error) {
	return nil, nil
}
func (m *memTaskRepo) ListDueScheduled(context.Context, time.Time, int) ([]*task.Task, error) {
	return nil, nil
}

type memPrincipalRepo struct {
	items map[contracts.PrincipalID]*principal.Principal
}

func (m *memPrincipalRepo) Create(_ context.Context, p *principal.Principal) error {
	if m.items == nil {
		m.items = map[contracts.PrincipalID]*principal.Principal{}
	}
	cp := *p
	m.items[p.ID] = &cp
	return nil
}
func (m *memPrincipalRepo) GetByID(_ context.Context, id contracts.PrincipalID) (*principal.Principal, error) {
	p, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *p
	return &cp, nil
}
func (m *memPrincipalRepo) List(context.Context, int, int) ([]*principal.Principal, error) {
	return nil, nil
}

type memScopeRepo struct {
	items map[contracts.ScopeID]*domainscope.Scope
}

func (m *memScopeRepo) Create(_ context.Context, s *domainscope.Scope) error {
	if m.items == nil {
		m.items = map[contracts.ScopeID]*domainscope.Scope{}
	}
	cp := *s
	m.items[s.ID] = &cp
	return nil
}
func (m *memScopeRepo) GetByID(_ context.Context, id contracts.ScopeID) (*domainscope.Scope, error) {
	s, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *s
	return &cp, nil
}
func (m *memScopeRepo) Update(context.Context, *domainscope.Scope) error { return nil }
func (m *memScopeRepo) ListByProject(context.Context, contracts.ProjectID, int, int) ([]*domainscope.Scope, error) {
	return nil, nil
}
func (m *memScopeRepo) ListAll(context.Context, int, int) ([]*domainscope.Scope, error) {
	return nil, nil
}

func setupEngine(t *testing.T) (*Engine, *memTaskRepo) {
	t.Helper()
	pr := &memPrincipalRepo{}
	sr := &memScopeRepo{}
	tr := &memTaskRepo{}
	p, _ := principal.NewHuman("user-1", "alice")
	_ = pr.Create(context.Background(), p)
	sc, _ := domainscope.New("scope-1", "proj-1", "Main", contracts.ScopeStatusActive)
	_ = sr.Create(context.Background(), sc)

	capReg := capability.NewRegistry(nil)
	c, _ := domaincap.New("recon.domain.enumerate", "Enum", contracts.CapabilityTypeRecon)
	_ = capReg.Register(context.Background(), c)

	eng := NewEngine(tr, identity.NewService(pr), appscope.NewService(sr), capReg)
	return eng, tr
}

func TestCreateTask(t *testing.T) {
	eng, _ := setupEngine(t)
	tk, err := eng.Create(context.Background(), CreateRequest{
		ID: "task-1", ProjectID: "proj-1", ScopeID: "scope-1",
		PrincipalID: "user-1", CapabilityID: "recon.domain.enumerate",
		Type: contracts.TaskTypeRecon, Title: "Enumerate",
	})
	if err != nil {
		t.Fatal(err)
	}
	if tk.State != contracts.TaskStateCreated {
		t.Errorf("state = %s", tk.State)
	}
}

func TestCreateUnknownCapability(t *testing.T) {
	eng, _ := setupEngine(t)
	_, err := eng.Create(context.Background(), CreateRequest{
		ID: "task-2", ProjectID: "proj-1", ScopeID: "scope-1",
		PrincipalID: "user-1", CapabilityID: "unknown.cap",
		Type: contracts.TaskTypeRecon, Title: "Bad",
	})
	if err == nil {
		t.Fatal("unknown capability should fail")
	}
	if !errors.Is(err, contracts.ErrCapabilityUnknown) {
		t.Errorf("expected ErrCapabilityUnknown, got %v", err)
	}
}

func TestCreateInactivePrincipal(t *testing.T) {
	pr := &memPrincipalRepo{}
	p, _ := principal.NewHuman("user-x", "x")
	p.Active = false
	_ = pr.Create(context.Background(), p)
	sr := &memScopeRepo{}
	sc, _ := domainscope.New("scope-1", "proj-1", "M", contracts.ScopeStatusActive)
	_ = sr.Create(context.Background(), sc)
	capReg := capability.NewRegistry(nil)
	c, _ := domaincap.New("cap.1", "C", contracts.CapabilityTypeUtility)
	_ = capReg.Register(context.Background(), c)

	eng := NewEngine(&memTaskRepo{}, identity.NewService(pr), appscope.NewService(sr), capReg)
	_, err := eng.Create(context.Background(), CreateRequest{
		ID: "t", ProjectID: "proj-1", ScopeID: "scope-1",
		PrincipalID: "user-x", CapabilityID: "cap.1",
		Type: contracts.TaskTypeCustom, Title: "T",
	})
	if err == nil {
		t.Fatal("inactive principal should fail")
	}
}

func TestCreateExpiredScope(t *testing.T) {
	pr := &memPrincipalRepo{}
	p, _ := principal.NewHuman("user-1", "a")
	_ = pr.Create(context.Background(), p)
	sr := &memScopeRepo{}
	sc, _ := domainscope.New("scope-exp", "proj-1", "E", contracts.ScopeStatusActive)
	past := time.Now().UTC().Add(-time.Hour)
	sc.ExpiresAt = &past
	_ = sr.Create(context.Background(), sc)
	capReg := capability.NewRegistry(nil)
	c, _ := domaincap.New("cap.1", "C", contracts.CapabilityTypeUtility)
	_ = capReg.Register(context.Background(), c)

	eng := NewEngine(&memTaskRepo{}, identity.NewService(pr), appscope.NewService(sr), capReg)
	_, err := eng.Create(context.Background(), CreateRequest{
		ID: "t", ProjectID: "proj-1", ScopeID: "scope-exp",
		PrincipalID: "user-1", CapabilityID: "cap.1",
		Type: contracts.TaskTypeCustom, Title: "T",
	})
	if err == nil {
		t.Fatal("expired scope should fail")
	}
}

func TestTransitionLegalAndIllegal(t *testing.T) {
	eng, _ := setupEngine(t)
	_, err := eng.Create(context.Background(), CreateRequest{
		ID: "task-tr", ProjectID: "proj-1", ScopeID: "scope-1",
		PrincipalID: "user-1", CapabilityID: "recon.domain.enumerate",
		Type: contracts.TaskTypeRecon, Title: "T",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Legal: CREATED → PENDING
	tk, err := eng.Transition(context.Background(), "task-tr", contracts.TaskStatePending)
	if err != nil {
		t.Fatalf("legal transition: %v", err)
	}
	if tk.State != contracts.TaskStatePending {
		t.Errorf("state = %s", tk.State)
	}

	// Illegal: PENDING → RUNNING
	_, err = eng.Transition(context.Background(), "task-tr", contracts.TaskStateRunning)
	if !errors.Is(err, contracts.ErrInvalidStateTransition) {
		t.Errorf("expected ErrInvalidStateTransition, got %v", err)
	}
	// State unchanged
	tk, _ = eng.Get(context.Background(), "task-tr")
	if tk.State != contracts.TaskStatePending {
		t.Errorf("state mutated on illegal transition: %s", tk.State)
	}
}
