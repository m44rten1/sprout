package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/m44rten1/sprout/internal/git"

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

		worktrees, err := git.ListWorktrees(repoRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list worktrees: %v\n", err)
			os.Exit(1)
		}

		// Filter to only sprout-managed worktrees (check ALL possible sprout roots)
		sproutRoots := getPossibleSproutRoots()
		var sproutWorktrees []git.Worktree
		for _, wt := range worktrees {
			for _, sproutRoot := range sproutRoots {
				if isUnderSproutRoot(wt.Path, sproutRoot) {
					sproutWorktrees = append(sproutWorktrees, wt)
					break
				}
			}
		}

		sproutWorktrees = filterExistingWorktrees(sproutWorktrees)
		if len(sproutWorktrees) == 0 {
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

		// Collect status in parallel
		lines := make([]outputLine, len(sproutWorktrees))
		var wg sync.WaitGroup

		for i, wt := range sproutWorktrees {
			wg.Add(1)
			go func(idx int, worktree git.Worktree) {
				defer wg.Done()

				branch := worktree.Branch
				if branch == "" {
					branch = "(detached)"
				}

				// Get status indicators
				wtStatus := git.GetWorktreeStatus(worktree.Path)
				statusEmojis := buildStatusEmojis(wtStatus)

				lines[idx] = outputLine{
					label:  "\033[32m" + branch + "\033[0m",
					status: statusEmojis,
					path:   "\033[90m" + worktree.Path + "\033[0m",
				}
			}(i, wt)
		}

		wg.Wait()

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

	// Pre-calculate total lines needed
	totalLines := 0
	for _, repo := range allRepos {
		if len(repo.Worktrees) > 0 {
			totalLines += 1 + len(repo.Worktrees) // repo header + worktrees
		}
	}

	first := true

	// Collect all worktrees with their indices for parallel processing
	type worktreeJob struct {
		lineIdx      int
		worktree     git.Worktree
		isRepoHeader bool
		repoName     string
		repoRoot     string
		needsBlank   bool
	}

	var jobs []worktreeJob
	lineIdx := 0

	for _, repo := range allRepos {
		if len(repo.Worktrees) == 0 {
			continue
		}

		repoName := filepath.Base(repo.RepoRoot)

		// Add repo header
		jobs = append(jobs, worktreeJob{
			lineIdx:      lineIdx,
			isRepoHeader: true,
			repoName:     repoName,
			repoRoot:     repo.RepoRoot,
			needsBlank:   !first,
		})
		lineIdx++
		first = false

		// Add worktree jobs
		for _, wt := range repo.Worktrees {
			jobs = append(jobs, worktreeJob{
				lineIdx:  lineIdx,
				worktree: wt,
			})
			lineIdx++
		}
	}

	// Process worktrees in parallel
	results := make([]outputLine, len(jobs))
	var wg sync.WaitGroup

	for _, job := range jobs {
		wg.Add(1)
		go func(j worktreeJob) {
			defer wg.Done()

			if j.isRepoHeader {
				results[j.lineIdx] = outputLine{
					label:            "📦 \033[1m" + j.repoName + "\033[0m",
					status:           "",
					path:             "\033[90m" + j.repoRoot + "\033[0m",
					needsBlankBefore: j.needsBlank,
				}
			} else {
				branch := j.worktree.Branch
				if branch == "" {
					branch = "(detached)"
				}

				// Get status indicators
				wtStatus := git.GetWorktreeStatus(j.worktree.Path)
				statusEmojis := buildStatusEmojis(wtStatus)

				results[j.lineIdx] = outputLine{
					label:  "  \033[32m" + branch + "\033[0m",
					status: statusEmojis,
					path:   "\033[90m" + j.worktree.Path + "\033[0m",
				}
			}
		}(job)
	}

	wg.Wait()
	lines := results

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
	// Get all possible sprout root directories
	sproutRoots := getPossibleSproutRoots()

	// Collect all repo directories from all sprout roots
	var repoDirs []string
	for _, sproutRoot := range sproutRoots {
		// Check if sprout root exists
		if _, err := os.Stat(sproutRoot); os.IsNotExist(err) {
			continue
		}

		// Read all repo directories
		entries, err := os.ReadDir(sproutRoot)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				repoDirs = append(repoDirs, filepath.Join(sproutRoot, entry.Name()))
			}
		}
	}

	// Process repos in parallel
	var mu sync.Mutex
	var wg sync.WaitGroup
	seenRepos := make(map[string]bool)
	var allRepos []RepoWorktrees

	for _, repoDir := range repoDirs {
		wg.Add(1)
		go func(dir string) {
			defer wg.Done()

			// Find any worktree in this repo dir (shallow scan)
			anyWorktree := findFirstWorktree(dir)
			if anyWorktree == "" {
				return
			}

			// Get all worktrees for this repo with ONE git command
			worktrees, err := git.ListWorktrees(anyWorktree)
			if err != nil {
				return
			}

			if len(worktrees) == 0 {
				return
			}

			// First worktree is always the main repo
			mainRepoRoot := worktrees[0].Path

			// Filter to only sprout-managed worktrees (exclude main repo)
			// Check against all possible sprout roots, not just the discovered directory
			var sproutWorktrees []git.Worktree
			sproutRoots := getPossibleSproutRoots()
			for _, wt := range worktrees {
				isUnderAnySproutRoot := false
				for _, sproutRoot := range sproutRoots {
					if isUnderSproutRoot(wt.Path, sproutRoot) {
						isUnderAnySproutRoot = true
						break
					}
				}
				if isUnderAnySproutRoot {
					sproutWorktrees = append(sproutWorktrees, wt)
				}
			}

			if len(sproutWorktrees) == 0 {
				return
			}

			// Add to results (thread-safe)
			mu.Lock()
			if !seenRepos[mainRepoRoot] {
				seenRepos[mainRepoRoot] = true
				allRepos = append(allRepos, RepoWorktrees{
					RepoRoot:  mainRepoRoot,
					Worktrees: sproutWorktrees,
				})
			}
			mu.Unlock()
		}(repoDir)
	}

	wg.Wait()
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

