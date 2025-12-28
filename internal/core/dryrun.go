package core

import (
	"fmt"
	"strings"
)

// FormatPlan converts a Plan into a human-readable description of what will happen.
// This is used for --dry-run mode to show users what would be executed.
// Deterministic: same plan always produces the same output.
func FormatPlan(plan Plan) string {
	if len(plan.Actions) == 0 {
		return "No actions to perform."
	}

	// Preallocate to avoid reallocations
	lines := make([]string, 0, len(plan.Actions)+1)
	lines = append(lines, "Planned actions:")

	for i, action := range plan.Actions {
		prefix := fmt.Sprintf("  %d. ", i+1)
		formatted := formatAction(action)
		lines = append(lines, prefix+formatted)
	}

	return strings.Join(lines, "\n")
}

// formatAction converts a single action into a human-readable description.
func formatAction(action Action) string {
	switch a := action.(type) {
	case NoOp:
		return "No operation"

	case PrintMessage:
		return fmt.Sprintf("Print: %q", truncate(a.Msg, 60))

	case PrintError:
		return fmt.Sprintf("Print error: %q", truncate(a.Msg, 60))

	case CreateDirectory:
		return fmt.Sprintf("Create directory: %s", a.Path)

	case RunGitCommand:
		// Handle empty args edge case
		if len(a.Args) == 0 {
			if a.Dir != "" {
				return fmt.Sprintf("Run git in %s", a.Dir)
			}
			return "Run git"
		}
		args := strings.Join(a.Args, " ")
		if a.Dir != "" {
			return fmt.Sprintf("Run git command in %s: git %s", a.Dir, args)
		}
		return fmt.Sprintf("Run git command: git %s", args)

	case OpenEditor:
		return fmt.Sprintf("Open editor: %s", a.Path)

	case RunHooks:
		hookCount := len(a.Commands)
		return fmt.Sprintf("Run %d %s hook(s) in %s", hookCount, a.Type, a.Path)

	case TrustRepo:
		return fmt.Sprintf("Trust repository: %s", a.RepoRoot)

	case UntrustRepo:
		return fmt.Sprintf("Untrust repository: %s", a.RepoRoot)

	case PromptTrust:
		return fmt.Sprintf("Prompt to trust repository: %s (%d %s hooks)", a.MainWorktreePath, len(a.HookCommands), a.HookType)

	case SelectInteractive:
		return "Interactive selection (should not appear in execution plans)"

	case Exit:
		return fmt.Sprintf("Exit with code %d", a.Code)

	default:
		return fmt.Sprintf("Unknown action: %T", action)
	}
}

// truncate shortens a string to maxLen characters (byte-based), adding "..." if truncated.
// Multiline strings are truncated to the first line only.
// Note: operates on bytes, not runes, so may split multibyte UTF-8 characters.
func truncate(s string, maxLen int) string {
	// Show only first line
	if idx := strings.IndexAny(s, "\n\r"); idx != -1 {
		s = s[:idx]
	}

	if len(s) <= maxLen {
		return s
	}

	if maxLen <= 3 {
		return "..."
	}

	return s[:maxLen-3] + "..."
}
