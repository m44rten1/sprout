package cmd

import (
	"testing"

	"github.com/m44rten1/sprout/internal/core"
	"github.com/m44rten1/sprout/internal/effects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUntrustCommand_EndToEnd(t *testing.T) {
	t.Run("untrust currently trusted repo", func(t *testing.T) {
		fx := effects.NewTestEffects()
		fx.MainWorktreePath = "/test/repo"
		fx.TrustedRepos = map[string]bool{
			"/test/repo": true,
		}

		// Build context
		ctx, err := BuildTrustContext(fx, "")
		require.NoError(t, err)
		assert.Equal(t, "/test/repo", ctx.RepoRoot)
		assert.True(t, ctx.AlreadyTrusted)

		// Plan
		plan := core.PlanUntrustCommand(ctx)

		// Execute
		err = effects.ExecutePlan(plan, fx)
		require.NoError(t, err)

		// Verify
		assert.Equal(t, 1, fx.UntrustRepoCalls)
		assert.Equal(t, []string{"/test/repo"}, fx.UntrustRepoRepos)
		assert.False(t, fx.TrustedRepos["/test/repo"])
		assert.Contains(t, fx.PrintedMsgs[0], "untrusted")
	})

	t.Run("untrust already untrusted repo", func(t *testing.T) {
		fx := effects.NewTestEffects()
		fx.MainWorktreePath = "/test/repo"
		fx.TrustedRepos = map[string]bool{}

		// Build context
		ctx, err := BuildTrustContext(fx, "")
		require.NoError(t, err)
		assert.False(t, ctx.AlreadyTrusted)

		// Plan
		plan := core.PlanUntrustCommand(ctx)

		// Execute
		err = effects.ExecutePlan(plan, fx)
		require.NoError(t, err)

		// Verify - should just print message, not call UntrustRepo
		assert.Equal(t, 0, fx.UntrustRepoCalls)
		assert.Contains(t, fx.PrintedMsgs[0], "not trusted")
	})

	t.Run("untrust specific path", func(t *testing.T) {
		fx := effects.NewTestEffects()
		fx.TrustedRepos = map[string]bool{
			"/other/repo": true,
		}
		fx.GitCommandOutput = map[string]string{
			"/other/repo\nrev-parse\n--show-toplevel": "/other/repo",
		}

		// Build context with explicit path
		ctx, err := BuildTrustContext(fx, "/other/repo")
		require.NoError(t, err)
		assert.Equal(t, "/other/repo", ctx.RepoRoot)
		assert.True(t, ctx.AlreadyTrusted)

		// Plan
		plan := core.PlanUntrustCommand(ctx)

		// Execute
		err = effects.ExecutePlan(plan, fx)
		require.NoError(t, err)

		// Verify
		assert.Equal(t, 1, fx.UntrustRepoCalls)
		assert.Equal(t, []string{"/other/repo"}, fx.UntrustRepoRepos)
		assert.False(t, fx.TrustedRepos["/other/repo"])
	})
}