// findFirstWorktree does a shallow scan to find any worktree in the repo directory.
// Sprout structure can be:
//   - <repo-dir>/<branch>/.git (flat, older structure)
//   - <repo-dir>/<branch>/<repo-slug>/.git (nested, newer structure)
//   - <repo-dir>/<branch>/<repo-slug>/<repo-slug>/.git (double-nested, migration artifact)
//
// We scan up to 3 levels deep and return the first WORKING worktree.
func findFirstWorktree(repoDir string) string {
	// Collect all .git files up to 3 levels deep
	var candidates []string

	// Level 1: <repo-dir>/
	level1Entries, err := os.ReadDir(repoDir)
	if err != nil {
		return ""
	}

	for _, entry1 := range level1Entries {
		if !entry1.IsDir() {
			continue
		}

		level1Path := filepath.Join(repoDir, entry1.Name())

		// Check level 1
		if _, err := os.Stat(filepath.Join(level1Path, ".git")); err == nil {
			candidates = append(candidates, level1Path)
		}

		// Level 2: <repo-dir>/<branch>/
		level2Entries, err := os.ReadDir(level1Path)
		if err != nil {
			continue
		}

		for _, entry2 := range level2Entries {
			if !entry2.IsDir() {
				continue
			}

			level2Path := filepath.Join(level1Path, entry2.Name())

			// Check level 2
			if _, err := os.Stat(filepath.Join(level2Path, ".git")); err == nil {
				candidates = append(candidates, level2Path)
			}

			// Level 3: <repo-dir>/<branch>/<slug>/
			level3Entries, err := os.ReadDir(level2Path)
			if err != nil {
				continue
			}

			for _, entry3 := range level3Entries {
				if !entry3.IsDir() {
					continue
				}

				level3Path := filepath.Join(level2Path, entry3.Name())

				// Check level 3
				if _, err := os.Stat(filepath.Join(level3Path, ".git")); err == nil {
					candidates = append(candidates, level3Path)
				}
			}
		}
	}

	// Try each candidate and return the first one where git worktree list works
	for _, candidate := range candidates {
		if _, err := git.ListWorktrees(candidate); err == nil {
			return candidate
		}
	}

	return ""
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
