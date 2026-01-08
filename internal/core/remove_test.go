package core

import (
	"strings"
	"testing"

	"github.com/m44rten1/sprout/internal/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanRemoveCommand(t *testing.T) {
	tests := []struct {
		name         string
		ctx          RemoveContext
		wantActions  int
		wantExit     bool
		wantExitCode int
		assertions   func(t *testing.T, plan Plan)
	}{
		{
			name: "remove worktree successfully",
			ctx: RemoveContext{
				RepoRoot:   "/test/repo",
				SproutRoot: "/test/repo/.sprout",
				TargetPath: "/test/repo/.sprout/feature",
				Force:      false,
			},
			wantActions: 3, // git remove + success message + prune
			wantExit:    false,
			assertions: func(t *testing.T, plan Plan) {
				// Action 1: git worktree remove
				gitCmd, ok := plan.Actions[0].(RunGitCommand)
				require.True(t, ok, "expected RunGitCommand at index 0")
				assert.Equal(t, "/test/repo", gitCmd.Dir)
				assert.Equal(t, []string{"worktree", "remove", "/test/repo/.sprout/feature"}, gitCmd.Args)

				// Action 2: success message
				msg, ok := plan.Actions[1].(PrintMessage)
				require.True(t, ok, "expected PrintMessage at index 1")
				assert.Contains(t, msg.Msg, "Removed worktree")
				assert.Contains(t, msg.Msg, "/test/repo/.sprout/feature")

				// Action 3: prune
				prune, ok := plan.Actions[2].(RunGitCommand)
				require.True(t, ok, "expected RunGitCommand at index 2")
				assert.Equal(t, []string{"worktree", "prune"}, prune.Args)
			},
		},
		{
			name: "remove worktree with force flag",
			ctx: RemoveContext{
				RepoRoot:   "/test/repo",
				SproutRoot: "/test/repo/.sprout",
				TargetPath: "/test/repo/.sprout/feature",
				Force:      true,
			},
			wantActions: 3,
			wantExit:    false,
			assertions: func(t *testing.T, plan Plan) {
				// Check --force flag is present in first action
				gitCmd, ok := plan.Actions[0].(RunGitCommand)
				require.True(t, ok, "expected RunGitCommand at index 0")
				assert.Equal(t, []string{"worktree", "remove", "--force", "/test/repo/.sprout/feature"}, gitCmd.Args)
			},
		},
		{
			name: "empty repo root returns error",
			ctx: RemoveContext{
				RepoRoot:   "",
				SproutRoot: "/test/repo/.sprout",
				TargetPath: "/test/repo/.sprout/feature",
			},
			wantActions:  2, // error + exit
			wantExit:     true,
			wantExitCode: 1,
			assertions: func(t *testing.T, plan Plan) {
				errMsg, ok := plan.Actions[0].(PrintError)
				require.True(t, ok, "expected PrintError at index 0")
				assert.Contains(t, errMsg.Msg, "repository root")
			},
		},
		{
			name: "empty target path returns error",
			ctx: RemoveContext{
				RepoRoot:   "/test/repo",
				SproutRoot: "/test/repo/.sprout",
				TargetPath: "",
			},
			wantActions:  2,
			wantExit:     true,
			wantExitCode: 1,
			assertions: func(t *testing.T, plan Plan) {
				errMsg, ok := plan.Actions[0].(PrintError)
				require.True(t, ok, "expected PrintError at index 0")
				assert.Contains(t, errMsg.Msg, "target path")
			},
		},
		{
			name: "non-sprout worktree returns error",
			ctx: RemoveContext{
				RepoRoot:   "/test/repo",
				SproutRoot: "/test/repo/.sprout",
				TargetPath: "/some/other/path", // Not under sprout root
			},
			wantActions:  2,
			wantExit:     true,
			wantExitCode: 1,
			assertions: func(t *testing.T, plan Plan) {
				errMsg, ok := plan.Actions[0].(PrintError)
				require.True(t, ok, "expected PrintError at index 0")
				assert.Contains(t, errMsg.Msg, "Refusing to remove non-sprout worktree")
				assert.Contains(t, errMsg.Msg, "/some/other/path")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := PlanRemoveCommand(tt.ctx)

			assert.Len(t, plan.Actions, tt.wantActions)

			// Check exit action behavior
			lastAction := plan.Actions[len(plan.Actions)-1]
			if tt.wantExit {
				exit, ok := lastAction.(Exit)
				require.True(t, ok, "expected Exit action for error plan")
				assert.Equal(t, tt.wantExitCode, exit.Code)
			} else {
				_, ok := lastAction.(Exit)
				assert.False(t, ok, "did not expect Exit action for success plan")
			}

			// Run custom assertions
			if tt.assertions != nil {
				tt.assertions(t, plan)
			}
		})
	}
}

