package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/m44rten1/sprout/internal/core"
	"github.com/m44rten1/sprout/internal/effects"
	"github.com/m44rten1/sprout/internal/git"

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
		fx := effects.NewRealEffects()

		// 1. Gather (imperative - uses Effects)
		ctx, err := BuildListContext(fx, listAllFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// 2. Format (pure - no I/O)
		output := core.FormatListOutput(ctx)

		// 3. Output (imperative)
		fx.Print(output)
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVar(&listAllFlag, "all", false, "List worktrees from all repositories")
}

// BuildListContext gathers all data needed for the list command.
// This is the imperative "gather" step of the FCIS sandwich.
func BuildListContext(fx effects.Effects, all bool) (core.ListContext, error) {
	var repos []core.RepoDisplay
	var err error

	if all {
		repos, err = collectAllReposWithEffects(fx)
	} else {
		var repo core.RepoDisplay
		var found bool
		repo, found, err = collectCurrentRepoWithEffects(fx)
		if err == nil && found {
			repos = []core.RepoDisplay{repo}
		}
	}

	if err != nil {
		return core.ListContext{}, err
	}

	home, _ := fx.UserHomeDir()

	return core.ListContext{
		Repos:   repos,
		Home:    home,
		ShowAll: all,
	}, nil
}

// collectCurrentRepoWithEffects gathers information about the current repository using Effects.
// Returns (repo, true, nil) if sprout worktrees exist.
// Returns (empty, false, nil) if no sprout worktrees exist (not an error).
// Returns (empty, false, err) if git introspection fails or repo has no worktrees at all.
func collectCurrentRepoWithEffects(fx effects.Effects) (core.RepoDisplay, bool, error) {
	repoRoot, err := fx.GetRepoRoot()
	if err != nil {
		return core.RepoDisplay{}, false, err
	}

	allWorktrees, err := fx.ListWorktrees(repoRoot)
	if err != nil {
		return core.RepoDisplay{}, false, fmt.Errorf("failed to list worktrees: %w", err)
	}

	if len(allWorktrees) == 0 {
		return core.RepoDisplay{}, false, fmt.Errorf("no worktrees found")
	}

	// First worktree is always the main repo
	mainWorktree := allWorktrees[0]

	// Filter to only sprout-managed worktrees
	sproutRoot, err := fx.GetSproutRoot()
	if err != nil {
		return core.RepoDisplay{}, false, fmt.Errorf("failed to get sprout root: %w", err)
	}
	sproutWorktrees := core.FilterSproutWorktrees(allWorktrees[1:], sproutRoot)
	sproutWorktrees = filterExistingWorktreesWithEffects(fx, sproutWorktrees)

	if len(sproutWorktrees) == 0 {
		return core.RepoDisplay{}, false, nil // No sprout worktrees is not an error
	}

	return buildRepoDisplayWithEffects(fx, filepath.Base(mainWorktree.Path), mainWorktree, sproutWorktrees), true, nil
}

// collectAllReposWithEffects discovers all sprout-managed repositories using Effects.
// Returns nil, nil if no repositories are found (not an error).
func collectAllReposWithEffects(fx effects.Effects) ([]core.RepoDisplay, error) {
	repoDirs, err := findAllRepoDirectoriesWithEffects(fx)
	if err != nil {
		return nil, fmt.Errorf("failed to scan sprout directories: %w", err)
	}
	if len(repoDirs) == 0 {
		return nil, nil
	}

	repoMap := discoverReposParallelWithEffects(fx, repoDirs)

	// Convert map to sorted slice
	repos := make([]core.RepoDisplay, 0, len(repoMap))
	for _, repo := range repoMap {
		repos = append(repos, repo)
	}

	sort.Slice(repos, func(i, j int) bool {
		return repos[i].MainPath < repos[j].MainPath
	})

	return repos, nil
}

