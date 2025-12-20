package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorktreeAddArgs(t *testing.T) {
	tests := []struct {
		name               string
		path               string
		branch             string
		localExists        bool
		remoteBranchExists bool
		hasOriginMain      bool
		want               []string
	}{
		{
			name:               "local branch exists - simple checkout",
			path:               "/path/to/worktree",
			branch:             "feature-123",
			localExists:        true,
			remoteBranchExists: false,
			hasOriginMain:      false,
			want:               []string{"worktree", "add", "/path/to/worktree", "feature-123"},
		},
		{
			name:               "remote branch exists - create local tracking remote (no --no-track)",
			path:               "/path/to/worktree",
			branch:             "feature-456",
			localExists:        false,
			remoteBranchExists: true,
			hasOriginMain:      false,
			want:               []string{"worktree", "add", "/path/to/worktree", "-b", "feature-456", "origin/feature-456"},
		},
		{
			name:               "new branch with origin/main - branch from origin/main with --no-track",
			path:               "/path/to/worktree",
			branch:             "new-feature",
			localExists:        false,
			remoteBranchExists: false,
			hasOriginMain:      true,
			want:               []string{"worktree", "add", "/path/to/worktree", "-b", "new-feature", "--no-track", "origin/main"},
		},
		{
			name:               "new branch without origin/main - branch from HEAD with --no-track",
			path:               "/path/to/worktree",
			branch:             "experimental",
			localExists:        false,
			remoteBranchExists: false,
			hasOriginMain:      false,
			want:               []string{"worktree", "add", "/path/to/worktree", "-b", "experimental", "--no-track", "HEAD"},
		},
		{
			name:               "local exists with remote also existing - prefer local",
			path:               "/path/to/worktree",
			branch:             "main",
			localExists:        true,
			remoteBranchExists: true,
			hasOriginMain:      true,
			want:               []string{"worktree", "add", "/path/to/worktree", "main"},
		},
		{
			name:               "remote exists with origin/main - prefer remote branch (with tracking)",
			path:               "/path/to/worktree",
			branch:             "develop",
			localExists:        false,
			remoteBranchExists: true,
			hasOriginMain:      true,
			want:               []string{"worktree", "add", "/path/to/worktree", "-b", "develop", "origin/develop"},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := WorktreeAddArgs(tt.path, tt.branch, tt.localExists, tt.remoteBranchExists, tt.hasOriginMain)
			assert.Equal(t, tt.want, got, "git command mismatch")
		})
	}
}

// indexOf returns the index of a value in a slice, or -1 if not found
func indexOf(xs []string, v string) int {
	for i, x := range xs {
		if x == v {
			return i
		}
	}
	return -1
}

func TestWorktreeAddArgs_ArgumentOrder(t *testing.T) {
	t.Run("existing local branch has no --no-track", func(t *testing.T) {
		result := WorktreeAddArgs("/path", "existing", true, false, false)
		assert.NotContains(t, result, "--no-track", "existing local branches should not have --no-track")
	})

	t.Run("remote branch has no --no-track (wants tracking)", func(t *testing.T) {
		result := WorktreeAddArgs("/path", "remote-branch", false, true, false)
		assert.NotContains(t, result, "--no-track", "remote branches should enable upstream tracking")
	})

	t.Run("new branch places --no-track AFTER -b (Git semantics)", func(t *testing.T) {
		result := WorktreeAddArgs("/path", "new", false, false, true)

		idxB := indexOf(result, "-b")
		idxNoTrack := indexOf(result, "--no-track")

		assert.NotEqual(t, -1, idxB, "should contain -b flag")
		assert.NotEqual(t, -1, idxNoTrack, "should contain --no-track flag")
		assert.Greater(t, idxNoTrack, idxB, "--no-track must come after -b for correct Git parsing")
	})

	t.Run("new branch from HEAD places --no-track AFTER -b", func(t *testing.T) {
		result := WorktreeAddArgs("/path", "new", false, false, false)

		idxB := indexOf(result, "-b")
		idxNoTrack := indexOf(result, "--no-track")

		assert.NotEqual(t, -1, idxB, "should contain -b flag")
		assert.NotEqual(t, -1, idxNoTrack, "should contain --no-track flag")
		assert.Greater(t, idxNoTrack, idxB, "--no-track must come after -b for correct Git parsing")
	})
}