func TestBuildRemoveWorktreeArgs(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		force    bool
		wantArgs []string
	}{
		{
			name:     "without force",
			path:     "/test/repo/.sprout/feature",
			force:    false,
			wantArgs: []string{"worktree", "remove", "/test/repo/.sprout/feature"},
		},
		{
			name:     "with force",
			path:     "/test/repo/.sprout/feature",
			force:    true,
			wantArgs: []string{"worktree", "remove", "--force", "/test/repo/.sprout/feature"},
		},
		{
			name:     "empty path without force",
			path:     "",
			force:    false,
			wantArgs: []string{"worktree", "remove", ""},
		},
		{
			name:     "empty path with force",
			path:     "",
			force:    true,
			wantArgs: []string{"worktree", "remove", "--force", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildRemoveWorktreeArgs(tt.path, tt.force)
			assert.Equal(t, tt.wantArgs, got)
		})
	}
}

// TestRemoveContext_Documentation verifies the documented fields are present
func TestRemoveContext_Documentation(t *testing.T) {
	// This test exists to document the RemoveContext structure and ensure
	// all fields are properly accessible
	ctx := RemoveContext{
		ArgProvided: true,
		Arg:         "feature",
		RepoRoot:    "/test/repo",
		SproutRoot:  "/test/repo/.sprout",
		Worktrees:   []git.Worktree{{Branch: "main"}, {Branch: "feature"}},
		TargetPath:  "/test/repo/.sprout/feature",
		Force:       true,
	}

	// Verify all fields are accessible
	assert.True(t, ctx.ArgProvided)
	assert.Equal(t, "feature", ctx.Arg)
	assert.Equal(t, "/test/repo", ctx.RepoRoot)
	assert.Equal(t, "/test/repo/.sprout", ctx.SproutRoot)
	assert.Len(t, ctx.Worktrees, 2)
	assert.Equal(t, "/test/repo/.sprout/feature", ctx.TargetPath)
	assert.True(t, ctx.Force)
}

