// Package main is the entry point for the CYBEROS CLI and optional API server.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	apiv1 "cyberos/internal/api/v1"
	"cyberos/internal/auth"
	"cyberos/internal/cli"
	"cyberos/internal/config"
	"cyberos/internal/sdk"
)

func main() {
	cfg := config.Load()

	if len(os.Args) > 1 && os.Args[1] == "serve" {
		runServe(cfg)
		return
	}

	client := sdk.NewInMemory()
	app := cli.New(client)
	os.Exit(app.Run(os.Args[1:]))
}

func runServe(cfg *config.Config) {
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
		mig := findMigrationsDir()
		prod, err = sdk.NewProduction(ctx, cfg, mig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "production startup failed (fail-closed): %v\n", err)
			os.Exit(1)
		}
		defer func() { _ = prod.Close() }()
		client = prod.Client
		cfg.AuthRequired = true
		fmt.Fprintf(os.Stderr, "CYBEROS mode: production (PostgreSQL + durable repositories)\n")
	} else {
		client = sdk.NewInMemory()
		fmt.Fprintf(os.Stderr, "CYBEROS mode: development (in-memory)\n")
	}

	// Fail closed: a network-accessible listener must have mandatory authentication.
	if err := config.ValidateListenSecurity(addr, cfg.AuthRequired || cfg.Env == "production"); err != nil {
		fmt.Fprintf(os.Stderr, "unsafe server configuration (fail-closed): %v\n", err)
		os.Exit(1)
	}

	srv := apiv1.NewServer(client)
	if cfg.AuthRequired || cfg.Env == "production" {
		var authenticator *auth.APIKeyAuthenticator
		if prod != nil && prod.Auth != nil {
			authenticator = prod.Auth
		} else {
			store := auth.NewMemoryKeyStore()
			authenticator = auth.NewAPIKeyAuthenticator(store)
			client.Auth = authenticator
		}
		plain, created, berr := authenticator.EnsureBootstrap(ctx, cfg.BootstrapKey, "")
		if berr != nil {
			fmt.Fprintf(os.Stderr, "auth bootstrap failed (fail-closed): %v\n", berr)
			os.Exit(1)
		}
		if created {
			fmt.Fprintf(os.Stderr, "CYBEROS bootstrap API key created (store securely; will not re-issue on restart if keys remain active)\n")
			fmt.Fprintf(os.Stderr, "bootstrap_key=%s\n", plain)
		} else if cfg.BootstrapKey != "" {
			fmt.Fprintf(os.Stderr, "CYBEROS auth: using CYBEROS_BOOTSTRAP_API_KEY / existing store (secret not printed)\n")
		} else {
			fmt.Fprintf(os.Stderr, "CYBEROS auth: existing active API key(s) present (not re-issuing)\n")
		}
		srv.Auth = auth.NewMiddleware(authenticator)
		fmt.Fprintf(os.Stderr, "CYBEROS API auth: REQUIRED\n")
	} else {
		fmt.Fprintf(os.Stderr, "CYBEROS API auth: optional (set CYBEROS_AUTH_REQUIRED=true or CYBEROS_ENV=production)\n")
	}

	fmt.Fprintf(os.Stderr, "CYBEROS API v1 listening on %s\n", addr)
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: cfg.APIReadHeaderTimeout,
		ReadTimeout:       cfg.APIReadTimeout,
		WriteTimeout:      cfg.APIWriteTimeout,
		IdleTimeout:       cfg.APIIdleTimeout,
	}
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
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
