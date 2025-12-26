package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/m44rten1/sprout/internal/core"
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
  ` + "\033[31m✗\033[0m" + `  Dirty - worktree has uncommitted changes
  ` + "\033[33m↑\033[0m" + `  Ahead - worktree has unpushed commits
  ` + "\033[36m↓\033[0m" + `  Behind - worktree needs to pull
  ` + "\033[35m↕\033[0m" + `  Unmerged - worktree has commits not in main/master branch

Multiple indicators can appear together (e.g., ` + "\033[31m✗\033[0m \033[35m↕\033[0m" + ` means dirty and unmerged).
Clean worktrees show no indicators.`,
	Run: func(cmd *cobra.Command, args []string) {
		if listAllFlag {
			repos, err := collectAllRepos()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if len(repos) == 0 {
				fmt.Println("\nNo sprout worktrees found.")
				return
			}
			printRepos(repos, true)
			return
		}

		// Single repo mode
		repo, found, err := collectCurrentRepo()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if !found {
			fmt.Println("\nNo sprout worktrees found for this repository.")
			return
		}
		printRepos([]RepoInfo{repo}, false)
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVar(&listAllFlag, "all", false, "List worktrees from all repositories")
}

// RepoInfo holds information about a repository and its worktrees.
type RepoInfo struct {
	Name      string
	MainPath  string
	Worktrees []WorktreeInfo
}

// WorktreeInfo holds information about a single worktree.
type WorktreeInfo struct {
	Worktree git.Worktree
	Status   git.WorktreeStatus
	IsMain   bool
}

// RepoWorktrees holds worktrees for a specific repository (legacy type for compatibility).
type RepoWorktrees struct {
	RepoRoot  string
	Worktrees []git.Worktree
}

// collectCurrentRepo gathers information about the current repository.
// Returns (repo, true, nil) if sprout worktrees exist.
// Returns (empty, false, nil) if no sprout worktrees exist (not an error).
// Returns (empty, false, err) if git introspection fails or repo has no worktrees at all.
func collectCurrentRepo() (RepoInfo, bool, error) {
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return RepoInfo{}, false, err
	}

	allWorktrees, err := git.ListWorktrees(repoRoot)
	if err != nil {
		return RepoInfo{}, false, fmt.Errorf("failed to list worktrees: %w", err)
	}

	if len(allWorktrees) == 0 {
		return RepoInfo{}, false, fmt.Errorf("no worktrees found")
	}

	// First worktree is always the main repo
	mainWorktree := allWorktrees[0]

	// Filter to only sprout-managed worktrees
	sproutRoot, err := sprout.GetSproutRoot()
	if err != nil {
		return RepoInfo{}, false, fmt.Errorf("failed to get sprout root: %w", err)
	}
	sproutWorktrees := core.FilterSproutWorktrees(allWorktrees[1:], sproutRoot)
	sproutWorktrees = filterExistingWorktrees(sproutWorktrees)

	if len(sproutWorktrees) == 0 {
		return RepoInfo{}, false, nil // No sprout worktrees is not an error
	}

	return buildRepoInfo(filepath.Base(mainWorktree.Path), mainWorktree, sproutWorktrees), true, nil
}

// collectAllRepos discovers all sprout-managed repositories.
// Returns nil, nil if no repositories are found (not an error).
func collectAllRepos() ([]RepoInfo, error) {
	repoDirs, err := findAllRepoDirectories()
	if err != nil {
		return nil, fmt.Errorf("failed to scan sprout directories: %w", err)
	}
	if len(repoDirs) == 0 {
		return nil, nil
	}

	repoMap := discoverReposParallel(repoDirs)

	// Convert map to sorted slice
	repos := make([]RepoInfo, 0, len(repoMap))
	for _, repo := range repoMap {
		repos = append(repos, repo)
	}

	sort.Slice(repos, func(i, j int) bool {
		return repos[i].MainPath < repos[j].MainPath
	})

	return repos, nil
}

// findAllRepoDirectories scans the sprout root for repository directories.
// Returns nil, nil if sprout directory doesn't exist (user hasn't used sprout yet).
// Returns error if sprout directory exists but can't be read (permissions, IO error).
func findAllRepoDirectories() ([]string, error) {
	sproutRoot, err := sprout.GetSproutRoot()
	if err != nil {
		return nil, fmt.Errorf("get sprout root: %w", err)
	}

	// Check if sprout directory exists
	if _, err := os.Stat(sproutRoot); err != nil {
		if os.IsNotExist(err) {
			// Not an error - user just hasn't created any worktrees yet
			return nil, nil
		}
		// Real error - permissions, IO problem, etc.
		return nil, fmt.Errorf("stat sprout directory: %w", err)
	}

	entries, err := os.ReadDir(sproutRoot)
	if err != nil {
		return nil, fmt.Errorf("read sprout directory: %w", err)
	}

	var repoDirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			repoDirs = append(repoDirs, filepath.Join(sproutRoot, entry.Name()))
		}
	}

	return repoDirs, nil
}

// discoverReposParallel processes repo directories in parallel and returns a map of repos.
func discoverReposParallel(repoDirs []string) map[string]RepoInfo {
	var mu sync.Mutex
	var wg sync.WaitGroup
	repoMap := make(map[string]RepoInfo)

	for _, repoDir := range repoDirs {
		wg.Add(1)
		go func(dir string) {
			defer wg.Done()

			repo, ok := processRepoDirectory(dir)
			if !ok {
				return
			}

			mu.Lock()
			if _, exists := repoMap[repo.MainPath]; !exists {
				repoMap[repo.MainPath] = repo
			}
			mu.Unlock()
		}(repoDir)
	}

	wg.Wait()
	return repoMap
}

// processRepoDirectory processes a single repo directory and returns repo info.
func processRepoDirectory(repoDir string) (RepoInfo, bool) {
	// Find any worktree in this repo dir
	anyWorktree := findFirstWorktree(repoDir)
	if anyWorktree == "" {
		return RepoInfo{}, false
	}

	// Get all worktrees for this repo
	allWorktrees, err := git.ListWorktrees(anyWorktree)
	if err != nil || len(allWorktrees) == 0 {
		return RepoInfo{}, false
	}

	// First worktree is the main repo
	mainWorktree := allWorktrees[0]

	// Filter to only sprout-managed worktrees
	sproutRoot, err := sprout.GetSproutRoot()
	if err != nil {
		return RepoInfo{}, false
	}
	sproutWorktrees := core.FilterSproutWorktrees(allWorktrees[1:], sproutRoot)
	sproutWorktrees = filterExistingWorktrees(sproutWorktrees)

	if len(sproutWorktrees) == 0 {
		return RepoInfo{}, false
	}

	repoName := filepath.Base(mainWorktree.Path)
	return buildRepoInfo(repoName, mainWorktree, sproutWorktrees), true
}

// buildRepoInfo creates a RepoInfo with parallel status collection.
func buildRepoInfo(name string, mainWorktree git.Worktree, sproutWorktrees []git.Worktree) RepoInfo {
	totalWorktrees := 1 + len(sproutWorktrees)
	worktrees := make([]WorktreeInfo, totalWorktrees)
	var wg sync.WaitGroup

	// Collect status for main worktree
	wg.Add(1)
	go func() {
		defer wg.Done()
		worktrees[0] = WorktreeInfo{
			Worktree: mainWorktree,
			Status:   git.GetWorktreeStatus(mainWorktree.Path),
			IsMain:   true,
		}
	}()

	// Collect status for sprout worktrees
	for i, wt := range sproutWorktrees {
		wg.Add(1)
		go func(idx int, worktree git.Worktree) {
			defer wg.Done()
			worktrees[idx+1] = WorktreeInfo{
				Worktree: worktree,
				Status:   git.GetWorktreeStatus(worktree.Path),
				IsMain:   false,
			}
		}(i, wt)
	}

	wg.Wait()

	return RepoInfo{
		Name:      name,
		MainPath:  mainWorktree.Path,
		Worktrees: worktrees,
	}
}

// printRepos prints repository information with proper formatting.
func printRepos(repos []RepoInfo, showRepoHeaders bool) {
	if len(repos) == 0 {
		fmt.Println("No sprout worktrees found.")
		return
	}

	fmt.Println() // Add spacing from prompt

	for i, repo := range repos {
		if showRepoHeaders {
			if i > 0 {
				fmt.Println() // Blank line between repos
			}
			fmt.Printf("\033[1m%s\033[0m\n", repo.Name)
		}

		for j, wt := range repo.Worktrees {
			isLast := j == len(repo.Worktrees)-1
			printWorktree(wt, isLast, showRepoHeaders)
		}
	}
}

// printWorktree prints a single worktree with formatting.
func printWorktree(wt WorktreeInfo, isLast bool, useTreeLines bool) {
	display := core.WorktreeDisplay{
		Branch:       wt.Worktree.Branch,
		Path:         core.ShortenPath(wt.Worktree.Path),
		StatusEmojis: core.BuildStatusEmojis(wt.Status),
		IsMain:       wt.IsMain,
		IsLast:       isLast,
		UseTreeLines: useTreeLines,
	}
	fmt.Println(core.FormatWorktree(display))
}

// findFirstWorktree does a shallow scan to find any worktree in the repo directory.
// Sprout structure can be:
//   - <repo-dir>/<branch>/.git (flat, older structure)
//   - <repo-dir>/<branch>/<repo-slug>/.git (nested, newer structure)
//   - <repo-dir>/<branch>/<repo-slug>/<repo-slug>/.git (double-nested, migration artifact)
//
// We scan up to 3 levels deep and return the first WORKING worktree.
func findFirstWorktree(repoDir string) string {
	candidates := scanForGitDirs(repoDir, 3)

	// Try each candidate and return the first one where git worktree list works
	for _, candidate := range candidates {
		if _, err := git.ListWorktrees(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

// scanForGitDirs recursively scans for directories containing .git up to maxDepth levels.
func scanForGitDirs(rootDir string, maxDepth int) []string {
	var candidates []string
	scanLevel(rootDir, 0, maxDepth, &candidates)
	return candidates
}

// scanLevel recursively scans a single level.
func scanLevel(dir string, currentDepth, maxDepth int, candidates *[]string) {
	if currentDepth >= maxDepth {
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		entryPath := filepath.Join(dir, entry.Name())

		// Check if this directory has .git
		if _, err := os.Stat(filepath.Join(entryPath, ".git")); err == nil {
			*candidates = append(*candidates, entryPath)
		}

		// Recurse to next level
		scanLevel(entryPath, currentDepth+1, maxDepth, candidates)
	}
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
