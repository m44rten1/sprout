package effects

import (
	"fmt"
	"testing"

	"github.com/m44rten1/sprout/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutePlan(t *testing.T) {
	t.Run("empty plan succeeds", func(t *testing.T) {
		fx := NewTestEffects()
		plan := core.Plan{Actions: []core.Action{}}

		err := ExecutePlan(plan, fx)

		assert.NoError(t, err)
	})

	t.Run("NoOp action does nothing", func(t *testing.T) {
		fx := NewTestEffects()
		plan := core.Plan{Actions: []core.Action{
			core.NoOp{},
		}}

		err := ExecutePlan(plan, fx)

		assert.NoError(t, err)
	})

	t.Run("PrintMessage calls Print", func(t *testing.T) {
		fx := NewTestEffects()
		plan := core.Plan{Actions: []core.Action{
			core.PrintMessage{Msg: "Hello, world!"},
		}}

		err := ExecutePlan(plan, fx)

		require.NoError(t, err)
		require.Len(t, fx.PrintedMsgs, 1)
		assert.Equal(t, "Hello, world!", fx.PrintedMsgs[0])
	})

	t.Run("PrintError calls PrintErr", func(t *testing.T) {
		fx := NewTestEffects()
		plan := core.Plan{Actions: []core.Action{
			core.PrintError{Msg: "Error occurred"},
		}}

		err := ExecutePlan(plan, fx)

		require.NoError(t, err)
		require.Len(t, fx.PrintedErrs, 1)
		assert.Equal(t, "Error occurred", fx.PrintedErrs[0])
	})

	t.Run("CreateDirectory calls MkdirAll", func(t *testing.T) {
		fx := NewTestEffects()
		plan := core.Plan{Actions: []core.Action{
			core.CreateDirectory{Path: "/test/dir", Perm: 0755},
		}}

		err := ExecutePlan(plan, fx)

		require.NoError(t, err)
		require.Len(t, fx.CreatedDirs, 1)
		assert.Equal(t, "/test/dir", fx.CreatedDirs[0])
		assert.True(t, fx.Files["/test/dir"], "Directory should be marked as existing")
	})

	t.Run("RunGitCommand executes git", func(t *testing.T) {
		fx := NewTestEffects()
		fx.GitCommandOutput["/repo\nworktree add"] = "success"
		plan := core.Plan{Actions: []core.Action{
			core.RunGitCommand{Dir: "/repo", Args: []string{"worktree", "add"}},
		}}

		err := ExecutePlan(plan, fx)

		require.NoError(t, err)
		require.Len(t, fx.GitCommands, 1)
		assert.Equal(t, "/repo", fx.GitCommands[0].Dir)
		assert.Equal(t, []string{"worktree", "add"}, fx.GitCommands[0].Args)
	})

	t.Run("OpenEditor calls OpenEditor", func(t *testing.T) {
		fx := NewTestEffects()
		plan := core.Plan{Actions: []core.Action{
			core.OpenEditor{Path: "/test/path"},
		}}

		err := ExecutePlan(plan, fx)

		require.NoError(t, err)
		require.Len(t, fx.OpenedPaths, 1)
		assert.Equal(t, "/test/path", fx.OpenedPaths[0])
	})

	t.Run("TrustRepo calls TrustRepo", func(t *testing.T) {
		fx := NewTestEffects()
		plan := core.Plan{Actions: []core.Action{
			core.TrustRepo{RepoRoot: "/test/repo"},
		}}

		err := ExecutePlan(plan, fx)

		require.NoError(t, err)
		assert.True(t, fx.TrustedRepos["/test/repo"], "Repo should be marked as trusted")
	})

	t.Run("Exit returns ExitError", func(t *testing.T) {
		fx := NewTestEffects()
		plan := core.Plan{Actions: []core.Action{
			core.Exit{Code: 42},
		}}

		err := ExecutePlan(plan, fx)

		require.Error(t, err)
		var exitErr ExitError
		require.ErrorAs(t, err, &exitErr)
		assert.Equal(t, 42, exitErr.Code)
	})

	t.Run("Exit stops execution", func(t *testing.T) {
		fx := NewTestEffects()
		plan := core.Plan{Actions: []core.Action{
			core.PrintMessage{Msg: "Before exit"},
			core.Exit{Code: 1},
			core.PrintMessage{Msg: "After exit"}, // Should not execute
		}}

		err := ExecutePlan(plan, fx)

		require.Error(t, err)
		var exitErr ExitError
		require.ErrorAs(t, err, &exitErr)
		assert.Equal(t, 1, exitErr.Code)

		// Only first message should print
		require.Len(t, fx.PrintedMsgs, 1)
		assert.Equal(t, "Before exit", fx.PrintedMsgs[0])
	})

	t.Run("multiple actions execute in order", func(t *testing.T) {
		fx := NewTestEffects()
		plan := core.Plan{Actions: []core.Action{
			core.PrintMessage{Msg: "First"},
			core.PrintMessage{Msg: "Second"},
			core.PrintError{Msg: "Error message"},
		}}

		err := ExecutePlan(plan, fx)

		require.NoError(t, err)
		assert.Equal(t, []string{"First", "Second"}, fx.PrintedMsgs)
		assert.Equal(t, []string{"Error message"}, fx.PrintedErrs)
	})

	t.Run("SelectInteractive returns error", func(t *testing.T) {
		fx := NewTestEffects()
		plan := core.Plan{Actions: []core.Action{
			core.SelectInteractive{Items: []any{"test"}, DisplayFunc: func(a any) string { return "" }},
		}}

		err := ExecutePlan(plan, fx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "SelectInteractive")
	})
}

