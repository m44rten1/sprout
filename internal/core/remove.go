package core

import (
	"errors"
	"fmt"

	"github.com/m44rten1/sprout/internal/git"
)

// Message constants for remove command
const (
	msgRemovedWorktree            = "Removed worktree at %s"
	msgWouldRemoveWorktree        = "Would remove worktree at %s"
	msgDeletedLocalBranch         = "Deleted local branch %s"
	msgWouldDeleteLocalBranch     = "Would delete local branch %s"
	msgDeletedRemoteBranch        = "Deleted remote branch %s/%s"
	msgWouldDeleteRemoteBranch    = "Would delete remote branch %s/%s"
	msgBranchNotFound             = "Branch %s not found, skipping branch deletion"
	msgRemoteBranchNotFound       = "Remote branch %s/%s not found, skipping"
	msgNoUpstreamAssumeOrigin     = "No upstream configured for %s, assuming %s/%s"
	msgUnpushedCommitsWarning     = "Warning: branch %s has unpushed commits"
	errRefuseNonSprout            = "Refusing to remove non-sprout worktree: %s"
	errDeleteRemoteRequiresBranch = "--delete-remote requires -d or -D flag"
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
	Force        bool // Force removal even if worktree has uncommitted changes (-f/--force)
	DeleteBranch bool // Delete local branch after removing worktree (-d)
	ForceDelete  bool // Force delete local branch even if not merged (-D, implies DeleteBranch)
	DeleteRemote bool // Also delete remote branch (--delete-remote)
	DryRun       bool // Preview actions without executing (--dry-run)

	// Branch info (populated by shell layer after gathering context)
	BranchName         string // Branch associated with the worktree
	DefaultBranch      string // Resolved default branch name (e.g. main, master, dev)
	BranchExists       bool   // Whether the local branch exists
	IsMerged           bool   // Whether branch is merged into main
	MergeCheckError    string // Error determining merge status for safe deletion
	RemoteName         string // e.g., "origin"
	RemoteBranch       string // e.g., "feature/login"
	RemoteBranchExists bool   // Whether the remote branch exists
	HasUpstream        bool   // Whether upstream is configured
	HasUnpushed        bool   // Whether branch has unpushed commits
}

// PlanRemoveCommand creates a plan to remove a worktree and optionally delete branches.
//
// The command flow is:
// 1. Validate target path is under a sprout root (safety check)
// 2. Validate flag combinations (--delete-remote requires -d or -D)
// 3. If dry-run: return plan with preview messages only
// 4. Remove the worktree using git
// 5. If -d or -D: attempt to delete local branch with safety checks
// 6. If --delete-remote: attempt to delete remote branch (fail-soft)
// 7. Prune stale worktree references
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

	// Validate flag combinations
	if ctx.DeleteRemote && !ctx.DeleteBranch && !ctx.ForceDelete {
		return errorPlan(errors.New(errDeleteRemoteRequiresBranch))
	}

	// -D implies -d
	wantDeleteBranch := ctx.DeleteBranch || ctx.ForceDelete
	forceDeleteBranch := ctx.ForceDelete

	// Dry-run mode: just show what would happen
	if ctx.DryRun {
		return planDryRun(ctx, wantDeleteBranch)
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
	}

	// Local branch deletion
	if wantDeleteBranch {
		actions = append(actions, planLocalBranchDeletion(ctx, forceDeleteBranch)...)
	}

	// Remote branch deletion
	if ctx.DeleteRemote && wantDeleteBranch {
		actions = append(actions, planRemoteBranchDeletion(ctx, forceDeleteBranch)...)
	}

	// Prune stale worktree references
	actions = append(actions, RunGitCommand{
		Dir:  ctx.RepoRoot,
		Args: []string{"worktree", "prune"},
	})

	return Plan{Actions: actions}
}

// planDryRun creates a plan that only prints what would happen.
func planDryRun(ctx RemoveContext, wantDeleteBranch bool) Plan {
	actions := []Action{
		PrintMessage{Msg: fmt.Sprintf(msgWouldRemoveWorktree, ctx.TargetPath)},
	}

	if wantDeleteBranch {
		if !ctx.BranchExists {
			actions = append(actions, PrintMessage{
				Msg: fmt.Sprintf(msgBranchNotFound, ctx.BranchName),
			})
		} else {
			actions = append(actions, PrintMessage{
				Msg: fmt.Sprintf(msgWouldDeleteLocalBranch, ctx.BranchName),
			})
		}
	}

	if ctx.DeleteRemote && wantDeleteBranch && ctx.BranchExists {
		if !ctx.RemoteBranchExists {
			actions = append(actions, PrintMessage{
				Msg: fmt.Sprintf(msgRemoteBranchNotFound, ctx.RemoteName, ctx.RemoteBranch),
			})
		} else {
			actions = append(actions, PrintMessage{
				Msg: fmt.Sprintf(msgWouldDeleteRemoteBranch, ctx.RemoteName, ctx.RemoteBranch),
			})
		}
	}

	return Plan{Actions: actions}
}

