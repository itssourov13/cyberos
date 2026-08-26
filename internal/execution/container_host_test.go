package execution_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"cyberos/internal/broker"
	"cyberos/internal/contracts"
	"cyberos/internal/execution"
)

type fakeRuntime struct {
	calls atomic.Int32
	last  execution.ContainerRunRequest
	err   error
	res   execution.ContainerRunResult
}

func (f *fakeRuntime) Run(ctx context.Context, req execution.ContainerRunRequest) (execution.ContainerRunResult, error) {
	f.calls.Add(1)
	f.last = req
	if f.err != nil {
		return f.res, f.err
	}
	if f.res.Stdout == nil && f.res.ExitCode == 0 {
		return execution.ContainerRunResult{ExitCode: 0, Stdout: []byte("ok")}, nil
	}
	return f.res, nil
}

func grantOK() contracts.ExecutionGrant {
	now := time.Now().UTC()
	return contracts.ExecutionGrant{
		ID: "g-c1", Status: contracts.ExecutionGrantStatusIssued,
		Principal:    contracts.PrincipalRef{ID: "p1", Type: contracts.PrincipalTypeHuman},
		ProjectID:    "proj-a",
		ScopeID:      "s1",
		TaskID:       "t1",
		CapabilityID: "cap.demo",
		Nonce:        "nc-" + now.Format("150405.000000"),
		IssuedAt:     now,
		ExpiresAt:    now.Add(time.Hour),
	}
}

func TestContainerHost_ImplementsExecutionHost(t *testing.T) {
	cfg := execution.DefaultContainerHostConfig()
	h, err := execution.NewContainerHost(cfg, &fakeRuntime{})
	if err != nil {
		t.Fatal(err)
	}
	var _ execution.ExecutionHost = h
	if h.ID() != execution.HostContainer {
		t.Fatal(h.ID())
	}
}

func TestContainerHost_ImageAllowlist(t *testing.T) {
	cfg := execution.DefaultContainerHostConfig()
	rt := &fakeRuntime{}
	h, _ := execution.NewContainerHost(cfg, rt)
	_, err := h.Execute(context.Background(), contracts.ExecutionRequest{
		Parameters: map[string]any{"image": "evil/image:latest"},
	})
	if err == nil {
		t.Fatal("unknown image must deny")
	}
	if rt.calls.Load() != 0 {
		t.Fatal("runtime must not run")
	}
}

func TestContainerHost_RejectUnsafeParams(t *testing.T) {
	cfg := execution.DefaultContainerHostConfig()
	rt := &fakeRuntime{}
	h, _ := execution.NewContainerHost(cfg, rt)
	for _, k := range []string{"privileged", "docker_sock", "host_network", "mounts"} {
		_, err := h.Execute(context.Background(), contracts.ExecutionRequest{
			Parameters: map[string]any{k: true},
		})
		if err == nil {
			t.Fatalf("param %s must deny", k)
		}
	}
	if rt.calls.Load() != 0 {
		t.Fatal("runtime invoked")
	}
}

func TestContainerHost_ResourceBounds(t *testing.T) {
	cfg := execution.DefaultContainerHostConfig()
	rt := &fakeRuntime{}
	h, _ := execution.NewContainerHost(cfg, rt)
	_, err := h.Execute(context.Background(), contracts.ExecutionRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if rt.last.MemoryBytes <= 0 || rt.last.PidsLimit <= 0 || !rt.last.NetworkNone {
		t.Fatalf("%+v", rt.last)
	}
	if rt.last.User == "" || rt.last.User == "0" || rt.last.User == "root" {
		t.Fatal("must not run as root by default")
	}
}

func TestForbiddenMount_DockerSock(t *testing.T) {
	if err := execution.RejectUnsafeMount("/var/run/docker.sock"); err == nil {
		t.Fatal("docker.sock must be forbidden")
	}
	if err := execution.RejectUnsafeMount("/"); err == nil {
		t.Fatal("root mount forbidden")
	}
}

func TestBuildHardenedArgs_NoPrivileged(t *testing.T) {
	args := execution.BuildHardenedArgs(execution.ContainerRunRequest{
		Image: "cyberos/tool-runtime:stable", Args: []string{"true"},
		NetworkNone: true, ReadOnlyRoot: true, NoNewPrivs: true, User: "65534:65534",
		MemoryBytes: 100, PidsLimit: 10,
	})
	joined := ""
	for _, a := range args {
		joined += a + " "
		if a == "--privileged" {
			t.Fatal("privileged flag present")
		}
		if a == "host" {
			t.Fatal("host network")
		}
	}
	_ = joined
}

func TestBroker_ContainerHost_GrantFirst(t *testing.T) {
	rt := &fakeRuntime{}
	cfg := execution.DefaultContainerHostConfig()
	ch, err := execution.NewContainerHost(cfg, rt)
	if err != nil {
		t.Fatal(err)
	}
	reg := execution.NewHostRegistry()
	_ = reg.Register(execution.NewLocalHost(&countingExec{}))
	_ = reg.Register(ch)
	b := broker.NewWithHosts(reg)

	// invalid grant → no runtime
	_, err = b.Execute(contracts.ExecutionRequest{HostID: "container"})
	if err == nil {
		t.Fatal("invalid grant")
	}
	if rt.calls.Load() != 0 {
		t.Fatal("container must not start without grant")
	}

	g := grantOK()
	res, err := b.Execute(contracts.ExecutionRequest{Grant: g, HostID: "container"})
	if err != nil {
		t.Fatal(err)
	}
	if res.HostID != "container" {
		t.Fatal(res.HostID)
	}
	if rt.calls.Load() != 1 {
		t.Fatal(rt.calls.Load())
	}
}

func TestBroker_DefaultStillLocal(t *testing.T) {
	rt := &fakeRuntime{}
	cfg := execution.DefaultContainerHostConfig()
	ch, _ := execution.NewContainerHost(cfg, rt)
	local := &countingExec{}
	reg := execution.NewHostRegistry()
	_ = reg.Register(execution.NewLocalHost(local))
	_ = reg.Register(ch)
	b := broker.NewWithHosts(reg)
	_, err := b.Execute(contracts.ExecutionRequest{Grant: grantOK()})
	if err != nil {
		t.Fatal(err)
	}
	if local.n.Load() != 1 || rt.calls.Load() != 0 {
		t.Fatal("empty HostID must stay local")
	}
}

func TestConfig_NetworkNoneRequired(t *testing.T) {
	cfg := execution.DefaultContainerHostConfig()
	cfg.NetworkNone = false
	if err := cfg.Validate(); err == nil {
		t.Fatal("network must be none in 17B")
	}
}
