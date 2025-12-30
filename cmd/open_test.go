package cmd

import (
	"errors"
	"fmt"
	"testing"

	"github.com/m44rten1/sprout/internal/config"
	"github.com/m44rten1/sprout/internal/core"
	"github.com/m44rten1/sprout/internal/effects"
	"github.com/m44rten1/sprout/internal/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// baseTestFxOpen creates a TestEffects with common defaults for open command tests.
// Sets XDG_DATA_HOME for predictable sprout root and cleans up after test.
func baseTestFxOpen(t *testing.T) *effects.TestEffects {
	t.Helper()

	fx := effects.NewTestEffects()
	fx.RepoRoot = "/test/repo"
	fx.MainWorktreePath = "/test/repo"
	fx.Config = &config.Config{Hooks: config.HooksConfig{}}
	// Set SproutRoot to match the expected test paths (/test/data/sprout)
	fx.SproutRoot = "/test/data/sprout"
	return fx
}

// TestBuildOpenContext tests the handler's logic for building OpenContext
// from effects (the "imperative shell" layer). This catches wiring bugs
// that pure planner tests miss.
func TestBuildOpenContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		noHooks       bool
		setupFx       func(*effects.TestEffects)
		wantCtx       *core.OpenContext // nil if error expected
		wantErr       bool
		assertEffects func(t *testing.T, fx *effects.TestEffects)
	}{
		{
			name:    "explicit path argument",
			args:    []string{"/test/repo/.sprout/feature"},
			noHooks: false,
			setupFx: func(fx *effects.TestEffects) {
				fx.Files["/test/repo/.sprout/feature"] = true
				fx.TrustedRepos["/test/repo"] = false
			},
			wantCtx: &core.OpenContext{
				TargetPath:       "/test/repo/.sprout/feature",
				RepoRoot:         "/test/repo",
				MainWorktreePath: "/test/repo",
				Config:           &config.Config{Hooks: config.HooksConfig{}},
				IsTrusted:        false,
				NoHooks:          false,
			},
			wantErr: false,
		},
		{
			name:    "explicit branch argument - finds worktree",
			args:    []string{"feature"},
			noHooks: false,
			setupFx: func(fx *effects.TestEffects) {
				fx.Worktrees = []git.Worktree{
					{Path: "/test/data/sprout/repo-abc123/feature/repo", Branch: "feature"},
					{Path: "/test/data/sprout/repo-abc123/main/repo", Branch: "main"},
				}
				// FileExists returns false for "feature" string (not a path)
				fx.Files["feature"] = false
				fx.TrustedRepos["/test/repo"] = false
			},
			wantCtx: &core.OpenContext{
				TargetPath:       "/test/data/sprout/repo-abc123/feature/repo",
				RepoRoot:         "/test/repo",
				MainWorktreePath: "/test/repo",
				Config:           &config.Config{Hooks: config.HooksConfig{}},
				IsTrusted:        false,
				NoHooks:          false,
			},
			wantErr: false,
		},
		{
			name:    "interactive mode - select worktree",
			args:    []string{}, // Empty = interactive
			noHooks: false,
			setupFx: func(fx *effects.TestEffects) {
				fx.Worktrees = []git.Worktree{
					{Path: "/test/data/sprout/repo-abc123/feature/repo", Branch: "feature"},
					{Path: "/test/data/sprout/repo-abc123/main/repo", Branch: "main"},
				}
				fx.SelectedWorktreeIndex = 0 // Select first worktree
				fx.TrustedRepos["/test/repo"] = false
			},
			wantCtx: &core.OpenContext{
				TargetPath:       "/test/data/sprout/repo-abc123/feature/repo",
				RepoRoot:         "/test/repo",
				MainWorktreePath: "/test/repo",
				Config:           &config.Config{Hooks: config.HooksConfig{}},
				IsTrusted:        false,
				NoHooks:          false,
			},
			wantErr: false,
		},
		{
			name:    "with hooks configured and trusted",
			args:    []string{"/test/repo/.sprout/feature"},
			noHooks: false,
			setupFx: func(fx *effects.TestEffects) {
				fx.Files["/test/repo/.sprout/feature"] = true
				fx.Config = &config.Config{
					Hooks: config.HooksConfig{
						OnOpen: []string{"echo 'opening'", "npm install"},
					},
				}
				fx.TrustedRepos["/test/repo"] = true
			},
			wantCtx: &core.OpenContext{
				TargetPath:       "/test/repo/.sprout/feature",
				RepoRoot:         "/test/repo",
				MainWorktreePath: "/test/repo",
				Config: &config.Config{
					Hooks: config.HooksConfig{
						OnOpen: []string{"echo 'opening'", "npm install"},
					},
				},
				IsTrusted: true,
				NoHooks:   false,
			},
			wantErr: false,
			assertEffects: func(t *testing.T, fx *effects.TestEffects) {
				// Should have checked trust status
				assert.Contains(t, fx.IsTrustedArgs, "/test/repo")
			},
		},
		{
			name:    "with --no-hooks flag",
			args:    []string{"/test/repo/.sprout/feature"},
			noHooks: true,
			setupFx: func(fx *effects.TestEffects) {
				fx.Files["/test/repo/.sprout/feature"] = true
				fx.Config = &config.Config{
					Hooks: config.HooksConfig{
						OnOpen: []string{"echo 'opening'"},
					},
				}
			},
			wantCtx: &core.OpenContext{
				TargetPath:       "/test/repo/.sprout/feature",
				RepoRoot:         "/test/repo",
				MainWorktreePath: "/test/repo",
				Config: &config.Config{
					Hooks: config.HooksConfig{
						OnOpen: []string{"echo 'opening'"},
					},
				},
				IsTrusted: false, // Not checked when noHooks=true
				NoHooks:   true,
			},
			wantErr: false,
			assertEffects: func(t *testing.T, fx *effects.TestEffects) {
				// Should NOT check trust status when hooks disabled
				assert.Empty(t, fx.IsTrustedArgs)
			},
		},
		{
			name:    "GetRepoRoot fails",
			args:    []string{"/test/repo/.sprout/feature"},
			noHooks: false,
			setupFx: func(fx *effects.TestEffects) {
				fx.RepoRoot = "" // Triggers error in GetRepoRoot
			},
			wantCtx: nil,
			wantErr: true,
		},
		{
			name:    "GetMainWorktreePath fails",
			args:    []string{"/test/repo/.sprout/feature"},
			noHooks: false,
			setupFx: func(fx *effects.TestEffects) {
				fx.MainWorktreePath = "" // Triggers error
			},
			wantCtx: nil,
			wantErr: true,
		},
		{
			name:    "interactive mode - no sprout worktrees",
			args:    []string{},
			noHooks: false,
			setupFx: func(fx *effects.TestEffects) {
				// Worktrees exist but none are under sprout root
				fx.Worktrees = []git.Worktree{
					{Path: "/other/location/feature", Branch: "feature"},
				}
			},
			wantCtx: nil,
			wantErr: true,
		},
		{
			name:    "interactive mode - selection cancelled",
			args:    []string{},
			noHooks: false,
			setupFx: func(fx *effects.TestEffects) {
				// Use correct sprout root path (/test/data/sprout) so worktree is found
				fx.Worktrees = []git.Worktree{
					{Path: "/test/data/sprout/repo-abc123/feature/repo", Branch: "feature"},
				}
				fx.SelectionError = errors.New("cancelled")
			},
			wantCtx: nil,
			wantErr: true,
		},
		{
			name:    "branch argument - no matching worktree",
			args:    []string{"nonexistent"},
			noHooks: false,
			setupFx: func(fx *effects.TestEffects) {
				fx.Worktrees = []git.Worktree{
					{Path: "/home/user/.local/share/sprout/repo-12345678/feature/repo", Branch: "feature"},
				}
				fx.Files["nonexistent"] = false
			},
			wantCtx: nil,
			wantErr: true,
		},
		{
			name:    "LoadConfig fails",
			args:    []string{"/test/repo/.sprout/feature"},
			noHooks: false,
			setupFx: func(fx *effects.TestEffects) {
				fx.Files["/test/repo/.sprout/feature"] = true
				fx.LoadConfigErr = errors.New("config parse error")
			},
			wantCtx: nil,
			wantErr: true,
		},
		{
			name:    "IsTrusted fails",
			args:    []string{"/test/repo/.sprout/feature"},
			noHooks: false,
			setupFx: func(fx *effects.TestEffects) {
				fx.Files["/test/repo/.sprout/feature"] = true
				fx.Config = &config.Config{
					Hooks: config.HooksConfig{
						OnOpen: []string{"echo 'opening'"},
					},
				}
				fx.IsTrustedErr = errors.New("trust check failed")
			},
			wantCtx: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fx := baseTestFxOpen(t)
			tt.setupFx(fx)

			ctx, err := BuildOpenContext(fx, tt.args, tt.noHooks)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, tt.wantCtx, "test setup error: wantCtx should not be nil when wantErr=false")

			// Verify context matches expectations
			assert.Equal(t, tt.wantCtx.TargetPath, ctx.TargetPath)
			assert.Equal(t, tt.wantCtx.RepoRoot, ctx.RepoRoot)
			assert.Equal(t, tt.wantCtx.MainWorktreePath, ctx.MainWorktreePath)
			assert.Equal(t, tt.wantCtx.IsTrusted, ctx.IsTrusted)
			assert.Equal(t, tt.wantCtx.NoHooks, ctx.NoHooks)

			// Verify Config is forwarded correctly from LoadConfig
			assert.Same(t, fx.Config, ctx.Config, "should forward loaded config unchanged")

			if tt.assertEffects != nil {
				tt.assertEffects(t, fx)
			}
		})
	}
}

