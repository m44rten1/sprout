package cmd

import (
	"errors"
	"testing"

	"github.com/m44rten1/sprout/internal/core"
	"github.com/m44rten1/sprout/internal/effects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errPermissionDenied = errors.New("permission denied")
	errDiskFull         = errors.New("disk full")
)

// stubGitRepoValid configures TestEffects to simulate a valid git repository
// at the given path. This encapsulates the git command format details.
func stubGitRepoValid(fx *effects.TestEffects, repoPath string) {
	key := repoPath + "\nrev-parse --show-toplevel"
	fx.GitCommandOutput[key] = repoPath
}

// TestPlanTrustCommand tests the pure planning logic.
// These tests validate PlanTrustCommand's decision-making with pre-built contexts.
func TestPlanTrustCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		ctx                core.TrustContext
		wantTrustRepoCalls int
		wantPrintCalls     int
		wantPrintErrCalls  int
		wantExit           bool
		wantExitCode       int
		assertOutput       func(t *testing.T, fx *effects.TestEffects)
	}{
		{
			name: "already trusted",
			ctx: core.TrustContext{
				RepoRoot:       "/test/repo",
				AlreadyTrusted: true,
			},
			wantTrustRepoCalls: 0,
			wantPrintCalls:     1,
			wantPrintErrCalls:  0,
			wantExit:           false,
			assertOutput: func(t *testing.T, fx *effects.TestEffects) {
				require.Len(t, fx.PrintedMsgs, 1)
				assert.Contains(t, fx.PrintedMsgs[0], "/test/repo")
				assert.Contains(t, fx.PrintedMsgs[0], "already trusted")
				// Verify no TrustRepo attempts recorded
				assert.Empty(t, fx.TrustRepoRepos)
			},
		},
		{
			name: "not yet trusted",
			ctx: core.TrustContext{
				RepoRoot:       "/test/repo",
				AlreadyTrusted: false,
			},
			wantTrustRepoCalls: 1,
			wantPrintCalls:     1,
			wantPrintErrCalls:  0,
			wantExit:           false,
			assertOutput: func(t *testing.T, fx *effects.TestEffects) {
				require.Len(t, fx.TrustRepoRepos, 1)
				assert.Equal(t, "/test/repo", fx.TrustRepoRepos[0])
				require.Len(t, fx.PrintedMsgs, 1)
				assert.Contains(t, fx.PrintedMsgs[0], "/test/repo")
				assert.Contains(t, fx.PrintedMsgs[0], "trusted")
				// Note: Testing "on_create"/"on_open" is brittle but intentional
				// These are stable UX contract strings that shouldn't change often
				assert.Contains(t, fx.PrintedMsgs[0], "on_create")
				assert.Contains(t, fx.PrintedMsgs[0], "on_open")
			},
		},
		{
			name: "custom repo path",
			ctx: core.TrustContext{
				RepoRoot:       "/custom/path/repo",
				AlreadyTrusted: false,
			},
			wantTrustRepoCalls: 1,
			wantPrintCalls:     1,
			wantPrintErrCalls:  0,
			wantExit:           false,
			assertOutput: func(t *testing.T, fx *effects.TestEffects) {
				require.Len(t, fx.TrustRepoRepos, 1)
				assert.Equal(t, "/custom/path/repo", fx.TrustRepoRepos[0])
				assert.Contains(t, fx.PrintedMsgs[0], "/custom/path/repo")
			},
		},
		{
			name: "empty repo root returns error",
			ctx: core.TrustContext{
				RepoRoot:       "",
				AlreadyTrusted: false,
			},
			wantTrustRepoCalls: 0,
			wantPrintCalls:     0,
			wantPrintErrCalls:  1,
			wantExit:           true,
			wantExitCode:       1,
			assertOutput: func(t *testing.T, fx *effects.TestEffects) {
				require.Len(t, fx.PrintedErrs, 1)
				assert.Contains(t, fx.PrintedErrs[0], "no repository root")
				// Verify no stdout (no contradictory output)
				assert.Empty(t, fx.PrintedMsgs)
				assert.Empty(t, fx.TrustRepoRepos)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fx := effects.NewTestEffects()
			plan := core.PlanTrustCommand(tt.ctx)
			err := effects.ExecutePlan(plan, fx)

			// Check exit behavior
			if tt.wantExit {
				exitCode, isExit := effects.IsExit(err)
				require.True(t, isExit, "Expected exit error")
				assert.Equal(t, tt.wantExitCode, exitCode)
			} else {
				assert.NoError(t, err)
			}

			// Verify call counts
			assert.Equal(t, tt.wantTrustRepoCalls, fx.TrustRepoCalls, "TrustRepo call count mismatch")
			assert.Equal(t, tt.wantPrintCalls, fx.PrintCalls, "Print call count mismatch")
			assert.Equal(t, tt.wantPrintErrCalls, fx.PrintErrCalls, "PrintErr call count mismatch")

			// Custom output assertions
			if tt.assertOutput != nil {
				tt.assertOutput(t, fx)
			}
		})
	}
}

