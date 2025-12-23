package cmd

import (
	"fmt"
	"os"

	"github.com/m44rten1/sprout/internal/core"
	"github.com/m44rten1/sprout/internal/effects"
	"github.com/m44rten1/sprout/internal/git"

	"github.com/spf13/cobra"
)

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

		// Gather inputs: determine repo root
		var repoRoot string
		var err error

		if len(args) > 0 {
			// Trust the specified path - verify it's a git repo
			repoRoot = args[0]
			_, err := git.RunGitCommand(repoRoot, "rev-parse", "--show-toplevel")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s is not a git repository\n", repoRoot)
				os.Exit(1)
			}
		} else {
			// Trust current repo - use main worktree path
			repoRoot, err = fx.GetMainWorktreePath()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}

		// Check if already trusted
		isTrusted, err := fx.IsTrusted(repoRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to check trust status: %v\n", err)
			os.Exit(1)
		}

		// Build context and plan
		ctx := core.TrustContext{
			RepoRoot:       repoRoot,
			AlreadyTrusted: isTrusted,
		}

		plan := core.PlanTrustCommand(ctx)

		// Execute plan
		if err := effects.ExecutePlan(plan, fx); err != nil {
			if code, ok := effects.IsExit(err); ok {
				os.Exit(code)
			}
			fmt.Fprintf(os.Stderr, "Execution failed: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(trustCmd)
}
