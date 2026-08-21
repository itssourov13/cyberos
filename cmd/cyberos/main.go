// Package main is the entry point for the CYBEROS CLI and optional API server.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

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
		addr = ":8080"
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

	srv := apiv1.NewServer(client)
	if cfg.AuthRequired || cfg.Env == "production" {
		var authenticator auth.Authenticator
		if prod != nil && prod.Auth != nil {
			authenticator = prod.Auth
			plain, _, ierr := prod.Auth.Issue(ctx, "bootstrap", "", 24*time.Hour)
			if ierr != nil {
				fmt.Fprintf(os.Stderr, "auth bootstrap failed (fail-closed): %v\n", ierr)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "CYBEROS bootstrap API key issued (store securely)\n")
			fmt.Fprintf(os.Stderr, "bootstrap_key=%s\n", plain)
		} else {
			store := auth.NewMemoryKeyStore()
			a := auth.NewAPIKeyAuthenticator(store)
			plain, _, ierr := a.Issue(ctx, "bootstrap", "", 0)
			if ierr != nil {
				fmt.Fprintf(os.Stderr, "auth bootstrap failed: %v\n", ierr)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "bootstrap_key=%s\n", plain)
			authenticator = a
			client.Auth = a
		}
		srv.Auth = auth.NewMiddleware(authenticator)
		fmt.Fprintf(os.Stderr, "CYBEROS API auth: REQUIRED\n")
	} else {
		fmt.Fprintf(os.Stderr, "CYBEROS API auth: optional (set CYBEROS_AUTH_REQUIRED=true or CYBEROS_ENV=production)\n")
	}

	fmt.Fprintf(os.Stderr, "CYBEROS API v1 listening on %s\n", addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
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
