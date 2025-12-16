package sprout

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
)

// GetSproutRoot returns the root directory for sprout worktrees.
// Respects XDG_DATA_HOME, falling back to ~/.local/share/sprout
func GetSproutRoot() (string, error) {
	roots := GetAllPossibleSproutRoots()
	if len(roots) == 0 {
		return "", fmt.Errorf("failed to get user home directory")
	}
	return roots[0], nil
}

// GetAllPossibleSproutRoots returns all possible sprout root directories,
// including legacy locations for backward compatibility.
// Order: XDG_DATA_HOME/sprout, ~/.local/share/sprout, ~/.sprout (legacy)
func GetAllPossibleSproutRoots() []string {
	var roots []string

	// Add XDG_DATA_HOME location if set
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		roots = append(roots, filepath.Join(xdgData, "sprout"))
	}

	// Add ~/.local/share/sprout and legacy ~/.sprout
	if home, err := os.UserHomeDir(); err == nil {
		roots = append(roots, filepath.Join(home, ".local", "share", "sprout"))
		// Add ~/.sprout for backward compatibility
		roots = append(roots, filepath.Join(home, ".sprout"))
	}

	return roots
}

// GetRepoID computes a stable identifier for a repository based on its absolute path.
// It returns the first 8 characters of the SHA1 hash of the path.
func GetRepoID(repoPath string) string {
	hash := sha1.Sum([]byte(repoPath))
	return fmt.Sprintf("%x", hash)[:8]
}

// GetWorktreeRoot returns the root directory for worktrees of a specific repository.
// Format: ~/.sprout/<repo-slug>-<repo-id>
func GetWorktreeRoot(repoPath string) (string, error) {
	sproutRoot, err := GetSproutRoot()
	if err != nil {
		return "", err
	}

	repoSlug := filepath.Base(repoPath)
	repoID := GetRepoID(repoPath)
	return filepath.Join(sproutRoot, fmt.Sprintf("%s-%s", repoSlug, repoID)), nil
}

// GetWorktreePath returns the full path for a worktree given the repo path and branch name.
// This now includes nesting the worktree inside a folder named after the repo.
// Format: ~/.sprout/<repo-slug>-<repo-id>/<branch>/<repo-slug>/
func GetWorktreePath(repoPath, branch string) (string, error) {
	root, err := GetWorktreeRoot(repoPath)
	if err != nil {
		return "", err
	}
	repoSlug := filepath.Base(repoPath)
	return filepath.Join(root, branch, repoSlug), nil
}
