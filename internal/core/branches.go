package core

import "github.com/m44rten1/sprout/internal/git"

// GetWorktreeAvailableBranches returns branches that can be used to create new worktrees.
// Excludes branches currently checked out in any worktree.
// Detached HEAD worktrees don't block any branch.
// Assumes Branch.Name and Worktree.Branch are local branch names (e.g. "main").
func GetWorktreeAvailableBranches(allBranches []git.Branch, worktrees []git.Worktree) []git.Branch {
	checkedOut := make(map[string]struct{}, len(worktrees))
	for _, wt := range worktrees {
		if wt.Branch != "" {
			checkedOut[wt.Branch] = struct{}{}
		}
	}

	available := make([]git.Branch, 0, len(allBranches))
	for _, branch := range allBranches {
		if branch.Name == "" {
			continue
		}
		if _, ok := checkedOut[branch.Name]; !ok {
			available = append(available, branch)
		}
	}
	return available
}
