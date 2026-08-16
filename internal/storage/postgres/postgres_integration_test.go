//go:build integration

package postgres_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/asset"
	"cyberos/internal/domain/capability"
	"cyberos/internal/domain/principal"
	"cyberos/internal/domain/project"
	"cyberos/internal/domain/scope"
	"cyberos/internal/domain/task"
	"cyberos/internal/storage/postgres"
)

func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("CYBEROS_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://cyberos:cyberos@localhost:5432/cyberos?sslmode=disable"
	}
	return dsn
}

func setupDB(t *testing.T) *postgres.DB {
	t.Helper()
	db, err := postgres.Open(testDSN(t))
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Locate migrations relative to module root.
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// Walk up to find migrations/
	dir := root
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "migrations")
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := postgres.ApplyMigrations(ctx, db, candidate); err != nil {
				t.Fatalf("migrate: %v", err)
			}
			return db
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("migrations directory not found")
	return nil
}

func TestProjectAssetScopeTaskRoundTrip(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	repos := postgres.NewRepositories(db)

	// Principal
	p, err := principal.NewHuman("user-it-1", "integration-user")
	if err != nil {
		t.Fatal(err)
	}
	if err := repos.Principals.Create(ctx, p); err != nil {
		t.Fatalf("create principal: %v", err)
	}
	gotP, err := repos.Principals.GetByID(ctx, p.ID)
	if err != nil || gotP.Name != p.Name {
		t.Fatalf("get principal: %v %+v", err, gotP)
	}

	// Project
	proj, err := project.New("proj-it-1", "Integration Project")
	if err != nil {
		t.Fatal(err)
	}
	if err := repos.Projects.Create(ctx, proj); err != nil {
		t.Fatalf("create project: %v", err)
	}
	gotProj, err := repos.Projects.GetByID(ctx, proj.ID)
	if err != nil || gotProj.Name != proj.Name {
		t.Fatalf("get project: %v", err)
	}

	// Asset
	a, err := asset.New("asset-it-1", proj.ID, contracts.AssetTypeDomain, "example.test", "example.test")
	if err != nil {
		t.Fatal(err)
	}
	if err := repos.Assets.Create(ctx, a); err != nil {
		t.Fatalf("create asset: %v", err)
	}

	// Scope
	sc, err := scope.New("scope-it-1", proj.ID, "IT Scope", contracts.ScopeStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	if err := repos.Scopes.Create(ctx, sc); err != nil {
		t.Fatalf("create scope: %v", err)
	}
	gotSc, err := repos.Scopes.GetByID(ctx, sc.ID)
	if err != nil || !gotSc.IsActive(time.Now().UTC()) {
		t.Fatalf("get scope: %v active=%v", err, gotSc != nil && gotSc.IsActive(time.Now().UTC()))
	}

	// Capability
	cap, err := capability.New("recon.domain.enumerate", "Domain Enum", contracts.CapabilityTypeRecon)
	if err != nil {
		t.Fatal(err)
	}
	if err := repos.Capabilities.Create(ctx, cap); err != nil {
		t.Fatalf("create capability: %v", err)
	}

	// Task
	tk, err := task.New("task-it-1", proj.ID, sc.ID, p.ID, cap.ID, contracts.TaskTypeRecon, "Enumerate")
	if err != nil {
		t.Fatal(err)
	}
	if err := repos.Tasks.Create(ctx, tk); err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := tk.TransitionTo(contracts.TaskStatePending); err != nil {
		t.Fatal(err)
	}
	if err := repos.Tasks.Update(ctx, tk); err != nil {
		t.Fatalf("update task: %v", err)
	}
	gotTk, err := repos.Tasks.GetByID(ctx, tk.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if gotTk.State != contracts.TaskStatePending {
		t.Errorf("expected PENDING, got %s", gotTk.State)
	}

	// Not found
	_, err = repos.Projects.GetByID(ctx, "does-not-exist")
	if err != contracts.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
