package cmd

import (
	"fmt"
	"os"

	"github.com/m44rten1/sprout/internal/git"
	"github.com/m44rten1/sprout/internal/sprout"
	"github.com/m44rten1/sprout/internal/tui"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove [branch-or-path]",
	Short: "Remove a worktree",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoRoot, err := git.GetRepoRoot()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		sproutRoot, err := sprout.GetWorktreeRoot(repoRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to determine sprout root: %v\n", err)
			os.Exit(1)
		}

		var targetPath string

		if len(args) == 0 {
			// Interactive mode
			worktrees, err := git.ListWorktrees(repoRoot)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to list worktrees: %v\n", err)
				os.Exit(1)
			}

			choices := filterSproutWorktrees(worktrees, sproutRoot)

			if len(choices) == 0 {
				fmt.Println("No sprout-managed worktrees found.")
				return
			}

			idx, err := tui.SelectOne(choices, func(wt git.Worktree) string {
				branch := wt.Branch
				if branch == "" {
					branch = "(detached)"
				}
				return fmt.Sprintf("%-30s %s", branch, wt.Path)
			}, nil)

			if err != nil {
				return
			}
			targetPath = choices[idx].Path

		} else {
			arg := args[0]
			// Check if it's a path
			if info, err := os.Stat(arg); err == nil && info.IsDir() {
				targetPath = arg
			} else {
				// Assume it's a branch
				path, err := sprout.GetWorktreePath(repoRoot, arg)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error calculating worktree path: %v\n", err)
					os.Exit(1)
				}
				// We don't necessarily need to check if it exists on disk, git will complain if not.
				// But let's check to be nice.
				if _, err := os.Stat(path); err != nil {
					fmt.Fprintf(os.Stderr, "No worktree found for branch '%s' at %s\n", arg, path)
					os.Exit(1)
				}
				targetPath = path
			}
		}

		if !isUnderSproutRoot(targetPath, sproutRoot) {
			fmt.Fprintf(os.Stderr, "Refusing to remove non-sprout worktree: %s\n", targetPath)
			os.Exit(1)
		}

		// TODO: Add force flag support
		force, _ := cmd.Flags().GetBool("force")

		if err := git.RemoveWorktree(repoRoot, targetPath, force); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to remove worktree: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Removed worktree at %s\n", targetPath)

		// Prune stale worktree references
		if err := git.PruneWorktrees(repoRoot); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to prune stale worktrees: %v\n", err)
		}
	},
}

func init() {
	removeCmd.Flags().Bool("force", false, "Force removal")
	rootCmd.AddCommand(removeCmd)
}
