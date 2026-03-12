package effects

import (
	"fmt"
	"os"
	"strings"

	"github.com/m44rten1/sprout/internal/config"
	"github.com/m44rten1/sprout/internal/core"
	"github.com/m44rten1/sprout/internal/git"
)

// TestEffects is a mock implementation of Effects for testing.
// It records all method calls and returns predefined values.
type TestEffects struct {
	// Predefined return values
	RepoRoot         string
	MainWorktreePath string
	CurrentBranch    string
	Worktrees        []git.Worktree
	Branches         []git.Branch
	Config           *config.Config
	TrustedRepos     map[string]bool
	Files            map[string]bool   // Paths that "exist"
	GitCommandOutput map[string]string // Key: "dir\nargs..." -> output
	GitCommandErrors map[string]error  // Key: "dir\nargs..." -> error
	DirtyWorktrees   map[string]bool   // path -> dirty state
	CreatedStashes   map[string]string // path -> stash OID
	AppliedStashes   map[string]string // path -> stash OID

	// Branch existence mocking
	LocalBranches  map[string]bool // branch name -> exists locally
	RemoteBranches map[string]bool // branch name -> exists on remote

	// Branch operations mocking
	DefaultBranch             string
	BranchMergedIntoMain      map[string]bool               // branch -> is merged into main
	BranchUpstreams           map[string]git.BranchUpstream // branch -> upstream info
	BranchHasUnpushedCommits  map[string]bool               // branch -> has unpushed commits
	GetDefaultBranchErr       error
	IsBranchMergedIntoMainErr error
	HasUnpushedCommitsErr     error
	DeleteLocalBranchErr      error
	DeleteRemoteBranchErr     error
	RemoteBranchExistsOnMap   map[string]bool // "remote/branch" -> exists
	RemoteBranchExistsOnErr   error

	// Worktree path calculation
	WorktreePaths      map[string]string // branch -> path mapping
	GetWorktreePathErr error

	// Sprout paths
	SproutRoot         string
	WorktreeRoot       string
	GetSproutRootErr   error
	GetWorktreeRootErr error

	// Filesystem (additional)
	DirEntries       map[string][]os.DirEntry // path -> entries
	UserHome         string
	WorktreeStatuses map[string]git.WorktreeStatus // path -> status

	// Error injection - set these to simulate failures
	GetRepoRootErr         error
	GetMainWorktreePathErr error
	GetCurrentBranchErr    error
	IsWorktreeDirtyErr     error
	ListWorktreesErr       error
	ListBranchesErr        error
	MkdirAllErr            error
	LoadConfigErr          error
	IsTrustedErr           error
	TrustRepoErr           error
	UntrustRepoErr         error
	OpenEditorErr          error
	RunHooksErr            error
	LocalBranchExistsErr   error
	RemoteBranchExistsErr  error
	PromptTrustRepoErr     error
	ReadDirErr             error
	UserHomeDirErr         error

	// Interaction results
	SelectedBranchIndex     int
	SelectedFromBranchIndex int
	SelectedWorktreeIndex   int
	SelectionError          error
	SelectFromBranchError   error

	// Call counters (structured tracking)
	GetRepoRootCalls            int
	GetMainWorktreePathCalls    int
	GetCurrentBranchCalls       int
	IsWorktreeDirtyCalls        int
	ListWorktreesCalls          int
	ListBranchesCalls           int
	RunGitCommandCalls          int
	CreateStashCalls            int
	ApplyStashCalls             int
	DropStashCalls              int
	FileExistsCalls             int
	MkdirAllCalls               int
	LoadConfigCalls             int
	IsTrustedCalls              int
	TrustRepoCalls              int
	UntrustRepoCalls            int
	OpenEditorCalls             int
	PrintCalls                  int
	PrintErrCalls               int
	SelectBranchCalls           int
	SelectFromBranchCalls       int
	SelectWorktreeCalls         int
	RunHooksCalls               int
	LocalBranchExistsCalls      int
	RemoteBranchExistsCalls     int
	GetWorktreePathCalls        int
	GetSproutRootCalls          int
	GetWorktreeRootCalls        int
	PromptTrustRepoCalls        int
	ReadDirCalls                int
	UserHomeDirCalls            int
	GetWorktreeStatusCalls      int
	GetDefaultBranchCalls       int
	IsBranchMergedIntoMainCalls int
	GetBranchUpstreamCalls      int
	HasUnpushedCommitsCalls     int
	DeleteLocalBranchCallCount  int
	DeleteRemoteBranchCallCount int
	RemoteBranchExistsOnCalls   int

	// Call tracking (captured side effects and arguments)
	ListWorktreesArgs          []string   // repoRoot args passed to ListWorktrees
	ListBranchesArgs           []string   // repoRoot args passed to ListBranches
	LoadConfigCurrentArgs      []string   // currentPath args passed to LoadConfig
	LoadConfigMainArgs         []string   // mainPath args passed to LoadConfig
	IsTrustedArgs              []string   // repoRoot args passed to IsTrusted
	IsWorktreeDirtyArgs        []string   // path args passed to IsWorktreeDirty
	TrustRepoRepos             []string   // Repos that had TrustRepo called
	UntrustRepoRepos           []string   // Repos that had UntrustRepo called
	PrintedMsgs                []string   // Messages printed via Print
	PrintedErrs                []string   // Messages printed via PrintErr
	GitCommands                []GitCmd   // Git commands executed
	OpenedPaths                []string   // Paths opened in editor
	CreatedDirs                []string   // Directories created via MkdirAll
	RunHooksInvocations        []HookCall // Hooks that were run
	LocalBranchExistsQueries   []BranchQuery
	RemoteBranchExistsQueries  []BranchQuery
	GetWorktreePathQueries     []WorktreePathQuery
	GetWorktreeRootArgs        []string // repoRoot args passed to GetWorktreeRoot
	PromptTrustRepoInvocations []PromptTrustCall
	ReadDirArgs                []string // path args passed to ReadDir
	GetWorktreeStatusArgs      []string // path args passed to GetWorktreeStatus
	CreateStashInvocations     []CreateStashCall
	ApplyStashInvocations      []StashCall
	DropStashInvocations       []StashCall

	// Branch operations tracking
	GetDefaultBranchQueries       []string
	IsBranchMergedIntoMainQueries []BranchQuery
	GetBranchUpstreamQueries      []BranchQuery
	HasUnpushedCommitsQueries     []UnpushedCommitsQuery
	DeleteLocalBranchCalls        []DeleteLocalBranchCall
	DeleteRemoteBranchCalls       []DeleteRemoteBranchCall
	RemoteBranchExistsOnQueries   []RemoteBranchExistsOnQuery
}