// TestOpenCommand_EndToEnd tests the full open command flow:
// BuildOpenContext → PlanOpenCommand → ExecutePlan.
// These tests verify behavioral outcomes, not implementation details.
func TestOpenCommand_EndToEnd(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		args           []string
		noHooks        bool
		setupFx        func(*effects.TestEffects)
		assertBehavior func(t *testing.T, fx *effects.TestEffects)
		wantErr        bool
		wantExit       bool
		wantExitCode   int
	}{
		{
			name:    "open worktree without hooks",
			args:    []string{"/test/repo/.sprout/feature"},
			noHooks: false,
			setupFx: func(fx *effects.TestEffects) {
				fx.Files["/test/repo/.sprout/feature"] = true
			},
			assertBehavior: func(t *testing.T, fx *effects.TestEffects) {
				// Editor opened
				require.Len(t, fx.OpenedPaths, 1)
				assert.Equal(t, "/test/repo/.sprout/feature", fx.OpenedPaths[0])

				// No hooks executed
				assert.Empty(t, fx.RunHooksInvocations)
			},
			wantErr: false,
		},
		{
			name:    "open worktree with hooks and trusted",
			args:    []string{"/test/repo/.sprout/feature"},
			noHooks: false,
			setupFx: func(fx *effects.TestEffects) {
				fx.Files["/test/repo/.sprout/feature"] = true
				fx.Config = &config.Config{
					Hooks: config.HooksConfig{
						OnOpen: []string{"echo 'opening'", "npm install"},
					},
				}
				fx.TrustedRepos["/test/repo"] = true
			},
			assertBehavior: func(t *testing.T, fx *effects.TestEffects) {
				// Editor opened first
				require.Len(t, fx.OpenedPaths, 1)
				assert.Equal(t, "/test/repo/.sprout/feature", fx.OpenedPaths[0])

				// Hooks executed with correct data
				require.Len(t, fx.RunHooksInvocations, 1)
				hookCall := fx.RunHooksInvocations[0]
				assert.Equal(t, core.HookTypeOnOpen, hookCall.HookType)
				assert.Equal(t, []string{"echo 'opening'", "npm install"}, hookCall.Commands)
				assert.Equal(t, "/test/repo/.sprout/feature", hookCall.WorktreePath)
				assert.Equal(t, "/test/repo", hookCall.RepoRoot)
				assert.Equal(t, "/test/repo", hookCall.MainWorktreePath)
			},
			wantErr: false,
		},
		{
			name:    "open worktree with hooks but untrusted - should fail",
			args:    []string{"/test/repo/.sprout/feature"},
			noHooks: false,
			setupFx: func(fx *effects.TestEffects) {
				fx.Files["/test/repo/.sprout/feature"] = true
				fx.Config = &config.Config{
					Hooks: config.HooksConfig{
						OnOpen: []string{"echo 'opening'"},
					},
				}
				fx.TrustedRepos["/test/repo"] = false
				// Make prompt fail (simulating non-interactive terminal)
				fx.PromptTrustRepoErr = fmt.Errorf("not a terminal: cannot prompt for trust interactively")
			},
			assertBehavior: func(t *testing.T, fx *effects.TestEffects) {
				// Should try to prompt
				assert.Equal(t, 1, fx.PromptTrustRepoCalls)

				// Should not open editor or run hooks (prompt failed)
				assert.Empty(t, fx.OpenedPaths)
				assert.Empty(t, fx.RunHooksInvocations)
			},
			wantErr:      true,
			wantExit:     false, // ExecutePlan returns regular error, not Exit
			wantExitCode: 0,
		},
		{
			name:    "open worktree with --no-hooks flag",
			args:    []string{"/test/repo/.sprout/feature"},
			noHooks: true,
			setupFx: func(fx *effects.TestEffects) {
				fx.Files["/test/repo/.sprout/feature"] = true
				fx.Config = &config.Config{
					Hooks: config.HooksConfig{
						OnOpen: []string{"echo 'opening'"},
					},
				}
			},
			assertBehavior: func(t *testing.T, fx *effects.TestEffects) {
				// Editor opened
				require.Len(t, fx.OpenedPaths, 1)

				// No hooks executed
				assert.Empty(t, fx.RunHooksInvocations)
			},
			wantErr: false,
		},
		{
			name:    "open by branch name",
			args:    []string{"feature"},
			noHooks: false,
			setupFx: func(fx *effects.TestEffects) {
				fx.Worktrees = []git.Worktree{
					{Path: "/test/data/sprout/repo-abc123/feature/repo", Branch: "feature"},
					{Path: "/test/data/sprout/repo-abc123/main/repo", Branch: "main"},
				}
				fx.Files["feature"] = false // Not a valid path
			},
			assertBehavior: func(t *testing.T, fx *effects.TestEffects) {
				// Editor opened to the correct worktree
				require.Len(t, fx.OpenedPaths, 1)
				assert.Equal(t, "/test/data/sprout/repo-abc123/feature/repo", fx.OpenedPaths[0])
			},
			wantErr: false,
		},
		{
			name:    "interactive selection",
			args:    []string{},
			noHooks: false,
			setupFx: func(fx *effects.TestEffects) {
				fx.Worktrees = []git.Worktree{
					{Path: "/test/data/sprout/repo-abc123/feature/repo", Branch: "feature"},
					{Path: "/test/data/sprout/repo-abc123/main/repo", Branch: "main"},
				}
				fx.SelectedWorktreeIndex = 1 // Select "main"
			},
			assertBehavior: func(t *testing.T, fx *effects.TestEffects) {
				// Editor opened to selected worktree
				require.Len(t, fx.OpenedPaths, 1)
				assert.Equal(t, "/test/data/sprout/repo-abc123/main/repo", fx.OpenedPaths[0])

				// Interactive selection was called
				assert.Equal(t, 1, fx.SelectWorktreeCalls)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fx := baseTestFxOpen(t)
			tt.setupFx(fx)

			// Build context from effects (simulating handler)
			ctx, err := BuildOpenContext(fx, tt.args, tt.noHooks)
			if tt.wantErr && err != nil {
				// Early error in context building
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Plan and execute
			plan := core.PlanOpenCommand(ctx)
			err = effects.ExecutePlan(plan, fx)

			// Check exit behavior
			if tt.wantExit {
				exitCode, isExit := effects.IsExit(err)
				require.True(t, isExit, "Expected exit error")
				assert.Equal(t, tt.wantExitCode, exitCode)
			} else if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Assert behavioral outcomes
			if tt.assertBehavior != nil {
				tt.assertBehavior(t, fx)
			}
		})
	}
}
