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
	Long: `List worktrees for the current repository or all repositories.

Status indicators show the git state of each worktree:
  🔴  Dirty - worktree has uncommitted changes
  ⬆️   Ahead - worktree has unpushed commits (ahead of remote)
  ⬇️   Behind - worktree needs to pull (behind remote)
  🔀  Unmerged - worktree has commits not in main/master branch

Multiple indicators can appear together (e.g., 🔴🔀 means dirty and unmerged).
Clean worktrees show no indicators.`,
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
		worktrees = filterExistingWorktrees(worktrees)
		if len(worktrees) == 0 {
			fmt.Println("No sprout-managed worktrees found.")
			return
		}

		// Add spacing from prompt
		fmt.Println()

		// Collect output lines with styling
		type outputLine struct {
			label  string
			status string
			path   string
		}

		var lines []outputLine
		for _, wt := range worktrees {
			branch := wt.Branch
			if branch == "" {
				branch = "(detached)"
			}

			// Get status indicators
			wtStatus := git.GetWorktreeStatus(wt.Path)
			statusEmojis := buildStatusEmojis(wtStatus)

			lines = append(lines, outputLine{
				label:  "\033[32m" + branch + "\033[0m",
				status: statusEmojis,
				path:   "\033[90m" + wt.Path + "\033[0m",
			})
		}

		// Calculate the maximum label width (without ANSI codes) for alignment
		maxLabelWidth := 0
		maxStatusWidth := 0
		hasAnyStatus := false
		for _, line := range lines {
			labelWidth := visualWidth(line.label)
			if labelWidth > maxLabelWidth {
				maxLabelWidth = labelWidth
			}
			statusWidth := visualWidth(line.status)
			if statusWidth > maxStatusWidth {
				maxStatusWidth = statusWidth
			}
			if line.status != "" {
				hasAnyStatus = true
			}
		}

		// Print with manual padding for consistent alignment
		for _, line := range lines {
			labelWidth := visualWidth(line.label)
			labelPadding := maxLabelWidth - labelWidth + 3

			if hasAnyStatus {
				statusWidth := visualWidth(line.status)
				statusPadding := maxStatusWidth - statusWidth + 3
				fmt.Printf("%s%*s%s%*s%s\n", line.label, labelPadding, "", line.status, statusPadding, "", line.path)
			} else {
				fmt.Printf("%s%*s%s\n", line.label, labelPadding, "", line.path)
			}
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
		status           string
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
			status:           "",
			path:             "\033[90m" + repo.RepoRoot + "\033[0m",
			needsBlankBefore: !first,
		})
		first = false

		for _, wt := range repo.Worktrees {
			branch := wt.Branch
			if branch == "" {
				branch = "(detached)"
			}

			// Get status indicators
			wtStatus := git.GetWorktreeStatus(wt.Path)
			statusEmojis := buildStatusEmojis(wtStatus)

			lines = append(lines, outputLine{
				label:  "  \033[32m" + branch + "\033[0m",
				status: statusEmojis,
				path:   "\033[90m" + wt.Path + "\033[0m",
			})
		}
	}

	// Calculate the maximum label and status width (without ANSI codes) for alignment
	maxLabelWidth := 0
	maxStatusWidth := 0
	hasAnyStatus := false
	for _, line := range lines {
		labelWidth := visualWidth(line.label)
		if labelWidth > maxLabelWidth {
			maxLabelWidth = labelWidth
		}
		statusWidth := visualWidth(line.status)
		if statusWidth > maxStatusWidth {
			maxStatusWidth = statusWidth
		}
		if line.status != "" {
			hasAnyStatus = true
		}
	}

	// Print with manual padding for consistent alignment
	for _, line := range lines {
		if line.needsBlankBefore {
			fmt.Println()
		}

		labelWidth := visualWidth(line.label)
		labelPadding := maxLabelWidth - labelWidth + 3

		if hasAnyStatus {
			statusWidth := visualWidth(line.status)
			statusPadding := maxStatusWidth - statusWidth + 3
			fmt.Printf("%s%*s%s%*s%s\n", line.label, labelPadding, "", line.status, statusPadding, "", line.path)
		} else {
			fmt.Printf("%s%*s%s\n", line.label, labelPadding, "", line.path)
		}
	}
}

