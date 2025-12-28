package core

import (
	"errors"
	"os"
)

// HookType represents the type of hook to execute.
type HookType string

// Hook type constants for RunHooks action
const (
	HookTypeOnCreate HookType = "on_create"
	HookTypeOnOpen   HookType = "on_open"
)

// Common error variables used across commands
var (
	ErrNoRepoRoot            = errors.New("no repository root provided")
	ErrEmptyRepoRoot         = ErrNoRepoRoot // Alias for consistency across commands
	ErrEmptyTargetPath       = errors.New("target path cannot be empty")
	ErrEmptyWorktreePath     = errors.New("worktree path cannot be empty")
	ErrEmptyMainWorktreePath = errors.New("main worktree path cannot be empty")
	ErrEmptyBranch           = errors.New("branch name cannot be empty")
	ErrNilConfig             = errors.New("config must not be nil")
	ErrUntrustedWithHooks    = errors.New("Repository not trusted. Cannot run hooks.\n\nThis repository has hooks defined that would run automatically.\nTo allow these hooks, run:\n  sprout trust\n\nTo skip hooks this time:\n  Add the --no-hooks flag")
	ErrNoSproutWorktrees     = errors.New("no sprout-managed worktrees found")
	ErrSelectionCancelled    = errors.New("selection cancelled")
)

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
	Type             HookType // HookTypeOnCreate, HookTypeOnOpen, etc.
	Commands         []string // Shell commands to execute
	Path             string   // Working directory for hooks (worktree path)
	RepoRoot         string   // Repository root
	MainWorktreePath string   // Main worktree path (for trust checks)
}

func (RunHooks) isAction() {}

// PromptTrust prompts the user to trust a repository interactively.
// Shows hooks that would run and asks for consent.
type PromptTrust struct {
	MainWorktreePath string   // Main worktree path (trust key)
	HookType         HookType // Type of hooks that would run
	HookCommands     []string // Commands that would be executed
}

func (PromptTrust) isAction() {}

// TrustRepo marks a repository as trusted.
type TrustRepo struct {
	RepoRoot string
}

func (TrustRepo) isAction() {}

// UntrustRepo removes trust from a repository.
type UntrustRepo struct {
	RepoRoot string
}

func (UntrustRepo) isAction() {}

// SelectInteractive represents an interactive selection.
// Note: Uses 'any' for flexibility, but this is intentionally "edge-only" - not
// executed by the standard effects executor. Interactive prompts are handled in
// the imperative shell (cmd layer) before planning, not as part of plan execution.
// If this pattern becomes more common, consider concrete types like SelectWorktree.
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
