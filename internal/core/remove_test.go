package core

import (
	"testing"

	"github.com/m44rten1/sprout/internal/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanRemoveCommand(t *testing.T) {
	tests := []struct {
		name         string
		ctx          RemoveContext
		wantActions  int
		wantExit     bool
		wantExitCode int
		assertions   func(t *testing.T, plan Plan)
	}{
		{
			name: "remove worktree successfully",
			ctx: RemoveContext{
				RepoRoot:   "/test/repo",
				SproutRoot: "/test/repo/.sprout",
				TargetPath: "/test/repo/.sprout/feature",
				Force:      false,
			},
			wantActions: 3, // git remove + success message + prune
			wantExit:    false,
			assertions: func(t *testing.T, plan Plan) {
				// Action 1: git worktree remove
				gitCmd, ok := plan.Actions[0].(RunGitCommand)
				require.True(t, ok, "expected RunGitCommand at index 0")
				assert.Equal(t, "/test/repo", gitCmd.Dir)
				assert.Equal(t, []string{"worktree", "remove", "/test/repo/.sprout/feature"}, gitCmd.Args)

				// Action 2: success message
				msg, ok := plan.Actions[1].(PrintMessage)
				require.True(t, ok, "expected PrintMessage at index 1")
				assert.Contains(t, msg.Msg, "Removed worktree")
				assert.Contains(t, msg.Msg, "/test/repo/.sprout/feature")

				// Action 3: prune
				prune, ok := plan.Actions[2].(RunGitCommand)
				require.True(t, ok, "expected RunGitCommand at index 2")
				assert.Equal(t, []string{"worktree", "prune"}, prune.Args)
			},
		},
		{
			name: "remove worktree with force flag",
			ctx: RemoveContext{
				RepoRoot:   "/test/repo",
				SproutRoot: "/test/repo/.sprout",
				TargetPath: "/test/repo/.sprout/feature",
				Force:      true,
			},
			wantActions: 3,
			wantExit:    false,
			assertions: func(t *testing.T, plan Plan) {
				// Check --force flag is present in first action
				gitCmd, ok := plan.Actions[0].(RunGitCommand)
				require.True(t, ok, "expected RunGitCommand at index 0")
				assert.Equal(t, []string{"worktree", "remove", "--force", "/test/repo/.sprout/feature"}, gitCmd.Args)
			},
		},
		{
			name: "empty repo root returns error",
			ctx: RemoveContext{
				RepoRoot:   "",
				SproutRoot: "/test/repo/.sprout",
				TargetPath: "/test/repo/.sprout/feature",
			},
			wantActions:  2, // error + exit
			wantExit:     true,
			wantExitCode: 1,
			assertions: func(t *testing.T, plan Plan) {
				errMsg, ok := plan.Actions[0].(PrintError)
				require.True(t, ok, "expected PrintError at index 0")
				assert.Contains(t, errMsg.Msg, "repository root")
			},
		},
		{
			name: "empty target path returns error",
			ctx: RemoveContext{
				RepoRoot:   "/test/repo",
				SproutRoot: "/test/repo/.sprout",
				TargetPath: "",
			},
			wantActions:  2,
			wantExit:     true,
			wantExitCode: 1,
			assertions: func(t *testing.T, plan Plan) {
				errMsg, ok := plan.Actions[0].(PrintError)
				require.True(t, ok, "expected PrintError at index 0")
				assert.Contains(t, errMsg.Msg, "target path")
			},
		},
		{
			name: "non-sprout worktree returns error",
			ctx: RemoveContext{
				RepoRoot:   "/test/repo",
				SproutRoot: "/test/repo/.sprout",
				TargetPath: "/some/other/path", // Not under sprout root
			},
			wantActions:  2,
			wantExit:     true,
			wantExitCode: 1,
			assertions: func(t *testing.T, plan Plan) {
				errMsg, ok := plan.Actions[0].(PrintError)
				require.True(t, ok, "expected PrintError at index 0")
				assert.Contains(t, errMsg.Msg, "Refusing to remove non-sprout worktree")
				assert.Contains(t, errMsg.Msg, "/some/other/path")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := PlanRemoveCommand(tt.ctx)

			assert.Len(t, plan.Actions, tt.wantActions)

			// Check exit action behavior
			lastAction := plan.Actions[len(plan.Actions)-1]
			if tt.wantExit {
				exit, ok := lastAction.(Exit)
				require.True(t, ok, "expected Exit action for error plan")
				assert.Equal(t, tt.wantExitCode, exit.Code)
			} else {
				_, ok := lastAction.(Exit)
				assert.False(t, ok, "did not expect Exit action for success plan")
			}

			// Run custom assertions
			if tt.assertions != nil {
				tt.assertions(t, plan)
			}
		})
	}
}

func TestBuildRemoveWorktreeArgs(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		force    bool
		wantArgs []string
	}{
		{
			name:     "without force",
			path:     "/test/repo/.sprout/feature",
			force:    false,
			wantArgs: []string{"worktree", "remove", "/test/repo/.sprout/feature"},
		},
		{
			name:     "with force",
			path:     "/test/repo/.sprout/feature",
			force:    true,
			wantArgs: []string{"worktree", "remove", "--force", "/test/repo/.sprout/feature"},
		},
		{
			name:     "empty path without force",
			path:     "",
			force:    false,
			wantArgs: []string{"worktree", "remove", ""},
		},
		{
			name:     "empty path with force",
			path:     "",
			force:    true,
			wantArgs: []string{"worktree", "remove", "--force", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildRemoveWorktreeArgs(tt.path, tt.force)
			assert.Equal(t, tt.wantArgs, got)
		})
	}
}

// TestRemoveContext_Documentation verifies the documented fields are present
func TestRemoveContext_Documentation(t *testing.T) {
	// This test exists to document the RemoveContext structure and ensure
	// all fields are properly accessible
	ctx := RemoveContext{
		ArgProvided: true,
		Arg:         "feature",
		RepoRoot:    "/test/repo",
		SproutRoot:  "/test/repo/.sprout",
		Worktrees:   []git.Worktree{{Branch: "main"}, {Branch: "feature"}},
		TargetPath:  "/test/repo/.sprout/feature",
		Force:       true,
	}

	// Verify all fields are accessible
	assert.True(t, ctx.ArgProvided)
	assert.Equal(t, "feature", ctx.Arg)
	assert.Equal(t, "/test/repo", ctx.RepoRoot)
	assert.Equal(t, "/test/repo/.sprout", ctx.SproutRoot)
	assert.Len(t, ctx.Worktrees, 2)
	assert.Equal(t, "/test/repo/.sprout/feature", ctx.TargetPath)
	assert.True(t, ctx.Force)
}
