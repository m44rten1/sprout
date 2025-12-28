package cmd

import (
	"fmt"
	"os"

	"github.com/m44rten1/sprout/internal/core"
	"github.com/m44rten1/sprout/internal/effects"

	"github.com/spf13/cobra"
)

// BuildTrustContext gathers all inputs needed to plan the trust command.
// It uses the provided effects to determine the repository root and trust status.
// If pathArg is empty, it uses the current repository.
func BuildTrustContext(fx effects.Effects, pathArg string) (core.TrustContext, error) {
	var repoRoot string
	var err error

	if pathArg != "" {
		// Trust the specified path - verify it's a git repo
		repoRoot = pathArg
		_, err := fx.RunGitCommand(repoRoot, "rev-parse", "--show-toplevel")
		if err != nil {
			return core.TrustContext{}, fmt.Errorf("not a git repository: %s", repoRoot)
		}
	} else {
		// Trust current repo - use main worktree path
		repoRoot, err = fx.GetMainWorktreePath()
		if err != nil {
			return core.TrustContext{}, fmt.Errorf("get main worktree: %w", err)
		}
	}

	// Check if already trusted
	isTrusted, err := fx.IsTrusted(repoRoot)
	if err != nil {
		return core.TrustContext{}, fmt.Errorf("check trust status: %w", err)
	}

	return core.TrustContext{
		RepoRoot:       repoRoot,
		AlreadyTrusted: isTrusted,
	}, nil
}

var trustCmd = &cobra.Command{
	Use:   "trust [path]",
	Short: "Trust a repository to run hooks",
	Long: `Mark a repository as trusted to allow running hooks defined in .sprout.yml.

If no path is provided, the current repository is trusted.

WARNING: Only trust repositories you control or have reviewed the .sprout.yml file for.
Hooks can execute arbitrary commands on your system.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Create effects
		fx := effects.NewRealEffects()

		// Determine path argument
		var pathArg string
		if len(args) > 0 {
			pathArg = args[0]
		}

		// Build context from effects
		ctx, err := BuildTrustContext(fx, pathArg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Plan and execute
		plan := core.PlanTrustCommand(ctx)
		runPlan(plan, fx)
	},
}

func init() {
	rootCmd.AddCommand(trustCmd)
}
