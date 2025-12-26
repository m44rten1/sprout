package core

import (
	"fmt"

	"github.com/m44rten1/sprout/internal/git"
)

// Message constants for remove command
const (
	msgRemovedWorktree = "Removed worktree at %s"
	errRefuseNonSprout = "Refusing to remove non-sprout worktree: %s"
)

// RemoveContext contains all inputs needed to plan a remove command.
//
// Note: This context mixes "shell layer" fields (ArgProvided, Arg, Worktrees)
// with "planner" fields (RepoRoot, SproutRoot, TargetPath, Force). This is
// intentional and consistent with other commands - the shell builds the full
// context, while the planner only uses the subset it needs.
type RemoveContext struct {
	// Inputs provided by caller (used by shell for resolution, not planner)
	ArgProvided bool   // Whether user provided an argument (branch or path)
	Arg         string // Branch name or path (if ArgProvided is true)

	// Context gathered from environment
	RepoRoot   string         // Repository root path
	SproutRoot string         // Sprout root directory for this repo
	Worktrees  []git.Worktree // All worktrees in the repo (used by shell, not planner)

	// Resolved target (after branch lookup or interactive selection)
	TargetPath string // Final worktree path to remove

	// Flags
	Force bool // Force removal even if worktree has uncommitted changes
}

// PlanRemoveCommand creates a plan to remove a worktree.
//
// The command flow is:
// 1. Validate target path is under a sprout root (safety check)
// 2. Remove the worktree using git
// 3. Print success message
// 4. Prune stale worktree references
//
// Note: Currently prune failures will fail the entire plan. To make this truly
// "best-effort" (warn but continue), we would need a RunGitCommandBestEffort
// action type that prints warnings instead of propagating errors.
//
// Returns an error plan if validation fails.
func PlanRemoveCommand(ctx RemoveContext) Plan {
	// Validation: empty repo root
	if ctx.RepoRoot == "" {
		return errorPlan(ErrEmptyRepoRoot)
	}

	// Validation: empty target path
	if ctx.TargetPath == "" {
		return errorPlan(ErrEmptyTargetPath)
	}

	// Safety check: verify target is under a sprout root
	if !IsUnderSproutRoot(ctx.TargetPath, ctx.SproutRoot) {
		return errorPlan(fmt.Errorf(errRefuseNonSprout, ctx.TargetPath))
	}

	// Build action sequence
	actions := []Action{
		// Remove the worktree
		RunGitCommand{
			Dir:  ctx.RepoRoot,
			Args: buildRemoveWorktreeArgs(ctx.TargetPath, ctx.Force),
		},
		// Print success message
		PrintMessage{
			Msg: fmt.Sprintf(msgRemovedWorktree, ctx.TargetPath),
		},
		// Prune stale worktree references
		RunGitCommand{
			Dir:  ctx.RepoRoot,
			Args: []string{"worktree", "prune"},
		},
	}

	return Plan{Actions: actions}
}

// buildRemoveWorktreeArgs constructs arguments for 'git worktree remove'.
// If force is true, adds --force flag to remove even with uncommitted changes.
func buildRemoveWorktreeArgs(path string, force bool) []string {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	return args
}
