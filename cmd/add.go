package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/m44rten1/sprout/internal/config"
	"github.com/m44rten1/sprout/internal/editor"
	"github.com/m44rten1/sprout/internal/git"
	"github.com/m44rten1/sprout/internal/hooks"
	"github.com/m44rten1/sprout/internal/sprout"
	"github.com/m44rten1/sprout/internal/tui"

	"github.com/spf13/cobra"
)

var (
	addNoHooksFlag bool
	addNoOpenFlag  bool
)

var addCmd = &cobra.Command{
	Use:   "add [branch]",
	Short: "Create a new worktree",
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

		// Get all branches
		branches, err := git.ListAllBranches(repoRoot)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		// Get all worktrees to exclude branches that are already checked out
		worktrees, err := git.ListWorktrees(repoRoot)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		// Build a set of branches that are already checked out
		checkedOutBranches := make(map[string]bool)
		for _, wt := range worktrees {
			if wt.Branch != "" {
				checkedOutBranches[wt.Branch] = true
			}
		}

		var completions []string
		for _, branch := range branches {
			// Skip branches that are already checked out in any worktree
			if checkedOutBranches[branch.DisplayName] {
				continue
			}
			// Skip "origin" which is a remote name, not a branch
			if branch.DisplayName == "origin" {
				continue
			}
			completions = append(completions, branch.DisplayName)
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		var branch string

		// Get repo root first for potential interactive mode
		repoRoot, err := git.GetRepoRoot()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Use main worktree path for consistent sprout root calculation
		mainRepoRoot, err := git.GetMainWorktreePath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get main worktree: %v\n", err)
			os.Exit(1)
		}

		// Interactive mode: select from existing branches
		if len(args) == 0 {
			branches, err := git.ListAllBranches(repoRoot)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to list branches: %v\n", err)
				os.Exit(1)
			}

			// Get all worktrees to exclude branches that are already checked out
			worktrees, err := git.ListWorktrees(repoRoot)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to list worktrees: %v\n", err)
				os.Exit(1)
			}

			// Build a set of branches that are already checked out
			checkedOutBranches := make(map[string]bool)
			for _, wt := range worktrees {
				if wt.Branch != "" {
					checkedOutBranches[wt.Branch] = true
				}
			}

			// Filter branches
			var filteredBranches []git.Branch
			for _, branch := range branches {
				// Skip branches that are already checked out in any worktree
				if checkedOutBranches[branch.DisplayName] {
					continue
				}
				// Skip "origin" which is a remote name, not a branch
				if branch.DisplayName == "origin" {
					continue
				}
				filteredBranches = append(filteredBranches, branch)
			}

			if len(filteredBranches) == 0 {
				fmt.Println("No available branches found.")
				return
			}

			idx, err := tui.SelectOne(filteredBranches, func(b git.Branch) string {
				return b.DisplayName
			}, nil)

			if err != nil {
				// User cancelled or error occurred
				return
			}

			branch = filteredBranches[idx].DisplayName
		} else {
			branch = args[0]
		}

		worktreePath, err := sprout.GetWorktreePath(mainRepoRoot, branch)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error calculating worktree path: %v\n", err)
			os.Exit(1)
		}

		// Check if worktree already exists
		if _, err := os.Stat(worktreePath); err == nil {
			fmt.Printf("Worktree already exists at %s\n", worktreePath)
			// TODO: Offer to open it
			if err := editor.Open(worktreePath); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to open editor: %v\n", err)
			}
			return
		}

		// Get main worktree path for config loading
		mainWorktreePath, err := git.GetMainWorktreePath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get main worktree path: %v\n", err)
			os.Exit(1)
		}

		// Check if .sprout.yml exists and has on_create hooks
		cfg, err := config.Load(repoRoot, mainWorktreePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
			os.Exit(1)
		}

		shouldRunHooks := cfg.HasCreateHooks() && !addNoHooksFlag

		// If hooks should run, verify repo is trusted before creating worktree
		if shouldRunHooks {
			untrusted, err := hooks.CheckAndPrintUntrusted(repoRoot, mainWorktreePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to check trust status: %v\n", err)
				os.Exit(1)
			}

			if untrusted {
				// Hooks exist but repo is not trusted - abort before creating worktree
				os.Exit(1)
			}
		}

		fmt.Printf("Creating worktree for %s at %s...\n", branch, worktreePath)

		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create parent directory: %v\n", err)
			os.Exit(1)
		}

		// Check if branch exists locally
		localExists, _ := git.LocalBranchExists(repoRoot, branch)

		// Build git worktree add command.
		gitArgs := []string{"worktree", "add"}

		// Only new branches should avoid configuring an upstream.
		if !localExists {
			gitArgs = append(gitArgs, "--no-track")
		}

		gitArgs = append(gitArgs, worktreePath)

		if localExists {
			// Branch exists locally, just checkout
			gitArgs = append(gitArgs, branch)
		} else {
			// Check if remote branch exists
			remoteBranch := "origin/" + branch
			exists, _ := git.BranchExists(repoRoot, remoteBranch)

			if exists {
				gitArgs = append(gitArgs, "-b", branch, remoteBranch)
			} else {
				// Check if origin/main exists
				if valid, _ := git.BranchExists(repoRoot, "main"); valid {
					// Check if it's remote or local
					// Actually BranchExists checks both.
					// Let's try origin/main first
					if remoteMain, _ := git.BranchExists(repoRoot, "origin/main"); remoteMain {
						gitArgs = append(gitArgs, "-b", branch, "origin/main")
					} else {
						// Fallback to local main or HEAD
						gitArgs = append(gitArgs, "-b", branch, "main")
					}
				} else {
					// Just use HEAD
					gitArgs = append(gitArgs, "-b", branch)
				}
			}
		}

		if _, err := git.RunGitCommand(repoRoot, gitArgs...); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create worktree: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Worktree created!")

		// Open editor first if not disabled and hooks will run
		// This allows user to start browsing code while hooks run in terminal
		if !addNoOpenFlag && shouldRunHooks {
			if err := editor.Open(worktreePath); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to open editor: %v\n", err)
			}
		}

		// Run on_create hooks automatically if they exist and repo is trusted
		if shouldRunHooks {
			if err := hooks.RunHooks(repoRoot, worktreePath, mainWorktreePath, hooks.OnCreate); err != nil {
				if _, ok := err.(*hooks.UntrustedError); ok {
					hooks.PrintUntrustedMessage(mainWorktreePath)
				} else {
					fmt.Fprintf(os.Stderr, "Error running hooks: %v\n", err)
					os.Exit(1)
				}
			}
		}

		// Open editor after everything if hooks didn't run and --no-open wasn't set
		if !shouldRunHooks && !addNoOpenFlag {
			if err := editor.Open(worktreePath); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to open editor: %v\n", err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().BoolVar(&addNoHooksFlag, "no-hooks", false, "Skip running on_create hooks even if .sprout.yml exists")
	addCmd.Flags().BoolVar(&addNoOpenFlag, "no-open", false, "Skip opening the worktree in an editor")
}
