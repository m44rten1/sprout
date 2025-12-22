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

	// Editor
	OpenEditor(path string) error

	// Output
	Print(msg string)
	PrintErr(msg string)

	// Interactive (kept at edge)
	// SelectOne displays items with custom formatting and returns selected index.
	// Items must be provided as []T where display converts T to string.
	// This maintains type safety while avoiding interface{} casting in callers.
	SelectBranch(branches []git.Branch) (int, error)
	SelectWorktree(worktrees []git.Worktree) (int, error)
}
