package core

import (
	"testing"

	"github.com/m44rten1/sprout/internal/git"
	"github.com/stretchr/testify/assert"
)

func TestGetWorktreeAvailableBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		allBranches []git.Branch
		worktrees   []git.Worktree
		want        []git.Branch
	}{
		{
			name:        "empty branches list",
			allBranches: []git.Branch{},
			worktrees:   []git.Worktree{},
			want:        []git.Branch{},
		},
		{
			name: "all branches available",
			allBranches: []git.Branch{
				MakeBranch("main"),
				MakeBranch("feature"),
				MakeBranch("bugfix"),
			},
			worktrees: []git.Worktree{},
			want: []git.Branch{
				MakeBranch("main"),
				MakeBranch("feature"),
				MakeBranch("bugfix"),
			},
		},
		{
			name: "all branches checked out",
			allBranches: []git.Branch{
				MakeBranch("main"),
				MakeBranch("feature"),
			},
			worktrees: []git.Worktree{
				MakeWorktree("/repo", "main"),
				MakeWorktree("/repo/worktrees/feature", "feature"),
			},
			want: []git.Branch{},
		},
		{
			name: "some branches checked out",
			allBranches: []git.Branch{
				MakeBranch("main"),
				MakeBranch("feature"),
				MakeBranch("bugfix"),
			},
			worktrees: []git.Worktree{
				MakeWorktree("/repo", "main"),
			},
			want: []git.Branch{
				MakeBranch("feature"),
				MakeBranch("bugfix"),
			},
		},
		{
			name: "worktree with detached HEAD doesn't block branches",
			allBranches: []git.Branch{
				MakeBranch("main"),
				MakeBranch("feature"),
			},
			worktrees: []git.Worktree{
				MakeWorktree("/repo", ""),        // detached HEAD
				MakeWorktree("/repo/bisect", ""), // another detached HEAD
			},
			want: []git.Branch{
				MakeBranch("main"),
				MakeBranch("feature"),
			},
		},
		{
			name: "mixed checked out and available",
			allBranches: []git.Branch{
				MakeBranch("main"),
				MakeBranch("develop"),
				MakeBranch("feature-a"),
				MakeBranch("feature-b"),
			},
			worktrees: []git.Worktree{
				MakeWorktree("/repo", "main"),
				MakeWorktree("/repo/worktrees/feature-a", "feature-a"),
			},
			want: []git.Branch{
				MakeBranch("develop"),
				MakeBranch("feature-b"),
			},
		},
		{
			name: "defensive: skip branches with empty Name field",
			allBranches: []git.Branch{
				MakeBranch("main"),
				{DisplayName: "broken", Name: ""}, // malformed entry
				MakeBranch("feature"),
			},
			worktrees: []git.Worktree{},
			want: []git.Branch{
				MakeBranch("main"),
				MakeBranch("feature"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := GetWorktreeAvailableBranches(tt.allBranches, tt.worktrees)
			assert.Equal(t, tt.want, got)
		})
	}
}
