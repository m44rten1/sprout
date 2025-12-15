package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/m44rten1/sprout/internal/git"
	"github.com/m44rten1/sprout/internal/sprout"

	"github.com/spf13/cobra"
)

var listAllFlag bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List worktrees",
	Run: func(cmd *cobra.Command, args []string) {
		if listAllFlag {
			listAllRepos()
			return
		}

		// Normal mode: list current repo only
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

		worktrees, err := git.ListWorktrees(repoRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list worktrees: %v\n", err)
			os.Exit(1)
		}

		worktrees = filterSproutWorktrees(worktrees, sproutRoot)
		if len(worktrees) == 0 {
			fmt.Println("No sprout-managed worktrees found.")
			return
		}

		// Add spacing from prompt
		fmt.Println()

		// Collect output lines with styling
		type outputLine struct {
			label string
			path  string
		}

		var lines []outputLine
		for _, wt := range worktrees {
			branch := wt.Branch
			if branch == "" {
				branch = "(detached)"
			}
			lines = append(lines, outputLine{
				label: "\033[32m" + branch + "\033[0m",
				path:  "\033[90m" + wt.Path + "\033[0m",
			})
		}

		// Calculate the maximum label width (without ANSI codes) for alignment
		maxWidth := 0
		for _, line := range lines {
			width := 0
			inAnsi := false
			for _, r := range line.label {
				if r == '\033' {
					inAnsi = true
				} else if inAnsi && r == 'm' {
					inAnsi = false
				} else if !inAnsi {
					if r > 127 {
						width += 2
					} else {
						width += 1
					}
				}
			}
			if width > maxWidth {
				maxWidth = width
			}
		}

		// Print with manual padding for consistent alignment
		for _, line := range lines {
			width := 0
			inAnsi := false
			for _, r := range line.label {
				if r == '\033' {
					inAnsi = true
				} else if inAnsi && r == 'm' {
					inAnsi = false
				} else if !inAnsi {
					if r > 127 {
						width += 2
					} else {
						width += 1
					}
				}
			}
			padding := maxWidth - width + 3
			fmt.Printf("%s%*s%s\n", line.label, padding, "", line.path)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVar(&listAllFlag, "all", false, "List worktrees from all repositories")
}

// RepoWorktrees holds worktrees for a specific repository
type RepoWorktrees struct {
	RepoRoot  string
	Worktrees []git.Worktree
}

// listAllRepos lists worktrees from all sprout-managed repositories
func listAllRepos() {
	allRepos, err := discoverAllSproutRepos()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to discover repositories: %v\n", err)
		os.Exit(1)
	}

	if len(allRepos) == 0 {
		fmt.Println("No sprout-managed worktrees found.")
		return
	}

	// Sort repos by path for consistent output
	sort.Slice(allRepos, func(i, j int) bool {
		return allRepos[i].RepoRoot < allRepos[j].RepoRoot
	})

	// Add spacing from prompt
	fmt.Println()

	// Collect all output lines first to ensure proper alignment
	type outputLine struct {
		label            string
		path             string
		needsBlankBefore bool
	}

	var lines []outputLine
	first := true

	for _, repo := range allRepos {
		if len(repo.Worktrees) == 0 {
			continue
		}

		repoName := filepath.Base(repo.RepoRoot)
		lines = append(lines, outputLine{
			label:            "📦 \033[1m" + repoName + "\033[0m",
			path:             "\033[90m" + repo.RepoRoot + "\033[0m",
			needsBlankBefore: !first,
		})
		first = false

		for _, wt := range repo.Worktrees {
			branch := wt.Branch
			if branch == "" {
				branch = "(detached)"
			}
			lines = append(lines, outputLine{
				label: "  \033[32m" + branch + "\033[0m",
				path:  "\033[90m" + wt.Path + "\033[0m",
			})
		}
	}

	// Calculate the maximum label width (without ANSI codes) for alignment
	maxWidth := 0
	for _, line := range lines {
		// Strip ANSI codes to get actual display width
		// Simple approach: count visible characters (emojis count as 2, other chars as 1)
		width := 0
		inAnsi := false
		for _, r := range line.label {
			if r == '\033' {
				inAnsi = true
			} else if inAnsi && r == 'm' {
				inAnsi = false
			} else if !inAnsi {
				if r > 127 {
					width += 2 // Emoji/unicode
				} else {
					width += 1
				}
			}
		}
		if width > maxWidth {
			maxWidth = width
		}
	}

	// Print with manual padding for consistent alignment
	for _, line := range lines {
		if line.needsBlankBefore {
			fmt.Println()
		}

		// Calculate padding needed
		width := 0
		inAnsi := false
		for _, r := range line.label {
			if r == '\033' {
				inAnsi = true
			} else if inAnsi && r == 'm' {
				inAnsi = false
			} else if !inAnsi {
				if r > 127 {
					width += 2
				} else {
					width += 1
				}
			}
		}
		padding := maxWidth - width + 3 // 3 spaces gap

		fmt.Printf("%s%*s%s\n", line.label, padding, "", line.path)
	}
}

