package cmd

import (
	"fmt"
	"os"

	"github.com/m44rten1/sprout/internal/git"

	"github.com/spf13/cobra"
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Prune stale worktrees",
	Run: func(cmd *cobra.Command, args []string) {
		repoRoot, err := git.GetRepoRoot()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := git.PruneWorktrees(repoRoot); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to prune worktrees: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Pruned stale worktrees.")
	},
}

func init() {
	rootCmd.AddCommand(pruneCmd)
}
