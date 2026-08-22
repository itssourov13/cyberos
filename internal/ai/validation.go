package ai

import (
	"fmt"
	"strings"

	"cyberos/internal/contracts"
)

// Forbidden parameter keys (executable/path/shell injection vectors).
var forbiddenParamKeys = map[string]struct{}{
	"binary": {}, "executable": {}, "path": {}, "shell": {},
	"command": {}, "argv": {}, "interpreter": {}, "cmd": {},
}

// ValidatePlan fail-closed structural validation. Does not authorize or execute.
func ValidatePlan(p *Plan) error {
	if p == nil {
		return fmt.Errorf("%w: plan is nil", contracts.ErrInvalidArgument)
	}
	cv := p.ContractVersion
	if cv == "" {
		cv = PlanContractV1
	}
	if cv != PlanContractV1 {
		return fmt.Errorf("%w: unsupported plan contract version %q", contracts.ErrInvalidArgument, p.ContractVersion)
	}
	if err := contracts.ValidateNonEmpty("plan.id", p.ID); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("plan.goal", p.Goal); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("plan.project_id", string(p.ProjectID)); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("plan.scope_id", string(p.ScopeID)); err != nil {
		return err
	}
	if len(p.Steps) == 0 {
		return fmt.Errorf("%w: plan requires at least one step", contracts.ErrInvalidArgument)
	}
	seen := map[string]struct{}{}
	for i, s := range p.Steps {
		if err := contracts.ValidateNonEmpty(fmt.Sprintf("steps[%d].id", i), s.ID); err != nil {
			return err
		}
		if _, ok := seen[s.ID]; ok {
			return fmt.Errorf("%w: duplicate step id %q", contracts.ErrInvalidArgument, s.ID)
		}
		seen[s.ID] = struct{}{}
		if err := contracts.ValidateNonEmpty(fmt.Sprintf("steps[%d].capability_id", i), string(s.CapabilityID)); err != nil {
			return err
		}
		if looksLikeExecutable(string(s.CapabilityID)) {
			return fmt.Errorf("%w: capability_id must not look like a path or binary", contracts.ErrInvalidArgument)
		}
		if err := validateParams(s.Parameters); err != nil {
			return fmt.Errorf("steps[%d]: %w", i, err)
		}
		for _, dep := range s.DependsOn {
			if dep == s.ID {
				return fmt.Errorf("%w: step %q depends on itself", contracts.ErrInvalidArgument, s.ID)
			}
		}
	}
	// dependency existence + cycle detection
	for _, s := range p.Steps {
		for _, dep := range s.DependsOn {
			if _, ok := seen[dep]; !ok {
				return fmt.Errorf("%w: step %q depends on unknown step %q", contracts.ErrInvalidArgument, s.ID, dep)
			}
		}
	}
	if err := detectCycle(p.Steps); err != nil {
		return err
	}
	return nil
}

func validateParams(params map[string]any) error {
	for k, v := range params {
		lk := strings.ToLower(k)
		if _, bad := forbiddenParamKeys[lk]; bad {
			return fmt.Errorf("%w: forbidden parameter key %q", contracts.ErrInvalidArgument, k)
		}
		if s, ok := v.(string); ok {
			if looksLikeShell(s) {
				return fmt.Errorf("%w: parameter %q looks like shell/command injection", contracts.ErrInvalidArgument, k)
			}
		}
	}
	return nil
}

func looksLikeExecutable(s string) bool {
	if strings.HasPrefix(s, "/") || strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../") {
		return true
	}
	if strings.Contains(s, "\\") {
		return true
	}
	// bare known shell names as "capability"
	switch strings.ToLower(s) {
	case "nmap", "curl", "bash", "sh", "cmd", "powershell":
		return true
	}
	return false
}

func looksLikeShell(s string) bool {
	if strings.Contains(s, "&&") || strings.Contains(s, "||") || strings.Contains(s, ";") {
		return true
	}
	if strings.Contains(s, "`") || strings.Contains(s, "$(") {
		return true
	}
	if strings.Contains(s, "sh -c") || strings.Contains(s, "bash -c") {
		return true
	}
	return false
}

func detectCycle(steps []PlanStep) error {
	adj := map[string][]string{}
	for _, s := range steps {
		adj[s.ID] = append([]string{}, s.DependsOn...)
	}
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := map[string]int{}
	var dfs func(string) error
	dfs = func(u string) error {
		color[u] = gray
		for _, v := range adj[u] {
			switch color[v] {
			case gray:
				return fmt.Errorf("%w: dependency cycle involving %q", contracts.ErrInvalidArgument, u)
			case white:
				if err := dfs(v); err != nil {
					return err
				}
			}
		}
		color[u] = black
		return nil
	}
	for _, s := range steps {
		if color[s.ID] == white {
			if err := dfs(s.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// ValidateBrainRequest checks request shape (not authorization).
func ValidateBrainRequest(r BrainRequest) error {
	if r.Operation == "" {
		return fmt.Errorf("%w: operation required", contracts.ErrInvalidArgument)
	}
	switch r.Operation {
	case OpPlan, OpAnalyze, OpExplain:
	default:
		return fmt.Errorf("%w: unknown operation %q", contracts.ErrInvalidArgument, r.Operation)
	}
	if err := contracts.ValidateNonEmpty("objective", r.Objective); err != nil {
		return err
	}
	return nil
}
