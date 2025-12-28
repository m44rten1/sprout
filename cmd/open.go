package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/m44rten1/sprout/internal/core"
	"github.com/m44rten1/sprout/internal/effects"
	"github.com/m44rten1/sprout/internal/git"
	"github.com/m44rten1/sprout/internal/sprout"

	"github.com/spf13/cobra"
)

var (
	openNoHooksFlag bool
)

var openCmd = &cobra.Command{
	Use:   "open [branch-or-path]",
	Short: "Open a worktree",
	Args:  cobra.MaximumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Only complete the first argument
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		repoRoot, err := git.GetRepoRoot()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		worktrees, err := git.ListWorktrees(repoRoot)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		// Filter to sprout worktrees
		sproutRoot, err := sprout.GetSproutRoot()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		choices := core.FilterSproutWorktrees(worktrees, sproutRoot)

		var completions []string
		for _, wt := range choices {
			if wt.Branch != "" {
				// Filter by what user has typed so far for smarter completion
				if toComplete == "" || strings.HasPrefix(wt.Branch, toComplete) {
					completions = append(completions, wt.Branch)
				}
			}
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		fx := effects.NewRealEffects()

		ctx, err := BuildOpenContext(fx, args, openNoHooksFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		plan := core.PlanOpenCommand(ctx)
		runPlan(plan, fx)
	},
}

// BuildOpenContext gathers all inputs needed to plan the open command.
// It handles interactive selection if no argument is provided.
func BuildOpenContext(fx effects.Effects, args []string, noHooks bool) (core.OpenContext, error) {
	// Get repo root
	repoRoot, err := fx.GetRepoRoot()
	if err != nil {
		return core.OpenContext{}, fmt.Errorf("not a git repository: %w", err)
	}

	// Get main worktree path for config loading and hooks
	mainWorktreePath, err := fx.GetMainWorktreePath()
	if err != nil {
		return core.OpenContext{}, fmt.Errorf("failed to get main worktree: %w", err)
	}

	// Get sprout root once - used for filtering worktrees
	sproutRoot, err := sprout.GetSproutRoot()
	if err != nil {
		return core.OpenContext{}, fmt.Errorf("failed to get sprout root: %w", err)
	}

	var targetPath string

	if len(args) == 0 {
		// Interactive mode: select from sprout worktrees
		worktrees, err := fx.ListWorktrees(repoRoot)
		if err != nil {
			return core.OpenContext{}, fmt.Errorf("failed to list worktrees: %w", err)
		}

		choices := core.FilterSproutWorktrees(worktrees, sproutRoot)

		if len(choices) == 0 {
			return core.OpenContext{}, fmt.Errorf(core.MsgNoSproutWorktrees)
		}

		idx, err := fx.SelectWorktree(choices)
		if err != nil {
			return core.OpenContext{}, fmt.Errorf("selection cancelled: %w", err)
		}

		targetPath = choices[idx].Path
	} else {
		arg := args[0]
		// Check if it's a path (paths take precedence over branch names)
		if fx.FileExists(arg) {
			targetPath = arg
		} else {
			// Assume it's a branch - search for it in worktrees
			worktrees, err := fx.ListWorktrees(repoRoot)
			if err != nil {
				return core.OpenContext{}, fmt.Errorf("failed to list worktrees: %w", err)
			}

			var found bool
			targetPath, found = core.FindWorktreeByBranch(worktrees, sproutRoot, arg)
			if !found {
				return core.OpenContext{}, fmt.Errorf("no sprout-managed worktree found for branch '%s'", arg)
			}
		}
	}

	// Load config
	cfg, err := fx.LoadConfig(repoRoot, mainWorktreePath)
	if err != nil {
		return core.OpenContext{}, fmt.Errorf("failed to load config: %w", err)
	}

	// Check trust status (only matters if hooks will run)
	isTrusted := false
	if cfg.HasOpenHooks() && !noHooks {
		isTrusted, err = fx.IsTrusted(mainWorktreePath)
		if err != nil {
			return core.OpenContext{}, fmt.Errorf("failed to check trust status: %w", err)
		}
	}

	return core.OpenContext{
		TargetPath:       targetPath,
		RepoRoot:         repoRoot,
		MainWorktreePath: mainWorktreePath,
		Config:           cfg,
		IsTrusted:        isTrusted,
		NoHooks:          noHooks,
	}, nil
}

func init() {
	rootCmd.AddCommand(openCmd)
	openCmd.Flags().BoolVar(&openNoHooksFlag, "no-hooks", false, "Skip running on_open hooks even if .sprout.yml exists")
}
