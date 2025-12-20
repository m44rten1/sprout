package core

// WorktreeAddArgs constructs git arguments for creating a worktree.
// It follows this priority: local branch > remote branch > new from origin/main > new from HEAD.
// When creating from a remote branch, upstream tracking is enabled by default.
// For truly new branches, --no-track is used to avoid configuring an upstream.
func WorktreeAddArgs(path, branch string, localExists bool, remoteBranchExists bool, hasOriginMain bool) []string {
	args := []string{"worktree", "add", path}

	// Case 1: Local branch exists - simple checkout
	if localExists {
		return append(args, branch)
	}

	// Case 2: Remote branch exists - create local tracking remote
	// We DO want upstream tracking here (no --no-track)
	if remoteBranchExists {
		return append(args, "-b", branch, "origin/"+branch)
	}

	// Case 3 & 4: Create truly new branch - avoid upstream config
	// --no-track must follow -b (it's a branch creation option, not worktree option)
	args = append(args, "-b", branch, "--no-track")

	if hasOriginMain {
		return append(args, "origin/main")
	}

	return append(args, "HEAD")
}
