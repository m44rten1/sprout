package sprout

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GetSproutRoot returns the root directory for sprout worktrees.
// Uses $XDG_DATA_HOME/sprout if XDG_DATA_HOME is set, otherwise ~/.local/share/sprout
func GetSproutRoot() (string, error) {
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		return filepath.Join(xdgData, "sprout"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(home, ".local", "share", "sprout"), nil
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
	// Validate branch name to prevent path traversal attacks
	if err := validateBranchName(branch); err != nil {
		return "", err
	}

	root, err := GetWorktreeRoot(repoPath)
	if err != nil {
		return "", err
	}
	repoSlug := filepath.Base(repoPath)
	return filepath.Join(root, branch, repoSlug), nil
}

// validateBranchName checks if a branch name contains dangerous path components
// that could allow escaping the sprout root directory.
func validateBranchName(branch string) error {
	if branch == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Check for absolute paths
	if filepath.IsAbs(branch) {
		return fmt.Errorf("branch name cannot be an absolute path: %s", branch)
	}

	// Normalize to forward slashes for consistent checking across platforms
	normalized := filepath.ToSlash(branch)

	// Check for path traversal attempts
	// This catches: "..", "../foo", "foo/..", "foo/../bar", etc.
	if normalized == ".." ||
		normalized == "." ||
		normalized == "../" ||
		normalized == "./" ||
		strings.HasPrefix(normalized, "../") ||
		strings.HasSuffix(normalized, "/..") ||
		strings.Contains(normalized, "/../") {
		return fmt.Errorf("branch name cannot contain '..' path components: %s", branch)
	}

	return nil
}