// UnpushedCommitsQuery represents a HasUnpushedCommits check.
type UnpushedCommitsQuery struct {
	RepoRoot     string
	Branch       string
	Remote       string
	RemoteBranch string
}

// DeleteLocalBranchCall represents a local branch deletion.
type DeleteLocalBranchCall struct {
	RepoRoot string
	Branch   string
	Force    bool
}

// DeleteRemoteBranchCall represents a remote branch deletion.
type DeleteRemoteBranchCall struct {
	RepoRoot string
	Remote   string
	Branch   string
}

// RemoteBranchExistsOnQuery represents a remote branch existence check.
type RemoteBranchExistsOnQuery struct {
	RepoRoot string
	Remote   string
	Branch   string
}

// GitCmd represents a recorded git command execution.
type GitCmd struct {
	Dir  string
	Args []string
}

// HookCall represents a recorded hook execution.
type HookCall struct {
	RepoRoot         string
	WorktreePath     string
	MainWorktreePath string
	Commands         []string
	HookType         core.HookType
}

// BranchQuery represents a branch existence check.
type BranchQuery struct {
	RepoRoot string
	Branch   string
}

// WorktreePathQuery represents a worktree path calculation.
type WorktreePathQuery struct {
	RepoPath string
	Branch   string
}

// PromptTrustCall represents a trust prompt invocation.
type PromptTrustCall struct {
	MainWorktreePath string
	HookType         string
	HookCommands     []string
}

// CreateStashCall represents a stash creation invocation.
type CreateStashCall struct {
	Path             string
	Message          string
	IncludeUntracked bool
}

// StashCall represents applying or dropping a stash.
type StashCall struct {
	Path     string
	StashOID string
}

