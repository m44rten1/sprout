package core

import (
	"testing"

	"github.com/m44rten1/sprout/internal/git"
	"github.com/stretchr/testify/assert"
)

func TestFilterSproutWorktrees(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		worktrees  []git.Worktree
		sproutRoot string
		want       []git.Worktree
	}{
		{
			name: "filters worktrees under sprout root",
			worktrees: []git.Worktree{
				MakeWorktree("/home/user/repos/myrepo", "main"),
				MakeWorktree("/home/user/.sprout/myrepo/feature-1", "feature-1"),
				MakeWorktree("/home/user/.sprout/myrepo/feature-2", "feature-2"),
				MakeWorktree("/tmp/other-worktree", "test"),
			},
			sproutRoot: "/home/user/.sprout",
			want: []git.Worktree{
				MakeWorktree("/home/user/.sprout/myrepo/feature-1", "feature-1"),
				MakeWorktree("/home/user/.sprout/myrepo/feature-2", "feature-2"),
			},
		},
		{
			name: "returns empty slice when no worktrees match",
			worktrees: []git.Worktree{
				MakeWorktree("/home/user/repos/myrepo", "main"),
				MakeWorktree("/tmp/other", "test"),
			},
			sproutRoot: "/home/user/.sprout",
			want:       []git.Worktree{},
		},
		{
			name:       "returns empty slice for empty worktrees",
			worktrees:  []git.Worktree{},
			sproutRoot: "/home/user/.sprout",
			want:       []git.Worktree{},
		},
		{
			name: "returns empty slice for empty sprout roots",
			worktrees: []git.Worktree{
				MakeWorktree("/home/user/.sprout/myrepo/feature", "feature"),
			},
			sproutRoot: "",
			want:       []git.Worktree{},
		},
		{
			name: "handles detached HEAD worktrees",
			worktrees: []git.Worktree{
				MakeWorktree("/home/user/.sprout/myrepo/detached", ""),
				MakeWorktree("/home/user/.sprout/myrepo/feature", "feature"),
			},
			sproutRoot: "/home/user/.sprout",
			want: []git.Worktree{
				MakeWorktree("/home/user/.sprout/myrepo/detached", ""),
				MakeWorktree("/home/user/.sprout/myrepo/feature", "feature"),
			},
		},
		{
			name: "skips worktrees with empty paths",
			worktrees: []git.Worktree{
				{Path: "", Branch: "invalid"},
				MakeWorktree("/home/user/.sprout/myrepo/valid", "valid"),
			},
			sproutRoot: "/home/user/.sprout",
			want: []git.Worktree{
				MakeWorktree("/home/user/.sprout/myrepo/valid", "valid"),
			},
		},
		{
			name: "excludes worktree at root level (not under root)",
			worktrees: []git.Worktree{
				MakeWorktree("/home/user/.sprout", "root-level"),
				MakeWorktree("/home/user/.sprout/myrepo/valid", "valid"),
			},
			sproutRoot: "/home/user/.sprout",
			want: []git.Worktree{
				MakeWorktree("/home/user/.sprout/myrepo/valid", "valid"),
			},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable for parallel test
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FilterSproutWorktrees(tt.worktrees, tt.sproutRoot)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFindWorktreeByBranch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		worktrees  []git.Worktree
		sproutRoot string
		branch     string
		wantPath   string
		wantFound  bool
	}{
		{
			name: "finds worktree by branch name",
			worktrees: []git.Worktree{
				MakeWorktree("/home/user/.sprout/myrepo/feature-1", "feature-1"),
				MakeWorktree("/home/user/.sprout/myrepo/feature-2", "feature-2"),
			},
			sproutRoot: "/home/user/.sprout",
			branch:     "feature-2",
			wantPath:   "/home/user/.sprout/myrepo/feature-2",
			wantFound:  true,
		},
		{
			name: "returns false when branch not found",
			worktrees: []git.Worktree{
				MakeWorktree("/home/user/.sprout/myrepo/feature-1", "feature-1"),
			},
			sproutRoot: "/home/user/.sprout",
			branch:     "nonexistent",
			wantPath:   "",
			wantFound:  false,
		},
		{
			name: "returns false when branch exists but not under sprout root",
			worktrees: []git.Worktree{
				MakeWorktree("/home/user/repos/myrepo", "feature"),
			},
			sproutRoot: "/home/user/.sprout",
			branch:     "feature",
			wantPath:   "",
			wantFound:  false,
		},
		{
			name: "returns false for empty branch name",
			worktrees: []git.Worktree{
				MakeWorktree("/home/user/.sprout/myrepo/detached", ""),
			},
			sproutRoot: "/home/user/.sprout",
			branch:     "",
			wantPath:   "",
			wantFound:  false,
		},
		{
			name: "returns false for detached HEAD worktrees",
			worktrees: []git.Worktree{
				MakeWorktree("/home/user/.sprout/myrepo/detached", ""),
			},
			sproutRoot: "/home/user/.sprout",
			branch:     "main",
			wantPath:   "",
			wantFound:  false,
		},
		{
			name:       "returns false for empty sprout roots",
			worktrees:  []git.Worktree{MakeWorktree("/home/user/.sprout/myrepo/feature", "feature")},
			sproutRoot: "",
			branch:     "feature",
			wantPath:   "",
			wantFound:  false,
		},
		{
			name:       "returns false for empty worktrees list",
			worktrees:  []git.Worktree{},
			sproutRoot: "/home/user/.sprout",
			branch:     "feature",
			wantPath:   "",
			wantFound:  false,
		},
		{
			name: "handles branch names with slashes",
			worktrees: []git.Worktree{
				MakeWorktree("/home/user/.sprout/myrepo/feat-abc", "feat/abc"),
			},
			sproutRoot: "/home/user/.sprout",
			branch:     "feat/abc",
			wantPath:   "/home/user/.sprout/myrepo/feat-abc",
			wantFound:  true,
		},
		{
			name: "skips worktrees with empty paths",
			worktrees: []git.Worktree{
				{Path: "", Branch: "feature"},
				MakeWorktree("/home/user/.sprout/myrepo/feature", "feature"),
			},
			sproutRoot: "/home/user/.sprout",
			branch:     "feature",
			wantPath:   "/home/user/.sprout/myrepo/feature",
			wantFound:  true,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable for parallel test
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotPath, gotFound := FindWorktreeByBranch(tt.worktrees, tt.sproutRoot, tt.branch)
			assert.Equal(t, tt.wantPath, gotPath, "path mismatch")
			assert.Equal(t, tt.wantFound, gotFound, "found flag mismatch")
		})
	}
}

func TestIsUnderSproutRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		path       string
		sproutRoot string
		want       bool
	}{
		{
			name:       "path is under sprout root",
			path:       "/home/user/.sprout/myrepo/feature",
			sproutRoot: "/home/user/.sprout",
			want:       true,
		},
		{
			name:       "path equals sprout root",
			path:       "/home/user/.sprout",
			sproutRoot: "/home/user/.sprout",
			want:       false,
		},
		{
			name:       "path is parent of sprout root",
			path:       "/home/user",
			sproutRoot: "/home/user/.sprout",
			want:       false,
		},
		{
			name:       "path is sibling of sprout root",
			path:       "/home/user/.config",
			sproutRoot: "/home/user/.sprout",
			want:       false,
		},
		{
			name:       "empty path returns false",
			path:       "",
			sproutRoot: "/home/user/.sprout",
			want:       false,
		},
		{
			name:       "empty sprout root returns false",
			path:       "/home/user/.sprout/myrepo",
			sproutRoot: "",
			want:       false,
		},
		{
			name:       "deeply nested path",
			path:       "/home/user/.sprout/org/repo/branch/subdir",
			sproutRoot: "/home/user/.sprout",
			want:       true,
		},
		{
			name:       "path with .. segments (normalized correctly)",
			path:       "/home/user/.sprout/myrepo/../myrepo/feature",
			sproutRoot: "/home/user/.sprout",
			want:       true,
		},
		{
			name:       "root with .. segments (normalized correctly)",
			path:       "/home/user/.sprout/myrepo/feature",
			sproutRoot: "/home/user/other/../.sprout",
			want:       true,
		},
		{
			name:       "trailing slash in root (normalized)",
			path:       "/home/user/.sprout/myrepo",
			sproutRoot: "/home/user/.sprout/",
			want:       true,
		},
		{
			name:       "trailing slash in path (normalized)",
			path:       "/home/user/.sprout/myrepo/",
			sproutRoot: "/home/user/.sprout",
			want:       true,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable for parallel test
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsUnderSproutRoot(tt.path, tt.sproutRoot)
			assert.Equal(t, tt.want, got)
		})
	}
}
