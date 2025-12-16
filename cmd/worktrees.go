package cmd

import (
	"path/filepath"
	"strings"

	"github.com/m44rten1/sprout/internal/git"
	"github.com/m44rten1/sprout/internal/sprout"
)

// filterSproutWorktrees keeps only worktrees located under the sprout-managed root.
func filterSproutWorktrees(worktrees []git.Worktree, sproutRoot string) []git.Worktree {
	var filtered []git.Worktree
	for _, wt := range worktrees {
		if isUnderSproutRoot(wt.Path, sproutRoot) {
			filtered = append(filtered, wt)
		}
	}
	return filtered
}

// filterSproutWorktreesAllRoots filters worktrees to only those under any possible sprout root.
func filterSproutWorktreesAllRoots(worktrees []git.Worktree) []git.Worktree {
	sproutRoots := sprout.GetAllPossibleSproutRoots()
	var filtered []git.Worktree
	for _, wt := range worktrees {
		for _, sproutRoot := range sproutRoots {
			if isUnderSproutRoot(wt.Path, sproutRoot) {
				filtered = append(filtered, wt)
				break
			}
		}
	}
	return filtered
}

// findWorktreeByBranch finds a sprout-managed worktree by branch name.
// Returns the worktree path and true if found, empty string and false otherwise.
func findWorktreeByBranch(worktrees []git.Worktree, branch string) (string, bool) {
	sproutRoots := sprout.GetAllPossibleSproutRoots()
	for _, wt := range worktrees {
		if wt.Branch == branch {
			for _, sproutRoot := range sproutRoots {
				if isUnderSproutRoot(wt.Path, sproutRoot) {
					return wt.Path, true
				}
			}
		}
	}
	return "", false
}

// isUnderSproutRoot reports whether the path is inside the sprout worktree root.
func isUnderSproutRoot(path, sproutRoot string) bool {
	if path == "" || sproutRoot == "" {
		return false
	}

	rel, err := filepath.Rel(sproutRoot, path)
	if err != nil {
		return false
	}

	if rel == "." {
		return false
	}

	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
