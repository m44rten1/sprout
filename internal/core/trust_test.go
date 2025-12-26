package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanTrustCommand(t *testing.T) {
	t.Run("empty repo root returns error", func(t *testing.T) {
		ctx := TrustContext{
			RepoRoot:       "",
			AlreadyTrusted: false,
		}

		plan := PlanTrustCommand(ctx)

		require.Len(t, plan.Actions, 2)

		// First action: print error
		printErr, ok := plan.Actions[0].(PrintError)
		require.True(t, ok, "Expected PrintError action")
		assert.Equal(t, ErrNoRepoRoot.Error(), printErr.Msg)

		// Second action: exit with failure code
		exit, ok := plan.Actions[1].(Exit)
		require.True(t, ok, "Expected Exit action")
		assert.Equal(t, 1, exit.Code)
	})

	t.Run("empty repo root with already trusted still fails", func(t *testing.T) {
		ctx := TrustContext{
			RepoRoot:       "",
			AlreadyTrusted: true, // Even if trusted, empty root is invalid
		}

		plan := PlanTrustCommand(ctx)

		require.Len(t, plan.Actions, 2)

		printErr, ok := plan.Actions[0].(PrintError)
		require.True(t, ok, "Expected PrintError action")
		assert.Equal(t, ErrNoRepoRoot.Error(), printErr.Msg)

		exit, ok := plan.Actions[1].(Exit)
		require.True(t, ok, "Expected Exit action")
		assert.Equal(t, 1, exit.Code)
	})

	t.Run("already trusted", func(t *testing.T) {
		ctx := TrustContext{
			RepoRoot:       "/test/repo",
			AlreadyTrusted: true,
		}

		plan := PlanTrustCommand(ctx)

		require.Len(t, plan.Actions, 1)

		printMsg, ok := plan.Actions[0].(PrintMessage)
		require.True(t, ok, "Expected PrintMessage action")
		assert.Contains(t, printMsg.Msg, "✅ Repository is already trusted: /test/repo")
	})

	t.Run("not yet trusted", func(t *testing.T) {
		ctx := TrustContext{
			RepoRoot:       "/test/repo",
			AlreadyTrusted: false,
		}

		plan := PlanTrustCommand(ctx)

		require.Len(t, plan.Actions, 2)

		// First action: trust the repo
		trustAction, ok := plan.Actions[0].(TrustRepo)
		require.True(t, ok, "Expected TrustRepo action")
		assert.Equal(t, "/test/repo", trustAction.RepoRoot)

		// Second action: print success message with hook instructions
		printMsg, ok := plan.Actions[1].(PrintMessage)
		require.True(t, ok, "Expected PrintMessage action")
		assert.Contains(t, printMsg.Msg, "✅ Repository trusted: /test/repo")
		assert.Contains(t, printMsg.Msg, "on_create hooks")
		assert.Contains(t, printMsg.Msg, "on_open hooks")
		assert.Contains(t, printMsg.Msg, "--no-hooks")
	})
}

func TestPlanUntrustCommand(t *testing.T) {
	t.Run("empty repo root returns error", func(t *testing.T) {
		ctx := TrustContext{
			RepoRoot:       "",
			AlreadyTrusted: false,
		}

		plan := PlanUntrustCommand(ctx)

		require.Len(t, plan.Actions, 2)

		// First action: print error
		printErr, ok := plan.Actions[0].(PrintError)
		require.True(t, ok, "Expected PrintError action")
		assert.Equal(t, ErrNoRepoRoot.Error(), printErr.Msg)

		// Second action: exit with failure code
		exit, ok := plan.Actions[1].(Exit)
		require.True(t, ok, "Expected Exit action")
		assert.Equal(t, 1, exit.Code)
	})

	t.Run("not trusted", func(t *testing.T) {
		ctx := TrustContext{
			RepoRoot:       "/test/repo",
			AlreadyTrusted: false,
		}

		plan := PlanUntrustCommand(ctx)

		require.Len(t, plan.Actions, 1)

		printMsg, ok := plan.Actions[0].(PrintMessage)
		require.True(t, ok, "Expected PrintMessage action")
		assert.Contains(t, printMsg.Msg, "not trusted: /test/repo")
	})

	t.Run("currently trusted", func(t *testing.T) {
		ctx := TrustContext{
			RepoRoot:       "/test/repo",
			AlreadyTrusted: true,
		}

		plan := PlanUntrustCommand(ctx)

		require.Len(t, plan.Actions, 2)

		// First action: untrust the repo
		untrustAction, ok := plan.Actions[0].(UntrustRepo)
		require.True(t, ok, "Expected UntrustRepo action")
		assert.Equal(t, "/test/repo", untrustAction.RepoRoot)

		// Second action: print success message
		printMsg, ok := plan.Actions[1].(PrintMessage)
		require.True(t, ok, "Expected PrintMessage action")
		assert.Contains(t, printMsg.Msg, "✅ Repository untrusted: /test/repo")
		assert.Contains(t, printMsg.Msg, "no longer run automatically")
		assert.Contains(t, printMsg.Msg, "sprout trust")
	})
}
