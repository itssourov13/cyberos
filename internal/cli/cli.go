// Package cli implements the CYBEROS command-line interface.
// CLI commands call the same SDK/services as the API — no duplicated business logic.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"cyberos/internal/contracts"
	"cyberos/internal/controlplane"
	"cyberos/internal/sdk"
	"cyberos/pkg/version"
)

// App is the CLI application.
type App struct {
	SDK *sdk.Client
}

// New constructs a CLI app.
func New(client *sdk.Client) *App {
	return &App{SDK: client}
}

// Run dispatches commands from args (typically os.Args[1:]).
func (a *App) Run(args []string) int {
	if len(args) == 0 {
		printHelp()
		return 0
	}
	switch args[0] {
	case "version", "--version", "-v":
		fmt.Println(version.String())
		return 0
	case "help", "--help", "-h":
		printHelp()
		return 0
	case "health":
		fmt.Println("ok")
		return 0
	case "capabilities":
		return a.cmdCapabilities(args[1:])
	case "evaluate":
		return a.cmdEvaluate(args[1:])
	case "grant":
		return a.cmdGrant(args[1:])
	case "execute":
		return a.cmdExecute(args[1:])
	case "projects":
		return a.cmdProjects()
	case "scopes":
		return a.cmdScopes(args[1:])
	case "tasks":
		return a.cmdTasks(args[1:])
	case "dashboard":
		return a.cmdDashboard()
	case "assessments":
		return a.cmdAssessments(args[1:])
	case "tools":
		return a.cmdTools(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		printHelp()
		return 1
	}
}

func printHelp() {
	fmt.Print(`CYBEROS CLI

Usage:
  cyberos version
  cyberos health
  cyberos capabilities list
  cyberos capabilities register <id> <name> <type>
  cyberos evaluate --principal <id> --scope <id> --project <id> --capability <id> [--task <id>]
  cyberos grant --principal <id> --scope <id> --project <id> --capability <id> [--task <id>]
  cyberos execute --grant-json <file>
  cyberos projects
  cyberos scopes [--project <id>]
  cyberos tasks [--project <id>] [--scope <id>]
  cyberos dashboard
  cyberos assessments list|create|plan|run|get
  cyberos tools list|health|info|packs
  cyberos serve

Sensitive operations use Control Plane → Grant → Broker → Sandbox.
`)
}

func (a *App) cmdCapabilities(args []string) int {
	ctx := context.Background()
	if len(args) == 0 || args[0] == "list" {
		list, err := a.SDK.ListCapabilities(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(list)
		return 0
	}
	if args[0] == "register" && len(args) >= 4 {
		err := a.SDK.RegisterCapability(ctx, args[1], args[2], contracts.CapabilityType(args[3]))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		fmt.Println("registered", args[1])
		return 0
	}
	fmt.Fprintln(os.Stderr, "usage: capabilities list | register <id> <name> <type>")
	return 1
}

func (a *App) cmdEvaluate(args []string) int {
	req, err := parseEvalFlags(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	d, err := a.SDK.Evaluate(context.Background(), req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(d)
	if !d.Allowed() {
		return 2
	}
	return 0
}

func (a *App) cmdGrant(args []string) int {
	req, err := parseEvalFlags(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	g, d, err := a.SDK.IssueGrant(context.Background(), req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		_ = json.NewEncoder(os.Stdout).Encode(d)
		return 2
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]any{"grant": g, "decision": d})
	return 0
}

func (a *App) cmdExecute(args []string) int {
	var path string
	for i := 0; i < len(args); i++ {
		if args[i] == "--grant-json" && i+1 < len(args) {
			path = args[i+1]
		}
	}
	if path == "" {
		fmt.Fprintln(os.Stderr, "usage: execute --grant-json <file>")
		return 1
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	var grant contracts.ExecutionGrant
	if err := json.Unmarshal(data, &grant); err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid grant json: %v\n", err)
		return 1
	}
	res, err := a.SDK.Execute(grant, nil)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(res)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 2
	}
	return 0
}

func parseEvalFlags(args []string) (controlplane.EvaluateRequest, error) {
	req := controlplane.EvaluateRequest{
		CorrelationID: contracts.CorrelationID("cli-" + strings.ReplaceAll(fmt.Sprint(os.Getpid()), " ", "")),
	}
	for i := 0; i < len(args); i++ {
		key := args[i]
		if !strings.HasPrefix(key, "--") || i+1 >= len(args) {
			continue
		}
		val := args[i+1]
		i++
		switch key {
		case "--principal":
			req.PrincipalID = contracts.PrincipalID(val)
		case "--project":
			req.ProjectID = contracts.ProjectID(val)
		case "--scope":
			req.ScopeID = contracts.ScopeID(val)
		case "--task":
			req.TaskID = contracts.TaskID(val)
		case "--capability":
			req.CapabilityID = contracts.CapabilityID(val)
		case "--correlation":
			req.CorrelationID = contracts.CorrelationID(val)
		}
	}
	if req.PrincipalID == "" || req.ScopeID == "" || req.CapabilityID == "" {
		return req, fmt.Errorf("--principal, --scope, and --capability are required")
	}
	return req, nil
}

func (a *App) cmdProjects() int {
	if a.SDK == nil || a.SDK.Catalog == nil {
		fmt.Println("[]")
		return 0
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(a.SDK.Catalog.ListProjects())
	return 0
}

func (a *App) cmdScopes(args []string) int {
	projectID := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--project" && i+1 < len(args) {
			projectID = args[i+1]
		}
	}
	if a.SDK == nil || a.SDK.Catalog == nil {
		fmt.Println("[]")
		return 0
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(a.SDK.Catalog.ListScopes(contracts.ProjectID(projectID)))
	return 0
}

func (a *App) cmdTasks(args []string) int {
	projectID, scopeID := "", ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--project" && i+1 < len(args) {
			projectID = args[i+1]
		}
		if args[i] == "--scope" && i+1 < len(args) {
			scopeID = args[i+1]
		}
	}
	if a.SDK == nil || a.SDK.Catalog == nil {
		fmt.Println("[]")
		return 0
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(a.SDK.Catalog.ListTasks(contracts.ProjectID(projectID), contracts.ScopeID(scopeID)))
	return 0
}

func (a *App) cmdDashboard() int {
	projects, scopes, tasks := 0, 0, 0
	if a.SDK != nil && a.SDK.Catalog != nil {
		projects = len(a.SDK.Catalog.ListProjects())
		scopes = len(a.SDK.Catalog.ListScopes(""))
		tasks = len(a.SDK.Catalog.ListTasks("", ""))
	}
	fmt.Printf("projects=%d scopes=%d tasks=%d\n", projects, scopes, tasks)
	fmt.Println("security=Scope→Principal→ControlPlane→Grant→Broker→Sandbox")
	return 0
}

func (a *App) cmdAssessments(args []string) int {
	if len(args) == 0 {
		fmt.Println("usage: assessments list|create|plan|run|get")
		return 1
	}
	switch args[0] {
	case "list":
		if a.SDK == nil || a.SDK.Catalog == nil {
			fmt.Println("[]")
			return 0
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(a.SDK.Catalog.ListAssessments())
		return 0
	case "get":
		if len(args) < 2 {
			fmt.Println("usage: assessments get <id>")
			return 1
		}
		asmt, err := a.SDK.GetAssessment(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(asmt)
		return 0
	case "create":
		// assessments create <id> <project> <scope> <principal> <title> [target]
		if len(args) < 6 {
			fmt.Println("usage: assessments create <id> <project> <scope> <principal> <title> [target]")
			return 1
		}
		target := ""
		if len(args) > 6 {
			target = args[6]
		}
		asmt, err := a.SDK.CreateAssessment(args[1], args[2], args[3], args[4], args[5], target)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(asmt.ID, asmt.State)
		return 0
	case "plan":
		// assessments plan <id> <capability>
		if len(args) < 3 {
			fmt.Println("usage: assessments plan <id> <capability_id>")
			return 1
		}
		steps := []contracts.AssessmentStep{{ID: "s1", CapabilityID: contracts.CapabilityID(args[2])}}
		asmt, err := a.SDK.PlanAssessment(args[1], steps)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(asmt.ID, asmt.State, len(asmt.Steps))
		return 0
	case "run":
		if len(args) < 2 {
			fmt.Println("usage: assessments run <id>")
			return 1
		}
		asmt, err := a.SDK.RunAssessment(context.Background(), args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(asmt.ID, asmt.State, len(asmt.StepResults))
		return 0
	default:
		fmt.Println("unknown assessments subcommand")
		return 1
	}
}

func (a *App) cmdTools(args []string) int {
	if a.SDK == nil || a.SDK.Platform == nil {
		fmt.Fprintln(os.Stderr, "tool platform not configured")
		return 1
	}
	p := a.SDK.Platform
	if len(args) == 0 {
		fmt.Println("usage: tools list|health|info <id>|packs")
		return 1
	}
	switch args[0] {
	case "list":
		for _, d := range p.Discover() {
			fmt.Printf("%s\t%s\t%s\t%s\n", d.ID, d.Name, d.Runtime, d.ExecutorID)
		}
		return 0
	case "health":
		st := p.HealthSweep(context.Background())
		for id, s := range st {
			fmt.Printf("%s\t%s\t%s\t%s\n", id, s.Availability, s.Health.Status, s.Health.Message)
		}
		return 0
	case "info":
		if len(args) < 2 {
			fmt.Println("usage: tools info <tool-id>")
			return 1
		}
		ad, err := p.Resolve(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(ad.Descriptor())
		return 0
	case "packs":
		for _, pack := range p.Packs().List() {
			fmt.Printf("%s\t%s\t%v\n", pack.ID, pack.Name, pack.ToolIDs)
		}
		return 0
	default:
		fmt.Println("unknown tools subcommand")
		return 1
	}
}