// TestPlanTrustCommand_TrustRepoFailure tests error propagation separately
// because it has different assertion patterns (ErrorIs, partial execution).
func TestPlanTrustCommand_TrustRepoFailure(t *testing.T) {
	t.Parallel()

	fx := effects.NewTestEffects()
	fx.TrustRepoErr = errPermissionDenied

	ctx := core.TrustContext{
		RepoRoot:       "/test/repo",
		AlreadyTrusted: false,
	}

	plan := core.PlanTrustCommand(ctx)
	err := effects.ExecutePlan(plan, fx)

	// Verify error is returned, root cause preserved, and context wrapped
	require.Error(t, err)
	assert.ErrorIs(t, err, errPermissionDenied, "Root cause should be preserved")
	assert.Contains(t, err.Error(), "trust repo", "Error should include context")

	// Verify TrustRepo was attempted
	assert.Equal(t, 1, fx.TrustRepoCalls)

	// Verify plan stopped (no success message printed, no stdout)
	assert.Equal(t, 0, fx.PrintCalls)
	assert.Empty(t, fx.PrintedMsgs)

	// Contract: planner doesn't print errors (caller does)
	// So PrintErrCalls should be 0 - error is returned, not printed
	assert.Equal(t, 0, fx.PrintErrCalls)
	assert.Empty(t, fx.PrintedErrs)
}

