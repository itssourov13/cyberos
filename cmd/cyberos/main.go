// Package main is the entry point for the CYBEROS CLI and optional API server.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	apiv1 "cyberos/internal/api/v1"
	"cyberos/internal/auth"
	"cyberos/internal/cli"
	"cyberos/internal/config"
	"cyberos/internal/sdk"
)

func main() {
	cfg := config.Load()
	client := sdk.NewInMemory()

	if len(os.Args) > 1 && os.Args[1] == "serve" {
		addr := cfg.APIAddr
		if addr == "" {
			addr = ":8080"
		}
		srv := apiv1.NewServer(client)
		if cfg.AuthRequired {
			store := auth.NewMemoryKeyStore()
			a := auth.NewAPIKeyAuthenticator(store)
			plain, _, err := a.Issue(context.Background(), "bootstrap", "", 0)
			if err != nil {
				fmt.Fprintf(os.Stderr, "auth bootstrap failed: %v\n", err)
				os.Exit(1)
			}
			if cfg.BootstrapKey != "" {
				fmt.Fprintf(os.Stderr, "CYBEROS_BOOTSTRAP_API_KEY is set; issued runtime key (prefer Issue API in production):\n")
			}
			fmt.Fprintf(os.Stderr, "CYBEROS bootstrap API key (store securely): %s\n", plain)
			srv.Auth = auth.NewMiddleware(a)
			client.Auth = a
			fmt.Fprintf(os.Stderr, "CYBEROS API auth: REQUIRED (env=%s)\n", cfg.Env)
		} else {
			fmt.Fprintf(os.Stderr, "CYBEROS API auth: optional — set CYBEROS_AUTH_REQUIRED=true or CYBEROS_ENV=production\n")
		}
		fmt.Fprintf(os.Stderr, "CYBEROS API v1 listening on %s\n", addr)
		if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	app := cli.New(client)
	os.Exit(app.Run(os.Args[1:]))
}
