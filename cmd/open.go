package cmd

import (
	"fmt"
	"os"

	"github.com/m44rten1/sprout/internal/config"
	"github.com/m44rten1/sprout/internal/core"
	"github.com/m44rten1/sprout/internal/editor"
	"github.com/m44rten1/sprout/internal/git"
	"github.com/m44rten1/sprout/internal/hooks"
	"github.com/m44rten1/sprout/internal/sprout"
	"github.com/m44rten1/sprout/internal/tui"

	"github.com/spf13/cobra"
)

var (
	openNoHooksFlag bool
)

var openCmd = &cobra.Command{
	Use:   "open [branch-or-path]",
	Short: "Open a worktree",
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
		sproutRoot, err := sprout.GetSproutRoot()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		choices := core.FilterSproutWorktrees(worktrees, sproutRoot)

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
			sproutRoot, err := sprout.GetSproutRoot()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to get sprout root: %v\n", err)
				os.Exit(1)
			}
			choices := core.FilterSproutWorktrees(worktrees, sproutRoot)

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
				// Assume it's a branch - search for it in worktrees
				worktrees, err := git.ListWorktrees(repoRoot)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to list worktrees: %v\n", err)
					os.Exit(1)
				}

				// Find matching sprout worktree by branch
				sproutRoot, err := sprout.GetSproutRoot()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to get sprout root: %v\n", err)
					os.Exit(1)
				}
				var found bool
				targetPath, found = core.FindWorktreeByBranch(worktrees, sproutRoot, arg)
				if !found {
					fmt.Fprintf(os.Stderr, "No sprout-managed worktree found for branch '%s'\n", arg)
					os.Exit(1)
				}
			}
		}

		// Get main worktree path for config loading
		mainWorktreePath, err := git.GetMainWorktreePath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get main worktree path: %v\n", err)
			os.Exit(1)
		}

		// Check if .sprout.yml exists and has on_open hooks
		cfg, err := config.Load(repoRoot, mainWorktreePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
			os.Exit(1)
		}

		shouldRunHooks := cfg.HasOpenHooks() && !openNoHooksFlag

		// If hooks should run, verify repo is trusted before opening
		if shouldRunHooks {
			untrusted, err := hooks.CheckAndPrintUntrusted(repoRoot, mainWorktreePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to check trust status: %v\n", err)
				os.Exit(1)
			}

			if untrusted {
				// Hooks exist but repo is not trusted - abort
				os.Exit(1)
			}
		}

		// Open editor first, then run hooks
		// This allows user to start browsing code while hooks run
		if err := editor.Open(targetPath); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open editor: %v\n", err)
			os.Exit(1)
		}

		// Run on_open hooks automatically if they exist and repo is trusted
		if shouldRunHooks {
			if err := hooks.RunHooks(repoRoot, targetPath, mainWorktreePath, hooks.OnOpen); err != nil {
				if _, ok := err.(*hooks.UntrustedError); ok {
					hooks.PrintUntrustedMessage(mainWorktreePath)
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
	openCmd.Flags().BoolVar(&openNoHooksFlag, "no-hooks", false, "Skip running on_open hooks even if .sprout.yml exists")
}
