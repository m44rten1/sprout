package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/m44rten1/sprout/internal/core"
	"github.com/m44rten1/sprout/internal/effects"
	"github.com/m44rten1/sprout/internal/git"
	"github.com/spf13/cobra"
)

var (
	addNoHooksFlag bool
	addNoOpenFlag  bool
)

var addCmd = &cobra.Command{
	Use:   "add [branch]",
	Short: "Create a new worktree",
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

		branches, err := git.ListAllBranches(repoRoot)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		worktrees, err := git.ListWorktrees(repoRoot)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		// Reuse core logic to filter available branches
		availableBranches := core.GetWorktreeAvailableBranches(branches, worktrees)

		var completions []string
		for _, branch := range availableBranches {
			// Filter by what user has typed so far
			if strings.HasPrefix(branch.DisplayName, toComplete) {
				completions = append(completions, branch.DisplayName)
			}
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		fx := effects.NewRealEffects()

		ctx, err := BuildAddContext(fx, args, addNoHooksFlag, addNoOpenFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		plan := core.PlanAddCommand(ctx)
		if err := effects.ExecutePlan(plan, fx); err != nil {
			if code, ok := effects.IsExit(err); ok {
				os.Exit(code)
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

// BuildAddContext gathers all inputs needed to plan the add command.
// It handles interactive branch selection if no branch is provided.
func BuildAddContext(fx effects.Effects, args []string, noHooks, noOpen bool) (core.AddContext, error) {
	// Get repo root
	repoRoot, err := fx.GetRepoRoot()
	if err != nil {
		return core.AddContext{}, fmt.Errorf("not a git repository: %w", err)
	}

	// Get main worktree path for config loading and hooks
	mainWorktreePath, err := fx.GetMainWorktreePath()
	if err != nil {
		return core.AddContext{}, fmt.Errorf("failed to get main worktree: %w", err)
	}

	// Determine branch name (interactive or from args)
	var branch string
	if len(args) == 0 {
		// Interactive mode: select from existing branches
		branches, err := fx.ListBranches(repoRoot)
		if err != nil {
			return core.AddContext{}, fmt.Errorf("failed to list branches: %w", err)
		}

		worktrees, err := fx.ListWorktrees(repoRoot)
		if err != nil {
			return core.AddContext{}, fmt.Errorf("failed to list worktrees: %w", err)
		}

		availableBranches := core.GetWorktreeAvailableBranches(branches, worktrees)
		if len(availableBranches) == 0 {
			return core.AddContext{}, fmt.Errorf("no available branches found")
		}

		idx, err := fx.SelectBranch(availableBranches)
		if err != nil {
			return core.AddContext{}, fmt.Errorf("branch selection cancelled: %w", err)
		}

		branch = availableBranches[idx].DisplayName
	} else {
		branch = args[0]
	}

	// Strip remote prefix if user provided it (e.g., "origin/feature" -> "feature")
	branch = strings.TrimPrefix(branch, "origin/")

	// Calculate worktree path
	worktreePath, err := fx.GetWorktreePath(mainWorktreePath, branch)
	if err != nil {
		return core.AddContext{}, fmt.Errorf("error calculating worktree path: %w", err)
	}

	// Check if worktree already exists
	worktreeExists := fx.FileExists(worktreePath)

	// Check branch existence
	localBranchExists, err := fx.LocalBranchExists(repoRoot, branch)
	if err != nil {
		return core.AddContext{}, fmt.Errorf("failed to check local branch: %w", err)
	}

	remoteBranchExists, err := fx.RemoteBranchExists(repoRoot, branch)
	if err != nil {
		return core.AddContext{}, fmt.Errorf("failed to check remote branch: %w", err)
	}

	// Check if origin/main exists (used as base for new branches)
	// Note: RemoteBranchExists automatically prepends "origin/" prefix
	hasRemoteMain, err := fx.RemoteBranchExists(repoRoot, "main")
	if err != nil {
		return core.AddContext{}, fmt.Errorf("failed to check origin/main: %w", err)
	}

	// Load config
	cfg, err := fx.LoadConfig(repoRoot, mainWorktreePath)
	if err != nil {
		return core.AddContext{}, fmt.Errorf("failed to load config: %w", err)
	}

	// Check trust status (only matters if hooks will run)
	isTrusted := false
	if cfg.HasCreateHooks() && !noHooks {
		isTrusted, err = fx.IsTrusted(mainWorktreePath)
		if err != nil {
			return core.AddContext{}, fmt.Errorf("failed to check trust status: %w", err)
		}
	}

	return core.AddContext{
		Branch:             branch,
		RepoRoot:           repoRoot,
		MainWorktreePath:   mainWorktreePath,
		WorktreePath:       worktreePath,
		WorktreeExists:     worktreeExists,
		LocalBranchExists:  localBranchExists,
		RemoteBranchExists: remoteBranchExists,
		HasOriginMain:      hasRemoteMain,
		Config:             cfg,
		IsTrusted:          isTrusted,
		NoHooks:            noHooks,
		NoOpen:             noOpen,
	}, nil
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().BoolVar(&addNoHooksFlag, "no-hooks", false, "Skip running on_create hooks even if .sprout.yml exists")
	addCmd.Flags().BoolVar(&addNoOpenFlag, "no-open", false, "Skip opening the worktree in an editor")
}
