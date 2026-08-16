package dependency

import (
	"testing"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/task"
)

type memLookup struct {
	items map[contracts.TaskID]*task.Task
}

func (m *memLookup) GetByID(id contracts.TaskID) (*task.Task, error) {
	t, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *t
	return &cp, nil
}

func put(m *memLookup, id string, deps ...contracts.TaskID) {
	t, _ := task.New(contracts.TaskID(id), "p", "s", "u", "c", contracts.TaskTypeCustom, id)
	t.DependsOn = deps
	if m.items == nil {
		m.items = map[contracts.TaskID]*task.Task{}
	}
	m.items[t.ID] = t
}

func TestValidateEdgesOK(t *testing.T) {
	m := &memLookup{}
	put(m, "a")
	put(m, "b")
	g := New(m)
	if err := g.ValidateEdges("b", []contracts.TaskID{"a"}); err != nil {
		t.Fatal(err)
	}
}

func TestRejectMissingDependency(t *testing.T) {
	m := &memLookup{}
	put(m, "b")
	g := New(m)
	if err := g.ValidateEdges("b", []contracts.TaskID{"missing"}); err == nil {
		t.Fatal("expected missing dep error")
	}
}

func TestRejectSelfDependency(t *testing.T) {
	m := &memLookup{}
	put(m, "a")
	g := New(m)
	if err := g.ValidateEdges("a", []contracts.TaskID{"a"}); err == nil {
		t.Fatal("expected self-dep error")
	}
}

func TestRejectCycle(t *testing.T) {
	m := &memLookup{}
	put(m, "a", "b")
	put(m, "b")
	g := New(m)
	// b depends on a, but a already depends on b → cycle
	if err := g.ValidateEdges("b", []contracts.TaskID{"a"}); err == nil {
		t.Fatal("expected cycle error")
	}
}

func TestSatisfiedRequiresSucceeded(t *testing.T) {
	m := &memLookup{}
	put(m, "a")
	put(m, "b", "a")
	g := New(m)
	b, _ := m.GetByID("b")
	ok, unsat, err := g.Satisfied(b)
	if err != nil || ok || len(unsat) != 1 {
		t.Fatalf("expected unsatisfied: ok=%v unsat=%v err=%v", ok, unsat, err)
	}
	// mark a succeeded
	a := m.items["a"]
	a.State = contracts.TaskStateSucceeded
	ok, _, err = g.Satisfied(b)
	if err != nil || !ok {
		t.Fatalf("expected satisfied: %v", err)
	}
}

func TestAssertRunnableBlocks(t *testing.T) {
	m := &memLookup{}
	put(m, "a")
	put(m, "b", "a")
	g := New(m)
	b, _ := m.GetByID("b")
	if err := g.AssertRunnable(b); err == nil {
		t.Fatal("should block")
	}
}
