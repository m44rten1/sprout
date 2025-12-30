package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/m44rten1/sprout/internal/effects"
	"github.com/m44rten1/sprout/internal/git"
	"github.com/stretchr/testify/assert"
)

// Note: BuildStatusEmojis and ShortenPath tests moved to internal/core/list_test.go

// Test helper to create directories, fails test on error
func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("failed to create directory %s: %v", path, err)
	}
}

// Test helper to create files, fails test on error
func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

func TestFilterExistingWorktrees(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create some existing paths
	existingPath1 := filepath.Join(tmpDir, "existing1")
	existingPath2 := filepath.Join(tmpDir, "existing2")
	nonExistingPath := filepath.Join(tmpDir, "nonexistent")

	mustMkdirAll(t, existingPath1)
	mustMkdirAll(t, existingPath2)

	worktrees := []git.Worktree{
		{Path: existingPath1, Branch: "branch1"},
		{Path: nonExistingPath, Branch: "branch2"},
		{Path: existingPath2, Branch: "branch3"},
	}

	fx := effects.NewTestEffects()
	fx.Files[existingPath1] = true
	fx.Files[existingPath2] = true
	result := filterExistingWorktreesWithEffects(fx, worktrees)

	assert.Len(t, result, 2, "should filter out non-existing path")
	assert.Equal(t, existingPath1, result[0].Path, "should preserve order")
	assert.Equal(t, existingPath2, result[1].Path, "should preserve order")
}

func TestScanForGitDirs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(t *testing.T, tmpDir string) []string // returns expected paths
		maxDepth int
	}{
		{
			name: "no git dirs",
			setup: func(t *testing.T, tmpDir string) []string {
				mustMkdirAll(t, filepath.Join(tmpDir, "a", "b", "c"))
				return nil
			},
			maxDepth: 3,
		},
		{
			name: "git dir at level 1",
			setup: func(t *testing.T, tmpDir string) []string {
				level1 := filepath.Join(tmpDir, "repo")
				mustMkdirAll(t, level1)
				mustWriteFile(t, filepath.Join(level1, ".git"), "gitdir: /fake")
				return []string{level1}
			},
			maxDepth: 3,
		},
		{
			name: "git dirs at multiple levels",
			setup: func(t *testing.T, tmpDir string) []string {
				level1 := filepath.Join(tmpDir, "l1")
				level2 := filepath.Join(level1, "l2")
				level3 := filepath.Join(level2, "l3")

				mustMkdirAll(t, level3)
				mustWriteFile(t, filepath.Join(level1, ".git"), "gitdir: /fake")
				mustWriteFile(t, filepath.Join(level2, ".git"), "gitdir: /fake")
				mustWriteFile(t, filepath.Join(level3, ".git"), "gitdir: /fake")

				return []string{level1, level2, level3}
			},
			maxDepth: 3,
		},
		{
			name: "respects max depth",
			setup: func(t *testing.T, tmpDir string) []string {
				level1 := filepath.Join(tmpDir, "l1")
				level2 := filepath.Join(level1, "l2")
				level3 := filepath.Join(level2, "l3")

				mustMkdirAll(t, level3)
				mustWriteFile(t, filepath.Join(level1, ".git"), "gitdir: /fake")
				mustWriteFile(t, filepath.Join(level2, ".git"), "gitdir: /fake")
				mustWriteFile(t, filepath.Join(level3, ".git"), "gitdir: /fake")

				// maxDepth 2 should only find level1 and level2
				return []string{level1, level2}
			},
			maxDepth: 2,
		},
		{
			name: "multiple repos at same level",
			setup: func(t *testing.T, tmpDir string) []string {
				repoA := filepath.Join(tmpDir, "repo-a")
				repoB := filepath.Join(tmpDir, "repo-b")

				mustMkdirAll(t, repoA)
				mustMkdirAll(t, repoB)
				mustWriteFile(t, filepath.Join(repoA, ".git"), "gitdir: /fake")
				mustWriteFile(t, filepath.Join(repoB, ".git"), "gitdir: /fake")

				return []string{repoA, repoB}
			},
			maxDepth: 1,
		},
		{
			name: "ignores files without .git",
			setup: func(t *testing.T, tmpDir string) []string {
				withGit := filepath.Join(tmpDir, "has-git")
				withoutGit := filepath.Join(tmpDir, "no-git")

				mustMkdirAll(t, withGit)
				mustMkdirAll(t, withoutGit)
				mustWriteFile(t, filepath.Join(withGit, ".git"), "gitdir: /fake")
				mustWriteFile(t, filepath.Join(withoutGit, "README.md"), "readme")

				return []string{withGit}
			},
			maxDepth: 1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			expected := tt.setup(t, tmpDir)

			fx := effects.NewRealEffects()
			result := scanForGitDirsWithEffects(fx, tmpDir, tt.maxDepth)

			// Use ElementsMatch since directory traversal order isn't guaranteed
			assert.ElementsMatch(t, expected, result)
		})
	}
}

func TestScanForGitDirs_EmptyDirectory(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	fx := effects.NewRealEffects()
	result := scanForGitDirsWithEffects(fx, tmpDir, 3)
	assert.Empty(t, result, "empty directory should return no results")
}

func TestScanForGitDirs_MaxDepthZero(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, ".git"), "gitdir: /fake")

	fx := effects.NewRealEffects()
	result := scanForGitDirsWithEffects(fx, tmpDir, 0)
	assert.Empty(t, result, "maxDepth 0 should not traverse into any directories")
}
