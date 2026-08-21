package sdk

import (
	"context"
	"fmt"
	"time"

	appscope "cyberos/internal/application/scope"
	"cyberos/internal/assets"
	"cyberos/internal/audit"
	"cyberos/internal/auth"
	"cyberos/internal/broker"
	"cyberos/internal/capability"
	"cyberos/internal/cases"
	"cyberos/internal/config"
	"cyberos/internal/controlplane"
	"cyberos/internal/events"
	"cyberos/internal/evidence"
	"cyberos/internal/execution"
	"cyberos/internal/execution/lifecycle"
	"cyberos/internal/execution/sandbox"
	"cyberos/internal/findings"
	"cyberos/internal/identity"
	"cyberos/internal/provenance"
	"cyberos/internal/reports"
	"cyberos/internal/riskprio"
	"cyberos/internal/storage/postgres"
	"cyberos/internal/tools"
)

// Production is a PostgreSQL-backed SDK client for durable deployments (Phase 7C).
type Production struct {
	Client *Client
	DB     *postgres.DB
	Auth   *auth.APIKeyAuthenticator
	Keys   auth.KeyStore
}

// Close closes the database connection.
func (p *Production) Close() error {
	if p == nil || p.DB == nil {
		return nil
	}
	return p.DB.Close()
}

// NewProduction opens PostgreSQL, applies migrations, and wires durable repositories.
// Fail-closed: never falls back to in-memory stores when this constructor is used.
func NewProduction(ctx context.Context, cfg *config.Config, migrationsDir string) (*Production, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config required")
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("CYBEROS_DATABASE_URL required for production")
	}
	db, err := postgres.Open(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("postgres open: %w", err)
	}
	if migrationsDir == "" {
		migrationsDir = "migrations"
	}
	if err := postgres.ApplyMigrations(ctx, db, migrationsDir); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	repos := postgres.NewRepositories(db)
	keyStore := postgres.NewAPIKeyStore(db)

	ev := evidence.NewService()
	evStore := events.NewStore().WithBackend(postgres.NewEventBackend(db))
	hub := events.NewHub()
	execReg := execution.NewRegistry()
	toolMgr := tools.NewManager(nil)
	_ = toolMgr.Register(tools.NewStubScanner(nil))
	_ = toolMgr.RegisterAllExecutors(execReg)
	inner := sandbox.New(&execution.Router{Reg: execReg})
	_ = execReg.Register(mockToolExecutor{})
	execReg.SetDefault("mock")
	brk := broker.New(inner)

	idSvc := identity.NewService(repos.Principals)
	scSvc := appscope.NewService(repos.Scopes)
	cp := controlplane.NewService(idSvc, scSvc)
	authenticator := auth.NewAPIKeyAuthenticator(keyStore)

	c := &Client{
		Identity:   idSvc,
		Scopes:     scSvc,
		Control:    cp,
		Caps:       capability.NewRegistry(nil),
		Issuer:     broker.NewIssuer(5 * time.Minute),
		Broker:     brk,
		Evidence:   ev.WithRepository(repos.Evidence),
		Findings:   findings.NewService(ev).WithRepository(repos.Findings),
		Provenance: provenance.NewService().WithRepository(repos.Provenance),
		Audit:      audit.NewService(evStore).WithDurable(postgres.NewAuditBackend(db)),
		Catalog:    NewCatalog(),
		Events:     evStore,
		Hub:        hub,
		Tools:      toolMgr,
		Platform:   tools.NewPlatform(toolMgr),
		Assets:     assets.NewInventory().WithStore(repos.Assets, repos.Endpoints, repos.Observations),
		RiskPrio:   riskprio.NewEngine().WithRepository(repos.RiskEvals),
		Cases:      cases.NewService().WithRepository(repos.Cases),
		Reports:    reports.NewService().WithRepository(repos.Reports),
		ExecReg:    execReg,
		Auth:       authenticator,
	}
	_ = c.Catalog.RefreshFromRepos(ctx, repos)
	c.WireAssessment()
	c.Lifecycle = lifecycle.New(brk,
		lifecycle.WithEvidence(c.Evidence),
		lifecycle.WithFindings(c.Findings),
		lifecycle.WithProvenance(c.Provenance),
		lifecycle.WithAudit(c.Audit),
		lifecycle.WithRepository(repos.Executions),
	)

	return &Production{Client: c, DB: db, Auth: authenticator, Keys: keyStore}, nil
}

// UseProduction reports whether config selects the durable production path.
func UseProduction(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	if cfg.Env == "production" {
		return true
	}
	return cfg.AuthRequired && cfg.DatabaseURL != "" && cfg.Env != "development"
}
