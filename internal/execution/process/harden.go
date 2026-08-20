package process

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"cyberos/internal/contracts"
	"cyberos/internal/secrets"
)

// ShellBinaries are rejected even if somehow allowlisted unless explicitly
// permitted via Policy.AllowShell (default false).
var ShellBinaries = []string{
	"sh", "bash", "zsh", "csh", "tcsh", "fish", "ksh", "dash",
	"cmd", "cmd.exe", "powershell", "pwsh",
}

// ValidateArgv rejects shell-like argument patterns and empty binary paths.
// Arguments are always passed as argv array — never through a shell.
func ValidateArgv(path string, args []string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("%w: empty executable path", contracts.ErrInvalidArgument)
	}
	if strings.ContainsAny(path, " \t\n|&;<>$`\\\"'") {
		return fmt.Errorf("%w: executable path contains unsafe characters", contracts.ErrInvalidArgument)
	}
	// Disallow shell -c / -command style as first-class "command string" patterns in args
	for i, a := range args {
		if a == "" {
			continue
		}
		// Null bytes
		if strings.ContainsRune(a, 0) {
			return fmt.Errorf("%w: argument %d contains NUL", contracts.ErrInvalidArgument, i)
		}
		// Reject obvious multi-command injection fragments when used with shell bins
		if looksLikeShellCompound(a) {
			return fmt.Errorf("%w: argument %d looks like shell compound command", contracts.ErrInvalidArgument, i)
		}
	}
	return nil
}

func looksLikeShellCompound(a string) bool {
	// Only flag when clearly a compound command string, not normal tool flags.
	if strings.Contains(a, "&&") || strings.Contains(a, "||") || strings.Contains(a, ";") {
		return true
	}
	if strings.Contains(a, "`") || strings.Contains(a, "$(") {
		return true
	}
	return false
}

// IsShellBinary reports whether basename is a shell/interpreter.
func IsShellBinary(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	for _, s := range ShellBinaries {
		if base == s {
			return true
		}
	}
	return false
}

// SanitizeEnv builds an explicit env list without inheriting the full host environment.
// Secret-looking keys are dropped; values are not logged by this function.
func SanitizeEnv(in map[string]string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for k, v := range in {
		if k == "" {
			continue
		}
		lk := strings.ToLower(k)
		if strings.Contains(lk, "password") || strings.Contains(lk, "secret") ||
			strings.Contains(lk, "token") || strings.Contains(lk, "api_key") ||
			strings.Contains(lk, "authorization") {
			// Do not pass secrets via env unless using secrets.Ref resolution at a higher layer.
			_ = secrets.Redacted
			continue
		}
		// Basic key charset
		if !isSafeEnvKey(k) {
			continue
		}
		out = append(out, k+"="+v)
	}
	return out
}

func isSafeEnvKey(k string) bool {
	for _, r := range k {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			continue
		}
		return false
	}
	return true
}
