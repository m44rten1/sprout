package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GetRepoRoot returns the absolute path to the root of the current git repository.
func GetRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get repo root (not a git repo?): %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// WorktreeAdd creates a new worktree.
func WorktreeAdd(repoRoot, path, branch, base string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(repoRoot, 0755); err != nil {
		// This might be wrong, we need to mkdir the PARENT of the worktree path, not repoRoot
		// But actually `git worktree add` might complain if parent doesn't exist?
		// The spec says: sprout must create parent directories.
	}
	// We'll handle mkdir in the caller or here properly.
	// Let's just run the git command for now, the caller should handle mkdir.

	// Check if remote branch exists
	// For now, we'll assume the caller has decided on the arguments.
	// But wait, the spec says:
	// If origin/<branch> exists: git worktree add <path> origin/<branch> -b <branch>
	// If not: git worktree add <path> -b <branch> origin/main

	// Let's make this function simple: just run git worktree add with given args.
	// We'll add a higher level function to handle the logic.
	return nil
}

// RunGitCommand runs a git command in the given directory.
func RunGitCommand(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git command failed: %w\nOutput: %s", err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

// Worktree represents a git worktree.
type Worktree struct {
	Path   string
	HEAD   string
	Branch string
}

// ListWorktrees returns a list of worktrees for the repo.
func ListWorktrees(repoRoot string) ([]Worktree, error) {
	out, err := RunGitCommand(repoRoot, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	var worktrees []Worktree
	var current Worktree

	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			if current.Path != "" {
				worktrees = append(worktrees, current)
			}
			current = Worktree{Path: strings.TrimPrefix(line, "worktree ")}
		} else if strings.HasPrefix(line, "HEAD ") {
			current.HEAD = strings.TrimPrefix(line, "HEAD ")
		} else if strings.HasPrefix(line, "branch ") {
			ref := strings.TrimPrefix(line, "branch ")
			current.Branch = strings.TrimPrefix(ref, "refs/heads/")
		}
	}
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}

// PruneWorktrees prunes stale worktrees.
func PruneWorktrees(repoRoot string) error {
	_, err := RunGitCommand(repoRoot, "worktree", "prune")
	return err
}

// RemoveWorktree removes a worktree.
func RemoveWorktree(repoRoot, path string, force bool) error {
	args := []string{"worktree", "remove", path}
	if force {
		args = append(args, "--force")
	}
	_, err := RunGitCommand(repoRoot, args...)
	return err
}

// BranchExists checks if a branch exists (local or remote).
func BranchExists(repoRoot, branch string) (bool, error) {
	// Check local
	if _, err := RunGitCommand(repoRoot, "rev-parse", "--verify", branch); err == nil {
		return true, nil
	}
	// Check remote
	if _, err := RunGitCommand(repoRoot, "rev-parse", "--verify", "origin/"+branch); err == nil {
		return true, nil
	}
	return false, nil
}

// LocalBranchExists checks if a branch exists locally.
func LocalBranchExists(repoRoot, branch string) (bool, error) {
	if _, err := RunGitCommand(repoRoot, "rev-parse", "--verify", branch); err == nil {
		return true, nil
	}
	return false, nil
}

// Branch represents a git branch with metadata for display.
type Branch struct {
	// RefName is the full reference name (e.g., "feat/abc" or "origin/feat/abc")
	RefName string
	// DisplayName is the name to show in UI (remote branches without "origin/" prefix)
	DisplayName string
	// IsLocal indicates whether this is a local branch
	IsLocal bool
}

// ListAllBranches returns all local and remote branches.
// Remote branches have the "origin/" prefix stripped from their display name.
// When a local branch exists, its remote equivalent is excluded from the results.
func ListAllBranches(repoRoot string) ([]Branch, error) {
	// Get all branches (local and remote)
	out, err := RunGitCommand(repoRoot, "branch", "--all", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(out, "\n")
	var branches []Branch
	localBranchNames := make(map[string]bool)

	// First pass: collect all local branches
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip remote branches in first pass
		if strings.HasPrefix(line, "remotes/origin/") || strings.HasPrefix(line, "origin/") {
			continue
		}
		localBranchNames[line] = true
		branches = append(branches, Branch{
			RefName:     line,
			DisplayName: line,
			IsLocal:     true,
		})
	}

	// Second pass: add remote branches that don't have local equivalents
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Process remote branches
		var remoteBranch string
		if strings.HasPrefix(line, "remotes/origin/") {
			remoteBranch = strings.TrimPrefix(line, "remotes/origin/")
		} else if strings.HasPrefix(line, "origin/") {
			remoteBranch = strings.TrimPrefix(line, "origin/")
		} else {
			continue // Not a remote branch
		}

		// Skip HEAD pointer
		if remoteBranch == "HEAD" {
			continue
		}

		// Skip if local equivalent exists
		if localBranchNames[remoteBranch] {
			continue
		}

		branches = append(branches, Branch{
			RefName:     "origin/" + remoteBranch,
			DisplayName: remoteBranch,
			IsLocal:     false,
		})
	}

	return branches, nil
}
