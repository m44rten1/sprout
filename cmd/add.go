package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/m44rten1/sprout/internal/editor"
	"github.com/m44rten1/sprout/internal/git"
	"github.com/m44rten1/sprout/internal/sprout"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [branch]",
	Short: "Create a new worktree",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		branch := args[0]

		repoRoot, err := git.GetRepoRoot()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
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
		if err := editor.Open(worktreePath); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open editor: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
