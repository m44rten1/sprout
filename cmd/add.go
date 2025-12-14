package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/m44rten1/sprout/internal/editor"
	"github.com/m44rten1/sprout/internal/git"
	"github.com/m44rten1/sprout/internal/hooks"
	"github.com/m44rten1/sprout/internal/sprout"
	"github.com/m44rten1/sprout/internal/tui"

	"github.com/spf13/cobra"
)

var (
	initFlag   bool
	noOpenFlag bool
)

var addCmd = &cobra.Command{
	Use:   "add [branch]",
	Short: "Create a new worktree",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var branch string

		// Get repo root first for potential interactive mode
		repoRoot, err := git.GetRepoRoot()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Interactive mode: select from existing branches
		if len(args) == 0 {
			branches, err := git.ListAllBranches(repoRoot)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to list branches: %v\n", err)
				os.Exit(1)
			}

			if len(branches) == 0 {
				fmt.Println("No branches found.")
				return
			}

			idx, err := tui.SelectOne(branches, func(b git.Branch) string {
				return b.DisplayName
			}, nil)

			if err != nil {
				// User cancelled or error occurred
				return
			}

			branch = branches[idx].DisplayName
		} else {
			branch = args[0]
		}

		worktreePath, err := sprout.GetWorktreePath(repoRoot, branch)
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

		// Open editor first if not disabled, then run hooks
		// This allows user to start browsing code while hooks run
		if !noOpenFlag && initFlag {
			if err := editor.Open(worktreePath); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to open editor: %v\n", err)
			}
		}

		// Run on_create hooks if --init flag is set
		if initFlag {
			mainWorktreePath, err := git.GetMainWorktreePath()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to get main worktree path: %v\n", err)
				os.Exit(1)
			}

			if err := hooks.RunHooks(repoRoot, worktreePath, mainWorktreePath, hooks.OnCreate); err != nil {
				if _, ok := err.(*hooks.UntrustedError); ok {
					hooks.PrintUntrustedMessage(mainWorktreePath)
				} else {
					fmt.Fprintf(os.Stderr, "Error running hooks: %v\n", err)
					os.Exit(1)
				}
			}
		}

		// Open editor after everything if not using --init or if --no-open wasn't set
		if !initFlag && !noOpenFlag {
			if err := editor.Open(worktreePath); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to open editor: %v\n", err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().BoolVar(&initFlag, "init", false, "Run on_create hooks after creating the worktree")
	addCmd.Flags().BoolVar(&noOpenFlag, "no-open", false, "Skip opening the worktree in an editor")
}
