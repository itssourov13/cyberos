package sdk

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	appscope "cyberos/internal/application/scope"
	"cyberos/internal/assets"
	"cyberos/internal/audit"
	"cyberos/internal/auth"
	"cyberos/internal/broker"
	"cyberos/internal/capability"
	"cyberos/internal/cases"
	"cyberos/internal/config"
	"cyberos/internal/contracts"
	"cyberos/internal/controlplane"
	"cyberos/internal/domain/principal"
	"cyberos/internal/events"
	"cyberos/internal/evidence"
	"cyberos/internal/execution"
	"cyberos/internal/execution/lifecycle"
	"cyberos/internal/execution/process"
	"cyberos/internal/execution/sandbox"
	"cyberos/internal/findings"
	"cyberos/internal/identity"
	"cyberos/internal/provenance"
	"cyberos/internal/reports"
	"cyberos/internal/riskprio"
	"cyberos/internal/storage/postgres"
	"cyberos/internal/taskengine"
	"cyberos/internal/tools"
	"cyberos/internal/tools/httpprobe"
	"cyberos/internal/tools/nmap"
	"cyberos/internal/tools/pythonprobe"
	"cyberos/internal/tools/rustprobe"
)

// Production is a PostgreSQL-backed SDK client for durable deployments (Phase 7C).
type Production struct {
	Client     *Client
	DB         *postgres.DB
	Auth       *auth.APIKeyAuthenticator
	Sessions   *postgres.SessionStore
	Keys       auth.KeyStore
	PackTrust  tools.TrustStorePort
	SignerKeys tools.SignerKeyRegistryPort
	TrustAdmin *tools.TrustAdmin
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
	if err := cfg.ValidateProduction(); err != nil {
		return nil, fmt.Errorf("invalid production configuration: %w", err)
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
	packTrust := postgres.NewPackTrustRepo(db)
	signerKeys := postgres.NewSignerKeyRepo(db)

	ev := evidence.NewService()
	evStore := events.NewStore().WithBackend(postgres.NewEventBackend(db))
	hub := events.NewHub()
	evStore = evStore.WithHub(hub)
	execReg := execution.NewRegistry()
	procRunner := process.NewRunner(process.PolicyFromConfig(cfg))
	toolMgr := tools.NewManager(procRunner)
	// Production contains only real, bounded adapters. External process
	// execution is disabled unless the operator explicitly enables and allowlists
	// the required binaries in config; no mock/stub can report fake success.
	for _, adapter := range []tools.Adapter{
		nmap.New(nmap.Options{Proc: procRunner}),
		httpprobe.New(httpprobe.Options{Proc: procRunner}),
		pythonprobe.New(pythonprobe.Options{Proc: procRunner}),
		rustprobe.New(rustprobe.Options{Proc: procRunner}),
	} {
		if err := toolMgr.Register(adapter); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("register production tool adapter %s: %w", adapter.Descriptor().ID, err)
		}
	}
	if err := toolMgr.RegisterAllExecutors(execReg); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("register production executors: %w", err)
	}
	// Production grants must carry an explicit executor hint. Never silently
	// route a generic grant to an arbitrary tool based on map iteration order.
	_ = execReg.SetDefault("")
	inner := sandbox.New(&execution.Router{Reg: execReg})
	brk := broker.New(inner)

	idSvc := identity.NewService(repos.Principals)
	// The bootstrap operator key is bound to a real active principal so it can
	// participate in Control Plane decisions, not merely pass HTTP middleware.
	bootstrap, bootstrapErr := idSvc.Get(ctx, "bootstrap")
	if errors.Is(bootstrapErr, contracts.ErrNotFound) {
		bootstrap, bootstrapErr = principal.NewHuman("bootstrap", "CYBEROS Bootstrap Operator")
		if bootstrapErr == nil {
			bootstrapErr = idSvc.Create(ctx, bootstrap)
		}
	}
	if bootstrapErr != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ensure bootstrap principal: %w", bootstrapErr)
	}
	if _, err := idSvc.RequireActive(ctx, "bootstrap"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("bootstrap principal is not active: %w", err)
	}
	scSvc := appscope.NewService(repos.Scopes)
	cp := controlplane.NewService(idSvc, scSvc)
	authenticator := auth.NewAPIKeyAuthenticator(keyStore)

	c := &Client{
		Identity:                    idSvc,
		Scopes:                      scSvc,
		Control:                     cp,
		Caps:                        capability.NewRegistry(nil),
		Issuer:                      broker.NewIssuer(5 * time.Minute),
		Broker:                      brk,
		Evidence:                    ev.WithRepository(repos.Evidence),
		Findings:                    findings.NewService(ev).WithRepository(repos.Findings),
		Provenance:                  provenance.NewService().WithRepository(repos.Provenance),
		Audit:                       audit.NewService(evStore).WithDurable(postgres.NewAuditBackend(db)),
		Catalog:                     NewCatalog(),
		Events:                      evStore,
		Hub:                         hub,
		Tools:                       toolMgr,
		Platform:                    tools.NewPlatform(toolMgr),
		Assets:                      assets.NewInventory().WithStore(repos.Assets, repos.Endpoints, repos.Observations),
		RiskPrio:                    riskprio.NewEngine().WithRepository(repos.RiskEvals),
		Cases:                       cases.NewService().WithRepository(repos.Cases),
		Reports:                     reports.NewService().WithRepository(repos.Reports),
		ExecReg:                     execReg,
		Auth:                        authenticator,
		RequireExplicitExecutorHint: true,
	}
	// Expose the real production adapters' capabilities through the API.
	seenCapabilities := make(map[contracts.CapabilityID]struct{})
	for _, descriptor := range toolMgr.List() {
		for _, capabilityID := range descriptor.Capabilities {
			if _, seen := seenCapabilities[capabilityID]; seen {
				continue
			}
			seenCapabilities[capabilityID] = struct{}{}
			kind := contracts.CapabilityTypeCustom
			switch {
			case strings.HasPrefix(string(capabilityID), "recon."):
				kind = contracts.CapabilityTypeRecon
			case strings.HasPrefix(string(capabilityID), "web."):
				kind = contracts.CapabilityTypeScan
			}
			if err := c.RegisterCapability(ctx, string(capabilityID), string(capabilityID), kind); err != nil {
				_ = db.Close()
				return nil, fmt.Errorf("register production capability %s: %w", capabilityID, err)
			}
		}
	}

	var grantKey []byte
	if cfg.GrantSigningKey != "" {
		grantKey, err = hex.DecodeString(strings.TrimSpace(cfg.GrantSigningKey))
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("CYBEROS_GRANT_SIGNING_KEY must be hex: %w", err)
		}
	} else if cfg.Env == "production" {
		_ = db.Close()
		return nil, fmt.Errorf("CYBEROS_GRANT_SIGNING_KEY required when CYBEROS_ENV=production")
	}
	if err := c.EnableGrantProvenance(grantKey); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable grant provenance: %w", err)
	}
	// Durable, cluster-safe grant registry: unissued/replayed/revoked grants
	// are rejected by the database, not by process memory.
	c.UseGrantLedger(postgres.NewGrantLedger(db))
	if err := c.Catalog.RefreshFromRepos(ctx, repos); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("load production project catalog: %w", err)
	}
	c.WireAssessment()
	if c.Assessment != nil && repos.Assessments != nil {
		c.Assessment.WithRepository(repos.Assessments)
	}
	c.Lifecycle = lifecycle.New(brk,
		lifecycle.WithEvidence(c.Evidence),
		lifecycle.WithFindings(c.Findings),
		lifecycle.WithProvenance(c.Provenance),
		lifecycle.WithAudit(c.Audit),
		lifecycle.WithRepository(repos.Executions),
	)
	c.Tasks = taskengine.NewEngine(repos.Tasks, idSvc, scSvc, c.Caps)
	c.WireWorkflow()
	if c.Workflow != nil && repos.WorkflowRuns != nil {
		c.Workflow.WithRepository(repos.WorkflowRuns)
	}
	trustAdmin := &tools.TrustAdmin{Store: packTrust, Keys: signerKeys, Audit: c.Audit}
	sessions := postgres.NewSessionStore(db, cfg.SessionTTL, cfg.SessionIdle)
	return &Production{
		Client: c, DB: db, Auth: authenticator, Sessions: sessions, Keys: keyStore,
		PackTrust: packTrust, SignerKeys: signerKeys, TrustAdmin: trustAdmin,
	}, nil
}

// UseProduction reports whether config selects the durable production path.
func UseProduction(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	if cfg.Env == "production" {
		return true
	}
	// Any non-development environment is production-like. Never allow a
	// staging/preview environment to silently fall back to in-memory storage.
	return cfg.Env != "" && cfg.Env != "development"
}
