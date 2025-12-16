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

		if len(worktrees) == 0 {
			fmt.Println("No worktrees found.")
			return
		}

		// First worktree is always the main repo
		mainWorktree := worktrees[0]

		// Filter to only sprout-managed worktrees (check ALL possible sprout roots)
		sproutRoots := getPossibleSproutRoots()
		var sproutWorktrees []git.Worktree
		for _, wt := range worktrees[1:] { // Skip main worktree
			for _, sproutRoot := range sproutRoots {
				if isUnderSproutRoot(wt.Path, sproutRoot) {
					sproutWorktrees = append(sproutWorktrees, wt)
					break
				}
			}
		}

		sproutWorktrees = filterExistingWorktrees(sproutWorktrees)

		// Add spacing from prompt
		fmt.Println()

		// Print repo header
		repoName := filepath.Base(mainWorktree.Path)
		fmt.Printf("📦 \033[1m%s\033[0m\n", repoName)
		fmt.Println()

		// Collect output lines with styling
		type outputLine struct {
			label  string
			status string
			path   string
			isMain bool
		}

		// Collect status in parallel (including main worktree)
		totalLines := 1 + len(sproutWorktrees) // main + sprouts
		lines := make([]outputLine, totalLines)
		var wg sync.WaitGroup

		// Process main worktree
		wg.Add(1)
		go func() {
			defer wg.Done()

			branch := mainWorktree.Branch
			if branch == "" {
				branch = "(detached)"
			}

			// Get status indicators for main branch too
			wtStatus := git.GetWorktreeStatus(mainWorktree.Path)
			statusEmojis := buildStatusEmojis(wtStatus)

			lines[0] = outputLine{
				label:  "  🏠 \033[32m" + branch + "\033[0m",
				status: statusEmojis,
				path:   "\033[90m" + mainWorktree.Path + "\033[0m",
				isMain: true,
			}
		}()

		// Process sprout worktrees
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

				lines[idx+1] = outputLine{
					label:  "  🌱 \033[32m" + branch + "\033[0m",
					status: statusEmojis,
					path:   "\033[90m" + worktree.Path + "\033[0m",
					isMain: false,
				}
			}(i, wt)
		}

		wg.Wait()

		// Print with multi-line format for better readability
		for _, line := range lines {
			// Branch + status on first line, path indented below
			if line.status != "" {
				fmt.Printf("%s   %s\n", line.label, line.status)
			} else {
				fmt.Printf("%s\n", line.label)
			}
			fmt.Printf("     %s\n", line.path)
			fmt.Println() // Blank line after each worktree
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
		isRepoHeader     bool
		isMain           bool
	}

	first := true

	// Collect all worktrees with their indices for parallel processing
	type worktreeJob struct {
		lineIdx      int
		worktree     git.Worktree
		isRepoHeader bool
		repoName     string
		needsBlank   bool
		isMain       bool
	}

	var jobs []worktreeJob
	lineIdx := 0

	for _, repo := range allRepos {
		if len(repo.Worktrees) == 0 {
			continue
		}

		// Get all worktrees for this repo to find main branch
		allWorktrees, err := git.ListWorktrees(repo.RepoRoot)
		if err != nil || len(allWorktrees) == 0 {
			continue
		}

		mainWorktree := allWorktrees[0]
		repoName := filepath.Base(mainWorktree.Path)

		// Add repo header
		jobs = append(jobs, worktreeJob{
			lineIdx:      lineIdx,
			isRepoHeader: true,
			repoName:     repoName,
			needsBlank:   !first,
		})
		lineIdx++
		first = false

		// Add main worktree
		jobs = append(jobs, worktreeJob{
			lineIdx:  lineIdx,
			worktree: mainWorktree,
			isMain:   true,
		})
		lineIdx++

		// Add sprout worktree jobs
		for _, wt := range repo.Worktrees {
			jobs = append(jobs, worktreeJob{
				lineIdx:  lineIdx,
				worktree: wt,
				isMain:   false,
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
					label:            "📦 \033[1m" + j.repoName + "\033[0m\n",
					status:           "",
					path:             "",
					needsBlankBefore: j.needsBlank,
					isRepoHeader:     true,
				}
			} else {
				branch := j.worktree.Branch
				if branch == "" {
					branch = "(detached)"
				}

				// Get status indicators
				wtStatus := git.GetWorktreeStatus(j.worktree.Path)
				statusEmojis := buildStatusEmojis(wtStatus)

				label := "  🏠 \033[32m" + branch + "\033[0m"
				if !j.isMain {
					label = "  🌱 \033[32m" + branch + "\033[0m"
				}

				results[j.lineIdx] = outputLine{
					label:        label,
					status:       statusEmojis,
					path:         "\033[90m" + j.worktree.Path + "\033[0m",
					isRepoHeader: false,
					isMain:       j.isMain,
				}
			}
		}(job)
	}

	wg.Wait()
	lines := results

	// Print with multi-line format for better readability
	for _, line := range lines {
		if line.needsBlankBefore {
			fmt.Println()
		}

		if line.isRepoHeader {
			// Repo header: just the name with icon
			fmt.Printf("%s\n", line.label)
		} else {
			// Worktree: branch + status on first line, path indented below
			if line.status != "" {
				fmt.Printf("%s   %s\n", line.label, line.status)
			} else {
				fmt.Printf("%s\n", line.label)
			}
			fmt.Printf("     %s\n", line.path)
			fmt.Println() // Blank line after each worktree
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
	var emojis []string
	if status.Dirty {
		emojis = append(emojis, "🔴")
	}
	if status.Ahead > 0 {
		emojis = append(emojis, "⬆️")
	}
	if status.Behind > 0 {
		emojis = append(emojis, "⬇️")
	}
	if status.Unmerged {
		emojis = append(emojis, "🔀")
	}

	result := ""
	for i, emoji := range emojis {
		if i > 0 {
			result += " "
		}
		result += emoji
	}
	return result
}