// planLocalBranchDeletion creates actions for deleting the local branch.
func planLocalBranchDeletion(ctx RemoveContext, forceDelete bool) []Action {
	// Branch doesn't exist: print skip message
	if !ctx.BranchExists {
		return []Action{
			PrintMessage{Msg: fmt.Sprintf(msgBranchNotFound, ctx.BranchName)},
		}
	}

	if !forceDelete && ctx.MergeCheckError != "" {
		return []Action{
			PrintError{Msg: mergeCheckErrorMessage(ctx.BranchName, ctx.MergeCheckError)},
			Exit{Code: 1},
		}
	}

	// Check if branch is merged (unless force delete)
	if !forceDelete && !ctx.IsMerged {
		return []Action{
			PrintError{Msg: branchNotMergedMessage(ctx.BranchName, ctx.DefaultBranch)},
			Exit{Code: 1},
		}
	}

	// Delete the branch
	actions := []Action{
		DeleteLocalBranch{
			RepoRoot: ctx.RepoRoot,
			Branch:   ctx.BranchName,
			Force:    forceDelete,
		},
	}

	// Success message
	if forceDelete && !ctx.IsMerged {
		actions = append(actions, PrintMessage{
			Msg: deletedLocalBranchForceMessage(ctx.BranchName, ctx.DefaultBranch),
		})
	} else {
		actions = append(actions, PrintMessage{
			Msg: fmt.Sprintf(msgDeletedLocalBranch, ctx.BranchName),
		})
	}

	return actions
}

func branchNotMergedMessage(branchName, defaultBranch string) string {
	return fmt.Sprintf(
		"Branch %s is not merged into %s.\nUse -D to force delete, or merge it first.",
		branchName,
		defaultBranchLabel(defaultBranch),
	)
}

func deletedLocalBranchForceMessage(branchName, defaultBranch string) string {
	return fmt.Sprintf(
		"Deleted local branch %s (was not merged into %s)",
		branchName,
		defaultBranchLabel(defaultBranch),
	)
}

func mergeCheckErrorMessage(branchName, mergeCheckError string) string {
	return fmt.Sprintf(
		"Could not verify whether branch %s is safe to delete.\n%s",
		branchName,
		mergeCheckError,
	)
}

func defaultBranchLabel(defaultBranch string) string {
	if defaultBranch == "" {
		return "the default branch"
	}
	return defaultBranch
}

// planRemoteBranchDeletion creates actions for deleting the remote branch.
func planRemoteBranchDeletion(ctx RemoveContext, forceDelete bool) []Action {
	var actions []Action

	// Print upstream fallback notice if applicable
	if !ctx.HasUpstream {
		actions = append(actions, PrintMessage{
			Msg: fmt.Sprintf(msgNoUpstreamAssumeOrigin, ctx.BranchName, ctx.RemoteName, ctx.RemoteBranch),
		})
	}

	// Remote branch doesn't exist: print skip message
	if !ctx.RemoteBranchExists {
		return append(actions, PrintMessage{
			Msg: fmt.Sprintf(msgRemoteBranchNotFound, ctx.RemoteName, ctx.RemoteBranch),
		})
	}

	// Check for unpushed commits (unless force delete)
	if !forceDelete && ctx.HasUnpushed {
		return append(actions,
			PrintError{Msg: fmt.Sprintf("Branch %s has unpushed commits.\nUse -D --delete-remote to force delete.", ctx.BranchName)},
			Exit{Code: 1},
		)
	}

	// Warn about unpushed commits if force deleting
	if forceDelete && ctx.HasUnpushed {
		actions = append(actions, PrintMessage{
			Msg: fmt.Sprintf(msgUnpushedCommitsWarning, ctx.BranchName),
		})
	}

	// Delete the remote branch
	actions = append(actions,
		DeleteRemoteBranch{
			RepoRoot:     ctx.RepoRoot,
			Remote:       ctx.RemoteName,
			RemoteBranch: ctx.RemoteBranch,
		},
		PrintMessage{
			Msg: fmt.Sprintf(msgDeletedRemoteBranch, ctx.RemoteName, ctx.RemoteBranch),
		},
	)

	return actions
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
