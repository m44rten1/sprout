package core

import (
	"testing"

	"github.com/m44rten1/sprout/internal/git"
	"github.com/stretchr/testify/assert"
)

// MakeBranch creates a Branch for testing
func MakeBranch(name string) git.Branch {
	return git.Branch{DisplayName: name}
}

// MakeWorktree creates a Worktree for testing
func MakeWorktree(path, branch string) git.Worktree {
	return git.Worktree{Path: path, Branch: branch}
}

// TestBasicSetup verifies test infrastructure is working
func TestBasicSetup(t *testing.T) {
	assert.Equal(t, 1, 1, "basic equality check")
	assert.True(t, true, "basic true check")
	assert.False(t, false, "basic false check")
}
