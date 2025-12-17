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
		choices := filterSproutWorktreesAllRoots(worktrees)

		var completions []string
		for _, wt := range choices {
			if wt.Branch != "" {
				completions = append(completions, wt.Branch)
			}
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		repoRoot, err := git.GetRepoRoot()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
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

			// Filter to sprout worktrees
			choices := filterSproutWorktreesAllRoots(worktrees)

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
				// Assume it's a branch - search for it in worktrees
				worktrees, err := git.ListWorktrees(repoRoot)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to list worktrees: %v\n", err)
					os.Exit(1)
				}

				// Find matching sprout worktree by branch
				var found bool
				targetPath, found = findWorktreeByBranch(worktrees, arg)
				if !found {
					fmt.Fprintf(os.Stderr, "No sprout-managed worktree found for branch '%s'\n", arg)
					os.Exit(1)
				}
			}
		}

		// Verify the target is under a sprout root
		isSproutWorktree := false
		for _, sproutRoot := range sprout.GetAllPossibleSproutRoots() {
			if isUnderSproutRoot(targetPath, sproutRoot) {
				isSproutWorktree = true
				break
			}
		}
	if !isSproutWorktree {
		fmt.Fprintf(os.Stderr, "Refusing to remove non-sprout worktree: %s\n", targetPath)
		os.Exit(1)
	}

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
