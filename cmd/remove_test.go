package cmd

import (
	"fmt"
	"testing"

	"github.com/m44rten1/sprout/internal/core"
	"github.com/m44rten1/sprout/internal/effects"
	"github.com/m44rten1/sprout/internal/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRemoveContext(t *testing.T) {
	tests := []struct {
		name       string
		setupFx    func(*effects.TestEffects)
		args       []string
		force      bool
		wantCtx    *core.RemoveContext
		wantErr    bool
		assertions func(t *testing.T, fx *effects.TestEffects)
	}{
		{
			name: "explicit path argument",
			setupFx: func(fx *effects.TestEffects) {
				fx.RepoRoot = "/test/repo"
				fx.WorktreeRoot = "/test/repo/.sprout"
				fx.Worktrees = []git.Worktree{
					{Path: "/test/repo", Branch: "main"},
					{Path: "/test/repo/.sprout/feature", Branch: "feature"},
				}
				fx.Files["/test/repo/.sprout/feature"] = true
			},
			args:  []string{"/test/repo/.sprout/feature"},
			force: false,
			wantCtx: &core.RemoveContext{
				ArgProvided: true,
				Arg:         "/test/repo/.sprout/feature",
				RepoRoot:    "/test/repo",
				SproutRoot:  "/test/repo/.sprout",
				TargetPath:  "/test/repo/.sprout/feature",
				Force:       false,
			},
			wantErr: false,
		},
		{
			name: "branch name argument",
			setupFx: func(fx *effects.TestEffects) {
				fx.RepoRoot = "/test/repo"
				fx.WorktreeRoot = "/test/repo/.sprout"
				fx.Worktrees = []git.Worktree{
					{Path: "/test/repo", Branch: "main"},
					{Path: "/test/repo/.sprout/feature", Branch: "feature"},
				}
			},
			args:  []string{"feature"},
			force: true,
			wantCtx: &core.RemoveContext{
				ArgProvided: true,
				Arg:         "feature",
				RepoRoot:    "/test/repo",
				SproutRoot:  "/test/repo/.sprout",
				TargetPath:  "/test/repo/.sprout/feature",
				Force:       true,
			},
			wantErr: false,
		},
		{
			name: "interactive selection",
			setupFx: func(fx *effects.TestEffects) {
				fx.RepoRoot = "/test/repo"
				fx.WorktreeRoot = "/test/repo/.sprout"
				fx.Worktrees = []git.Worktree{
					{Path: "/test/repo", Branch: "main"},
					{Path: "/test/repo/.sprout/feature", Branch: "feature"},
					{Path: "/test/repo/.sprout/bugfix", Branch: "bugfix"},
				}
				fx.SelectedWorktreeIndex = 1 // Select "bugfix"
			},
			args:  []string{},
			force: false,
			wantCtx: &core.RemoveContext{
				ArgProvided: false,
				Arg:         "",
				RepoRoot:    "/test/repo",
				SproutRoot:  "/test/repo/.sprout",
				TargetPath:  "/test/repo/.sprout/bugfix",
				Force:       false,
			},
			wantErr: false,
			assertions: func(t *testing.T, fx *effects.TestEffects) {
				assert.Equal(t, 1, fx.SelectWorktreeCalls)
			},
		},
		{
			name: "interactive selection cancelled",
			setupFx: func(fx *effects.TestEffects) {
				fx.RepoRoot = "/test/repo"
				fx.WorktreeRoot = "/test/repo/.sprout"
				fx.Worktrees = []git.Worktree{
					{Path: "/test/repo/.sprout/feature", Branch: "feature"},
				}
				fx.SelectionError = fmt.Errorf("cancelled")
			},
			args:    []string{},
			force:   false,
			wantCtx: nil,
			wantErr: true,
		},
		{
			name: "no sprout worktrees for interactive",
			setupFx: func(fx *effects.TestEffects) {
				fx.RepoRoot = "/test/repo"
				fx.WorktreeRoot = "/test/repo/.sprout"
				fx.Worktrees = []git.Worktree{
					{Path: "/test/repo", Branch: "main"},
				}
			},
			args:    []string{},
			force:   false,
			wantCtx: nil,
			wantErr: true,
		},
		{
			name: "branch not found",
			setupFx: func(fx *effects.TestEffects) {
				fx.RepoRoot = "/test/repo"
				fx.WorktreeRoot = "/test/repo/.sprout"
				fx.Worktrees = []git.Worktree{
					{Path: "/test/repo/.sprout/feature", Branch: "feature"},
				}
			},
			args:    []string{"nonexistent"},
			force:   false,
			wantCtx: nil,
			wantErr: true,
			assertions: func(t *testing.T, fx *effects.TestEffects) {
				// Should not call SelectWorktree since arg was provided
				assert.Equal(t, 0, fx.SelectWorktreeCalls)
			},
		},
		{
			name: "GetRepoRoot fails",
			setupFx: func(fx *effects.TestEffects) {
				fx.GetRepoRootErr = fmt.Errorf("not a git repo")
			},
			args:    []string{},
			force:   false,
			wantCtx: nil,
			wantErr: true,
		},
		{
			name: "GetWorktreeRoot fails",
			setupFx: func(fx *effects.TestEffects) {
				fx.RepoRoot = "/test/repo"
				fx.GetWorktreeRootErr = fmt.Errorf("sprout root error")
			},
			args:    []string{},
			force:   false,
			wantCtx: nil,
			wantErr: true,
		},
		{
			name: "ListWorktrees fails",
			setupFx: func(fx *effects.TestEffects) {
				fx.RepoRoot = "/test/repo"
				fx.WorktreeRoot = "/test/repo/.sprout"
				fx.ListWorktreesErr = fmt.Errorf("git error")
			},
			args:    []string{},
			force:   false,
			wantCtx: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fx := effects.NewTestEffects()
			tt.setupFx(fx)

			got, err := BuildRemoveContext(fx, tt.args, tt.force)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, tt.wantCtx)

				assert.Equal(t, tt.wantCtx.ArgProvided, got.ArgProvided)
				assert.Equal(t, tt.wantCtx.Arg, got.Arg)
				assert.Equal(t, tt.wantCtx.RepoRoot, got.RepoRoot)
				assert.Equal(t, tt.wantCtx.SproutRoot, got.SproutRoot)
				assert.Equal(t, tt.wantCtx.TargetPath, got.TargetPath)
				assert.Equal(t, tt.wantCtx.Force, got.Force)
				// Note: Worktrees field not checked here - it's passed through
				// from fx.Worktrees but the specific content doesn't affect behavior
			}

			// Run custom assertions for both success and error paths
			if tt.assertions != nil {
				tt.assertions(t, fx)
			}
		})
	}
}

