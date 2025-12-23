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
		assert.Equal(t, errNoRepoRoot, printErr.Msg)

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
		assert.Equal(t, errNoRepoRoot, printErr.Msg)

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
		assert.Equal(t, msgRepoAlreadyTrusted, printMsg.Msg)
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
		assert.Equal(t, "/test/repo", trustAction.Repo)

		// Second action: print success message
		printMsg, ok := plan.Actions[1].(PrintMessage)
		require.True(t, ok, "Expected PrintMessage action")
		assert.Equal(t, msgRepoTrusted, printMsg.Msg)
	})
}
