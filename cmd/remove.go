package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/m44rten1/sprout/internal/core"
	"github.com/m44rten1/sprout/internal/effects"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove [branch-or-path]",
	Short: "Remove a worktree",
	Long: `Remove a worktree and optionally delete associated branches.

By default, only the worktree is removed. Use flags to also delete branches:
  -d    Delete local branch (safe: requires branch to be merged into main)
  -D    Force delete local branch (even if not merged)
  --delete-remote  Also delete remote branch (requires -d or -D)

Examples:
  sprout remove feature/login           # Remove worktree only
  sprout remove -d feature/login        # Remove worktree + delete local branch (safe)
  sprout remove -D feature/login        # Remove worktree + force delete local branch
  sprout remove -d --delete-remote feature/login  # Remove worktree + local + remote branch
  sprout remove --dry-run -D --delete-remote feature/login  # Preview what would happen`,
	Args: cobra.MaximumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Only complete the first argument
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		// Use effects interface for consistency
		fx := effects.NewRealEffects()

		repoRoot, err := fx.GetRepoRoot()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		worktrees, err := fx.ListWorktrees(repoRoot)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		sproutRoot, err := fx.GetWorktreeRoot(repoRoot)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		choices := core.FilterSproutWorktrees(worktrees, sproutRoot)

		var completions []string
		for _, wt := range choices {
			if wt.Branch != "" {
				completions = append(completions, wt.Branch)
			}
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		fx := effects.NewRealEffects()

		// Get all flags (errors impossible for registered bool flags)
		force, _ := cmd.Flags().GetBool("force")
		deleteBranch, _ := cmd.Flags().GetBool("delete-branch")
		forceDelete, _ := cmd.Flags().GetBool("force-delete")
		deleteRemote, _ := cmd.Flags().GetBool("delete-remote")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// Build context
		ctx, err := BuildRemoveContext(fx, args, RemoveFlags{
			Force:        force,
			DeleteBranch: deleteBranch,
			ForceDelete:  forceDelete,
			DeleteRemote: deleteRemote,
			DryRun:       dryRun,
		})
		if err != nil {
			// Handle specific errors with better UX
			if errors.Is(err, core.ErrNoSproutWorktrees) {
				fmt.Println("No sprout-managed worktrees found.")
				os.Exit(1)
			}
			if errors.Is(err, core.ErrSelectionCancelled) {
				// Silent exit for cancelled selection (user pressed Ctrl+C)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Plan and execute
		plan := core.PlanRemoveCommand(ctx)
		runPlan(plan, fx)
	},
}

// RemoveFlags holds all flag values for the remove command.
type RemoveFlags struct {
	Force        bool // -f/--force: force remove worktree with uncommitted changes
	DeleteBranch bool // -d/--delete-branch: delete local branch (safe)
	ForceDelete  bool // -D: force delete local branch (implies DeleteBranch)
	DeleteRemote bool // --delete-remote: also delete remote branch
	DryRun       bool // --dry-run: preview actions without executing
}

// BuildRemoveContext gathers all inputs needed for the remove command.
//
// It handles three input modes:
// - Interactive selection if no argument provided
// - Branch name lookup (tries to match against worktree branches)
// - Direct path (if argument is an existing file/directory)
//
// Path vs branch disambiguation: If the argument exists as a file/directory,
// it's treated as a path; otherwise it's treated as a branch name. This means
// a branch name that matches a file in CWD will be interpreted as a path.
// This is acceptable for a worktree tool where explicit paths are uncommon.
func BuildRemoveContext(fx effects.Effects, args []string, flags RemoveFlags) (core.RemoveContext, error) {
	// Get repository root
	repoRoot, err := fx.GetRepoRoot()
	if err != nil {
		return core.RemoveContext{}, fmt.Errorf("failed to get repository root: %w", err)
	}

	// Get sprout root
	sproutRoot, err := fx.GetWorktreeRoot(repoRoot)
	if err != nil {
		return core.RemoveContext{}, fmt.Errorf("failed to get sprout root: %w", err)
	}

	// Get all worktrees
	worktrees, err := fx.ListWorktrees(repoRoot)
	if err != nil {
		return core.RemoveContext{}, fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Filter to sprout-managed worktrees
	sproutWorktrees := core.FilterSproutWorktrees(worktrees, sproutRoot)

	var targetPath string
	var branchName string
	var argProvided bool
	var arg string

	if len(args) == 0 {
		// Interactive mode
		if len(sproutWorktrees) == 0 {
			return core.RemoveContext{}, core.ErrNoSproutWorktrees
		}

		idx, err := fx.SelectWorktree(sproutWorktrees)
		if err != nil {
			return core.RemoveContext{}, core.ErrSelectionCancelled
		}
		targetPath = sproutWorktrees[idx].Path
		branchName = sproutWorktrees[idx].Branch

	} else {
		// Argument provided
		argProvided = true
		arg = args[0]

		// Disambiguate: path (if exists) vs branch name
		if fx.FileExists(arg) {
			targetPath = arg
			// Find branch name for this path
			for _, wt := range worktrees {
				if wt.Path == arg {
					branchName = wt.Branch
					break
				}
			}
		} else {
			// Assume it's a branch - search for it in worktrees
			var found bool
			targetPath, found = core.FindWorktreeByBranch(worktrees, sproutRoot, arg)
			if !found {
				return core.RemoveContext{}, fmt.Errorf("no sprout-managed worktree found for branch '%s'", arg)
			}
			branchName = arg
		}
	}

	ctx := core.RemoveContext{
		ArgProvided:  argProvided,
		Arg:          arg,
		RepoRoot:     repoRoot,
		SproutRoot:   sproutRoot,
		Worktrees:    worktrees,
		TargetPath:   targetPath,
		Force:        flags.Force,
		DeleteBranch: flags.DeleteBranch,
		ForceDelete:  flags.ForceDelete,
		DeleteRemote: flags.DeleteRemote,
		DryRun:       flags.DryRun,
		BranchName:   branchName,
	}

	// If we need to delete branches, gather branch info
	wantDeleteBranch := flags.DeleteBranch || flags.ForceDelete
	if wantDeleteBranch && branchName != "" {
		ctx = gatherBranchInfo(fx, ctx)
	}

	return ctx, nil
}

// gatherBranchInfo populates branch-related fields in the context.
// This is the "imperative shell" part - gathering data before planning.
//
// Errors are intentionally swallowed here (best-effort gathering):
// - If we can't check branch existence, we assume it doesn't exist
// - If we can't check merge status, we assume not merged (safer)
// - The planner will handle missing/default values gracefully
func gatherBranchInfo(fx effects.Effects, ctx core.RemoveContext) core.RemoveContext {
	// Check if local branch exists (best-effort: assume false on error)
	exists, err := fx.LocalBranchExists(ctx.RepoRoot, ctx.BranchName)
	if err == nil {
		ctx.BranchExists = exists
	}

	// Check if merged into main (best-effort: assume false on error, which is safer)
	if ctx.BranchExists {
		merged, err := fx.IsBranchMergedIntoMain(ctx.RepoRoot, ctx.BranchName)
		if err == nil {
			ctx.IsMerged = merged
		}
	}

	// Get upstream info for remote deletion
	if ctx.DeleteRemote {
		upstream := fx.GetBranchUpstream(ctx.RepoRoot, ctx.BranchName)
		ctx.RemoteName = upstream.Remote
		ctx.RemoteBranch = upstream.RemoteBranch
		ctx.HasUpstream = upstream.Configured

		// Check if remote branch exists (best-effort: assume false on error)
		exists, err := fx.RemoteBranchExistsOn(ctx.RepoRoot, upstream.Remote, upstream.RemoteBranch)
		if err == nil {
			ctx.RemoteBranchExists = exists
		}

		// Check for unpushed commits (best-effort: assume false on error)
		if ctx.BranchExists && ctx.RemoteBranchExists {
			hasUnpushed, err := fx.HasUnpushedCommits(ctx.RepoRoot, ctx.BranchName, upstream.Remote, upstream.RemoteBranch)
			if err == nil {
				ctx.HasUnpushed = hasUnpushed
			}
		}
	}

	return ctx
}

func init() {
	// Worktree removal flags
	removeCmd.Flags().BoolP("force", "f", false, "Force removal even if worktree has uncommitted changes")

	// Branch deletion flags
	removeCmd.Flags().BoolP("delete-branch", "d", false, "Delete local branch (safe: requires merge into main)")
	removeCmd.Flags().BoolP("force-delete", "D", false, "Force delete local branch even if not merged (implies -d)")
	removeCmd.Flags().Bool("delete-remote", false, "Also delete remote branch (requires -d or -D)")

	// Preview flag
	removeCmd.Flags().Bool("dry-run", false, "Preview what would be deleted without actually doing it")

	rootCmd.AddCommand(removeCmd)
}
