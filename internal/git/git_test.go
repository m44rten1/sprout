package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateStashIncludesUntrackedFiles(t *testing.T) {
	repo := initTestRepo(t)

	writeFile(t, filepath.Join(repo, "tracked.txt"), "updated\n")
	writeFile(t, filepath.Join(repo, "notes.txt"), "todo\n")

	stashOID, err := CreateStash(repo, "move current changes", true)
	if err != nil {
		t.Fatalf("CreateStash() error = %v", err)
	}
	if stashOID == "" {
		t.Fatal("CreateStash() returned empty stash OID")
	}

	dirty, err := IsDirty(repo)
	if err != nil {
		t.Fatalf("IsDirty() error = %v", err)
	}
	if dirty {
		t.Fatal("expected repository to be clean after stashing changes")
	}

	stashList := runGit(t, repo, "stash", "list", "--format=%H")
	if !strings.Contains(stashList, stashOID) {
		t.Fatalf("expected stash list to contain %s, got %q", stashOID, stashList)
	}
}

func TestApplyAndDropStashAcrossWorktrees(t *testing.T) {
	repo := initTestRepo(t)

	writeFile(t, filepath.Join(repo, "tracked.txt"), "updated\n")
	writeFile(t, filepath.Join(repo, "notes.txt"), "todo\n")

	stashOID, err := CreateStash(repo, "move current changes", true)
	if err != nil {
		t.Fatalf("CreateStash() error = %v", err)
	}

	worktreePath := filepath.Join(t.TempDir(), "feature-worktree")
	runGit(t, repo, "worktree", "add", worktreePath, "-b", "feature")

	if err := ApplyStash(worktreePath, stashOID); err != nil {
		t.Fatalf("ApplyStash() error = %v", err)
	}

	trackedContent, err := os.ReadFile(filepath.Join(worktreePath, "tracked.txt"))
	if err != nil {
		t.Fatalf("reading tracked file: %v", err)
	}
	if string(trackedContent) != "updated\n" {
		t.Fatalf("expected applied tracked content, got %q", trackedContent)
	}

	untrackedContent, err := os.ReadFile(filepath.Join(worktreePath, "notes.txt"))
	if err != nil {
		t.Fatalf("reading untracked file: %v", err)
	}
	if string(untrackedContent) != "todo\n" {
		t.Fatalf("expected applied untracked content, got %q", untrackedContent)
	}

	if err := DropStash(repo, stashOID); err != nil {
		t.Fatalf("DropStash() error = %v", err)
	}

	stashList := strings.TrimSpace(runGit(t, repo, "stash", "list"))
	if stashList != "" {
		t.Fatalf("expected stash list to be empty after drop, got %q", stashList)
	}
}

func TestDropStashReturnsErrorWhenEntryMissing(t *testing.T) {
	repo := initTestRepo(t)

	err := DropStash(repo, "deadbeef")
	if err == nil {
		t.Fatal("expected DropStash() to fail for a missing stash entry")
	}
}

func TestGetDefaultBranchFallsBackToMainWorktreeBranch(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init", "-b", "dev")
	runGit(t, repo, "config", "user.name", "Test User")
	runGit(t, repo, "config", "user.email", "test@example.com")

	writeFile(t, filepath.Join(repo, "tracked.txt"), "initial\n")
	runGit(t, repo, "add", "tracked.txt")
	runGit(t, repo, "commit", "-m", "initial commit")

	branch, err := GetDefaultBranch(repo)
	if err != nil {
		t.Fatalf("GetDefaultBranch() error = %v", err)
	}
	if branch != "dev" {
		t.Fatalf("expected default branch dev, got %q", branch)
	}
}

func TestGetDefaultBranchErrorsWhenItCannotBeDetermined(t *testing.T) {
	repo := initTestRepo(t)
	runGit(t, repo, "checkout", "--detach")

	_, err := GetDefaultBranch(repo)
	if err == nil {
		t.Fatal("expected GetDefaultBranch() to fail in detached HEAD without origin/HEAD")
	}
}

func TestIsBranchMergedIntoMainUsesLocalDefaultBranch(t *testing.T) {
	repo := initTestRepo(t)
	remote := t.TempDir()
	runGit(t, remote, "init", "--bare")

	runGit(t, repo, "remote", "add", "origin", remote)
	runGit(t, repo, "push", "-u", "origin", "main")

	runGit(t, repo, "checkout", "-b", "feature")
	writeFile(t, filepath.Join(repo, "tracked.txt"), "feature change\n")
	runGit(t, repo, "add", "tracked.txt")
	runGit(t, repo, "commit", "-m", "feature commit")

	runGit(t, repo, "checkout", "main")
	runGit(t, repo, "merge", "--ff-only", "feature")

	merged, err := IsBranchMergedIntoMain(repo, "feature")
	if err != nil {
		t.Fatalf("IsBranchMergedIntoMain() error = %v", err)
	}
	if !merged {
		t.Fatal("expected branch merged into local main to be considered merged")
	}
}

func initTestRepo(t *testing.T) string {
	t.Helper()

	repo := t.TempDir()
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.name", "Test User")
	runGit(t, repo, "config", "user.email", "test@example.com")

	writeFile(t, filepath.Join(repo, "tracked.txt"), "initial\n")
	runGit(t, repo, "add", "tracked.txt")
	runGit(t, repo, "commit", "-m", "initial commit")

	return repo
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}
