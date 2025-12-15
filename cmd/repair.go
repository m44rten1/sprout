package cmd

import (
	"fmt"
	"os"

	"github.com/m44rten1/sprout/internal/git"
	"github.com/spf13/cobra"
)

var repairPruneFlag bool

var repairCmd = &cobra.Command{
	Use:   "repair",
	Short: "Repair git metadata for moved worktrees",
	Long: `Repair git metadata for all sprout-managed repositories.

This command runs 'git worktree repair' on all discovered repositories to fix
metadata for worktrees that have been moved (e.g., from ~/.sprout to ~/.local/share/sprout).

Note: Run 'repair' first, then optionally 'repair --prune' to remove truly deleted worktrees.
Pruning before repair may cause loss of metadata for moved worktrees.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Discover all repositories
		allRepos, err := discoverAllSproutRepos()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to discover repositories: %v\n", err)
			os.Exit(1)
		}

		if len(allRepos) == 0 {
			fmt.Println("No sprout-managed repositories found.")
			return
		}

		fmt.Printf("Found %d repository(ies) to repair...\n\n", len(allRepos))

		repairedCount := 0
		prunedCount := 0
		errorCount := 0

		for _, repo := range allRepos {
			repoName := getRepoName(repo.RepoRoot)
			fmt.Printf("📦 %s\n", repoName)

			// Run git worktree repair
			_, err := git.RunGitCommand(repo.RepoRoot, "worktree", "repair")
			if err != nil {
				fmt.Printf("   ⚠️  Failed to repair: %v\n", err)
				errorCount++
			} else {
				fmt.Printf("   ✅ Repaired worktree metadata\n")
				repairedCount++
			}

			// Optionally prune stale worktrees
			if repairPruneFlag {
				err := git.PruneWorktrees(repo.RepoRoot)
				if err != nil {
					fmt.Printf("   ⚠️  Failed to prune: %v\n", err)
				} else {
					fmt.Printf("   🧹 Pruned stale worktree references\n")
					prunedCount++
				}
			}

			fmt.Println()
		}

		// Summary
		fmt.Println("Summary:")
		fmt.Printf("  ✅ Repaired: %d\n", repairedCount)
		if repairPruneFlag {
			fmt.Printf("  🧹 Pruned: %d\n", prunedCount)
		}
		if errorCount > 0 {
			fmt.Printf("  ⚠️  Errors: %d\n", errorCount)
		}
	},
}

func init() {
	rootCmd.AddCommand(repairCmd)
	repairCmd.Flags().BoolVarP(&repairPruneFlag, "prune", "p", false, "Also prune stale worktree references")
}

// getRepoName extracts a simple name from the repo root path
func getRepoName(repoRoot string) string {
	parts := []rune(repoRoot)
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == '/' {
			return string(parts[i+1:])
		}
	}
	return repoRoot
}

