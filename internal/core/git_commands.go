package core

// WorktreeAddArgs constructs git arguments for creating a worktree.
// It follows this priority: local branch > remote branch > new from fromRef.
// When creating from a remote branch, upstream tracking is enabled by default.
// For truly new branches, --no-track is used to avoid configuring an upstream.
// The caller must resolve fromRef before calling (e.g. "origin/main", "HEAD", or a custom ref).
func WorktreeAddArgs(path, branch string, localExists bool, remoteBranchExists bool, fromRef string) []string {
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

	// Case 3: Create truly new branch from the resolved base ref
	// --no-track must follow -b (it's a branch creation option, not worktree option)
	return append(args, "-b", branch, "--no-track", fromRef)
}
