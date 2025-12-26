package cmd

import (
	"errors"
	"testing"

	"github.com/m44rten1/sprout/internal/config"
	"github.com/m44rten1/sprout/internal/core"
	"github.com/m44rten1/sprout/internal/effects"
	"github.com/m44rten1/sprout/internal/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// baseTestFx creates a TestEffects with common defaults for add command tests.
// This reduces duplication and makes test case differences stand out.
func baseTestFx() *effects.TestEffects {
	fx := effects.NewTestEffects()
	fx.RepoRoot = "/test/repo"
	fx.MainWorktreePath = "/test/repo"
	fx.RemoteBranches["main"] = true
	fx.Config = &config.Config{Hooks: config.HooksConfig{}}
	return fx
}

// TestBuildAddContext tests the handler's logic for building AddContext
// from effects (the "imperative shell" layer). This catches wiring bugs
// that pure planner tests miss.
func TestBuildAddContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		noHooks       bool
		noOpen        bool
		setupFx       func(*effects.TestEffects)
		wantCtx       *core.AddContext // nil if error expected
		wantErr       bool
		assertEffects func(t *testing.T, fx *effects.TestEffects)
	}{
		{
			name:    "explicit branch with new worktree",
			args:    []string{"feature"},
			noHooks: false,
			noOpen:  false,
			setupFx: func(fx *effects.TestEffects) {
				fx.LocalBranches["feature"] = false
				fx.RemoteBranches["feature"] = false
				fx.WorktreePaths["feature"] = "/test/repo-sprout/feature"
				fx.Files["/test/repo-sprout/feature"] = false
				fx.TrustedRepos["/test/repo"] = false
			},
			wantCtx: &core.AddContext{
				Branch:             "feature",
				RepoRoot:           "/test/repo",
				MainWorktreePath:   "/test/repo",
				WorktreePath:       "/test/repo-sprout/feature",
				WorktreeExists:     false,
				LocalBranchExists:  false,
				RemoteBranchExists: false,
				HasOriginMain:      true,
				Config:             &config.Config{Hooks: config.HooksConfig{}},
				IsTrusted:          false,
				NoHooks:            false,
				NoOpen:             false,
			},
			wantErr: false,
			// Happy path: verify captured data, not exact call counts
		},
		{
			name:    "explicit branch with origin prefix stripped",
			args:    []string{"origin/feature"},
			noHooks: false,
			noOpen:  false,
			setupFx: func(fx *effects.TestEffects) {
				fx.LocalBranches["feature"] = false
				fx.RemoteBranches["feature"] = true
				fx.WorktreePaths["feature"] = "/test/repo-sprout/feature"
				fx.Files["/test/repo-sprout/feature"] = false
				fx.TrustedRepos["/test/repo"] = false
			},
			wantCtx: &core.AddContext{
				Branch:             "feature",
				RepoRoot:           "/test/repo",
				MainWorktreePath:   "/test/repo",
				WorktreePath:       "/test/repo-sprout/feature",
				WorktreeExists:     false,
				LocalBranchExists:  false,
				RemoteBranchExists: true,
				HasOriginMain:      true,
				Config:             &config.Config{Hooks: config.HooksConfig{}},
				IsTrusted:          false,
				NoHooks:            false,
				NoOpen:             false,
			},
			wantErr: false,
		},
		{
			name:    "interactive branch selection",
			args:    []string{}, // Empty args = interactive
			noHooks: false,
			noOpen:  false,
			setupFx: func(fx *effects.TestEffects) {
				fx.Branches = []git.Branch{
					{RefName: "feature", Name: "feature", DisplayName: "feature", IsLocal: true},
					{RefName: "origin/bugfix", Name: "bugfix", DisplayName: "bugfix", IsLocal: false},
				}
				fx.Worktrees = []git.Worktree{}
				fx.SelectedBranchIndex = 1 // Select "bugfix"
				fx.LocalBranches["bugfix"] = false
				fx.RemoteBranches["bugfix"] = true
				fx.WorktreePaths["bugfix"] = "/test/repo-sprout/bugfix"
				fx.Files["/test/repo-sprout/bugfix"] = false
			},
			wantCtx: &core.AddContext{
				Branch:             "bugfix",
				RepoRoot:           "/test/repo",
				MainWorktreePath:   "/test/repo",
				WorktreePath:       "/test/repo-sprout/bugfix",
				WorktreeExists:     false,
				LocalBranchExists:  false,
				RemoteBranchExists: true,
				HasOriginMain:      true,
				Config:             &config.Config{Hooks: config.HooksConfig{}},
				IsTrusted:          false,
				NoHooks:            false,
				NoOpen:             false,
			},
			wantErr: false,
			assertEffects: func(t *testing.T, fx *effects.TestEffects) {
				// Verify interactive flow happened
				assert.Greater(t, fx.SelectBranchCalls, 0)
				assert.Greater(t, fx.ListBranchesCalls, 0)
				assert.Greater(t, fx.ListWorktreesCalls, 0)
			},
		},
		{
			name:    "worktree exists",
			args:    []string{"existing"},
			noHooks: false,
			noOpen:  false,
			setupFx: func(fx *effects.TestEffects) {
				fx.WorktreePaths["existing"] = "/test/repo-sprout/existing"
				fx.Files["/test/repo-sprout/existing"] = true // Exists!
				fx.LocalBranches["existing"] = true
				fx.RemoteBranches["existing"] = false
			},
			wantCtx: &core.AddContext{
				Branch:             "existing",
				RepoRoot:           "/test/repo",
				MainWorktreePath:   "/test/repo",
				WorktreePath:       "/test/repo-sprout/existing",
				WorktreeExists:     true,
				LocalBranchExists:  true,
				RemoteBranchExists: false,
				HasOriginMain:      true,
				Config:             &config.Config{Hooks: config.HooksConfig{}},
				IsTrusted:          false,
				NoHooks:            false,
				NoOpen:             false,
			},
			wantErr: false,
		},
		{
			name:    "hooks configured and trusted",
			args:    []string{"feature"},
			noHooks: false,
			noOpen:  false,
			setupFx: func(fx *effects.TestEffects) {
				fx.LocalBranches["feature"] = false
				fx.RemoteBranches["feature"] = false
				fx.WorktreePaths["feature"] = "/test/repo-sprout/feature"
				fx.Files["/test/repo-sprout/feature"] = false
				fx.Config = &config.Config{
					Hooks: config.HooksConfig{
						OnCreate: []string{"npm install"},
					},
				}
				fx.TrustedRepos["/test/repo"] = true // Trusted!
			},
			wantCtx: &core.AddContext{
				Branch:             "feature",
				RepoRoot:           "/test/repo",
				MainWorktreePath:   "/test/repo",
				WorktreePath:       "/test/repo-sprout/feature",
				WorktreeExists:     false,
				LocalBranchExists:  false,
				RemoteBranchExists: false,
				HasOriginMain:      true,
				Config: &config.Config{
					Hooks: config.HooksConfig{
						OnCreate: []string{"npm install"},
					},
				},
				IsTrusted: true,
				NoHooks:   false,
				NoOpen:    false,
			},
			wantErr: false,
			assertEffects: func(t *testing.T, fx *effects.TestEffects) {
				// Security-critical: trust check must happen when hooks configured
				assert.Greater(t, fx.IsTrustedCalls, 0, "Trust check is required for hooks")
				require.Len(t, fx.IsTrustedArgs, 1)
				assert.Equal(t, "/test/repo", fx.IsTrustedArgs[0])
			},
		},
		{
			name:    "hooks configured but --no-hooks flag",
			args:    []string{"feature"},
			noHooks: true, // Flag overrides hooks
			noOpen:  false,
			setupFx: func(fx *effects.TestEffects) {
				fx.LocalBranches["feature"] = false
				fx.RemoteBranches["feature"] = false
				fx.WorktreePaths["feature"] = "/test/repo-sprout/feature"
				fx.Files["/test/repo-sprout/feature"] = false
				fx.Config = &config.Config{
					Hooks: config.HooksConfig{
						OnCreate: []string{"npm install"},
					},
				}
			},
			wantCtx: &core.AddContext{
				Branch:             "feature",
				RepoRoot:           "/test/repo",
				MainWorktreePath:   "/test/repo",
				WorktreePath:       "/test/repo-sprout/feature",
				WorktreeExists:     false,
				LocalBranchExists:  false,
				RemoteBranchExists: false,
				HasOriginMain:      true,
				Config: &config.Config{
					Hooks: config.HooksConfig{
						OnCreate: []string{"npm install"},
					},
				},
				IsTrusted: false,
				NoHooks:   true,
				NoOpen:    false,
			},
			wantErr: false,
			assertEffects: func(t *testing.T, fx *effects.TestEffects) {
				// Security-critical: --no-hooks must skip trust check
				assert.Equal(t, 0, fx.IsTrustedCalls, "Trust check must be skipped with --no-hooks")
			},
		},
		{
			name:    "GetRepoRoot fails",
			args:    []string{"feature"},
			noHooks: false,
			noOpen:  false,
			setupFx: func(fx *effects.TestEffects) {
				fx.GetRepoRootErr = errors.New("not a git repository")
			},
			wantCtx: nil, // Error expected
			wantErr: true,
			assertEffects: func(t *testing.T, fx *effects.TestEffects) {
				assert.Equal(t, 1, fx.GetRepoRootCalls)
				// Should not proceed further
				assert.Equal(t, 0, fx.GetMainWorktreePathCalls)
			},
		},
		{
			name:    "GetMainWorktreePath fails",
			args:    []string{"feature"},
			noHooks: false,
			noOpen:  false,
			setupFx: func(fx *effects.TestEffects) {
				fx.RepoRoot = "/test/repo"
				fx.GetMainWorktreePathErr = errDiskFull
			},
			wantCtx: nil, // Error expected
			wantErr: true,
			assertEffects: func(t *testing.T, fx *effects.TestEffects) {
				assert.Equal(t, 1, fx.GetRepoRootCalls)
				assert.Equal(t, 1, fx.GetMainWorktreePathCalls)
				// Should not proceed to branch selection
				assert.Equal(t, 0, fx.SelectBranchCalls)
			},
		},
		{
			name:    "interactive selection cancelled",
			args:    []string{}, // Interactive mode
			noHooks: false,
			noOpen:  false,
			setupFx: func(fx *effects.TestEffects) {
				fx.Branches = []git.Branch{
					{RefName: "feature", Name: "feature", DisplayName: "feature", IsLocal: true},
				}
				fx.Worktrees = []git.Worktree{}
				fx.SelectionError = errors.New("user cancelled")
			},
			wantCtx: nil, // Error expected
			wantErr: true,
			assertEffects: func(t *testing.T, fx *effects.TestEffects) {
				assert.Equal(t, 1, fx.SelectBranchCalls)
			},
		},
		{
			name:    "no available branches in interactive mode",
			args:    []string{}, // Interactive mode
			noHooks: false,
			noOpen:  false,
			setupFx: func(fx *effects.TestEffects) {
				fx.Branches = []git.Branch{
					{RefName: "main", Name: "main", DisplayName: "main", IsLocal: true},
				}
				fx.Worktrees = []git.Worktree{
					{Branch: "main", Path: "/test/repo"}, // main is taken
				}
			},
			wantCtx: nil, // Error expected
			wantErr: true,
			assertEffects: func(t *testing.T, fx *effects.TestEffects) {
				assert.Equal(t, 1, fx.ListBranchesCalls)
				assert.Equal(t, 1, fx.ListWorktreesCalls)
				// Should not proceed to selection
				assert.Equal(t, 0, fx.SelectBranchCalls)
			},
		},
		{
			name:    "LoadConfig fails",
			args:    []string{"feature"},
			noHooks: false,
			noOpen:  false,
			setupFx: func(fx *effects.TestEffects) {
				fx.LocalBranches["feature"] = false
				fx.RemoteBranches["feature"] = false
				fx.WorktreePaths["feature"] = "/test/repo-sprout/feature"
				fx.Files["/test/repo-sprout/feature"] = false
				fx.LoadConfigErr = errors.New("config parse error")
			},
			wantCtx: nil, // Error expected
			wantErr: true,
			assertEffects: func(t *testing.T, fx *effects.TestEffects) {
				assert.Equal(t, 1, fx.LoadConfigCalls)
			},
		},
		{
			name:    "IsTrusted fails",
			args:    []string{"feature"},
			noHooks: false,
			noOpen:  false,
			setupFx: func(fx *effects.TestEffects) {
				fx.LocalBranches["feature"] = false
				fx.RemoteBranches["feature"] = false
				fx.WorktreePaths["feature"] = "/test/repo-sprout/feature"
				fx.Files["/test/repo-sprout/feature"] = false
				fx.Config = &config.Config{
					Hooks: config.HooksConfig{
						OnCreate: []string{"npm install"},
					},
				}
				fx.IsTrustedErr = errPermissionDenied
			},
			wantCtx: nil, // Error expected
			wantErr: true,
			assertEffects: func(t *testing.T, fx *effects.TestEffects) {
				assert.Equal(t, 1, fx.IsTrustedCalls)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fx := baseTestFx()
			tt.setupFx(fx)

			ctx, err := BuildAddContext(fx, tt.args, tt.noHooks, tt.noOpen)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, tt.wantCtx, "Test misconfiguration: wantCtx should not be nil for success cases")
				assert.Equal(t, *tt.wantCtx, ctx)
			}

			if tt.assertEffects != nil {
				tt.assertEffects(t, fx)
			}
		})
	}
}

