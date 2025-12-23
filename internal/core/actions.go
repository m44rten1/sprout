package core

import "os"

// Action represents a single operation to perform.
// This is a sum type implemented via interface for type safety.
type Action interface {
	isAction()
}

// NoOp represents a no-operation action (useful for conditional logic).
type NoOp struct{}

func (NoOp) isAction() {}

// PrintMessage prints a message to stdout.
type PrintMessage struct {
	Msg string
}

func (PrintMessage) isAction() {}

// PrintError prints an error message to stderr.
type PrintError struct {
	Msg string
}

func (PrintError) isAction() {}

// CreateDirectory creates a directory with the given permissions.
type CreateDirectory struct {
	Path string
	Perm os.FileMode
}

func (CreateDirectory) isAction() {}

// RunGitCommand executes a git command in the specified directory.
type RunGitCommand struct {
	Dir  string
	Args []string
}

func (RunGitCommand) isAction() {}

// OpenEditor opens the specified path in the user's editor.
type OpenEditor struct {
	Path string
}

func (OpenEditor) isAction() {}

// RunHooks executes hook commands in the specified directory.
type RunHooks struct {
	Type     string   // "on_create", "on_remove", etc.
	Commands []string // Shell commands to execute
	Path     string   // Working directory for hooks
}

func (RunHooks) isAction() {}

// TrustRepo marks a repository as trusted.
type TrustRepo struct {
	Repo string
}

func (TrustRepo) isAction() {}

// SelectInteractive represents an interactive selection (kept at edge).
// This might be split into specific types later (SelectBranch, SelectWorktree).
type SelectInteractive struct {
	Items       []any
	DisplayFunc func(any) string
}

func (SelectInteractive) isAction() {}

// Exit terminates the command with the specified exit code.
type Exit struct {
	Code int
}

func (Exit) isAction() {}

// Plan represents a sequence of actions to execute.
type Plan struct {
	Actions []Action
}
