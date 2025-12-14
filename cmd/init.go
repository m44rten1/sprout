package cmd

import (
	"fmt"
	"os"

	"github.com/m44rten1/sprout/internal/git"
	"github.com/m44rten1/sprout/internal/hooks"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Run on_create hooks in the current worktree",
	Long: `Run the on_create hooks defined in .sprout.yml for the current worktree.

This is useful for:
- Running bootstrap manually in an existing worktree
- Recovering from a failed initial run`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get current directory
		worktreePath, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get current directory: %v\n", err)
			os.Exit(1)
		}

		// Get repo root
		repoRoot, err := git.GetRepoRoot()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Get main worktree path for config fallback
		mainWorktreePath, err := git.GetMainWorktreePath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get main worktree path: %v\n", err)
			os.Exit(1)
		}

		// Run on_create hooks
		if err := hooks.RunHooks(repoRoot, worktreePath, mainWorktreePath, hooks.OnCreate); err != nil {
			if _, ok := err.(*hooks.UntrustedError); ok {
				hooks.PrintUntrustedMessage(mainWorktreePath)
				os.Exit(1)
			} else {
				fmt.Fprintf(os.Stderr, "Error running hooks: %v\n", err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
