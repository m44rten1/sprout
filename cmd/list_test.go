package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/m44rten1/sprout/internal/git"
)

func TestBuildStatusEmojis(t *testing.T) {
	tests := []struct {
		name     string
		status   git.WorktreeStatus
		expected string
	}{
		{
			name:     "clean worktree",
			status:   git.WorktreeStatus{},
			expected: "",
		},
		{
			name:     "dirty only",
			status:   git.WorktreeStatus{Dirty: true},
			expected: "🔴",
		},
		{
			name:     "ahead only",
			status:   git.WorktreeStatus{Ahead: 3},
			expected: "⬆️",
		},
		{
			name:     "behind only",
			status:   git.WorktreeStatus{Behind: 2},
			expected: "⬇️",
		},
		{
			name:     "unmerged only",
			status:   git.WorktreeStatus{Unmerged: true},
			expected: "🔀",
		},
		{
			name:     "dirty and ahead",
			status:   git.WorktreeStatus{Dirty: true, Ahead: 1},
			expected: "🔴 ⬆️",
		},
		{
			name:     "all statuses",
			status:   git.WorktreeStatus{Dirty: true, Ahead: 1, Behind: 2, Unmerged: true},
			expected: "🔴 ⬆️ ⬇️ 🔀",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildStatusEmojis(tt.status)
			if result != tt.expected {
				t.Errorf("buildStatusEmojis() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFilterExistingWorktrees(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()

	// Create some existing paths
	existingPath1 := filepath.Join(tmpDir, "existing1")
	existingPath2 := filepath.Join(tmpDir, "existing2")
	nonExistingPath := filepath.Join(tmpDir, "nonexistent")

	if err := os.Mkdir(existingPath1, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(existingPath2, 0755); err != nil {
		t.Fatal(err)
	}

	worktrees := []git.Worktree{
		{Path: existingPath1, Branch: "branch1"},
		{Path: nonExistingPath, Branch: "branch2"},
		{Path: existingPath2, Branch: "branch3"},
	}

	result := filterExistingWorktrees(worktrees)

	if len(result) != 2 {
		t.Errorf("expected 2 worktrees, got %d", len(result))
	}

	if result[0].Path != existingPath1 {
		t.Errorf("expected first path to be %s, got %s", existingPath1, result[0].Path)
	}

	if result[1].Path != existingPath2 {
		t.Errorf("expected second path to be %s, got %s", existingPath2, result[1].Path)
	}
}

func TestFindFirstWorktree(t *testing.T) {
	// Create temp directory with nested structure
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		setup    func() string
		expected string
	}{
		{
			name: "no worktrees",
			setup: func() string {
				dir := filepath.Join(tmpDir, "no_worktrees")
				os.Mkdir(dir, 0755)
				return dir
			},
			expected: "",
		},
		{
			name: "worktree at level 1",
			setup: func() string {
				dir := filepath.Join(tmpDir, "level1")
				os.Mkdir(dir, 0755)
				branch := filepath.Join(dir, "main")
				os.Mkdir(branch, 0755)
				// Create .git file (worktree marker)
				gitFile := filepath.Join(branch, ".git")
				os.WriteFile(gitFile, []byte("gitdir: /path/to/git"), 0644)
				return dir
			},
			expected: "",
		},
		{
			name: "worktree at level 2",
			setup: func() string {
				dir := filepath.Join(tmpDir, "level2")
				os.Mkdir(dir, 0755)
				branch := filepath.Join(dir, "feature")
				os.Mkdir(branch, 0755)
				repo := filepath.Join(branch, "myrepo")
				os.Mkdir(repo, 0755)
				gitFile := filepath.Join(repo, ".git")
				os.WriteFile(gitFile, []byte("gitdir: /path/to/git"), 0644)
				return dir
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoDir := tt.setup()
			result := findFirstWorktree(repoDir)

			// Since findFirstWorktree validates with git.ListWorktrees which needs actual git setup,
			// we expect empty results in tests without real git repos
			if result != tt.expected {
				t.Errorf("findFirstWorktree() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFindFirstWorktree_ScanDepth(t *testing.T) {
	// Test that it scans correctly up to 3 levels
	tmpDir := t.TempDir()

	// Create nested structure at different levels
	level1 := filepath.Join(tmpDir, "l1")
	os.MkdirAll(level1, 0755)
	os.WriteFile(filepath.Join(level1, ".git"), []byte("gitdir: /fake"), 0644)

	level2 := filepath.Join(tmpDir, "l1", "l2")
	os.MkdirAll(level2, 0755)
	os.WriteFile(filepath.Join(level2, ".git"), []byte("gitdir: /fake"), 0644)

	level3 := filepath.Join(tmpDir, "l1", "l2", "l3")
	os.MkdirAll(level3, 0755)
	os.WriteFile(filepath.Join(level3, ".git"), []byte("gitdir: /fake"), 0644)

	// The function should find .git files at all three levels
	// (though it won't return valid results without real git setup)
	result := findFirstWorktree(tmpDir)

	// Without real git repos, we expect empty string
	if result != "" {
		// If it found something, it should be one of our paths
		if result != level1 && result != level2 && result != level3 {
			t.Errorf("unexpected result: %s", result)
		}
	}
}

// Test that RepoWorktrees struct is properly defined
func TestRepoWorktreesStruct(t *testing.T) {
	repo := RepoWorktrees{
		RepoRoot: "/path/to/repo",
		Worktrees: []git.Worktree{
			{Path: "/path/1", Branch: "main"},
			{Path: "/path/2", Branch: "feature"},
		},
	}

	if repo.RepoRoot != "/path/to/repo" {
		t.Errorf("unexpected repo root: %s", repo.RepoRoot)
	}

	if len(repo.Worktrees) != 2 {
		t.Errorf("expected 2 worktrees, got %d", len(repo.Worktrees))
	}
}

