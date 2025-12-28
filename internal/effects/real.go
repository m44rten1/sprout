package effects

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/m44rten1/sprout/internal/config"
	"github.com/m44rten1/sprout/internal/editor"
	"github.com/m44rten1/sprout/internal/git"
	"github.com/m44rten1/sprout/internal/hooks"
	"github.com/m44rten1/sprout/internal/sprout"
	"github.com/m44rten1/sprout/internal/trust"
	"github.com/m44rten1/sprout/internal/tui"
	"golang.org/x/term"
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

func (r *RealEffects) UntrustRepo(repoRoot string) error {
	return trust.UntrustRepo(repoRoot)
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
	// Pre-compute statuses for all worktrees in parallel
	statuses := make([]git.WorktreeStatus, len(worktrees))
	var wg sync.WaitGroup
	for i, wt := range worktrees {
		wg.Add(1)
		go func(idx int, path string) {
			defer wg.Done()
			statuses[idx] = git.GetWorktreeStatus(path)
		}(i, wt.Path)
	}
	wg.Wait()

	// Create label function with pre-computed statuses
	labelFunc := func(w git.Worktree) string {
		// Find index of this worktree to get its status
		for i, wt := range worktrees {
			if wt.Path == w.Path {
				return worktreeLabelWithStatus(w, statuses[i])
			}
		}
		return worktreeLabel(w)
	}

	return tui.SelectOne(worktrees, labelFunc, nil)
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

// worktreeLabelWithStatus returns a display label for a worktree with status icons.
func worktreeLabelWithStatus(w git.Worktree, status git.WorktreeStatus) string {
	label := worktreeLabel(w)
	statusIcons := buildPlainStatusIcons(status)
	if statusIcons != "" {
		return label + " " + statusIcons
	}
	return label
}

// buildPlainStatusIcons builds status icons without ANSI color codes.
// Used for fuzzy finder which doesn't support ANSI escapes.
func buildPlainStatusIcons(status git.WorktreeStatus) string {
	var icons []string
	if status.Dirty {
		icons = append(icons, "✗")
	}
	if status.Ahead > 0 {
		icons = append(icons, "↑")
	}
	if status.Behind > 0 {
		icons = append(icons, "↓")
	}
	if status.Unmerged {
		icons = append(icons, "↕")
	}
	return strings.Join(icons, " ")
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

func (r *RealEffects) PromptTrustRepo(mainWorktreePath, hookType string, hookCommands []string) error {
	// Check if stdin is a terminal (interactive mode)
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		// Not a terminal - return error with helpful guidance for non-interactive environments
		var guidance strings.Builder
		guidance.WriteString("\nRepository has hooks but is not trusted.\n\n")
		guidance.WriteString(fmt.Sprintf("Hooks that would run on '%s':\n", hookType))
		for _, cmd := range hookCommands {
			guidance.WriteString(fmt.Sprintf("  • %s\n", cmd))
		}
		guidance.WriteString("\nTo allow these hooks, run:\n")
		guidance.WriteString("  sprout trust\n\n")
		guidance.WriteString("To skip hooks this time:\n")
		guidance.WriteString("  Use the --no-hooks flag")
		return fmt.Errorf("%s", guidance.String())
	}

	// Display warning and hooks
	fmt.Fprintln(os.Stderr, "\n⚠️  This repository defines Sprout hooks in .sprout.yml:")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintf(os.Stderr, "  %s:\n", hookType)
	for _, cmd := range hookCommands {
		fmt.Fprintf(os.Stderr, "    - %s\n", cmd)
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "These commands will be executed automatically when a worktree is created.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Do you want to allow hooks from this repository?")
	fmt.Fprintln(os.Stderr, "Press 'y' to run them, or run again with --no-hooks to skip.")
	fmt.Fprintln(os.Stderr, "")

	// Prompt for consent
	fmt.Fprint(os.Stderr, "Allow hooks? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))

	if response == "y" || response == "yes" {
		// Trust the repository using the effects layer for consistency
		if err := r.TrustRepo(mainWorktreePath); err != nil {
			return fmt.Errorf("failed to trust repository: %w", err)
		}
		fmt.Fprintln(os.Stderr, "✓ Repository trusted")
		return nil
	}

	// User declined
	return fmt.Errorf("repository not trusted: user declined")
}
