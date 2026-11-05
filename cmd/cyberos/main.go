// Package main is the entry point for the CYBEROS CLI and optional API server.
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	apiv1 "cyberos/internal/api/v1"
	"cyberos/internal/auth"
	"cyberos/internal/cli"
	"cyberos/internal/config"
	"cyberos/internal/execution"
	"cyberos/internal/execution/mtls"
	"cyberos/internal/sdk"
	"cyberos/internal/storage/memory"
	"cyberos/internal/storage/postgres"
)

func main() {
	cfg := config.Load()
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		if err := runServe(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "CYBEROS serve failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	client := sdk.NewInMemory()
	app := cli.New(client)
	os.Exit(app.Run(os.Args[1:]))
}

func runServe(cfg *config.Config) error {
	addr := cfg.APIAddr
	if addr == "" {
		addr = config.DefaultAPIAddr
	}
	ctx := context.Background()

	var (
		client *sdk.Client
		prod   *sdk.Production
		err    error
	)
	if cfg.Env == "production" || sdk.UseProduction(cfg) {
		prod, err = sdk.NewProduction(ctx, cfg, findMigrationsDir())
		if err != nil {
			return fmt.Errorf("production startup (fail-closed): %w", err)
		}
		defer func() { _ = prod.Close() }()
		client = prod.Client
		cfg.AuthRequired = true
		fmt.Fprintln(os.Stderr, "CYBEROS mode: durable PostgreSQL")
	} else {
		client = sdk.NewInMemory()
		fmt.Fprintln(os.Stderr, "CYBEROS mode: development (in-memory)")
	}

	if err := config.ValidateListenSecurity(addr, cfg.AuthRequired || cfg.Env == "production"); err != nil {
		return fmt.Errorf("unsafe server configuration: %w", err)
	}

	srv := apiv1.NewServer(client)
	if prod != nil {
		srv.TrustAdmin = prod.TrustAdmin
	}
	if client.Hub != nil {
		srv.WithEventHub(client.Hub)
	}
	if prod != nil && prod.DB != nil {
		srv.Workers = execution.NewWorkerRegistry(postgres.NewWorkerRepo(prod.DB))
		srv.Assignments = execution.NewAssignmentService(postgres.NewAssignmentRepo(prod.DB))
		srv.WithReadinessCheck(func(ctx context.Context) error {
			return prod.DB.PingContext(ctx)
		})
	} else {
		srv.Workers = execution.NewWorkerRegistry(memory.NewWorkerRepo())
		srv.Assignments = execution.NewAssignmentService(memory.NewAssignmentRepo())
		srv.WithReadinessCheck(func(context.Context) error { return nil })
	}

	if cfg.AuthRequired || sdk.UseProduction(cfg) {
		var authenticator *auth.APIKeyAuthenticator
		if prod != nil && prod.Auth != nil {
			authenticator = prod.Auth
		} else {
			store := auth.NewMemoryKeyStore()
			authenticator = auth.NewAPIKeyAuthenticator(store)
			client.Auth = authenticator
		}

		if prod != nil && cfg.BootstrapKey == "" {
			operators, countErr := authenticator.Store.CountActiveOperators(ctx)
			if countErr != nil {
				return fmt.Errorf("count active operator API keys: %w", countErr)
			}
			if operators == 0 {
				return fmt.Errorf("no active platform operator key exists: set CYBEROS_BOOTSTRAP_API_KEY for first startup; secrets are never printed to logs")
			}
		}
		plain, created, err := authenticator.EnsureBootstrap(ctx, cfg.BootstrapKey, "")
		if err != nil {
			return fmt.Errorf("auth bootstrap (fail-closed): %w", err)
		}
		if created && prod != nil {
			// Defensive invariant: production must never emit a newly generated
			// credential to stdout/stderr or a container log collector.
			return fmt.Errorf("production bootstrap unexpectedly generated an implicit key; refusing to start")
		}
		if created {
			fmt.Fprintln(os.Stderr, "development bootstrap API key created; store the following secret securely (shown once)")
			fmt.Fprintf(os.Stderr, "bootstrap_key=%s\n", plain)
		}
		srv.Auth = auth.NewMiddleware(authenticator)
		if prod != nil && prod.Sessions != nil {
			srv.Auth.Sessions = prod.Sessions
		} else if cfg.SessionBackend == "memory" && cfg.Env == "development" {
			srv.Auth.Sessions = auth.NewSessionStore(cfg.SessionTTL, cfg.SessionIdle)
		} else {
			return fmt.Errorf("durable session storage is required when authentication protects a network listener")
		}
		srv.Auth.AllowedOrigins = cfg.AllowedOrigins
		srv.Auth.CookieSecure = cfg.CookieSecure || cfg.Env == "production"
		fmt.Fprintln(os.Stderr, "CYBEROS API auth: required")
	}

	var tlsConfig *tls.Config
	if cfg.TLSCertFile != "" {
		tlsConfig, err = mtls.OptionalClientCertTLSConfig(cfg.TLSCertFile, cfg.TLSKeyFile, cfg.WorkerCACertFile)
		if err != nil {
			return fmt.Errorf("configure HTTPS/mTLS: %w", err)
		}
		if cfg.WorkerCACertFile != "" {
			fmt.Fprintln(os.Stderr, "CYBEROS worker mTLS: enabled (client certificates verified against configured worker CA)")
		} else {
			fmt.Fprintln(os.Stderr, "CYBEROS worker mTLS: disabled (no worker CA configured)")
		}
	} else if cfg.WorkerCACertFile != "" {
		return fmt.Errorf("worker mTLS requires direct HTTPS: configure TLS cert and key files")
	}

	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: cfg.APIReadHeaderTimeout,
		ReadTimeout:       cfg.APIReadTimeout,
		WriteTimeout:      cfg.APIWriteTimeout,
		IdleTimeout:       cfg.APIIdleTimeout,
		TLSConfig:         tlsConfig,
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}
	if tlsConfig != nil {
		listener = tls.NewListener(listener, tlsConfig)
		fmt.Fprintf(os.Stderr, "CYBEROS HTTPS API listening on %s\n", addr)
	} else {
		fmt.Fprintf(os.Stderr, "CYBEROS HTTP API listening on %s (place behind a trusted TLS reverse proxy in production)\n", addr)
	}

	serveErr := make(chan error, 1)
	go func() { serveErr <- httpSrv.Serve(listener) }()
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP server: %w", err)
		}
		return nil
	case <-signalCtx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			_ = httpSrv.Close()
			return fmt.Errorf("graceful HTTP shutdown: %w", err)
		}
		if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP server shutdown: %w", err)
		}
		return nil
	}
}

func findMigrationsDir() string {
	dir, _ := os.Getwd()
	for i := 0; i < 6; i++ {
		p := filepath.Join(dir, "migrations")
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
		dir = filepath.Dir(dir)
	}
	return "migrations"
}
