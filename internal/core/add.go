package core

import (
	"fmt"
	"path/filepath"

	"github.com/m44rten1/sprout/internal/config"
)

// Message constants for consistent UX
const (
	msgWorktreeExists   = "Worktree already exists at %s"
	msgCreatingWorktree = "Creating worktree for %s at %s..."
	msgWorktreeCreated  = "Worktree created!"
)

// AddContext contains all inputs needed to plan the add command.
// Config must not be nil.
type AddContext struct {
	Branch             string
	RepoRoot           string
	MainWorktreePath   string // Required for hooks
	WorktreePath       string
	WorktreeExists     bool
	LocalBranchExists  bool
	RemoteBranchExists bool
	HasOriginMain      bool
	Config             *config.Config // Must not be nil
	IsTrusted          bool
	NoHooks            bool
	NoOpen             bool
}

// PlanAddCommand creates a plan for adding/opening a worktree.
//
// Logic:
//  1. Validate inputs
//  2. If worktree exists, optionally open it (respecting NoOpen)
//  3. If creating new worktree with hooks, check trust
//  4. Build action sequence: create dir → git worktree add → editor/hooks (order varies)
func PlanAddCommand(ctx AddContext) Plan {
	// Validate inputs
	if ctx.RepoRoot == "" {
		return errorPlan(ErrEmptyRepoRoot)
	}
	if ctx.WorktreePath == "" {
		return errorPlan(ErrEmptyWorktreePath)
	}
	if ctx.Branch == "" {
		return errorPlan(ErrEmptyBranch)
	}
	if ctx.Config == nil {
		return errorPlan(ErrNilConfig)
	}

	// If worktree already exists, optionally open it (respecting NoOpen flag)
	if ctx.WorktreeExists {
		actions := []Action{
			PrintMessage{Msg: fmt.Sprintf(msgWorktreeExists, ctx.WorktreePath)},
		}
		if !ctx.NoOpen {
			actions = append(actions, OpenEditor{Path: ctx.WorktreePath})
		}
		return Plan{Actions: actions}
	}

	// Check trust requirements before creating worktree
	shouldRunHooks := ctx.Config.HasCreateHooks() && !ctx.NoHooks
	if shouldRunHooks {
		if ctx.MainWorktreePath == "" {
			return errorPlan(ErrEmptyMainWorktreePath)
		}
		if !ctx.IsTrusted {
			return errorPlan(ErrUntrustedWithHooks)
		}
	}

	// Build action sequence
	actions := []Action{
		PrintMessage{Msg: fmt.Sprintf(msgCreatingWorktree, ctx.Branch, ctx.WorktreePath)},
		CreateDirectory{
			Path: filepath.Dir(ctx.WorktreePath),
			Perm: 0755,
		},
		RunGitCommand{
			Dir:  ctx.RepoRoot,
			Args: WorktreeAddArgs(ctx.WorktreePath, ctx.Branch, ctx.LocalBranchExists, ctx.RemoteBranchExists, ctx.HasOriginMain),
		},
		PrintMessage{Msg: msgWorktreeCreated},
	}

	// Add hooks and editor based on configuration
	// Note: When hooks run, editor opens FIRST so user can browse while hooks execute in terminal
	// NoOpen flag is always respected, regardless of hooks
	if shouldRunHooks {
		if !ctx.NoOpen {
			actions = append(actions, OpenEditor{Path: ctx.WorktreePath})
		}
		actions = append(actions, RunHooks{
			Type:             HookTypeOnCreate,
			Commands:         ctx.Config.Hooks.OnCreate,
			Path:             ctx.WorktreePath,
			RepoRoot:         ctx.RepoRoot,
			MainWorktreePath: ctx.MainWorktreePath,
		})
	} else if !ctx.NoOpen {
		// No hooks: open editor after creation
		actions = append(actions, OpenEditor{Path: ctx.WorktreePath})
	}

	return Plan{Actions: actions}
}

// errorPlan creates a plan that prints an error and exits.
func errorPlan(err error) Plan {
	return Plan{Actions: []Action{
		PrintError{Msg: err.Error()},
		Exit{Code: 1},
	}}
}
