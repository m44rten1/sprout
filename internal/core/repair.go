package core

// RepairContext contains repositories that may need worktree repair.
type RepairContext struct {
	// Repos are absolute paths to git repository roots that may need repair
	Repos []string
}

// PlanRepair creates a Plan that will run `git worktree repair`
// for all repositories in the context when executed.
// Deterministic: same input always produces the same Plan.
func PlanRepair(ctx RepairContext) Plan {
	if len(ctx.Repos) == 0 {
		return Plan{Actions: nil}
	}

	actions := make([]Action, 0, len(ctx.Repos))
	for _, repoPath := range ctx.Repos {
		actions = append(actions, RunGitCommand{
			Dir:  repoPath,
			Args: []string{"worktree", "repair"},
		})
	}

	return Plan{Actions: actions}
}
