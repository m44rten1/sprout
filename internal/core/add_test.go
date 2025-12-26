package core

import (
	"testing"

	"github.com/m44rten1/sprout/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanAddCommand(t *testing.T) {
	tests := []struct {
		name         string
		ctx          AddContext
		wantActions  int
		checkActions func(t *testing.T, actions []Action)
	}{
		{
			name: "worktree already exists - opens editor",
			ctx: AddContext{
				Branch:             "feature",
				RepoRoot:           "/repo",
				WorktreePath:       "/sprout/feature",
				WorktreeExists:     true,
				LocalBranchExists:  true,
				RemoteBranchExists: false,
				HasOriginMain:      true,
				Config:             &config.Config{},
				IsTrusted:          true,
				NoHooks:            false,
				NoOpen:             false,
			},
			wantActions: 2,
			checkActions: func(t *testing.T, actions []Action) {
				assert.IsType(t, PrintMessage{}, actions[0])
				msg := actions[0].(PrintMessage)
				assert.Contains(t, msg.Msg, "already exists")
				assert.Contains(t, msg.Msg, "/sprout/feature")

				assert.IsType(t, OpenEditor{}, actions[1])
				editor := actions[1].(OpenEditor)
				assert.Equal(t, "/sprout/feature", editor.Path)
			},
		},
		{
			name: "worktree already exists with --no-open - only prints message",
			ctx: AddContext{
				Branch:             "feature",
				RepoRoot:           "/repo",
				WorktreePath:       "/sprout/feature",
				WorktreeExists:     true,
				LocalBranchExists:  true,
				RemoteBranchExists: false,
				HasOriginMain:      true,
				Config:             &config.Config{},
				IsTrusted:          true,
				NoHooks:            false,
				NoOpen:             true,
			},
			wantActions: 1,
			checkActions: func(t *testing.T, actions []Action) {
				assert.IsType(t, PrintMessage{}, actions[0])
				msg := actions[0].(PrintMessage)
				assert.Contains(t, msg.Msg, "already exists")
				assert.Contains(t, msg.Msg, "/sprout/feature")

				// Verify no editor action
				for _, action := range actions {
					_, isOpenEditor := action.(OpenEditor)
					assert.False(t, isOpenEditor, "should not open editor when --no-open flag is set")
				}
			},
		},
		{
			name: "new branch with hooks - trusted repo",
			ctx: AddContext{
				Branch:             "feature",
				RepoRoot:           "/repo",
				MainWorktreePath:   "/repo",
				WorktreePath:       "/sprout/feature",
				WorktreeExists:     false,
				LocalBranchExists:  false,
				RemoteBranchExists: false,
				HasOriginMain:      true,
				Config:             &config.Config{Hooks: config.HooksConfig{OnCreate: []string{"npm install"}}},
				IsTrusted:          true,
				NoHooks:            false,
				NoOpen:             false,
			},
			wantActions: 6,
			checkActions: func(t *testing.T, actions []Action) {
				assert.IsType(t, PrintMessage{}, actions[0], "should print creating message")
				assert.Contains(t, actions[0].(PrintMessage).Msg, "Creating worktree")

				assert.IsType(t, CreateDirectory{}, actions[1], "should create directory")
				dir := actions[1].(CreateDirectory)
				assert.Equal(t, "/sprout", dir.Path)

				assert.IsType(t, RunGitCommand{}, actions[2], "should run git worktree add")
				git := actions[2].(RunGitCommand)
				assert.Equal(t, "/repo", git.Dir)
				assert.Contains(t, git.Args, "worktree")
				assert.Contains(t, git.Args, "add")

				assert.IsType(t, PrintMessage{}, actions[3], "should print success message")
				assert.Contains(t, actions[3].(PrintMessage).Msg, "created")

				assert.IsType(t, OpenEditor{}, actions[4], "should open editor before hooks")
				assert.Equal(t, "/sprout/feature", actions[4].(OpenEditor).Path)

				assert.IsType(t, RunHooks{}, actions[5], "should run hooks")
				hooks := actions[5].(RunHooks)
				assert.Equal(t, HookTypeOnCreate, hooks.Type)
				assert.Equal(t, []string{"npm install"}, hooks.Commands)
				assert.Equal(t, "/sprout/feature", hooks.Path)
			},
		},
		{
			name: "new branch with hooks - untrusted repo",
			ctx: AddContext{
				Branch:             "feature",
				RepoRoot:           "/repo",
				MainWorktreePath:   "/repo",
				WorktreePath:       "/sprout/feature",
				WorktreeExists:     false,
				LocalBranchExists:  false,
				RemoteBranchExists: false,
				HasOriginMain:      true,
				Config:             &config.Config{Hooks: config.HooksConfig{OnCreate: []string{"npm install"}}},
				IsTrusted:          false,
				NoHooks:            false,
				NoOpen:             false,
			},
			wantActions: 2,
			checkActions: func(t *testing.T, actions []Action) {
				assert.IsType(t, PrintError{}, actions[0])
				err := actions[0].(PrintError)
				assert.Contains(t, err.Msg, "not trusted")
				assert.Contains(t, err.Msg, "sprout trust")

				assert.IsType(t, Exit{}, actions[1])
				exit := actions[1].(Exit)
				assert.Equal(t, 1, exit.Code)
			},
		},
		{
			name: "new branch with hooks but empty main worktree path",
			ctx: AddContext{
				Branch:             "feature",
				RepoRoot:           "/repo",
				MainWorktreePath:   "", // Empty - required for hooks
				WorktreePath:       "/sprout/feature",
				WorktreeExists:     false,
				LocalBranchExists:  false,
				RemoteBranchExists: false,
				HasOriginMain:      true,
				Config:             &config.Config{Hooks: config.HooksConfig{OnCreate: []string{"npm install"}}},
				IsTrusted:          true,
				NoHooks:            false,
				NoOpen:             false,
			},
			wantActions: 2,
			checkActions: func(t *testing.T, actions []Action) {
				assert.IsType(t, PrintError{}, actions[0])
				err := actions[0].(PrintError)
				assert.Equal(t, ErrEmptyMainWorktreePath.Error(), err.Msg)

				assert.IsType(t, Exit{}, actions[1])
				exit := actions[1].(Exit)
				assert.Equal(t, 1, exit.Code)
			},
		},
		{
			name: "new branch with hooks and --no-open - runs hooks without editor",
			ctx: AddContext{
				Branch:             "feature",
				RepoRoot:           "/repo",
				MainWorktreePath:   "/repo",
				WorktreePath:       "/sprout/feature",
				WorktreeExists:     false,
				LocalBranchExists:  false,
				RemoteBranchExists: false,
				HasOriginMain:      true,
				Config:             &config.Config{Hooks: config.HooksConfig{OnCreate: []string{"npm install"}}},
				IsTrusted:          true,
				NoHooks:            false,
				NoOpen:             true, // User explicitly skipped editor
			},
			wantActions: 5,
			checkActions: func(t *testing.T, actions []Action) {
				assert.IsType(t, PrintMessage{}, actions[0])
				assert.IsType(t, CreateDirectory{}, actions[1])
				assert.IsType(t, RunGitCommand{}, actions[2])
				assert.IsType(t, PrintMessage{}, actions[3])
				assert.IsType(t, RunHooks{}, actions[4], "should run hooks")

				hooks := actions[4].(RunHooks)
				assert.Equal(t, HookTypeOnCreate, hooks.Type)
				assert.Equal(t, []string{"npm install"}, hooks.Commands)
				assert.Equal(t, "/sprout/feature", hooks.Path)

				// Verify no editor action
				for _, action := range actions {
					_, isOpenEditor := action.(OpenEditor)
					assert.False(t, isOpenEditor, "should not open editor when --no-open flag is set")
				}
			},
		},
		{
			name: "new branch without hooks - no trust check",
			ctx: AddContext{
				Branch:             "feature",
				RepoRoot:           "/repo",
				WorktreePath:       "/sprout/feature",
				WorktreeExists:     false,
				LocalBranchExists:  false,
				RemoteBranchExists: false,
				HasOriginMain:      true,
				Config:             &config.Config{},
				IsTrusted:          false, // Not trusted, but no hooks so it's fine
				NoHooks:            false,
				NoOpen:             false,
			},
			wantActions: 5,
			checkActions: func(t *testing.T, actions []Action) {
				assert.IsType(t, PrintMessage{}, actions[0])
				assert.IsType(t, CreateDirectory{}, actions[1])
				assert.IsType(t, RunGitCommand{}, actions[2])
				assert.IsType(t, PrintMessage{}, actions[3])
				assert.IsType(t, OpenEditor{}, actions[4])
			},
		},
		{
			name: "new branch with --no-hooks flag",
			ctx: AddContext{
				Branch:             "feature",
				RepoRoot:           "/repo",
				WorktreePath:       "/sprout/feature",
				WorktreeExists:     false,
				LocalBranchExists:  false,
				RemoteBranchExists: false,
				HasOriginMain:      true,
				Config:             &config.Config{Hooks: config.HooksConfig{OnCreate: []string{"npm install"}}},
				IsTrusted:          true,
				NoHooks:            true, // User explicitly skipped hooks
				NoOpen:             false,
			},
			wantActions: 5,
			checkActions: func(t *testing.T, actions []Action) {
				assert.IsType(t, PrintMessage{}, actions[0])
				assert.IsType(t, CreateDirectory{}, actions[1])
				assert.IsType(t, RunGitCommand{}, actions[2])
				assert.IsType(t, PrintMessage{}, actions[3])
				assert.IsType(t, OpenEditor{}, actions[4])

				// Verify no hooks action
				for _, action := range actions {
					_, isRunHooks := action.(RunHooks)
					assert.False(t, isRunHooks, "should not run hooks when --no-hooks flag is set")
				}
			},
		},
		{
			name: "new branch with --no-open flag",
			ctx: AddContext{
				Branch:             "feature",
				RepoRoot:           "/repo",
				WorktreePath:       "/sprout/feature",
				WorktreeExists:     false,
				LocalBranchExists:  false,
				RemoteBranchExists: false,
				HasOriginMain:      true,
				Config:             &config.Config{},
				IsTrusted:          true,
				NoHooks:            false,
				NoOpen:             true, // User explicitly skipped editor
			},
			wantActions: 4,
			checkActions: func(t *testing.T, actions []Action) {
				assert.IsType(t, PrintMessage{}, actions[0])
				assert.IsType(t, CreateDirectory{}, actions[1])
				assert.IsType(t, RunGitCommand{}, actions[2])
				assert.IsType(t, PrintMessage{}, actions[3])

				// Verify no editor action
				for _, action := range actions {
					_, isOpenEditor := action.(OpenEditor)
					assert.False(t, isOpenEditor, "should not open editor when --no-open flag is set")
				}
			},
		},
		{
			name: "new branch with --no-open and --no-hooks flags",
			ctx: AddContext{
				Branch:             "feature",
				RepoRoot:           "/repo",
				WorktreePath:       "/sprout/feature",
				WorktreeExists:     false,
				LocalBranchExists:  false,
				RemoteBranchExists: false,
				HasOriginMain:      true,
				Config:             &config.Config{Hooks: config.HooksConfig{OnCreate: []string{"npm install"}}},
				IsTrusted:          true,
				NoHooks:            true,
				NoOpen:             true,
			},
			wantActions: 4,
			checkActions: func(t *testing.T, actions []Action) {
				assert.IsType(t, PrintMessage{}, actions[0])
				assert.IsType(t, CreateDirectory{}, actions[1])
				assert.IsType(t, RunGitCommand{}, actions[2])
				assert.IsType(t, PrintMessage{}, actions[3])

				// Verify neither hooks nor editor
				for _, action := range actions {
					_, isOpenEditor := action.(OpenEditor)
					_, isRunHooks := action.(RunHooks)
					assert.False(t, isOpenEditor, "should not open editor")
					assert.False(t, isRunHooks, "should not run hooks")
				}
			},
		},
		{
			name: "local branch exists - checkout existing",
			ctx: AddContext{
				Branch:             "feature",
				RepoRoot:           "/repo",
				WorktreePath:       "/sprout/feature",
				WorktreeExists:     false,
				LocalBranchExists:  true,
				RemoteBranchExists: false,
				HasOriginMain:      true,
				Config:             &config.Config{},
				IsTrusted:          true,
				NoHooks:            false,
				NoOpen:             false,
			},
			wantActions: 5,
			checkActions: func(t *testing.T, actions []Action) {
				assert.IsType(t, RunGitCommand{}, actions[2])
				git := actions[2].(RunGitCommand)
				// Should NOT have -b flag for existing branch
				assert.NotContains(t, git.Args, "-b")
			},
		},
		{
			name: "remote branch exists - track remote",
			ctx: AddContext{
				Branch:             "feature",
				RepoRoot:           "/repo",
				WorktreePath:       "/sprout/feature",
				WorktreeExists:     false,
				LocalBranchExists:  false,
				RemoteBranchExists: true,
				HasOriginMain:      true,
				Config:             &config.Config{},
				IsTrusted:          true,
				NoHooks:            false,
				NoOpen:             false,
			},
			wantActions: 5,
			checkActions: func(t *testing.T, actions []Action) {
				assert.IsType(t, RunGitCommand{}, actions[2])
				git := actions[2].(RunGitCommand)
				// Should create branch tracking remote
				assert.Contains(t, git.Args, "-b")
				assert.Contains(t, git.Args, "origin/feature")
			},
		},
		{
			name: "empty repo root - returns error",
			ctx: AddContext{
				Branch:             "feature",
				RepoRoot:           "", // Invalid
				WorktreePath:       "/sprout/feature",
				WorktreeExists:     false,
				LocalBranchExists:  false,
				RemoteBranchExists: false,
				HasOriginMain:      true,
				Config:             &config.Config{},
				IsTrusted:          true,
				NoHooks:            false,
				NoOpen:             false,
			},
			wantActions: 2,
			checkActions: func(t *testing.T, actions []Action) {
				assert.IsType(t, PrintError{}, actions[0])
				assert.Contains(t, actions[0].(PrintError).Msg, "repository root")

				assert.IsType(t, Exit{}, actions[1])
				assert.Equal(t, 1, actions[1].(Exit).Code)
			},
		},
		{
			name: "empty worktree path - returns error",
			ctx: AddContext{
				Branch:             "feature",
				RepoRoot:           "/repo",
				WorktreePath:       "", // Invalid
				WorktreeExists:     false,
				LocalBranchExists:  false,
				RemoteBranchExists: false,
				HasOriginMain:      true,
				Config:             &config.Config{},
				IsTrusted:          true,
				NoHooks:            false,
				NoOpen:             false,
			},
			wantActions: 2,
			checkActions: func(t *testing.T, actions []Action) {
				assert.IsType(t, PrintError{}, actions[0])
				assert.Contains(t, actions[0].(PrintError).Msg, "worktree path")

				assert.IsType(t, Exit{}, actions[1])
				assert.Equal(t, 1, actions[1].(Exit).Code)
			},
		},
		{
			name: "empty branch name - returns error",
			ctx: AddContext{
				Branch:             "", // Invalid
				RepoRoot:           "/repo",
				WorktreePath:       "/sprout/feature",
				WorktreeExists:     false,
				LocalBranchExists:  false,
				RemoteBranchExists: false,
				HasOriginMain:      true,
				Config:             &config.Config{},
				IsTrusted:          true,
				NoHooks:            false,
				NoOpen:             false,
			},
			wantActions: 2,
			checkActions: func(t *testing.T, actions []Action) {
				assert.IsType(t, PrintError{}, actions[0])
				assert.Contains(t, actions[0].(PrintError).Msg, "branch name")

				assert.IsType(t, Exit{}, actions[1])
				assert.Equal(t, 1, actions[1].(Exit).Code)
			},
		},
		{
			name: "nil config - returns error",
			ctx: AddContext{
				Branch:             "feature",
				RepoRoot:           "/repo",
				WorktreePath:       "/sprout/feature",
				WorktreeExists:     false,
				LocalBranchExists:  false,
				RemoteBranchExists: false,
				HasOriginMain:      true,
				Config:             nil, // Invalid
				IsTrusted:          true,
				NoHooks:            false,
				NoOpen:             false,
			},
			wantActions: 2,
			checkActions: func(t *testing.T, actions []Action) {
				assert.IsType(t, PrintError{}, actions[0])
				assert.Contains(t, actions[0].(PrintError).Msg, "config")

				assert.IsType(t, Exit{}, actions[1])
				assert.Equal(t, 1, actions[1].(Exit).Code)
			},
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable for parallel tests
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plan := PlanAddCommand(tt.ctx)

			require.Len(t, plan.Actions, tt.wantActions, "incorrect number of actions")
			tt.checkActions(t, plan.Actions)
		})
	}
}
