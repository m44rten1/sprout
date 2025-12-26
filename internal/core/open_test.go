package core

import (
	"testing"

	"github.com/m44rten1/sprout/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertErrorPlan verifies that a plan consists of PrintError followed by Exit.
func assertErrorPlan(t *testing.T, actions []Action, expectedErr error, expectedCode int) {
	t.Helper()

	require.Len(t, actions, 2)

	printErr, ok := actions[0].(PrintError)
	require.True(t, ok, "first action should be PrintError")
	assert.Equal(t, expectedErr.Error(), printErr.Msg)

	exitAction, ok := actions[1].(Exit)
	require.True(t, ok, "second action should be Exit")
	assert.Equal(t, expectedCode, exitAction.Code)
}

func TestPlanOpenCommand(t *testing.T) {
	tests := []struct {
		name         string
		ctx          OpenContext
		wantActions  int
		checkActions func(t *testing.T, actions []Action)
		wantExit     bool
		wantExitCode int
	}{
		{
			name: "open without hooks",
			ctx: OpenContext{
				TargetPath:       "/test/repo/.sprout/feature",
				RepoRoot:         "/test/repo",
				MainWorktreePath: "/test/repo",
				Config:           &config.Config{},
				IsTrusted:        false, // doesn't matter, no hooks
				NoHooks:          false,
			},
			wantActions: 1,
			checkActions: func(t *testing.T, actions []Action) {
				require.Len(t, actions, 1)

				openEditor, ok := actions[0].(OpenEditor)
				require.True(t, ok, "first action should be OpenEditor")
				assert.Equal(t, "/test/repo/.sprout/feature", openEditor.Path)
			},
		},
		{
			name: "open with hooks and trusted",
			ctx: OpenContext{
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
			wantActions: 2,
			checkActions: func(t *testing.T, actions []Action) {
				require.Len(t, actions, 2)

				openEditor, ok := actions[0].(OpenEditor)
				require.True(t, ok, "first action should be OpenEditor")
				assert.Equal(t, "/test/repo/.sprout/feature", openEditor.Path)

				runHooks, ok := actions[1].(RunHooks)
				require.True(t, ok, "second action should be RunHooks")
				assert.Equal(t, HookTypeOnOpen, runHooks.Type)
				assert.Equal(t, []string{"echo 'opening'", "npm install"}, runHooks.Commands)
				assert.Equal(t, "/test/repo/.sprout/feature", runHooks.Path)
				assert.Equal(t, "/test/repo", runHooks.RepoRoot)
				assert.Equal(t, "/test/repo", runHooks.MainWorktreePath)
			},
		},
		{
			name: "open with hooks but untrusted",
			ctx: OpenContext{
				TargetPath:       "/test/repo/.sprout/feature",
				RepoRoot:         "/test/repo",
				MainWorktreePath: "/test/repo",
				Config: &config.Config{
					Hooks: config.HooksConfig{
						OnOpen: []string{"echo 'opening'"},
					},
				},
				IsTrusted: false,
				NoHooks:   false,
			},
			wantActions:  2,
			wantExit:     true,
			wantExitCode: 1,
			checkActions: func(t *testing.T, actions []Action) {
				require.Len(t, actions, 2)

				printErr, ok := actions[0].(PrintError)
				require.True(t, ok, "first action should be PrintError")
				assert.Contains(t, printErr.Msg, "not trusted")

				exitAction, ok := actions[1].(Exit)
				require.True(t, ok, "second action should be Exit")
				assert.Equal(t, 1, exitAction.Code)
			},
		},
		{
			name: "open with hooks but empty main worktree path",
			ctx: OpenContext{
				TargetPath:       "/test/repo/.sprout/feature",
				RepoRoot:         "/test/repo",
				MainWorktreePath: "", // Empty - required for hooks
				Config: &config.Config{
					Hooks: config.HooksConfig{
						OnOpen: []string{"echo 'opening'"},
					},
				},
				IsTrusted: true,
				NoHooks:   false,
			},
			wantActions:  2,
			wantExit:     true,
			wantExitCode: 1,
			checkActions: func(t *testing.T, actions []Action) {
				assertErrorPlan(t, actions, ErrEmptyMainWorktreePath, 1)
			},
		},
		{
			name: "open with --no-hooks flag",
			ctx: OpenContext{
				TargetPath:       "/test/repo/.sprout/feature",
				RepoRoot:         "/test/repo",
				MainWorktreePath: "/test/repo",
				Config: &config.Config{
					Hooks: config.HooksConfig{
						OnOpen: []string{"echo 'opening'"},
					},
				},
				IsTrusted: true,
				NoHooks:   true, // explicitly disabled
			},
			wantActions: 1,
			checkActions: func(t *testing.T, actions []Action) {
				require.Len(t, actions, 1)

				openEditor, ok := actions[0].(OpenEditor)
				require.True(t, ok, "only action should be OpenEditor")
				assert.Equal(t, "/test/repo/.sprout/feature", openEditor.Path)
			},
		},
		{
			name: "open untrusted with --no-hooks flag",
			ctx: OpenContext{
				TargetPath:       "/test/repo/.sprout/feature",
				RepoRoot:         "/test/repo",
				MainWorktreePath: "/test/repo",
				Config: &config.Config{
					Hooks: config.HooksConfig{
						OnOpen: []string{"echo 'opening'"},
					},
				},
				IsTrusted: false, // untrusted, but irrelevant because NoHooks
				NoHooks:   true,  // explicitly disabled
			},
			wantActions: 1,
			checkActions: func(t *testing.T, actions []Action) {
				require.Len(t, actions, 1)

				openEditor, ok := actions[0].(OpenEditor)
				require.True(t, ok, "only action should be OpenEditor - trust irrelevant when hooks disabled")
				assert.Equal(t, "/test/repo/.sprout/feature", openEditor.Path)
			},
		},
		{
			name: "empty target path",
			ctx: OpenContext{
				TargetPath:       "",
				RepoRoot:         "/test/repo",
				MainWorktreePath: "/test/repo",
				Config:           &config.Config{},
				IsTrusted:        false,
				NoHooks:          false,
			},
			wantActions:  2,
			wantExit:     true,
			wantExitCode: 1,
			checkActions: func(t *testing.T, actions []Action) {
				assertErrorPlan(t, actions, ErrEmptyTargetPath, 1)
			},
		},
		{
			name: "empty repo root",
			ctx: OpenContext{
				TargetPath:       "/test/repo/.sprout/feature",
				RepoRoot:         "",
				MainWorktreePath: "/test/repo",
				Config:           &config.Config{},
				IsTrusted:        false,
				NoHooks:          false,
			},
			wantActions:  2,
			wantExit:     true,
			wantExitCode: 1,
			checkActions: func(t *testing.T, actions []Action) {
				assertErrorPlan(t, actions, ErrNoRepoRoot, 1)
			},
		},
		{
			name: "nil config",
			ctx: OpenContext{
				TargetPath:       "/test/repo/.sprout/feature",
				RepoRoot:         "/test/repo",
				MainWorktreePath: "/test/repo",
				Config:           nil,
				IsTrusted:        false,
				NoHooks:          false,
			},
			wantActions:  2,
			wantExit:     true,
			wantExitCode: 1,
			checkActions: func(t *testing.T, actions []Action) {
				assertErrorPlan(t, actions, ErrNilConfig, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := PlanOpenCommand(tt.ctx)

			require.Len(t, plan.Actions, tt.wantActions, "unexpected number of actions")

			if tt.checkActions != nil {
				tt.checkActions(t, plan.Actions)
			}

			// Verify exit action if expected
			if tt.wantExit {
				lastAction, ok := plan.Actions[len(plan.Actions)-1].(Exit)
				require.True(t, ok, "last action should be Exit")
				assert.Equal(t, tt.wantExitCode, lastAction.Code)
			}
		})
	}
}
