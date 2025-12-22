package effects

import (
	"fmt"
	"os"
	"strings"

	"github.com/m44rten1/sprout/internal/config"
	"github.com/m44rten1/sprout/internal/git"
)

// TestEffects is a mock implementation of Effects for testing.
// It records all method calls and returns predefined values.
type TestEffects struct {
	// Predefined return values
	RepoRoot         string
	MainWorktreePath string
	Worktrees        []git.Worktree
	Branches         []git.Branch
	Config           *config.Config
	TrustedRepos     map[string]bool
	Files            map[string]bool   // Paths that "exist"
	GitCommandOutput map[string]string // Key: "dir\nargs..." -> output
	GitCommandErrors map[string]error  // Key: "dir\nargs..." -> error

	// Interaction results
	SelectedBranchIndex   int
	SelectedWorktreeIndex int
	SelectionError        error

	// Call counters (structured tracking)
	GetRepoRootCalls         int
	GetMainWorktreePathCalls int
	ListWorktreesCalls       int
	ListBranchesCalls        int
	RunGitCommandCalls       int
	FileExistsCalls          int
	MkdirAllCalls            int
	LoadConfigCalls          int
	IsTrustedCalls           int
	TrustRepoCalls           int
	OpenEditorCalls          int
	PrintCalls               int
	PrintErrCalls            int
	SelectBranchCalls        int
	SelectWorktreeCalls      int

	// Call tracking (captured side effects and arguments)
	ListWorktreesArgs     []string // repoRoot args passed to ListWorktrees
	ListBranchesArgs      []string // repoRoot args passed to ListBranches
	LoadConfigCurrentArgs []string // currentPath args passed to LoadConfig
	LoadConfigMainArgs    []string // mainPath args passed to LoadConfig
	IsTrustedArgs         []string // repoRoot args passed to IsTrusted
	TrustRepoRepos        []string // Repos that had TrustRepo called
	PrintedMsgs           []string // Messages printed via Print
	PrintedErrs           []string // Messages printed via PrintErr
	GitCommands           []GitCmd // Git commands executed
	OpenedPaths           []string // Paths opened in editor
	CreatedDirs           []string // Directories created via MkdirAll
}

// GitCmd represents a recorded git command execution.
type GitCmd struct {
	Dir  string
	Args []string
}

// NewTestEffects creates a new TestEffects with sensible defaults.
func NewTestEffects() *TestEffects {
	return &TestEffects{
		RepoRoot:              "/test/repo",
		MainWorktreePath:      "/test/repo",
		Worktrees:             []git.Worktree{},
		Branches:              []git.Branch{},
		Config:                &config.Config{},
		TrustedRepos:          make(map[string]bool),
		Files:                 make(map[string]bool),
		GitCommandOutput:      make(map[string]string),
		GitCommandErrors:      make(map[string]error),
		ListWorktreesArgs:     []string{},
		ListBranchesArgs:      []string{},
		LoadConfigCurrentArgs: []string{},
		LoadConfigMainArgs:    []string{},
		IsTrustedArgs:         []string{},
		TrustRepoRepos:        []string{},
		PrintedMsgs:           []string{},
		PrintedErrs:           []string{},
		GitCommands:           []GitCmd{},
		OpenedPaths:           []string{},
		CreatedDirs:           []string{},
	}
}

func (t *TestEffects) GetRepoRoot() (string, error) {
	t.GetRepoRootCalls++
	if t.RepoRoot == "" {
		return "", fmt.Errorf("not a git repo")
	}
	return t.RepoRoot, nil
}

func (t *TestEffects) GetMainWorktreePath() (string, error) {
	t.GetMainWorktreePathCalls++
	if t.MainWorktreePath == "" {
		return "", fmt.Errorf("no worktrees found")
	}
	return t.MainWorktreePath, nil
}

func (t *TestEffects) ListWorktrees(repoRoot string) ([]git.Worktree, error) {
	t.ListWorktreesCalls++
	t.ListWorktreesArgs = append(t.ListWorktreesArgs, repoRoot)
	return t.Worktrees, nil
}

func (t *TestEffects) ListBranches(repoRoot string) ([]git.Branch, error) {
	t.ListBranchesCalls++
	t.ListBranchesArgs = append(t.ListBranchesArgs, repoRoot)
	return t.Branches, nil
}

func (t *TestEffects) RunGitCommand(dir string, args ...string) (string, error) {
	t.RunGitCommandCalls++
	// Copy args to avoid slice aliasing bugs
	argsCopy := append([]string(nil), args...)
	t.GitCommands = append(t.GitCommands, GitCmd{Dir: dir, Args: argsCopy})

	// Look up predefined output/error by dir + command args
	key := dir + "\n" + strings.Join(argsCopy, " ")
	if err, exists := t.GitCommandErrors[key]; exists {
		return "", err
	}
	if output, exists := t.GitCommandOutput[key]; exists {
		return output, nil
	}

	// Default: success with empty output
	return "", nil
}

func (t *TestEffects) FileExists(path string) bool {
	t.FileExistsCalls++
	return t.Files[path]
}

func (t *TestEffects) MkdirAll(path string, perm os.FileMode) error {
	t.MkdirAllCalls++
	t.CreatedDirs = append(t.CreatedDirs, path)
	// Automatically mark directory as existing
	t.Files[path] = true
	return nil
}

func (t *TestEffects) LoadConfig(currentPath, mainPath string) (*config.Config, error) {
	t.LoadConfigCalls++
	t.LoadConfigCurrentArgs = append(t.LoadConfigCurrentArgs, currentPath)
	t.LoadConfigMainArgs = append(t.LoadConfigMainArgs, mainPath)
	if t.Config == nil {
		return &config.Config{}, nil
	}
	return t.Config, nil
}

func (t *TestEffects) IsTrusted(repoRoot string) (bool, error) {
	t.IsTrustedCalls++
	t.IsTrustedArgs = append(t.IsTrustedArgs, repoRoot)
	return t.TrustedRepos[repoRoot], nil
}

func (t *TestEffects) TrustRepo(repoRoot string) error {
	t.TrustRepoCalls++
	t.TrustRepoRepos = append(t.TrustRepoRepos, repoRoot)
	t.TrustedRepos[repoRoot] = true
	return nil
}

func (t *TestEffects) OpenEditor(path string) error {
	t.OpenEditorCalls++
	t.OpenedPaths = append(t.OpenedPaths, path)
	return nil
}

func (t *TestEffects) Print(msg string) {
	t.PrintCalls++
	t.PrintedMsgs = append(t.PrintedMsgs, msg)
}

func (t *TestEffects) PrintErr(msg string) {
	t.PrintErrCalls++
	t.PrintedErrs = append(t.PrintedErrs, msg)
}

func (t *TestEffects) SelectBranch(branches []git.Branch) (int, error) {
	t.SelectBranchCalls++
	if t.SelectionError != nil {
		return -1, t.SelectionError
	}
	if t.SelectedBranchIndex < 0 || t.SelectedBranchIndex >= len(branches) {
		return -1, fmt.Errorf("invalid selection index")
	}
	return t.SelectedBranchIndex, nil
}

func (t *TestEffects) SelectWorktree(worktrees []git.Worktree) (int, error) {
	t.SelectWorktreeCalls++
	if t.SelectionError != nil {
		return -1, t.SelectionError
	}
	if t.SelectedWorktreeIndex < 0 || t.SelectedWorktreeIndex >= len(worktrees) {
		return -1, fmt.Errorf("invalid selection index")
	}
	return t.SelectedWorktreeIndex, nil
}
