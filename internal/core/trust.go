package core

import (
	"fmt"
)

// TrustContext contains all inputs needed to plan the trust command.
type TrustContext struct {
	RepoRoot       string
	AlreadyTrusted bool
}

// PlanTrustCommand generates a plan for trusting a repository.
// It returns a plan with PrintMessage if already trusted,
// or TrustRepo + PrintMessage if trust needs to be added.
func PlanTrustCommand(ctx TrustContext) Plan {
	if ctx.RepoRoot == "" {
		return Plan{Actions: []Action{
			PrintError{Msg: ErrNoRepoRoot.Error()},
			Exit{Code: 1},
		}}
	}

	if ctx.AlreadyTrusted {
		return Plan{Actions: []Action{
			PrintMessage{Msg: fmt.Sprintf("✅ Repository is already trusted: %s", ctx.RepoRoot)},
		}}
	}

	successMsg := fmt.Sprintf(
		`✅ Repository trusted: %s

Hooks defined in .sprout.yml will now run automatically:
  - sprout add           (runs on_create hooks)
  - sprout open          (runs on_open hooks)

Use --no-hooks flag to skip automatic execution.
`, ctx.RepoRoot)

	return Plan{Actions: []Action{
		TrustRepo{RepoRoot: ctx.RepoRoot},
		PrintMessage{Msg: successMsg},
	}}
}

// PlanUntrustCommand generates a plan for untrusting a repository.
// It returns a plan with PrintMessage if not trusted,
// or UntrustRepo + PrintMessage if trust needs to be removed.
func PlanUntrustCommand(ctx TrustContext) Plan {
	if ctx.RepoRoot == "" {
		return Plan{Actions: []Action{
			PrintError{Msg: ErrNoRepoRoot.Error()},
			Exit{Code: 1},
		}}
	}

	if !ctx.AlreadyTrusted {
		return Plan{Actions: []Action{
			PrintMessage{Msg: fmt.Sprintf("ℹ️  Repository is not trusted: %s", ctx.RepoRoot)},
		}}
	}

	successMsg := fmt.Sprintf(
		`✅ Repository untrusted: %s

Hooks will no longer run automatically. You can still:
  - Create worktrees with: sprout add <branch> --no-hooks
  - Run 'sprout trust' again to re-enable hooks
`, ctx.RepoRoot)

	return Plan{Actions: []Action{
		UntrustRepo{RepoRoot: ctx.RepoRoot},
		PrintMessage{Msg: successMsg},
	}}
}