// TestAddCommand_EndToEnd tests the full flow: BuildAddContext → plan → execute.
// This catches integration bugs across all layers.
func TestAddCommand_EndToEnd(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		args           []string
		noHooks        bool
		noOpen         bool
		setupFx        func(*effects.TestEffects)
		assertBehavior func(t *testing.T, fx *effects.TestEffects)
		wantErr        bool
		wantExit       bool
		wantExitCode   int
	}{
		{
			name:    "create new worktree with local branch",
			args:    []string{"feature"},
			noHooks: false,
			noOpen:  false,
			setupFx: func(fx *effects.TestEffects) {
				fx.LocalBranches["feature"] = true
				fx.RemoteBranches["feature"] = false
				fx.WorktreePaths["feature"] = "/test/repo-sprout/feature"
				fx.Files["/test/repo-sprout/feature"] = false
			},
			assertBehavior: func(t *testing.T, fx *effects.TestEffects) {
				// Directory created
				require.Len(t, fx.CreatedDirs, 1)
				assert.Equal(t, "/test/repo-sprout", fx.CreatedDirs[0])

				// Git command executed with correct args
				require.Len(t, fx.GitCommands, 1)
				cmd := fx.GitCommands[0]
				assert.Equal(t, "/test/repo", cmd.Dir)
				assert.Contains(t, cmd.Args, "worktree")
				assert.Contains(t, cmd.Args, "add")
				assert.Contains(t, cmd.Args, "/test/repo-sprout/feature")

				// Editor opened
				require.Len(t, fx.OpenedPaths, 1)
				assert.Equal(t, "/test/repo-sprout/feature", fx.OpenedPaths[0])

				// No hooks run (none configured)
				assert.Equal(t, 0, fx.RunHooksCalls)

				// Success messages printed
				assert.Contains(t, fx.PrintedMsgs[0], "Creating")
				assert.Contains(t, fx.PrintedMsgs[1], "created")
			},
			wantErr: false,
		},
		{
			name:    "create new worktree with remote branch",
			args:    []string{"origin/bugfix"},
			noHooks: false,
			noOpen:  false,
			setupFx: func(fx *effects.TestEffects) {
				fx.LocalBranches["bugfix"] = false
				fx.RemoteBranches["bugfix"] = true
				fx.WorktreePaths["bugfix"] = "/test/repo-sprout/bugfix"
				fx.Files["/test/repo-sprout/bugfix"] = false
			},
			assertBehavior: func(t *testing.T, fx *effects.TestEffects) {
				// Git command should track remote branch (not use --no-track)
				require.Len(t, fx.GitCommands, 1)
				cmd := fx.GitCommands[0]
				assert.Contains(t, cmd.Args, "worktree")
				assert.Contains(t, cmd.Args, "add")
				assert.NotContains(t, cmd.Args, "--no-track")

				// Editor opened
				require.Len(t, fx.OpenedPaths, 1)
			},
			wantErr: false,
		},
		{
			name:    "create new worktree with hooks (trusted)",
			args:    []string{"feature"},
			noHooks: false,
			noOpen:  false,
			setupFx: func(fx *effects.TestEffects) {
				fx.LocalBranches["feature"] = false
				fx.RemoteBranches["feature"] = false
				fx.WorktreePaths["feature"] = "/test/repo-sprout/feature"
				fx.Files["/test/repo-sprout/feature"] = false
				fx.Config = &config.Config{
					Hooks: config.HooksConfig{
						OnCreate: []string{"npm install", "npm test"},
					},
				}
				fx.TrustedRepos["/test/repo"] = true
			},
			assertBehavior: func(t *testing.T, fx *effects.TestEffects) {
				// Worktree created
				require.Len(t, fx.GitCommands, 1)

				// Editor opened
				require.Len(t, fx.OpenedPaths, 1)

				// Hooks executed with correct data
				require.Len(t, fx.RunHooksInvocations, 1)
				hookCall := fx.RunHooksInvocations[0]
				assert.Equal(t, core.HookTypeOnCreate, hookCall.HookType)
				assert.Equal(t, []string{"npm install", "npm test"}, hookCall.Commands)
				assert.Equal(t, "/test/repo-sprout/feature", hookCall.WorktreePath)
				assert.Equal(t, "/test/repo", hookCall.RepoRoot)
				assert.Equal(t, "/test/repo", hookCall.MainWorktreePath)
			},
			wantErr: false,
		},
		{
			name:    "create new worktree with hooks (untrusted) - should fail",
			args:    []string{"feature"},
			noHooks: false,
			noOpen:  false,
			setupFx: func(fx *effects.TestEffects) {
				fx.LocalBranches["feature"] = false
				fx.RemoteBranches["feature"] = false
				fx.WorktreePaths["feature"] = "/test/repo-sprout/feature"
				fx.Files["/test/repo-sprout/feature"] = false
				fx.Config = &config.Config{
					Hooks: config.HooksConfig{
						OnCreate: []string{"npm install"},
					},
				}
				fx.TrustedRepos["/test/repo"] = false
			},
			assertBehavior: func(t *testing.T, fx *effects.TestEffects) {
				// Should not create worktree
				assert.Empty(t, fx.GitCommands)
				assert.Empty(t, fx.OpenedPaths)
				assert.Empty(t, fx.RunHooksInvocations)

				// Error message printed
				require.Len(t, fx.PrintedErrs, 1)
				assert.Contains(t, fx.PrintedErrs[0], "not trusted")
			},
			wantErr:      true,
			wantExit:     true,
			wantExitCode: 1,
		},
		{
			name:    "worktree exists - just open it",
			args:    []string{"existing"},
			noHooks: false,
			noOpen:  false,
			setupFx: func(fx *effects.TestEffects) {
				fx.LocalBranches["existing"] = true
				fx.RemoteBranches["existing"] = false
				fx.WorktreePaths["existing"] = "/test/repo-sprout/existing"
				fx.Files["/test/repo-sprout/existing"] = true
			},
			assertBehavior: func(t *testing.T, fx *effects.TestEffects) {
				// Should NOT create directory or run git command
				assert.Empty(t, fx.CreatedDirs)
				assert.Empty(t, fx.GitCommands)

				// Should open editor
				require.Len(t, fx.OpenedPaths, 1)
				assert.Equal(t, "/test/repo-sprout/existing", fx.OpenedPaths[0])

				// Message about existing worktree
				assert.Contains(t, fx.PrintedMsgs[0], "already exists")
			},
			wantErr: false,
		},
		{
			name:    "worktree exists with --no-open",
			args:    []string{"existing"},
			noHooks: false,
			noOpen:  true,
			setupFx: func(fx *effects.TestEffects) {
				fx.LocalBranches["existing"] = true
				fx.RemoteBranches["existing"] = false
				fx.WorktreePaths["existing"] = "/test/repo-sprout/existing"
				fx.Files["/test/repo-sprout/existing"] = true
			},
			assertBehavior: func(t *testing.T, fx *effects.TestEffects) {
				// Should NOT open editor
				assert.Empty(t, fx.OpenedPaths)

				// Message still printed
				assert.Contains(t, fx.PrintedMsgs[0], "already exists")
			},
			wantErr: false,
		},
		{
			name:    "create with --no-open and --no-hooks",
			args:    []string{"feature"},
			noHooks: true,
			noOpen:  true,
			setupFx: func(fx *effects.TestEffects) {
				fx.LocalBranches["feature"] = false
				fx.RemoteBranches["feature"] = false
				fx.WorktreePaths["feature"] = "/test/repo-sprout/feature"
				fx.Files["/test/repo-sprout/feature"] = false
				fx.Config = &config.Config{
					Hooks: config.HooksConfig{
						OnCreate: []string{"npm install"},
					},
				}
			},
			assertBehavior: func(t *testing.T, fx *effects.TestEffects) {
				// Worktree created
				require.Len(t, fx.GitCommands, 1)

				// No editor, no hooks
				assert.Empty(t, fx.OpenedPaths)
				assert.Empty(t, fx.RunHooksInvocations)

				// Success messages printed
				assert.Contains(t, fx.PrintedMsgs[0], "Creating")
				assert.Contains(t, fx.PrintedMsgs[1], "created")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fx := baseTestFx()
			tt.setupFx(fx)

			// Build context from effects (simulating handler)
			ctx, err := BuildAddContext(fx, tt.args, tt.noHooks, tt.noOpen)
			if tt.wantErr && err != nil {
				// Early error in context building
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Plan and execute
			plan := core.PlanAddCommand(ctx)
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

			// Verify behavioral outcomes
			if tt.assertBehavior != nil {
				tt.assertBehavior(t, fx)
			}
		})
	}
}
