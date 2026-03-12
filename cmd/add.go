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
	addNoHooksFlag            bool
	addNoOpenFlag             bool
	addFromFlag               string
	addFromCurrentFlag        bool
	addMoveCurrentChangesFlag bool
)

const fromPickerValue = "?"

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

		fromChanged := cmd.Flags().Changed("from")
		ctx, err := BuildAddContext(fx, args, addNoHooksFlag, addNoOpenFlag, addFromFlag, fromChanged, addFromCurrentFlag, addMoveCurrentChangesFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		runAddContext(ctx, fx)
	},
}

// BuildAddContext gathers all inputs needed to plan the add command.
// It handles interactive branch selection if no branch is provided.
// fromValue and fromChanged represent the --from flag state:
//   - fromChanged=false: default behavior (origin/main or HEAD)
//   - fromChanged=true, fromValue="?": interactive from-branch picker
//   - fromChanged=true, fromValue=<branch>: use that branch as base
//
// fromCurrent=true uses the currently checked-out branch as the base ref.
func BuildAddContext(fx effects.Effects, args []string, noHooks, noOpen bool, fromValue string, fromChanged bool, fromCurrent bool, moveCurrentChanges bool) (core.AddContext, error) {
	// Validate: --from and --from-current are mutually exclusive
	if fromChanged && fromCurrent {
		return core.AddContext{}, fmt.Errorf("--from and --from-current cannot be used together")
	}

	// Validate: --from / --from-current require an explicit branch name
	if (fromChanged || fromCurrent) && len(args) == 0 {
		return core.AddContext{}, fmt.Errorf("--from/--from-current requires a branch name argument\n\nUsage: sprout add <branch> --from [base-branch]")
	}

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

	// Resolve base ref for new branches
	fromRef, err := resolveFromRef(fx, repoRoot, fromValue, fromChanged, fromCurrent)
	if err != nil {
		return core.AddContext{}, err
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

	currentWorktreeDirty := false
	if moveCurrentChanges {
		currentWorktreeDirty, err = fx.IsWorktreeDirty(repoRoot)
		if err != nil {
			return core.AddContext{}, fmt.Errorf("failed to check current worktree status: %w", err)
		}
	}

	return core.AddContext{
		Branch:               branch,
		RepoRoot:             repoRoot,
		MainWorktreePath:     mainWorktreePath,
		WorktreePath:         worktreePath,
		WorktreeExists:       worktreeExists,
		CurrentWorktreeDirty: currentWorktreeDirty,
		LocalBranchExists:    localBranchExists,
		RemoteBranchExists:   remoteBranchExists,
		FromRef:              fromRef,
		Config:               cfg,
		IsTrusted:            isTrusted,
		NoHooks:              noHooks,
		NoOpen:               noOpen,
		MoveCurrentChanges:   moveCurrentChanges,
	}, nil
}

// resolveFromRef determines the base ref for new branch creation.
func resolveFromRef(fx effects.Effects, repoRoot string, fromValue string, fromChanged bool, fromCurrent bool) (string, error) {
	// --from-current: use the currently checked-out branch
	if fromCurrent {
		branch, err := fx.GetCurrentBranch(repoRoot)
		if err != nil {
			return "", fmt.Errorf("failed to get current branch: %w", err)
		}
		return branch, nil
	}

	// Explicit --from <branch> (not the picker trigger)
	if fromChanged && fromValue != fromPickerValue {
		return fromValue, nil
	}

	// Interactive --from ? : show picker
	if fromChanged {
		branches, err := fx.ListBranches(repoRoot)
		if err != nil {
			return "", fmt.Errorf("failed to list branches: %w", err)
		}
		if len(branches) == 0 {
			return "", fmt.Errorf("no branches found")
		}

		idx, err := fx.SelectFromBranch(branches)
		if err != nil {
			return "", fmt.Errorf("base branch selection cancelled: %w", err)
		}

		return branches[idx].DisplayName, nil
	}

	// Default: origin/main or HEAD
	hasRemoteMain, err := fx.RemoteBranchExists(repoRoot, "main")
	if err != nil {
		return "", fmt.Errorf("failed to check origin/main: %w", err)
	}
	if hasRemoteMain {
		return "origin/main", nil
	}
	return "HEAD", nil
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().BoolVar(&addNoHooksFlag, "no-hooks", false, "Skip running on_create hooks even if .sprout.yml exists")
	addCmd.Flags().BoolVar(&addNoOpenFlag, "no-open", false, "Skip opening the worktree in an editor")
	addCmd.Flags().StringVar(&addFromFlag, "from", "", `Base branch to create the new branch from (use "?" for interactive picker)`)
	addCmd.Flags().BoolVar(&addFromCurrentFlag, "from-current", false, "Use the currently checked-out branch as the base")
	addCmd.Flags().BoolVar(&addMoveCurrentChangesFlag, "move-current-changes", false, "Move current uncommitted changes into the new worktree")
}