// discoverAllSproutRepos scans all possible sprout directories and discovers all managed repositories
func discoverAllSproutRepos() ([]RepoWorktrees, error) {
	var allRepos []RepoWorktrees
	seenRepos := make(map[string]bool)

	// Get all possible sprout root directories
	sproutRoots := getPossibleSproutRoots()

	// Scan each sprout root directory
	for _, sproutRoot := range sproutRoots {
		// Check if sprout root exists
		if _, err := os.Stat(sproutRoot); os.IsNotExist(err) {
			continue
		}

		// Read all repo directories
		entries, err := os.ReadDir(sproutRoot)
		if err != nil {
			// Skip directories we can't read
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			// Each directory should match pattern: <slug>-<8-char-hash>
			repoDir := filepath.Join(sproutRoot, entry.Name())

			// Find all valid worktrees in this directory
			worktreesInDir := findWorktreesInDirectory(repoDir)

			// Group worktrees by their main repository
			for _, wt := range worktreesInDir {
				// Get the main repo root
				repoRoot, err := getMainRepoRoot(wt.Path)
				if err != nil {
					continue
				}

				// Skip if we've already processed this repo, otherwise add the worktree
				if !seenRepos[repoRoot] {
					seenRepos[repoRoot] = true
					allRepos = append(allRepos, RepoWorktrees{
						RepoRoot:  repoRoot,
						Worktrees: []git.Worktree{wt},
					})
				} else {
					// Add to existing repo
					for i := range allRepos {
						if allRepos[i].RepoRoot == repoRoot {
							allRepos[i].Worktrees = append(allRepos[i].Worktrees, wt)
							break
						}
					}
				}
			}
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

// findWorktreesInDirectory finds all valid git worktrees in a directory
func findWorktreesInDirectory(dir string) []git.Worktree {
	var worktrees []git.Worktree

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Look for .git files (worktree markers)
		if !info.IsDir() && info.Name() == ".git" {
			worktreeDir := filepath.Dir(path)

			// Try to get branch info from this worktree
			branch, _ := git.RunGitCommand(worktreeDir, "rev-parse", "--abbrev-ref", "HEAD")
			head, _ := git.RunGitCommand(worktreeDir, "rev-parse", "HEAD")

			// Only add if we can successfully get git info
			if head != "" {
				wt := git.Worktree{
					Path:   worktreeDir,
					HEAD:   head,
					Branch: branch,
				}
				worktrees = append(worktrees, wt)
			}
		}
		return nil
	})

	return worktrees
}

// getMainRepoRoot gets the main repository root for a given worktree
func getMainRepoRoot(worktreePath string) (string, error) {
	// Get all worktrees from this worktree's perspective
	worktrees, err := git.ListWorktrees(worktreePath)
	if err != nil {
		return "", err
	}

	if len(worktrees) == 0 {
		return "", fmt.Errorf("no worktrees found")
	}

	// First worktree is always the main repo
	return worktrees[0].Path, nil
}

// findRepoRootFromWorktrees finds the main repo root by examining worktrees in the given directory
func findRepoRootFromWorktrees(repoDir string) (string, error) {
	// Walk through the directory to find any .git file (worktree marker)
	var gitFiles []string
	err := filepath.Walk(repoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() && info.Name() == ".git" {
			gitFiles = append(gitFiles, path)
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	if len(gitFiles) == 0 {
		return "", fmt.Errorf("no worktrees found in %s", repoDir)
	}

	// Try each .git file until we find one that works
	for _, gitFile := range gitFiles {
		worktreeDir := filepath.Dir(gitFile)

		// Use git worktree list to find the main worktree (first in the list)
		// This is more reliable than rev-parse --show-toplevel
		worktrees, err := git.ListWorktrees(worktreeDir)
		if err != nil {
			// This worktree might be broken, try the next one
			continue
		}

		if len(worktrees) > 0 {
			// The first worktree is always the main worktree
			return worktrees[0].Path, nil
		}
	}

	return "", fmt.Errorf("no valid worktrees found in %s", repoDir)
}

// filterExistingWorktrees filters out worktrees whose paths don't exist on the filesystem.
func filterExistingWorktrees(worktrees []git.Worktree) []git.Worktree {
	var existing []git.Worktree
	for _, wt := range worktrees {
		if _, err := os.Stat(wt.Path); err == nil {
			existing = append(existing, wt)
		}
	}
	return existing
}

// buildStatusEmojis builds a string of status emoji indicators.
func buildStatusEmojis(status git.WorktreeStatus) string {
	var emojis string
	if status.Dirty {
		emojis += "🔴"
	}
	if status.Ahead > 0 {
		emojis += "⬆️"
	}
	if status.Behind > 0 {
		emojis += "⬇️"
	}
	if status.Unmerged {
		emojis += "🔀"
	}
	return emojis
}

// visualWidth calculates the display width of a string, accounting for ANSI codes and emojis.
func visualWidth(s string) int {
	width := 0
	inAnsi := false
	for _, r := range s {
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
	return width
}
