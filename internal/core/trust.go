package core

const (
	msgRepoAlreadyTrusted = "Repository already trusted"
	msgRepoTrusted        = "Repository trusted"
	errNoRepoRoot         = "No repository root provided"
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
	// Validate inputs
	if ctx.RepoRoot == "" {
		return Plan{Actions: []Action{
			PrintError{Msg: errNoRepoRoot},
			Exit{Code: 1},
		}}
	}

	if ctx.AlreadyTrusted {
		return Plan{Actions: []Action{
			PrintMessage{Msg: msgRepoAlreadyTrusted},
		}}
	}

	return Plan{Actions: []Action{
		TrustRepo{Repo: ctx.RepoRoot},
		PrintMessage{Msg: msgRepoTrusted},
	}}
}
