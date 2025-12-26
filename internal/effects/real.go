package effects

import (
	"fmt"
	"os"

	"github.com/m44rten1/sprout/internal/config"
	"github.com/m44rten1/sprout/internal/editor"
	"github.com/m44rten1/sprout/internal/git"
	"github.com/m44rten1/sprout/internal/hooks"
	"github.com/m44rten1/sprout/internal/sprout"
	"github.com/m44rten1/sprout/internal/trust"
	"github.com/m44rten1/sprout/internal/tui"
)

// RealEffects implements Effects by delegating to existing packages.
// This is the production implementation used by CLI commands.
type RealEffects struct{}

// NewRealEffects creates a new RealEffects instance.
func NewRealEffects() *RealEffects {
	return &RealEffects{}
}

func (r *RealEffects) GetRepoRoot() (string, error) {
	return git.GetRepoRoot()
}

func (r *RealEffects) GetMainWorktreePath() (string, error) {
	return git.GetMainWorktreePath()
}

func (r *RealEffects) ListWorktrees(repoRoot string) ([]git.Worktree, error) {
	return git.ListWorktrees(repoRoot)
}

func (r *RealEffects) ListBranches(repoRoot string) ([]git.Branch, error) {
	return git.ListAllBranches(repoRoot)
}

func (r *RealEffects) RunGitCommand(dir string, args ...string) (string, error) {
	return git.RunGitCommand(dir, args...)
}

// FileExists returns true only if the path exists.
// Returns false for permission errors, broken symlinks, and other stat failures.
func (r *RealEffects) FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func (r *RealEffects) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (r *RealEffects) LoadConfig(currentPath, mainPath string) (*config.Config, error) {
	return config.Load(currentPath, mainPath)
}

func (r *RealEffects) IsTrusted(repoRoot string) (bool, error) {
	return trust.IsRepoTrusted(repoRoot)
}

func (r *RealEffects) TrustRepo(repoRoot string) error {
	return trust.TrustRepo(repoRoot)
}

func (r *RealEffects) OpenEditor(path string) error {
	return editor.Open(path)
}

func (r *RealEffects) Print(msg string) {
	fmt.Println(msg)
}

func (r *RealEffects) PrintErr(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}

func (r *RealEffects) SelectBranch(branches []git.Branch) (int, error) {
	return tui.SelectOne(branches, branchLabel, nil)
}

func (r *RealEffects) SelectWorktree(worktrees []git.Worktree) (int, error) {
	return tui.SelectOne(worktrees, worktreeLabel, nil)
}

// branchLabel returns the display name for a branch.
func branchLabel(b git.Branch) string {
	return b.DisplayName
}

// worktreeLabel returns a display label for a worktree.
// Shows branch name if available, otherwise falls back to path (detached HEAD).
func worktreeLabel(w git.Worktree) string {
	if w.Branch != "" {
		return w.Branch
	}
	return w.Path
}

func (r *RealEffects) RunHooks(repoRoot, worktreePath, mainWorktreePath string, commands []string, hookType string) error {
	return hooks.RunHooks(repoRoot, worktreePath, mainWorktreePath, hooks.HookType(hookType))
}

func (r *RealEffects) LocalBranchExists(repoRoot, branch string) (bool, error) {
	return git.LocalBranchExists(repoRoot, branch)
}

func (r *RealEffects) RemoteBranchExists(repoRoot, branch string) (bool, error) {
	return git.BranchExists(repoRoot, "origin/"+branch)
}

func (r *RealEffects) GetWorktreePath(repoPath, branch string) (string, error) {
	return sprout.GetWorktreePath(repoPath, branch)
}

func (r *RealEffects) GetSproutRoot() (string, error) {
	return sprout.GetSproutRoot()
}

func (r *RealEffects) GetWorktreeRoot(repoRoot string) (string, error) {
	return sprout.GetWorktreeRoot(repoRoot)
}
