package core

import (
	"github.com/m44rten1/sprout/internal/config"
)

// Message constants for consistent UX
const (
	MsgNoSproutWorktrees = "No sprout-managed worktrees found."
)

// OpenContext contains all inputs needed to plan the open command.
// Config must not be nil.
type OpenContext struct {
	TargetPath       string
	RepoRoot         string
	MainWorktreePath string         // Required for hooks
	Config           *config.Config // Must not be nil
	IsTrusted        bool
	NoHooks          bool
}

// PlanOpenCommand creates a plan for opening a worktree.
//
// If open hooks are configured and not disabled via --no-hooks, the repository
// must be trusted or the plan will prompt for trust interactively. This security
// check prevents untrusted repositories from executing arbitrary commands. In
// interactive terminals, users can grant trust inline; in non-interactive environments,
// they must run `sprout trust` explicitly or use --no-hooks.
//
// Logic:
//  1. Validate inputs
//  2. Open editor in target path
//  3. If hooks configured, trusted, and not disabled: run on_open hooks
func PlanOpenCommand(ctx OpenContext) Plan {
	// Validate inputs
	if ctx.TargetPath == "" {
		return errorPlan(ErrEmptyTargetPath)
	}
	if ctx.RepoRoot == "" {
		return errorPlan(ErrNoRepoRoot)
	}
	if ctx.Config == nil {
		return errorPlan(ErrNilConfig)
	}

	// Check trust requirements before any actions
	// This is a security feature: if hooks are configured and enabled,
	// we fail fast before opening the editor to avoid partial success states
	shouldRunHooks := ctx.Config.HasOpenHooks() && !ctx.NoHooks
	if shouldRunHooks {
		if ctx.MainWorktreePath == "" {
			return errorPlan(ErrEmptyMainWorktreePath)
		}
		if !ctx.IsTrusted {
			// Return a plan that prompts for trust interactively
			// If prompt fails (non-interactive), it will error with helpful guidance
			return Plan{Actions: []Action{
				PromptTrust{
					MainWorktreePath: ctx.MainWorktreePath,
					HookType:         HookTypeOnOpen,
					HookCommands:     ctx.Config.Hooks.OnOpen,
				},
				OpenEditor{Path: ctx.TargetPath},
				RunHooks{
					Type:             HookTypeOnOpen,
					Commands:         ctx.Config.Hooks.OnOpen,
					Path:             ctx.TargetPath,
					RepoRoot:         ctx.RepoRoot,
					MainWorktreePath: ctx.MainWorktreePath,
				},
			}}
		}
	}

	// Open editor first, then run hooks
	// This allows user to start browsing code while hooks run
	actions := []Action{
		OpenEditor{Path: ctx.TargetPath},
	}

	// Run on_open hooks if configured, trusted, and not disabled
	if shouldRunHooks {
		actions = append(actions, RunHooks{
			Type:             HookTypeOnOpen,
			Commands:         ctx.Config.Hooks.OnOpen,
			Path:             ctx.TargetPath,
			RepoRoot:         ctx.RepoRoot,
			MainWorktreePath: ctx.MainWorktreePath,
		})
	}

	return Plan{Actions: actions}
}
