package cmd

import (
	"path/filepath"
	"strings"

	"github.com/m44rten1/sprout/internal/git"
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
