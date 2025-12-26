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
	Args:  cobra.MaximumNArgs(1),
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

		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Build context
		ctx, err := BuildRemoveContext(fx, args, force)
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

		// Plan
		plan := core.PlanRemoveCommand(ctx)

		// Dry-run mode: print plan instead of executing
		if dryRunFlag {
			fmt.Println(core.FormatPlan(plan))
			return
		}

		// Execute plan
		if err := effects.ExecutePlan(plan, fx); err != nil{
			if code, ok := effects.IsExit(err); ok {
				os.Exit(code)
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
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
func BuildRemoveContext(fx effects.Effects, args []string, force bool) (core.RemoveContext, error) {
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

	} else {
		// Argument provided
		argProvided = true
		arg = args[0]

		// Disambiguate: path (if exists) vs branch name
		if fx.FileExists(arg) {
			targetPath = arg
		} else {
			// Assume it's a branch - search for it in worktrees
			var found bool
			targetPath, found = core.FindWorktreeByBranch(worktrees, sproutRoot, arg)
			if !found {
				return core.RemoveContext{}, fmt.Errorf("no sprout-managed worktree found for branch '%s'", arg)
			}
		}
	}

	return core.RemoveContext{
		ArgProvided: argProvided,
		Arg:         arg,
		RepoRoot:    repoRoot,
		SproutRoot:  sproutRoot,
		Worktrees:   worktrees,
		TargetPath:  targetPath,
		Force:       force,
	}, nil
}

func init() {
	removeCmd.Flags().Bool("force", false, "Force removal")
	rootCmd.AddCommand(removeCmd)
}