func TestPlanRemoveCommand_BranchDeletion(t *testing.T) {
	tests := []struct {
		name         string
		ctx          RemoveContext
		wantExit     bool
		wantExitCode int
		assertions   func(t *testing.T, plan Plan)
	}{
		{
			name: "delete local branch with -d (merged)",
			ctx: RemoveContext{
				RepoRoot:     "/test/repo",
				SproutRoot:   "/test/repo/.sprout",
				TargetPath:   "/test/repo/.sprout/feature",
				DeleteBranch: true,
				BranchName:   "feature",
				BranchExists: true,
				IsMerged:     true,
			},
			wantExit: false,
			assertions: func(t *testing.T, plan Plan) {
				// Should have: remove worktree, success msg, delete branch, branch deleted msg, prune
				require.Len(t, plan.Actions, 5)

				// Check delete branch action
				delBranch, ok := plan.Actions[2].(DeleteLocalBranch)
				require.True(t, ok, "expected DeleteLocalBranch at index 2")
				assert.Equal(t, "feature", delBranch.Branch)
				assert.False(t, delBranch.Force)

				// Check success message
				msg, ok := plan.Actions[3].(PrintMessage)
				require.True(t, ok, "expected PrintMessage at index 3")
				assert.Contains(t, msg.Msg, "Deleted local branch feature")
			},
		},
		{
			name: "delete local branch with -d (not merged) fails",
			ctx: RemoveContext{
				RepoRoot:     "/test/repo",
				SproutRoot:   "/test/repo/.sprout",
				TargetPath:   "/test/repo/.sprout/feature",
				DeleteBranch: true,
				BranchName:   "feature",
				BranchExists: true,
				IsMerged:     false, // Not merged!
			},
			wantExit:     true,
			wantExitCode: 1,
			assertions: func(t *testing.T, plan Plan) {
				// Should have: remove worktree, success msg, error about not merged, exit, prune
				// The error+exit is in the middle, not at the end
				foundError := false
				for _, action := range plan.Actions {
					if errMsg, ok := action.(PrintError); ok {
						assert.Contains(t, errMsg.Msg, "not merged into main")
						assert.Contains(t, errMsg.Msg, "Use -D")
						foundError = true
						break
					}
				}
				assert.True(t, foundError, "expected error message about not merged")
			},
		},
		{
			name: "force delete local branch with -D (not merged) succeeds",
			ctx: RemoveContext{
				RepoRoot:     "/test/repo",
				SproutRoot:   "/test/repo/.sprout",
				TargetPath:   "/test/repo/.sprout/feature",
				ForceDelete:  true, // -D flag
				BranchName:   "feature",
				BranchExists: true,
				IsMerged:     false,
			},
			wantExit: false,
			assertions: func(t *testing.T, plan Plan) {
				// Should have: remove worktree, success msg, delete branch, branch deleted msg, prune
				require.Len(t, plan.Actions, 5)

				// Check delete branch action with force
				delBranch, ok := plan.Actions[2].(DeleteLocalBranch)
				require.True(t, ok, "expected DeleteLocalBranch at index 2")
				assert.True(t, delBranch.Force, "expected Force=true for -D flag")

				// Check message mentions "was not merged"
				msg, ok := plan.Actions[3].(PrintMessage)
				require.True(t, ok, "expected PrintMessage at index 3")
				assert.Contains(t, msg.Msg, "was not merged into main")
			},
		},
		{
			name: "branch not found prints skip message",
			ctx: RemoveContext{
				RepoRoot:     "/test/repo",
				SproutRoot:   "/test/repo/.sprout",
				TargetPath:   "/test/repo/.sprout/feature",
				DeleteBranch: true,
				BranchName:   "feature",
				BranchExists: false, // Branch doesn't exist
			},
			wantExit: false,
			assertions: func(t *testing.T, plan Plan) {
				// Should have: remove worktree, success msg, skip msg, prune
				require.Len(t, plan.Actions, 4)

				// Check skip message
				msg, ok := plan.Actions[2].(PrintMessage)
				require.True(t, ok, "expected PrintMessage at index 2")
				assert.Contains(t, msg.Msg, "not found")
				assert.Contains(t, msg.Msg, "skipping")
			},
		},
		{
			name: "--delete-remote without -d or -D fails",
			ctx: RemoveContext{
				RepoRoot:     "/test/repo",
				SproutRoot:   "/test/repo/.sprout",
				TargetPath:   "/test/repo/.sprout/feature",
				DeleteRemote: true,
				DeleteBranch: false,
				ForceDelete:  false,
			},
			wantExit:     true,
			wantExitCode: 1,
			assertions: func(t *testing.T, plan Plan) {
				errMsg, ok := plan.Actions[0].(PrintError)
				require.True(t, ok, "expected PrintError at index 0")
				assert.Contains(t, errMsg.Msg, "--delete-remote requires -d or -D")
			},
		},
		{
			name: "delete local and remote branch",
			ctx: RemoveContext{
				RepoRoot:           "/test/repo",
				SproutRoot:         "/test/repo/.sprout",
				TargetPath:         "/test/repo/.sprout/feature",
				DeleteBranch:       true,
				DeleteRemote:       true,
				BranchName:         "feature",
				BranchExists:       true,
				IsMerged:           true,
				RemoteName:         "origin",
				RemoteBranch:       "feature",
				RemoteBranchExists: true,
				HasUpstream:        true,
			},
			wantExit: false,
			assertions: func(t *testing.T, plan Plan) {
				// Should have: remove worktree, success msg, delete local, local msg, delete remote, remote msg, prune
				require.Len(t, plan.Actions, 7)

				// Check delete remote action
				delRemote, ok := plan.Actions[4].(DeleteRemoteBranch)
				require.True(t, ok, "expected DeleteRemoteBranch at index 4")
				assert.Equal(t, "origin", delRemote.Remote)
				assert.Equal(t, "feature", delRemote.RemoteBranch)
			},
		},
		{
			name: "delete remote with no upstream shows fallback message",
			ctx: RemoveContext{
				RepoRoot:           "/test/repo",
				SproutRoot:         "/test/repo/.sprout",
				TargetPath:         "/test/repo/.sprout/feature",
				DeleteBranch:       true,
				DeleteRemote:       true,
				BranchName:         "feature",
				BranchExists:       true,
				IsMerged:           true,
				RemoteName:         "origin",
				RemoteBranch:       "feature",
				RemoteBranchExists: true,
				HasUpstream:        false, // No upstream configured
			},
			wantExit: false,
			assertions: func(t *testing.T, plan Plan) {
				// Should show fallback message before deleting
				msg, ok := plan.Actions[4].(PrintMessage)
				require.True(t, ok, "expected PrintMessage at index 4")
				assert.Contains(t, msg.Msg, "No upstream configured")
				assert.Contains(t, msg.Msg, "assuming origin")
			},
		},
		{
			name: "remote branch not found prints skip message",
			ctx: RemoveContext{
				RepoRoot:           "/test/repo",
				SproutRoot:         "/test/repo/.sprout",
				TargetPath:         "/test/repo/.sprout/feature",
				DeleteBranch:       true,
				DeleteRemote:       true,
				BranchName:         "feature",
				BranchExists:       true,
				IsMerged:           true,
				RemoteName:         "origin",
				RemoteBranch:       "feature",
				RemoteBranchExists: false, // Remote doesn't exist
				HasUpstream:        true,
			},
			wantExit: false,
			assertions: func(t *testing.T, plan Plan) {
				// Should show skip message for remote
				found := false
				for _, action := range plan.Actions {
					if msg, ok := action.(PrintMessage); ok {
						if strings.Contains(msg.Msg, "not found") && strings.Contains(msg.Msg, "skipping") {
							found = true
							break
						}
					}
				}
				assert.True(t, found, "expected skip message for remote branch not found")
			},
		},
		{
			name: "unpushed commits with --delete-remote and -d fails",
			ctx: RemoveContext{
				RepoRoot:           "/test/repo",
				SproutRoot:         "/test/repo/.sprout",
				TargetPath:         "/test/repo/.sprout/feature",
				DeleteBranch:       true,
				DeleteRemote:       true,
				BranchName:         "feature",
				BranchExists:       true,
				IsMerged:           true,
				RemoteName:         "origin",
				RemoteBranch:       "feature",
				RemoteBranchExists: true,
				HasUpstream:        true,
				HasUnpushed:        true, // Unpushed commits!
			},
			wantExit:     true,
			wantExitCode: 1,
			assertions: func(t *testing.T, plan Plan) {
				// Should have error about unpushed commits somewhere in the plan
				found := false
				for _, action := range plan.Actions {
					if errMsg, ok := action.(PrintError); ok {
						if strings.Contains(errMsg.Msg, "unpushed commits") {
							found = true
							break
						}
					}
				}
				assert.True(t, found, "expected error about unpushed commits")
			},
		},
		{
			name: "unpushed commits with -D --delete-remote succeeds with warning",
			ctx: RemoveContext{
				RepoRoot:           "/test/repo",
				SproutRoot:         "/test/repo/.sprout",
				TargetPath:         "/test/repo/.sprout/feature",
				ForceDelete:        true,
				DeleteRemote:       true,
				BranchName:         "feature",
				BranchExists:       true,
				IsMerged:           false, // Not merged
				RemoteName:         "origin",
				RemoteBranch:       "feature",
				RemoteBranchExists: true,
				HasUpstream:        true,
				HasUnpushed:        true, // Unpushed commits
			},
			wantExit: false,
			assertions: func(t *testing.T, plan Plan) {
				// Should have warning message about unpushed commits
				found := false
				for _, action := range plan.Actions {
					if msg, ok := action.(PrintMessage); ok {
						if strings.Contains(msg.Msg, "Warning") || strings.Contains(msg.Msg, "unpushed") {
							found = true
							break
						}
					}
				}
				assert.True(t, found, "expected warning about unpushed commits")

				// Should still have DeleteRemoteBranch action
				hasDeleteRemote := false
				for _, action := range plan.Actions {
					if _, ok := action.(DeleteRemoteBranch); ok {
						hasDeleteRemote = true
						break
					}
				}
				assert.True(t, hasDeleteRemote, "expected DeleteRemoteBranch action despite unpushed commits with -D")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := PlanRemoveCommand(tt.ctx)

			// Check for Exit action anywhere in the plan
			if tt.wantExit {
				found := false
				for _, action := range plan.Actions {
					if exit, ok := action.(Exit); ok {
						assert.Equal(t, tt.wantExitCode, exit.Code)
						found = true
						break
					}
				}
				assert.True(t, found, "expected Exit action in plan")
			}

			// Run custom assertions
			if tt.assertions != nil {
				tt.assertions(t, plan)
			}
		})
	}
}

