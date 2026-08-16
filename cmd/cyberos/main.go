// Package main is the entry point for the CYBEROS CLI and optional API server.
package main

import (
	"fmt"
	"net/http"
	"os"

	apiv1 "cyberos/internal/api/v1"
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
