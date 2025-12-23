package core

import (
	"errors"
	"fmt"
)

var ErrNoRepoRoot = errors.New("no repository root provided")

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