// NewTestEffects creates a new TestEffects with sensible defaults.
func NewTestEffects() *TestEffects {
	return &TestEffects{
		RepoRoot:                   "/test/repo",
		MainWorktreePath:           "/test/repo",
		Worktrees:                  []git.Worktree{},
		Branches:                   []git.Branch{},
		Config:                     &config.Config{},
		TrustedRepos:               make(map[string]bool),
		Files:                      make(map[string]bool),
		GitCommandOutput:           make(map[string]string),
		GitCommandErrors:           make(map[string]error),
		DirtyWorktrees:             make(map[string]bool),
		CreatedStashes:             make(map[string]string),
		AppliedStashes:             make(map[string]string),
		LocalBranches:              make(map[string]bool),
		RemoteBranches:             make(map[string]bool),
		WorktreePaths:              make(map[string]string),
		ListWorktreesArgs:          []string{},
		ListBranchesArgs:           []string{},
		LoadConfigCurrentArgs:      []string{},
		LoadConfigMainArgs:         []string{},
		IsTrustedArgs:              []string{},
		IsWorktreeDirtyArgs:        []string{},
		TrustRepoRepos:             []string{},
		PrintedMsgs:                []string{},
		PrintedErrs:                []string{},
		GitCommands:                []GitCmd{},
		OpenedPaths:                []string{},
		CreatedDirs:                []string{},
		RunHooksInvocations:        []HookCall{},
		LocalBranchExistsQueries:   []BranchQuery{},
		RemoteBranchExistsQueries:  []BranchQuery{},
		GetWorktreePathQueries:     []WorktreePathQuery{},
		GetWorktreeRootArgs:        []string{},
		PromptTrustRepoInvocations: []PromptTrustCall{},
		CreateStashInvocations:     []CreateStashCall{},
		ApplyStashInvocations:      []StashCall{},
		DropStashInvocations:       []StashCall{},
		SproutRoot:                 "/home/user/.local/share/sprout",
		WorktreeRoot:               "/home/user/.local/share/sprout/test-12345678",
		DirEntries:                 make(map[string][]os.DirEntry),
		UserHome:                   "/home/user",
		WorktreeStatuses:           make(map[string]git.WorktreeStatus),
		ReadDirArgs:                []string{},
		GetWorktreeStatusArgs:      []string{},
		// Branch operations
		DefaultBranch:                 "main",
		BranchMergedIntoMain:          make(map[string]bool),
		BranchUpstreams:               make(map[string]git.BranchUpstream),
		BranchHasUnpushedCommits:      make(map[string]bool),
		RemoteBranchExistsOnMap:       make(map[string]bool),
		GetDefaultBranchQueries:       []string{},
		IsBranchMergedIntoMainQueries: []BranchQuery{},
		GetBranchUpstreamQueries:      []BranchQuery{},
		HasUnpushedCommitsQueries:     []UnpushedCommitsQuery{},
		DeleteLocalBranchCalls:        []DeleteLocalBranchCall{},
		DeleteRemoteBranchCalls:       []DeleteRemoteBranchCall{},
		RemoteBranchExistsOnQueries:   []RemoteBranchExistsOnQuery{},
	}
}

func (t *TestEffects) GetRepoRoot() (string, error) {
	t.GetRepoRootCalls++
	if t.GetRepoRootErr != nil {
		return "", t.GetRepoRootErr
	}
	if t.RepoRoot == "" {
		return "", fmt.Errorf("not a git repo")
	}
	return t.RepoRoot, nil
}

func (t *TestEffects) GetMainWorktreePath() (string, error) {
	t.GetMainWorktreePathCalls++
	if t.GetMainWorktreePathErr != nil {
		return "", t.GetMainWorktreePathErr
	}
	if t.MainWorktreePath == "" {
		return "", fmt.Errorf("no worktrees found")
	}
	return t.MainWorktreePath, nil
}

func (t *TestEffects) GetCurrentBranch(repoRoot string) (string, error) {
	t.GetCurrentBranchCalls++
	if t.GetCurrentBranchErr != nil {
		return "", t.GetCurrentBranchErr
	}
	if t.CurrentBranch == "" {
		return "", fmt.Errorf("not on any branch (detached HEAD)")
	}
	return t.CurrentBranch, nil
}

func (t *TestEffects) IsWorktreeDirty(path string) (bool, error) {
	t.IsWorktreeDirtyCalls++
	t.IsWorktreeDirtyArgs = append(t.IsWorktreeDirtyArgs, path)
	if t.IsWorktreeDirtyErr != nil {
		return false, t.IsWorktreeDirtyErr
	}
	return t.DirtyWorktrees[path], nil
}

func (t *TestEffects) ListWorktrees(repoRoot string) ([]git.Worktree, error) {
	t.ListWorktreesCalls++
	t.ListWorktreesArgs = append(t.ListWorktreesArgs, repoRoot)
	if t.ListWorktreesErr != nil {
		return nil, t.ListWorktreesErr
	}
	return t.Worktrees, nil
}