func TestRemoveCommand_EndToEnd(t *testing.T) {
	tests := []struct {
		name       string
		setupFx    func(*effects.TestEffects)
		args       []string
		force      bool
		assertions func(t *testing.T, fx *effects.TestEffects)
	}{
		{
			name: "remove worktree by branch name",
			setupFx: func(fx *effects.TestEffects) {
				fx.RepoRoot = "/test/repo"
				fx.WorktreeRoot = "/test/repo/.sprout"
				fx.Worktrees = []git.Worktree{
					{Path: "/test/repo", Branch: "main"},
					{Path: "/test/repo/.sprout/feature", Branch: "feature"},
				}
			},
			args:  []string{"feature"},
			force: false,
			assertions: func(t *testing.T, fx *effects.TestEffects) {
				// Should have run git commands: remove + prune
				require.Len(t, fx.GitCommands, 2)

				// Check remove command
				assert.Equal(t, "/test/repo", fx.GitCommands[0].Dir)
				assert.Equal(t, []string{"worktree", "remove", "/test/repo/.sprout/feature"}, fx.GitCommands[0].Args)

				// Check prune command
				assert.Equal(t, "/test/repo", fx.GitCommands[1].Dir)
				assert.Equal(t, []string{"worktree", "prune"}, fx.GitCommands[1].Args)

				// Should have printed success message
				require.Len(t, fx.PrintedMsgs, 1)
				assert.Contains(t, fx.PrintedMsgs[0], "Removed worktree")
				assert.Contains(t, fx.PrintedMsgs[0], "/test/repo/.sprout/feature")
			},
		},
		{
			name: "remove worktree by path",
			setupFx: func(fx *effects.TestEffects) {
				fx.RepoRoot = "/test/repo"
				fx.WorktreeRoot = "/test/repo/.sprout"
				fx.Worktrees = []git.Worktree{
					{Path: "/test/repo", Branch: "main"},
					{Path: "/test/repo/.sprout/feature", Branch: "feature"},
				}
				fx.Files["/test/repo/.sprout/feature"] = true
			},
			args:  []string{"/test/repo/.sprout/feature"},
			force: false,
			assertions: func(t *testing.T, fx *effects.TestEffects) {
				require.Len(t, fx.GitCommands, 2)
				assert.Equal(t, []string{"worktree", "remove", "/test/repo/.sprout/feature"}, fx.GitCommands[0].Args)
			},
		},
		{
			name: "remove worktree with force",
			setupFx: func(fx *effects.TestEffects) {
				fx.RepoRoot = "/test/repo"
				fx.WorktreeRoot = "/test/repo/.sprout"
				fx.Worktrees = []git.Worktree{
					{Path: "/test/repo/.sprout/feature", Branch: "feature"},
				}
			},
			args:  []string{"feature"},
			force: true,
			assertions: func(t *testing.T, fx *effects.TestEffects) {
				require.Len(t, fx.GitCommands, 2)
				// Should have --force flag
				assert.Equal(t, []string{"worktree", "remove", "--force", "/test/repo/.sprout/feature"}, fx.GitCommands[0].Args)
			},
		},
		{
			name: "remove non-sprout worktree should fail",
			setupFx: func(fx *effects.TestEffects) {
				fx.RepoRoot = "/test/repo"
				fx.WorktreeRoot = "/test/repo/.sprout"
				fx.Worktrees = []git.Worktree{
					{Path: "/test/repo", Branch: "main"},
				}
				fx.Files["/some/other/path"] = true
			},
			args:  []string{"/some/other/path"},
			force: false,
			assertions: func(t *testing.T, fx *effects.TestEffects) {
				// Should not have executed any git commands
				assert.Len(t, fx.GitCommands, 0)

				// Should have printed error
				require.Len(t, fx.PrintedErrs, 1)
				assert.Contains(t, fx.PrintedErrs[0], "Refusing to remove non-sprout worktree")
			},
		},
		{
			name: "interactive selection",
			setupFx: func(fx *effects.TestEffects) {
				fx.RepoRoot = "/test/repo"
				fx.WorktreeRoot = "/test/repo/.sprout"
				fx.Worktrees = []git.Worktree{
					{Path: "/test/repo", Branch: "main"},
					{Path: "/test/repo/.sprout/feature", Branch: "feature"},
				}
				fx.SelectedWorktreeIndex = 0 // Select feature worktree (filtered, so index 0)
			},
			args:  []string{},
			force: false,
			assertions: func(t *testing.T, fx *effects.TestEffects) {
				// Should have called SelectWorktree
				assert.Equal(t, 1, fx.SelectWorktreeCalls)

				// Should have removed the selected worktree
				require.Len(t, fx.GitCommands, 2)
				assert.Equal(t, []string{"worktree", "remove", "/test/repo/.sprout/feature"}, fx.GitCommands[0].Args)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fx := effects.NewTestEffects()
			tt.setupFx(fx)

			// Build context
			ctx, err := BuildRemoveContext(fx, tt.args, tt.force)
			require.NoError(t, err, "context building should succeed in all current test cases")

			// Plan
			plan := core.PlanRemoveCommand(ctx)

			// Execute
			err = effects.ExecutePlan(plan, fx)
			// Note: We don't assert on err here because error plans return ExitError,
			// which is expected behavior. Tests verify outcomes via recorded effects.

			// Verify results
			tt.assertions(t, fx)
		})
	}
}
