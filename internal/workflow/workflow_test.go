package workflow_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"cyberos/internal/capability"
	"cyberos/internal/contracts"
	domaincap "cyberos/internal/domain/capability"
	domaintask "cyberos/internal/domain/task"
	"cyberos/internal/taskengine"
	"cyberos/internal/tools"
	"cyberos/internal/workflow"
)

type noopAdapter struct {
	d tools.Descriptor
}

func (n noopAdapter) Descriptor() tools.Descriptor { return n.d }
func (n noopAdapter) Health(context.Context) tools.HealthReport {
	return tools.HealthReport{Status: tools.HealthHealthy}
}
func (n noopAdapter) Run(context.Context, contracts.CapabilityID, map[string]any) (tools.NormalizedResult, error) {
	return tools.NormalizedResult{Success: true}, nil
}

type memTasks struct {
	mu sync.Mutex
	m  map[contracts.TaskID]*domaintask.Task
}

func (m *memTasks) Create(_ context.Context, t *domaintask.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.m == nil {
		m.m = map[contracts.TaskID]*domaintask.Task{}
	}
	cp := *t
	m.m[t.ID] = &cp
	return nil
}
func (m *memTasks) GetByID(_ context.Context, id contracts.TaskID) (*domaintask.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.m[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *t
	return &cp, nil
}
func (m *memTasks) Update(_ context.Context, t *domaintask.Task) error {
	return m.Create(context.Background(), t)
}
func (m *memTasks) ListByProject(context.Context, contracts.ProjectID, int, int) ([]*domaintask.Task, error) {
	return nil, nil
}
func (m *memTasks) ListByScope(context.Context, contracts.ScopeID, int, int) ([]*domaintask.Task, error) {
	return nil, nil
}
func (m *memTasks) ListByState(context.Context, contracts.TaskState, int, int) ([]*domaintask.Task, error) {
	return nil, nil
}
func (m *memTasks) ListDueScheduled(context.Context, time.Time, int) ([]*domaintask.Task, error) {
	return nil, nil
}
func (m *memTasks) List(context.Context, int, int) ([]*domaintask.Task, error) {
	return nil, nil
}

func TestDefinitionValidate(t *testing.T) {
	d := workflow.Definition{
		ID: "wf1", Name: "Test", Version: "1.0.0", ProjectID: "p1", ScopeID: "s1",
		Steps: []workflow.Step{
			{ID: "a", CapabilityID: "cap.a"},
			{ID: "b", CapabilityID: "cap.b", DependsOn: []string{"a"}},
		},
	}
	if err := d.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestDefinitionDuplicateStep(t *testing.T) {
	d := workflow.Definition{
		ID: "wf1", Name: "T", Version: "1", ProjectID: "p", ScopeID: "s",
		Steps: []workflow.Step{{ID: "a", CapabilityID: "c"}, {ID: "a", CapabilityID: "c"}},
	}
	if err := d.Validate(); err == nil {
		t.Fatal("duplicate")
	}
}

func TestDefinitionCycle(t *testing.T) {
	d := workflow.Definition{
		ID: "wf1", Name: "T", Version: "1", ProjectID: "p", ScopeID: "s",
		Steps: []workflow.Step{
			{ID: "a", CapabilityID: "c", DependsOn: []string{"b"}},
			{ID: "b", CapabilityID: "c", DependsOn: []string{"a"}},
		},
	}
	if err := d.Validate(); err == nil {
		t.Fatal("cycle")
	}
}

func TestDefinitionUnknownDep(t *testing.T) {
	d := workflow.Definition{
		ID: "wf1", Name: "T", Version: "1", ProjectID: "p", ScopeID: "s",
		Steps: []workflow.Step{{ID: "a", CapabilityID: "c", DependsOn: []string{"x"}}},
	}
	if err := d.Validate(); err == nil {
		t.Fatal("unknown dep")
	}
}

func TestEmptyWorkflowInvalid(t *testing.T) {
	d := workflow.Definition{ID: "w", Name: "n", Version: "1", ProjectID: "p", ScopeID: "s"}
	if err := d.Validate(); err == nil {
		t.Fatal("empty steps")
	}
}

func TestCapabilityResolveDeterministic(t *testing.T) {
	mgr := tools.NewManager(nil)
	_ = mgr.Register(noopAdapter{d: tools.Descriptor{
		ID: "tool.z", Name: "z", Version: "1", Runtime: contracts.PluginRuntimeGo,
		ExecutorID: "tool.z", Capabilities: []contracts.CapabilityID{"cap.scan"},
	}})
	_ = mgr.Register(noopAdapter{d: tools.Descriptor{
		ID: "tool.a", Name: "a", Version: "1", Runtime: contracts.PluginRuntimeGo,
		ExecutorID: "tool.a", Capabilities: []contracts.CapabilityID{"cap.scan"},
	}})
	plat := tools.NewPlatform(mgr)
	_ = plat.RegisterPack(tools.Pack{
		ID: "pack.x", Name: "X", Version: "1.0.0", State: tools.PackStateActive,
		ToolIDs: []string{"tool.z", "tool.a"},
	})
	res := workflow.NewCapabilityResolver(plat)
	rt, err := res.ResolvePrimary("cap.scan")
	if err != nil || rt.ToolID != "tool.a" {
		t.Fatal(err, rt)
	}
}

func TestCapabilityUnknown(t *testing.T) {
	res := workflow.NewCapabilityResolver(tools.NewPlatform(nil))
	if _, err := res.Resolve("cap.missing"); err == nil {
		t.Fatal("expected not found")
	}
}

func TestDisabledPackRejected(t *testing.T) {
	mgr := tools.NewManager(nil)
	_ = mgr.Register(noopAdapter{d: tools.Descriptor{
		ID: "tool.x", Name: "x", Version: "1", Runtime: contracts.PluginRuntimeGo,
		ExecutorID: "tool.x", Capabilities: []contracts.CapabilityID{"cap.x"},
	}})
	plat := tools.NewPlatform(mgr)
	_ = plat.RegisterPack(tools.Pack{
		ID: "pack.d", Name: "D", Version: "1.0.0", State: tools.PackStateDisabled,
		ToolIDs: []string{"tool.x"},
	})
	res := workflow.NewCapabilityResolver(plat)
	if _, err := res.Resolve("cap.x"); err == nil {
		t.Fatal("disabled pack tools must not resolve")
	}
}

func TestMaterializeCreatesTasksAndDeps(t *testing.T) {
	capReg := capability.NewRegistry(nil)
	c, err := domaincap.New("cap.scan", "Scan", contracts.CapabilityTypeRecon)
	if err != nil {
		t.Fatal(err)
	}
	_ = capReg.Register(context.Background(), c)

	mgr := tools.NewManager(nil)
	_ = mgr.Register(noopAdapter{d: tools.Descriptor{
		ID: "tool.scan", Name: "scan", Version: "1", Runtime: contracts.PluginRuntimeGo,
		ExecutorID: "tool.scan", Capabilities: []contracts.CapabilityID{"cap.scan"},
	}})
	plat := tools.NewPlatform(mgr)
	_ = plat.RegisterPack(tools.Pack{
		ID: "pack.r", Name: "R", Version: "1.0.0", State: tools.PackStateActive,
		ToolIDs: []string{"tool.scan"},
	})

	repo := &memTasks{}
	eng := taskengine.NewEngine(repo, nil, nil, capReg)
	mat := workflow.NewMaterializer(eng, workflow.NewCapabilityResolver(plat))

	def := workflow.Definition{
		ID: "wf-mat", Name: "Mat", Version: "1.0.0", ProjectID: "proj-a", ScopeID: "scope-a",
		Steps: []workflow.Step{
			{ID: "s1", CapabilityID: "cap.scan", Title: "first"},
			{ID: "s2", CapabilityID: "cap.scan", DependsOn: []string{"s1"}, MaxAttempts: 2},
		},
	}
	out, err := mat.Materialize(context.Background(), def, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Tasks) != 2 {
		t.Fatal(len(out.Tasks))
	}
	if out.Tasks[0].ProjectID != "proj-a" || out.Tasks[0].ScopeID != "scope-a" {
		t.Fatal("project/scope attribution")
	}
	if out.Resolutions["s1"].ToolID != "tool.scan" {
		t.Fatal(out.Resolutions)
	}
	t2, err := repo.GetByID(context.Background(), out.StepToTask["s2"])
	if err != nil {
		t.Fatal(err)
	}
	if len(t2.DependsOn) != 1 || t2.DependsOn[0] != out.StepToTask["s1"] {
		t.Fatalf("deps: %+v", t2.DependsOn)
	}
}

func TestMaterializeUnknownCapabilityFailClosed(t *testing.T) {
	eng := taskengine.NewEngine(&memTasks{}, nil, nil, capability.NewRegistry(nil))
	mat := workflow.NewMaterializer(eng, workflow.NewCapabilityResolver(tools.NewPlatform(nil)))
	def := workflow.Definition{
		ID: "wf", Name: "n", Version: "1", ProjectID: "p", ScopeID: "s",
		Steps: []workflow.Step{{ID: "s1", CapabilityID: "cap.unknown"}},
	}
	if _, err := mat.Materialize(context.Background(), def, "u1"); err == nil {
		t.Fatal("unknown capability")
	}
}
