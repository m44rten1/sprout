package effects

import (
	"errors"
	"fmt"

	"github.com/m44rten1/sprout/internal/core"
)

// ExitError is returned when a plan includes an Exit action.
// The shell can check for this type and use the exit code.
type ExitError struct {
	Code int
}

func (e ExitError) Error() string {
	return fmt.Sprintf("exit code %d", e.Code)
}

// IsExit checks if an error is an ExitError and returns the code.
// This provides ergonomic error checking at call sites:
//
//	if code, ok := IsExit(err); ok {
//	    os.Exit(code)
//	}
func IsExit(err error) (code int, ok bool) {
	var exitErr ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code, true
	}
	return 0, false
}

// ExecutePlan executes all actions in a plan using the provided Effects.
// It stops and returns an error on the first failure (fail-fast semantics).
// If an Exit action is encountered, it returns an ExitError with the code.
func ExecutePlan(plan core.Plan, fx Effects) error {
	for _, action := range plan.Actions {
		if err := executeAction(action, fx); err != nil {
			return err
		}
	}
	return nil
}

// executeAction executes a single action using type switches.
// Returns an error if the action fails or encounters an Exit action.
func executeAction(action core.Action, fx Effects) error {
	switch a := action.(type) {
	case core.NoOp:
		// Do nothing
		return nil

	case core.PrintMessage:
		// Print is best-effort (does not fail on broken pipe)
		fx.Print(a.Msg)
		return nil

	case core.PrintError:
		// PrintErr is best-effort (does not fail on broken pipe)
		fx.PrintErr(a.Msg)
		return nil

	case core.CreateDirectory:
		if err := fx.MkdirAll(a.Path, a.Perm); err != nil {
			return fmt.Errorf("create directory %s: %w", a.Path, err)
		}
		return nil

	case core.RunGitCommand:
		// Note: Output is intentionally discarded here.
		// This executor handles "command for side-effect" git operations.
		// If you need to query git for data (e.g., parse branches), use Effects methods
		// like ListBranches() instead of RunGitCommand in the plan.
		_, err := fx.RunGitCommand(a.Dir, a.Args...)
		if err != nil {
			return fmt.Errorf("git command in %s failed: %w", a.Dir, err)
		}
		return nil

	case core.OpenEditor:
		if err := fx.OpenEditor(a.Path); err != nil {
			return fmt.Errorf("open editor for %s: %w", a.Path, err)
		}
		return nil

	case core.RunHooks:
		if err := fx.RunHooks(a.RepoRoot, a.Path, a.MainWorktreePath, a.Commands, string(a.Type)); err != nil {
			return fmt.Errorf("run %s hooks: %w", a.Type, err)
		}
		return nil

	case core.TrustRepo:
		if err := fx.TrustRepo(a.RepoRoot); err != nil {
			return fmt.Errorf("trust repo %s: %w", a.RepoRoot, err)
		}
		return nil

	case core.SelectInteractive:
		// SelectInteractive is a planning-time artifact, not an executable action.
		// Interactive selection should happen in the shell BEFORE plan generation.
		// If this appears in an execution plan, it indicates a bug in the planner.
		return fmt.Errorf("SelectInteractive should not appear in execution plans (selection belongs in shell)")

	case core.Exit:
		return ExitError{Code: a.Code}

	default:
		return fmt.Errorf("unknown action type: %T", action)
	}
}