func TestPlanRemoveCommand_DryRun(t *testing.T) {
	tests := []struct {
		name       string
		ctx        RemoveContext
		assertions func(t *testing.T, plan Plan)
	}{
		{
			name: "dry run shows what would happen",
			ctx: RemoveContext{
				RepoRoot:   "/test/repo",
				SproutRoot: "/test/repo/.sprout",
				TargetPath: "/test/repo/.sprout/feature",
				DryRun:     true,
			},
			assertions: func(t *testing.T, plan Plan) {
				require.Len(t, plan.Actions, 1)
				msg, ok := plan.Actions[0].(PrintMessage)
				require.True(t, ok, "expected PrintMessage")
				assert.Contains(t, msg.Msg, "Would remove worktree")
			},
		},
		{
			name: "dry run with branch deletion",
			ctx: RemoveContext{
				RepoRoot:     "/test/repo",
				SproutRoot:   "/test/repo/.sprout",
				TargetPath:   "/test/repo/.sprout/feature",
				DryRun:       true,
				DeleteBranch: true,
				BranchName:   "feature",
				BranchExists: true,
			},
			assertions: func(t *testing.T, plan Plan) {
				require.Len(t, plan.Actions, 2)

				msg1 := plan.Actions[0].(PrintMessage)
				assert.Contains(t, msg1.Msg, "Would remove worktree")

				msg2 := plan.Actions[1].(PrintMessage)
				assert.Contains(t, msg2.Msg, "Would delete local branch")
			},
		},
		{
			name: "dry run with remote branch deletion",
			ctx: RemoveContext{
				RepoRoot:           "/test/repo",
				SproutRoot:         "/test/repo/.sprout",
				TargetPath:         "/test/repo/.sprout/feature",
				DryRun:             true,
				DeleteBranch:       true,
				DeleteRemote:       true,
				BranchName:         "feature",
				BranchExists:       true,
				RemoteName:         "origin",
				RemoteBranch:       "feature",
				RemoteBranchExists: true,
			},
			assertions: func(t *testing.T, plan Plan) {
				require.Len(t, plan.Actions, 3)

				msg3 := plan.Actions[2].(PrintMessage)
				assert.Contains(t, msg3.Msg, "Would delete remote branch")
			},
		},
		{
			name: "dry run shows branch not found",
			ctx: RemoveContext{
				RepoRoot:     "/test/repo",
				SproutRoot:   "/test/repo/.sprout",
				TargetPath:   "/test/repo/.sprout/feature",
				DryRun:       true,
				DeleteBranch: true,
				BranchName:   "feature",
				BranchExists: false,
			},
			assertions: func(t *testing.T, plan Plan) {
				require.Len(t, plan.Actions, 2)

				msg := plan.Actions[1].(PrintMessage)
				assert.Contains(t, msg.Msg, "not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := PlanRemoveCommand(tt.ctx)
			tt.assertions(t, plan)
		})
	}
}
