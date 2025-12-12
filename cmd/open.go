package cmd

import (
	"fmt"
	"os"

	"github.com/m44rten1/sprout/internal/editor"
	"github.com/m44rten1/sprout/internal/git"
	"github.com/m44rten1/sprout/internal/hooks"
	"github.com/m44rten1/sprout/internal/sprout"
	"github.com/m44rten1/sprout/internal/tui"

	"github.com/spf13/cobra"
)

var (
	syncFlag bool
)

var openCmd = &cobra.Command{
	Use:   "open [branch-or-path]",
	Short: "Open a worktree",
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
				// Cancelled or error
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
				if _, err := os.Stat(path); err != nil {
					fmt.Fprintf(os.Stderr, "No worktree found for branch '%s' at %s\n", arg, path)
					os.Exit(1)
				}
				targetPath = path
			}
		}

		// Open editor first, then run hooks
		// This allows user to start browsing code while hooks run
		if err := editor.Open(targetPath); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open editor: %v\n", err)
			os.Exit(1)
		}

		// Run on_open hooks if --sync flag is set
		if syncFlag {
			if err := hooks.RunHooks(repoRoot, targetPath, hooks.OnOpen); err != nil {
				if _, ok := err.(*hooks.UntrustedError); ok {
					hooks.PrintUntrustedMessage(repoRoot)
				} else {
					fmt.Fprintf(os.Stderr, "Error running hooks: %v\n", err)
					os.Exit(1)
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(openCmd)
	openCmd.Flags().BoolVar(&syncFlag, "sync", false, "Run on_open hooks before opening the worktree")
}
