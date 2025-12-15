package cmd

import (
	"fmt"
	"os"

	"github.com/m44rten1/sprout/internal/config"
	"github.com/m44rten1/sprout/internal/git"
	"github.com/m44rten1/sprout/internal/hooks"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Run on_open hooks in the current worktree",
	Long: `Run the on_open hooks defined in .sprout.yml for the current worktree.

This is useful for:
- Freshening up a worktree before working (e.g., type-check, codegen)
- Running lightweight sync operations`,
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

		// Check if hooks exist but are untrusted
		untrusted, err := hooks.CheckAndPrintUntrusted(repoRoot, mainWorktreePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to check trust status: %v\n", err)
			os.Exit(1)
		}

		if untrusted {
			// Hooks exist but repo is not trusted
			os.Exit(1)
		}

		// Verify on_open hooks actually exist
		cfg, err := config.Load(repoRoot, mainWorktreePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
			os.Exit(1)
		}

		if !cfg.HasOpenHooks() {
			fmt.Fprintf(os.Stderr, "Error: no on_open hooks configured for this repository\n")
			os.Exit(1)
		}

		// Run on_open hooks
		if err := hooks.RunHooks(repoRoot, worktreePath, mainWorktreePath, hooks.OnOpen); err != nil {
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
	rootCmd.AddCommand(syncCmd)
}
