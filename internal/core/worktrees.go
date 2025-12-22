// Package core contains pure functions for sprout's business logic.
// All path operations use string-based comparison and do not resolve symlinks.
package core

import (
	"path/filepath"
	"strings"

	"github.com/m44rten1/sprout/internal/git"
)

// FilterSproutWorktrees returns worktrees located under the given sprout root.
// Worktrees at the root level itself are excluded (must be descendants).
func FilterSproutWorktrees(worktrees []git.Worktree, sproutRoot string) []git.Worktree {
	filtered := make([]git.Worktree, 0, len(worktrees))
	for _, wt := range worktrees {
		if IsUnderSproutRoot(wt.Path, sproutRoot) {
			filtered = append(filtered, wt)
		}
	}
	return filtered
}

// FindWorktreeByBranch finds the first worktree under the sprout root matching the given branch.
// Returns the worktree path and true if found, empty string and false otherwise.
// Empty branch name never matches (excludes detached HEAD worktrees).
func FindWorktreeByBranch(worktrees []git.Worktree, sproutRoot string, branch string) (string, bool) {
	if branch == "" {
		return "", false
	}

	for _, wt := range worktrees {
		if wt.Branch == branch && IsUnderSproutRoot(wt.Path, sproutRoot) {
			return wt.Path, true
		}
	}
	return "", false
}

// IsUnderSproutRoot reports whether path is a descendant of sproutRoot.
// Returns false if path equals sproutRoot (not a descendant, but the root itself).
// Both paths are normalized and converted to absolute paths for consistent comparison.
// Note: This is lexical (string-based) and does not resolve symlinks.
func IsUnderSproutRoot(path, sproutRoot string) bool {
	if path == "" || sproutRoot == "" {
		return false
	}

	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return false
	}
	absRoot, err := filepath.Abs(filepath.Clean(sproutRoot))
	if err != nil {
		return false
	}

	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return false
	}

	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}

	return true
}