func (t *TestEffects) ListBranches(repoRoot string) ([]git.Branch, error) {
	t.ListBranchesCalls++
	t.ListBranchesArgs = append(t.ListBranchesArgs, repoRoot)
	if t.ListBranchesErr != nil {
		return nil, t.ListBranchesErr
	}
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

func (t *TestEffects) CreateStash(path, message string, includeUntracked bool) (string, error) {
	t.CreateStashCalls++
	t.CreateStashInvocations = append(t.CreateStashInvocations, CreateStashCall{
		Path:             path,
		Message:          message,
		IncludeUntracked: includeUntracked,
	})

	key := path + "\nstash-create"
	if err, exists := t.GitCommandErrors[key]; exists {
		return "", err
	}

	stashOID := t.CreatedStashes[path]
	if stashOID == "" {
		stashOID = "stash@mock"
	}

	t.DirtyWorktrees[path] = false
	return stashOID, nil
}

func (t *TestEffects) ApplyStash(path, stashOID string) error {
	t.ApplyStashCalls++
	t.ApplyStashInvocations = append(t.ApplyStashInvocations, StashCall{
		Path:     path,
		StashOID: stashOID,
	})

	key := path + "\nstash-apply " + stashOID
	if err, exists := t.GitCommandErrors[key]; exists {
		return err
	}

	t.AppliedStashes[path] = stashOID
	t.DirtyWorktrees[path] = true
	return nil
}

func (t *TestEffects) DropStash(path, stashOID string) error {
	t.DropStashCalls++
	t.DropStashInvocations = append(t.DropStashInvocations, StashCall{
		Path:     path,
		StashOID: stashOID,
	})

	key := path + "\nstash-drop " + stashOID
	if err, exists := t.GitCommandErrors[key]; exists {
		return err
	}

	delete(t.CreatedStashes, path)
	return nil
}

func (t *TestEffects) FileExists(path string) bool {
	t.FileExistsCalls++
	return t.Files[path]
}

func (t *TestEffects) MkdirAll(path string, perm os.FileMode) error {
	t.MkdirAllCalls++
	t.CreatedDirs = append(t.CreatedDirs, path)
	if t.MkdirAllErr != nil {
		return t.MkdirAllErr
	}
	// Automatically mark directory as existing
	t.Files[path] = true
	return nil
}

func (t *TestEffects) LoadConfig(currentPath, mainPath string) (*config.Config, error) {
	t.LoadConfigCalls++
	t.LoadConfigCurrentArgs = append(t.LoadConfigCurrentArgs, currentPath)
	t.LoadConfigMainArgs = append(t.LoadConfigMainArgs, mainPath)
	if t.LoadConfigErr != nil {
		return nil, t.LoadConfigErr
	}
	if t.Config == nil {
		return &config.Config{}, nil
	}
	return t.Config, nil
}

func (t *TestEffects) IsTrusted(repoRoot string) (bool, error) {
	t.IsTrustedCalls++
	t.IsTrustedArgs = append(t.IsTrustedArgs, repoRoot)
	if t.IsTrustedErr != nil {
		return false, t.IsTrustedErr
	}
	return t.TrustedRepos[repoRoot], nil
}

func (t *TestEffects) TrustRepo(repoRoot string) error {
	t.TrustRepoCalls++
	t.TrustRepoRepos = append(t.TrustRepoRepos, repoRoot)
	if t.TrustRepoErr != nil {
		return t.TrustRepoErr
	}
	t.TrustedRepos[repoRoot] = true
	return nil
}

func (t *TestEffects) UntrustRepo(repoRoot string) error {
	t.UntrustRepoCalls++
	t.UntrustRepoRepos = append(t.UntrustRepoRepos, repoRoot)
	if t.UntrustRepoErr != nil {
		return t.UntrustRepoErr
	}
	delete(t.TrustedRepos, repoRoot)
	return nil
}

func (t *TestEffects) OpenEditor(path string) error {
	t.OpenEditorCalls++
	t.OpenedPaths = append(t.OpenedPaths, path)
	if t.OpenEditorErr != nil {
		return t.OpenEditorErr
	}
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

func (t *TestEffects) SelectFromBranch(branches []git.Branch) (int, error) {
	t.SelectFromBranchCalls++
	if t.SelectFromBranchError != nil {
		return -1, t.SelectFromBranchError
	}
	if t.SelectedFromBranchIndex < 0 || t.SelectedFromBranchIndex >= len(branches) {
		return -1, fmt.Errorf("invalid selection index")
	}
	return t.SelectedFromBranchIndex, nil
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

func (t *TestEffects) RunHooks(repoRoot, worktreePath, mainWorktreePath string, commands []string, hookType string) error {
	t.RunHooksCalls++
	t.RunHooksInvocations = append(t.RunHooksInvocations, HookCall{
		RepoRoot:         repoRoot,
		WorktreePath:     worktreePath,
		MainWorktreePath: mainWorktreePath,
		Commands:         commands,
		HookType:         core.HookType(hookType),
	})
	return t.RunHooksErr
}

func (t *TestEffects) LocalBranchExists(repoRoot, branch string) (bool, error) {
	t.LocalBranchExistsCalls++
	t.LocalBranchExistsQueries = append(t.LocalBranchExistsQueries, BranchQuery{
		RepoRoot: repoRoot,
		Branch:   branch,
	})
	if t.LocalBranchExistsErr != nil {
		return false, t.LocalBranchExistsErr
	}
	return t.LocalBranches[branch], nil
}

func (t *TestEffects) RemoteBranchExists(repoRoot, branch string) (bool, error) {
	t.RemoteBranchExistsCalls++
	t.RemoteBranchExistsQueries = append(t.RemoteBranchExistsQueries, BranchQuery{
		RepoRoot: repoRoot,
		Branch:   branch,
	})
	if t.RemoteBranchExistsErr != nil {
		return false, t.RemoteBranchExistsErr
	}
	return t.RemoteBranches[branch], nil
}

func (t *TestEffects) GetWorktreePath(repoPath, branch string) (string, error) {
	t.GetWorktreePathCalls++
	t.GetWorktreePathQueries = append(t.GetWorktreePathQueries, WorktreePathQuery{
		RepoPath: repoPath,
		Branch:   branch,
	})
	if t.GetWorktreePathErr != nil {
		return "", t.GetWorktreePathErr
	}
	if path, ok := t.WorktreePaths[branch]; ok {
		return path, nil
	}
	// Default: generate a simple path
	return fmt.Sprintf("%s/worktrees/%s", repoPath, branch), nil
}

func (t *TestEffects) GetSproutRoot() (string, error) {
	t.GetSproutRootCalls++
	if t.GetSproutRootErr != nil {
		return "", t.GetSproutRootErr
	}
	if t.SproutRoot == "" {
		return "", fmt.Errorf("failed to get sprout root")
	}
	return t.SproutRoot, nil
}

func (t *TestEffects) GetWorktreeRoot(repoRoot string) (string, error) {
	t.GetWorktreeRootCalls++
	t.GetWorktreeRootArgs = append(t.GetWorktreeRootArgs, repoRoot)
	if t.GetWorktreeRootErr != nil {
		return "", t.GetWorktreeRootErr
	}
	if t.WorktreeRoot == "" {
		return "", fmt.Errorf("failed to get worktree root")
	}
	return t.WorktreeRoot, nil
}

func (t *TestEffects) PromptTrustRepo(mainWorktreePath, hookType string, hookCommands []string) error {
	t.PromptTrustRepoCalls++
	t.PromptTrustRepoInvocations = append(t.PromptTrustRepoInvocations, PromptTrustCall{
		MainWorktreePath: mainWorktreePath,
		HookType:         hookType,
		HookCommands:     hookCommands,
	})
	if t.PromptTrustRepoErr != nil {
		return t.PromptTrustRepoErr
	}
	// Auto-trust on success (simulates user saying yes)
	t.TrustedRepos[mainWorktreePath] = true
	return nil
}

func (t *TestEffects) ReadDir(path string) ([]os.DirEntry, error) {
	t.ReadDirCalls++
	t.ReadDirArgs = append(t.ReadDirArgs, path)
	if t.ReadDirErr != nil {
		return nil, t.ReadDirErr
	}
	if entries, ok := t.DirEntries[path]; ok {
		return entries, nil
	}
	// Default: empty directory
	return []os.DirEntry{}, nil
}

func (t *TestEffects) UserHomeDir() (string, error) {
	t.UserHomeDirCalls++
	if t.UserHomeDirErr != nil {
		return "", t.UserHomeDirErr
	}
	if t.UserHome == "" {
		return "", fmt.Errorf("failed to get home directory")
	}
	return t.UserHome, nil
}

func (t *TestEffects) GetWorktreeStatus(path string) git.WorktreeStatus {
	t.GetWorktreeStatusCalls++
	t.GetWorktreeStatusArgs = append(t.GetWorktreeStatusArgs, path)
	if status, ok := t.WorktreeStatuses[path]; ok {
		return status
	}
	// Default: clean worktree
	return git.WorktreeStatus{}
}

func (t *TestEffects) GetDefaultBranch(repoRoot string) (string, error) {
	t.GetDefaultBranchCalls++
	t.GetDefaultBranchQueries = append(t.GetDefaultBranchQueries, repoRoot)
	if t.GetDefaultBranchErr != nil {
		return "", t.GetDefaultBranchErr
	}
	if t.DefaultBranch == "" {
		return "", fmt.Errorf("could not determine default branch")
	}
	return t.DefaultBranch, nil
}

func (t *TestEffects) IsBranchMergedIntoMain(repoRoot, branch string) (bool, error) {
	t.IsBranchMergedIntoMainCalls++
	t.IsBranchMergedIntoMainQueries = append(t.IsBranchMergedIntoMainQueries, BranchQuery{
		RepoRoot: repoRoot,
		Branch:   branch,
	})
	if t.IsBranchMergedIntoMainErr != nil {
		return false, t.IsBranchMergedIntoMainErr
	}
	return t.BranchMergedIntoMain[branch], nil
}

func (t *TestEffects) GetBranchUpstream(repoRoot, branch string) git.BranchUpstream {
	t.GetBranchUpstreamCalls++
	t.GetBranchUpstreamQueries = append(t.GetBranchUpstreamQueries, BranchQuery{
		RepoRoot: repoRoot,
		Branch:   branch,
	})
	if upstream, ok := t.BranchUpstreams[branch]; ok {
		return upstream
	}
	// Default: origin fallback
	return git.BranchUpstream{
		Remote:       "origin",
		RemoteBranch: branch,
		Configured:   false,
	}
}

func (t *TestEffects) HasUnpushedCommits(repoRoot, branch, remote, remoteBranch string) (bool, error) {
	t.HasUnpushedCommitsCalls++
	t.HasUnpushedCommitsQueries = append(t.HasUnpushedCommitsQueries, UnpushedCommitsQuery{
		RepoRoot:     repoRoot,
		Branch:       branch,
		Remote:       remote,
		RemoteBranch: remoteBranch,
	})
	if t.HasUnpushedCommitsErr != nil {
		return false, t.HasUnpushedCommitsErr
	}
	return t.BranchHasUnpushedCommits[branch], nil
}

func (t *TestEffects) DeleteLocalBranch(repoRoot, branch string, force bool) error {
	t.DeleteLocalBranchCallCount++
	t.DeleteLocalBranchCalls = append(t.DeleteLocalBranchCalls, DeleteLocalBranchCall{
		RepoRoot: repoRoot,
		Branch:   branch,
		Force:    force,
	})
	if t.DeleteLocalBranchErr != nil {
		return t.DeleteLocalBranchErr
	}
	// Remove from local branches
	delete(t.LocalBranches, branch)
	return nil
}

func (t *TestEffects) DeleteRemoteBranch(repoRoot, remote, branch string) error {
	t.DeleteRemoteBranchCallCount++
	t.DeleteRemoteBranchCalls = append(t.DeleteRemoteBranchCalls, DeleteRemoteBranchCall{
		RepoRoot: repoRoot,
		Remote:   remote,
		Branch:   branch,
	})
	if t.DeleteRemoteBranchErr != nil {
		return t.DeleteRemoteBranchErr
	}
	// Remove from remote branches
	delete(t.RemoteBranchExistsOnMap, remote+"/"+branch)
	return nil
}

func (t *TestEffects) RemoteBranchExistsOn(repoRoot, remote, branch string) (bool, error) {
	t.RemoteBranchExistsOnCalls++
	t.RemoteBranchExistsOnQueries = append(t.RemoteBranchExistsOnQueries, RemoteBranchExistsOnQuery{
		RepoRoot: repoRoot,
		Remote:   remote,
		Branch:   branch,
	})
	if t.RemoteBranchExistsOnErr != nil {
		return false, t.RemoteBranchExistsOnErr
	}
	return t.RemoteBranchExistsOnMap[remote+"/"+branch], nil
}