// discoverAllSproutRepos scans ~/.sprout and discovers all managed repositories
func discoverAllSproutRepos() ([]RepoWorktrees, error) {
	sproutRoot, err := sprout.GetSproutRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to get sprout root: %w", err)
	}

	// Check if sprout root exists
	if _, err := os.Stat(sproutRoot); os.IsNotExist(err) {
		return []RepoWorktrees{}, nil
	}

	// Read all repo directories
	entries, err := os.ReadDir(sproutRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to read sprout root: %w", err)
	}

	var allRepos []RepoWorktrees
	seenRepos := make(map[string]bool)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Each directory should match pattern: <slug>-<8-char-hash>
		repoDir := filepath.Join(sproutRoot, entry.Name())

		// Find the main repo root by looking at any worktree in this directory
		repoRoot, err := findRepoRootFromWorktrees(repoDir)
		if err != nil {
			// Skip directories that don't contain valid worktrees
			continue
		}

		// Skip if we've already processed this repo
		if seenRepos[repoRoot] {
			continue
		}
		seenRepos[repoRoot] = true

		// List all worktrees for this repo
		worktrees, err := git.ListWorktrees(repoRoot)
		if err != nil {
			// Skip repos that can't be listed
			continue
		}

		// Filter to only sprout-managed worktrees
		// We need to check all possible sprout roots since worktrees might have been created
		// with different configurations (XDG_DATA_HOME vs ~/.sprout)
		sproutRoots := getPossibleSproutRoots()

		var filtered []git.Worktree
		for _, wt := range worktrees {
			if isSproutManagedWorktree(wt.Path, sproutRoots, repoRoot) {
				filtered = append(filtered, wt)
			}
		}
		worktrees = filtered

		if len(worktrees) > 0 {
			allRepos = append(allRepos, RepoWorktrees{
				RepoRoot:  repoRoot,
				Worktrees: worktrees,
			})
		}
	}

	return allRepos, nil
}

// getPossibleSproutRoots returns all possible sprout root directories
func getPossibleSproutRoots() []string {
	var roots []string

	// Add XDG_DATA_HOME location if set
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		roots = append(roots, filepath.Join(xdgData, "sprout"))
	}

	// Add ~/.local/share/sprout
	if home, err := os.UserHomeDir(); err == nil {
		roots = append(roots, filepath.Join(home, ".local", "share", "sprout"))
		// Add ~/.sprout for backward compatibility
		roots = append(roots, filepath.Join(home, ".sprout"))
	}

	return roots
}

// isSproutManagedWorktree checks if a worktree is managed by sprout
func isSproutManagedWorktree(worktreePath string, sproutRoots []string, mainRepoRoot string) bool {
	// Skip the main repository root
	if worktreePath == mainRepoRoot {
		return false
	}

	// Check if the worktree is under any of the sprout roots
	for _, root := range sproutRoots {
		if isUnderSproutRoot(worktreePath, root) {
			return true
		}
	}

	return false
}

// findRepoRootFromWorktrees finds the main repo root by examining worktrees in the given directory
func findRepoRootFromWorktrees(repoDir string) (string, error) {
	// Walk through the directory to find any .git file (worktree marker)
	var gitFile string
	err := filepath.Walk(repoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() && info.Name() == ".git" {
			gitFile = path
			return filepath.SkipAll // Found one, stop walking
		}
		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return "", err
	}

	if gitFile == "" {
		return "", fmt.Errorf("no worktrees found in %s", repoDir)
	}

	// Get the directory containing the .git file
	worktreeDir := filepath.Dir(gitFile)

	// Use git worktree list to find the main worktree (first in the list)
	// This is more reliable than rev-parse --show-toplevel
	worktrees, err := git.ListWorktrees(worktreeDir)
	if err != nil {
		return "", fmt.Errorf("failed to list worktrees: %w", err)
	}

	if len(worktrees) == 0 {
		return "", fmt.Errorf("no worktrees found")
	}

	// The first worktree is always the main worktree
	return worktrees[0].Path, nil
}
