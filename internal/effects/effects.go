package effects

import (
	"os"

	"github.com/m44rten1/sprout/internal/config"
	"github.com/m44rten1/sprout/internal/git"
)

// Effects defines all side effects that commands can perform.
// This interface enables testing by allowing mock implementations.
type Effects interface {
	// Git operations
	GetRepoRoot() (string, error)
	GetMainWorktreePath() (string, error)
	ListWorktrees(repoRoot string) ([]git.Worktree, error)
	ListBranches(repoRoot string) ([]git.Branch, error)
	RunGitCommand(dir string, args ...string) (string, error)

	// File system
	FileExists(path string) bool
	MkdirAll(path string, perm os.FileMode) error

	// Config
	LoadConfig(currentPath, mainPath string) (*config.Config, error)

	// Trust
	IsTrusted(repoRoot string) (bool, error)
	TrustRepo(repoRoot string) error
	UntrustRepo(repoRoot string) error
	// PromptTrustRepo prompts the user to trust a repository interactively.
	// Shows hooks that will run and asks for consent.
	// Returns error if stdin is not a terminal or user declined.
	PromptTrustRepo(mainWorktreePath, hookType string, hookCommands []string) error

	// Editor
	OpenEditor(path string) error

	// Output
	// Print and PrintErr are best-effort operations that write to stdout/stderr.
	// They do not return errors for broken pipes or other output failures.
	// If precise output handling is required, use a buffered writer with error checking.
	Print(msg string)
	PrintErr(msg string)

	// Interactive (kept at edge)
	// SelectOne displays items with custom formatting and returns selected index.
	// Items must be provided as []T where display converts T to string.
	// This maintains type safety while avoiding interface{} casting in callers.
	SelectBranch(branches []git.Branch) (int, error)
	SelectWorktree(worktrees []git.Worktree) (int, error)

	// Hooks
	// RunHooks executes hook commands in the given worktree.
	// RepoRoot and MainWorktreePath are used for trust verification.
	RunHooks(repoRoot, worktreePath, mainWorktreePath string, commands []string, hookType string) error

	// Branch existence checks
	LocalBranchExists(repoRoot, branch string) (bool, error)
	// RemoteBranchExists checks if a branch exists on the remote (automatically prepends "origin/")
	RemoteBranchExists(repoRoot, branch string) (bool, error)

	// Path calculation
	GetWorktreePath(repoPath, branch string) (string, error)

	// Sprout paths
	GetSproutRoot() (string, error)
	GetWorktreeRoot(repoRoot string) (string, error)

	// Filesystem (additional)
	ReadDir(path string) ([]os.DirEntry, error)
	UserHomeDir() (string, error)

	// Git status
	GetWorktreeStatus(path string) git.WorktreeStatus
}
