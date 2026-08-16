// Package main is the entry point for the CYBEROS CLI binary.
// Phase 0 Foundation: minimal placeholder only.
// No domain logic, no execution, no tools.
package main

import (
	"fmt"
	"os"

	"cyberos/pkg/version"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "version" || os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println(version.String())
		return
	}

	fmt.Println("CYBEROS — Cybersecurity Operating System")
	fmt.Println("Foundation phase. Core not yet implemented.")
	fmt.Println("Run with 'version' for build information.")
	fmt.Println()
	fmt.Println("Security invariant is non-bypassable.")
	fmt.Println("See docs/ARCHITECTURE.md and docs/SECURITY_MODEL.md")
}
