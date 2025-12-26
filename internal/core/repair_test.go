package core_test

import (
	"testing"

	"github.com/m44rten1/sprout/internal/core"
	"github.com/stretchr/testify/assert"
)

func TestPlanRepair_NoRepos(t *testing.T) {
	ctx := core.RepairContext{Repos: []string{}}
	plan := core.PlanRepair(ctx)

	assert.Len(t, plan.Actions, 0, "empty context should produce empty plan")
}

func TestPlanRepair_SingleRepo(t *testing.T) {
	ctx := core.RepairContext{
		Repos: []string{"/path/to/repo"},
	}
	plan := core.PlanRepair(ctx)

	assert.Len(t, plan.Actions, 1)
	assert.IsType(t, core.RunGitCommand{}, plan.Actions[0], "action should be RunGitCommand")
	action := plan.Actions[0].(core.RunGitCommand)
	assert.Equal(t, "/path/to/repo", action.Dir)
	assert.Equal(t, []string{"worktree", "repair"}, action.Args)
}

func TestPlanRepair_MultipleRepos(t *testing.T) {
	ctx := core.RepairContext{
		Repos: []string{"/repo1", "/repo2", "/repo3"},
	}
	plan := core.PlanRepair(ctx)

	assert.Len(t, plan.Actions, 3)
	for i, repoPath := range ctx.Repos {
		assert.IsType(t, core.RunGitCommand{}, plan.Actions[i], "action %d should be RunGitCommand", i)
		action := plan.Actions[i].(core.RunGitCommand)
		assert.Equal(t, repoPath, action.Dir)
		assert.Equal(t, []string{"worktree", "repair"}, action.Args)
	}
}

func TestPlanRepair_Deterministic(t *testing.T) {
	ctx := core.RepairContext{
		Repos: []string{"/repo1", "/repo2"},
	}

	plan1 := core.PlanRepair(ctx)
	plan2 := core.PlanRepair(ctx)

	// Pure function: same input produces same output
	assert.Equal(t, plan1, plan2)
}