// findAllRepoDirectoriesWithEffects scans the sprout root for repository directories using Effects.
// Returns nil, nil if sprout directory doesn't exist (user hasn't used sprout yet).
// Returns error if sprout directory exists but can't be read (permissions, IO error).
func findAllRepoDirectoriesWithEffects(fx effects.Effects) ([]string, error) {
	sproutRoot, err := fx.GetSproutRoot()
	if err != nil {
		return nil, fmt.Errorf("get sprout root: %w", err)
	}

	// Check if sprout directory exists
	if !fx.FileExists(sproutRoot) {
		// Not an error - user just hasn't created any worktrees yet
		return nil, nil
	}

	entries, err := fx.ReadDir(sproutRoot)
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

// discoverReposParallelWithEffects processes repo directories in parallel and returns a map of repos.
func discoverReposParallelWithEffects(fx effects.Effects, repoDirs []string) map[string]core.RepoDisplay {
	var mu sync.Mutex
	var wg sync.WaitGroup
	repoMap := make(map[string]core.RepoDisplay)

	for _, repoDir := range repoDirs {
		wg.Add(1)
		go func(dir string) {
			defer wg.Done()

			repo, ok := processRepoDirectoryWithEffects(fx, dir)
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

// processRepoDirectoryWithEffects processes a single repo directory and returns repo info.
func processRepoDirectoryWithEffects(fx effects.Effects, repoDir string) (core.RepoDisplay, bool) {
	// Find any worktree in this repo dir
	anyWorktree := findFirstWorktreeWithEffects(fx, repoDir)
	if anyWorktree == "" {
		return core.RepoDisplay{}, false
	}

	// Get all worktrees for this repo
	allWorktrees, err := fx.ListWorktrees(anyWorktree)
	if err != nil || len(allWorktrees) == 0 {
		return core.RepoDisplay{}, false
	}

	// First worktree is the main repo
	mainWorktree := allWorktrees[0]

	// Filter to only sprout-managed worktrees
	sproutRoot, err := fx.GetSproutRoot()
	if err != nil {
		return core.RepoDisplay{}, false
	}
	sproutWorktrees := core.FilterSproutWorktrees(allWorktrees[1:], sproutRoot)
	sproutWorktrees = filterExistingWorktreesWithEffects(fx, sproutWorktrees)

	if len(sproutWorktrees) == 0 {
		return core.RepoDisplay{}, false
	}

	repoName := filepath.Base(mainWorktree.Path)
	return buildRepoDisplayWithEffects(fx, repoName, mainWorktree, sproutWorktrees), true
}

// buildRepoDisplayWithEffects creates a RepoDisplay with parallel status collection.
func buildRepoDisplayWithEffects(fx effects.Effects, name string, mainWorktree git.Worktree, sproutWorktrees []git.Worktree) core.RepoDisplay {
	totalWorktrees := 1 + len(sproutWorktrees)
	worktrees := make([]core.WorktreeDisplayItem, totalWorktrees)
	var wg sync.WaitGroup

	// Collect status for main worktree
	wg.Add(1)
	go func() {
		defer wg.Done()
		worktrees[0] = core.WorktreeDisplayItem{
			Branch: mainWorktree.Branch,
			Path:   mainWorktree.Path,
			Status: fx.GetWorktreeStatus(mainWorktree.Path),
			IsMain: true,
		}
	}()

	// Collect status for sprout worktrees
	for i, wt := range sproutWorktrees {
		wg.Add(1)
		go func(idx int, worktree git.Worktree) {
			defer wg.Done()
			worktrees[idx+1] = core.WorktreeDisplayItem{
				Branch: worktree.Branch,
				Path:   worktree.Path,
				Status: fx.GetWorktreeStatus(worktree.Path),
				IsMain: false,
			}
		}(i, wt)
	}

	wg.Wait()

	return core.RepoDisplay{
		Name:      name,
		MainPath:  mainWorktree.Path,
		Worktrees: worktrees,
	}
}

// findFirstWorktreeWithEffects does a shallow scan to find any worktree in the repo directory.
// Sprout structure can be:
//   - <repo-dir>/<branch>/.git (flat, older structure)
//   - <repo-dir>/<branch>/<repo-slug>/.git (nested, newer structure)
//   - <repo-dir>/<branch>/<repo-slug>/<repo-slug>/.git (double-nested, migration artifact)
//
// We scan up to 3 levels deep and return the first WORKING worktree.
func findFirstWorktreeWithEffects(fx effects.Effects, repoDir string) string {
	candidates := scanForGitDirsWithEffects(fx, repoDir, 3)

	// Try each candidate and return the first one where git worktree list works
	for _, candidate := range candidates {
		if _, err := fx.ListWorktrees(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

// scanForGitDirsWithEffects recursively scans for directories containing .git up to maxDepth levels.
func scanForGitDirsWithEffects(fx effects.Effects, rootDir string, maxDepth int) []string {
	var candidates []string
	scanLevelWithEffects(fx, rootDir, 0, maxDepth, &candidates)
	return candidates
}

// scanLevelWithEffects recursively scans a single level.
func scanLevelWithEffects(fx effects.Effects, dir string, currentDepth, maxDepth int, candidates *[]string) {
	if currentDepth >= maxDepth {
		return
	}

	entries, err := fx.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		entryPath := filepath.Join(dir, entry.Name())

		// Check if this directory has .git
		if fx.FileExists(filepath.Join(entryPath, ".git")) {
			*candidates = append(*candidates, entryPath)
		}

		// Recurse to next level
		scanLevelWithEffects(fx, entryPath, currentDepth+1, maxDepth, candidates)
	}
}

// filterExistingWorktreesWithEffects filters out worktrees whose paths don't exist on the filesystem.
func filterExistingWorktreesWithEffects(fx effects.Effects, worktrees []git.Worktree) []git.Worktree {
	var existing []git.Worktree
	for _, wt := range worktrees {
		if fx.FileExists(wt.Path) {
			existing = append(existing, wt)
		}
	}
	return existing
}