// TestBuildTrustContext tests the handler's logic for building TrustContext
// from effects (the "imperative shell" layer). This catches wiring bugs
// that pure planner tests miss.
func TestBuildTrustContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		pathArg       string
		setupFx       func(*effects.TestEffects)
		wantCtx       *core.TrustContext // nil if error expected
		wantErr       bool
		assertEffects func(t *testing.T, fx *effects.TestEffects)
	}{
		{
			name:    "current repo not trusted",
			pathArg: "", // Empty = use current repo
			setupFx: func(fx *effects.TestEffects) {
				fx.MainWorktreePath = "/test/repo"
				fx.TrustedRepos["/test/repo"] = false
			},
			wantCtx: &core.TrustContext{
				RepoRoot:       "/test/repo",
				AlreadyTrusted: false,
			},
			wantErr: false,
			assertEffects: func(t *testing.T, fx *effects.TestEffects) {
				assert.Equal(t, 1, fx.GetMainWorktreePathCalls)
				assert.Equal(t, 1, fx.IsTrustedCalls)
				require.Len(t, fx.IsTrustedArgs, 1)
				assert.Equal(t, "/test/repo", fx.IsTrustedArgs[0])
			},
		},
		{
			name:    "current repo already trusted",
			pathArg: "",
			setupFx: func(fx *effects.TestEffects) {
				fx.MainWorktreePath = "/home/user/projects/myrepo"
				fx.TrustedRepos["/home/user/projects/myrepo"] = true
			},
			wantCtx: &core.TrustContext{
				RepoRoot:       "/home/user/projects/myrepo",
				AlreadyTrusted: true,
			},
			wantErr: false,
			assertEffects: func(t *testing.T, fx *effects.TestEffects) {
				assert.Equal(t, 1, fx.GetMainWorktreePathCalls)
				assert.Equal(t, 1, fx.IsTrustedCalls)
			},
		},
		{
			name:    "explicit path argument",
			pathArg: "/explicit/repo",
			setupFx: func(fx *effects.TestEffects) {
				// Stub git repo validation using helper (encapsulates format)
				stubGitRepoValid(fx, "/explicit/repo")
				fx.TrustedRepos["/explicit/repo"] = false
			},
			wantCtx: &core.TrustContext{
				RepoRoot:       "/explicit/repo",
				AlreadyTrusted: false,
			},
			wantErr: false,
			assertEffects: func(t *testing.T, fx *effects.TestEffects) {
				// Should NOT call GetMainWorktreePath when path is explicit
				assert.Equal(t, 0, fx.GetMainWorktreePathCalls)
				assert.Equal(t, 1, fx.IsTrustedCalls)
				require.Len(t, fx.IsTrustedArgs, 1)
				assert.Equal(t, "/explicit/repo", fx.IsTrustedArgs[0])
			},
		},
		{
			name:    "GetMainWorktreePath fails",
			pathArg: "",
			setupFx: func(fx *effects.TestEffects) {
				fx.GetMainWorktreePathErr = errDiskFull
			},
			wantCtx: nil, // Error expected, no valid context
			wantErr: true,
			assertEffects: func(t *testing.T, fx *effects.TestEffects) {
				assert.Equal(t, 1, fx.GetMainWorktreePathCalls)
				// Should not proceed to IsTrusted check
				assert.Equal(t, 0, fx.IsTrustedCalls)
			},
		},
		{
			name:    "IsTrusted fails",
			pathArg: "",
			setupFx: func(fx *effects.TestEffects) {
				fx.MainWorktreePath = "/test/repo"
				fx.IsTrustedErr = errPermissionDenied
			},
			wantCtx: nil, // Error expected, no valid context
			wantErr: true,
			assertEffects: func(t *testing.T, fx *effects.TestEffects) {
				assert.Equal(t, 1, fx.GetMainWorktreePathCalls)
				assert.Equal(t, 1, fx.IsTrustedCalls)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fx := effects.NewTestEffects()
			tt.setupFx(fx)

			ctx, err := BuildTrustContext(fx, tt.pathArg)

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

// TestTrustCommand_EndToEnd tests the full flow: BuildTrustContext → plan → execute.
// This catches integration bugs across all layers.
func TestTrustCommand_EndToEnd(t *testing.T) {
	t.Parallel()

	fx := effects.NewTestEffects()
	fx.MainWorktreePath = "/test/repo"
	fx.TrustedRepos["/test/repo"] = false

	// Verify initial state
	isTrusted, err := fx.IsTrusted("/test/repo")
	require.NoError(t, err)
	require.False(t, isTrusted, "Initial state: should not be trusted")

	// Build context from effects (simulating handler)
	ctx, err := BuildTrustContext(fx, "")
	require.NoError(t, err)
	assert.Equal(t, "/test/repo", ctx.RepoRoot)
	assert.False(t, ctx.AlreadyTrusted)

	// Plan and execute
	plan := core.PlanTrustCommand(ctx)
	err = effects.ExecutePlan(plan, fx)
	require.NoError(t, err)

	// Verify meaningful behavioral outcomes (not test bookkeeping)
	// 1. State transition: repo becomes trusted
	isTrusted, err = fx.IsTrusted("/test/repo")
	require.NoError(t, err)
	assert.True(t, isTrusted, "Final state: should be trusted after execution")

	// 2. TrustRepo action was executed
	assert.Equal(t, 1, fx.TrustRepoCalls)
	require.Len(t, fx.TrustRepoRepos, 1)
	assert.Equal(t, "/test/repo", fx.TrustRepoRepos[0])

	// 3. Success message was printed
	assert.Equal(t, 1, fx.PrintCalls)
	require.Len(t, fx.PrintedMsgs, 1)
	assert.Contains(t, fx.PrintedMsgs[0], "trusted")
	assert.Contains(t, fx.PrintedMsgs[0], "/test/repo")

	// Note: We DON'T assert exact IsTrustedCalls count - that's test bookkeeping,
	// not behavior. The meaningful contract is "repo becomes trusted", not
	// "IsTrusted called exactly N times".
}