func TestExecutePlan_ErrorHandling(t *testing.T) {
	t.Run("CreateDirectory error stops execution", func(t *testing.T) {
		fx := NewTestEffects()
		fx.MkdirAllErr = fmt.Errorf("permission denied")

		plan := core.Plan{Actions: []core.Action{
			core.PrintMessage{Msg: "Before error"},
			core.CreateDirectory{Path: "/test/dir", Perm: 0755},
			core.PrintMessage{Msg: "After error"}, // Should not execute
		}}

		err := ExecutePlan(plan, fx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "create directory")
		assert.Contains(t, err.Error(), "permission denied")

		// Only first message should print
		require.Len(t, fx.PrintedMsgs, 1)
		assert.Equal(t, "Before error", fx.PrintedMsgs[0])
	})

	t.Run("RunGitCommand error stops execution", func(t *testing.T) {
		fx := NewTestEffects()
		fx.GitCommandErrors["/repo\nworktree add"] = fmt.Errorf("worktree already exists")

		plan := core.Plan{Actions: []core.Action{
			core.PrintMessage{Msg: "Before git"},
			core.RunGitCommand{Dir: "/repo", Args: []string{"worktree", "add"}},
			core.PrintMessage{Msg: "After git"},
		}}

		err := ExecutePlan(plan, fx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "git command")
		assert.Contains(t, err.Error(), "worktree already exists")

		require.Len(t, fx.PrintedMsgs, 1)
		assert.Equal(t, "Before git", fx.PrintedMsgs[0])
	})

	t.Run("OpenEditor error stops execution", func(t *testing.T) {
		fx := NewTestEffects()
		fx.OpenEditorErr = fmt.Errorf("no editor found")

		plan := core.Plan{Actions: []core.Action{
			core.OpenEditor{Path: "/test/path"},
			core.PrintMessage{Msg: "Should not print"},
		}}

		err := ExecutePlan(plan, fx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "open editor")
		assert.Contains(t, err.Error(), "no editor found")

		assert.Empty(t, fx.PrintedMsgs)
	})

	t.Run("TrustRepo error stops execution", func(t *testing.T) {
		fx := NewTestEffects()
		fx.TrustRepoErr = fmt.Errorf("failed to write trust file")

		plan := core.Plan{Actions: []core.Action{
			core.TrustRepo{RepoRoot: "/test/repo"},
			core.PrintMessage{Msg: "Should not print"},
		}}

		err := ExecutePlan(plan, fx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "trust repo")
		assert.Contains(t, err.Error(), "failed to write trust file")

		assert.Empty(t, fx.PrintedMsgs)
	})
}

func TestExitError(t *testing.T) {
	t.Run("Error returns formatted message", func(t *testing.T) {
		err := ExitError{Code: 42}
		assert.Equal(t, "exit code 42", err.Error())
	})
}

func TestIsExit(t *testing.T) {
	t.Run("recognizes ExitError", func(t *testing.T) {
		err := ExitError{Code: 42}
		code, ok := IsExit(err)
		assert.True(t, ok)
		assert.Equal(t, 42, code)
	})

	t.Run("returns false for non-ExitError", func(t *testing.T) {
		err := assert.AnError
		code, ok := IsExit(err)
		assert.False(t, ok)
		assert.Equal(t, 0, code)
	})

	t.Run("returns false for nil error", func(t *testing.T) {
		code, ok := IsExit(nil)
		assert.False(t, ok)
		assert.Equal(t, 0, code)
	})

	t.Run("works with wrapped errors", func(t *testing.T) {
		wrapped := fmt.Errorf("context: %w", ExitError{Code: 7})
		code, ok := IsExit(wrapped)
		assert.True(t, ok)
		assert.Equal(t, 7, code)
	})
}
