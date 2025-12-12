package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/m44rten1/sprout/internal/config"
	"github.com/m44rten1/sprout/internal/trust"
)

// HookType represents the type of hook to run
type HookType string

const (
	OnCreate HookType = "on_create"
	OnOpen   HookType = "on_open"
)

// RunHooks executes hooks for the given hook type
func RunHooks(repoRoot, worktreePath string, hookType HookType) error {
	// Check if repo is trusted
	trusted, err := trust.IsRepoTrusted(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to check trust status: %w", err)
	}

	if !trusted {
		return &UntrustedError{RepoRoot: repoRoot}
	}

	// Load config
	cfg, err := config.Load(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get commands for this hook type
	var commands []string
	switch hookType {
	case OnCreate:
		commands = cfg.Hooks.OnCreate
	case OnOpen:
		commands = cfg.Hooks.OnOpen
	default:
		return fmt.Errorf("unknown hook type: %s", hookType)
	}

	if len(commands) == 0 {
		// No hooks to run
		return nil
	}

	fmt.Printf("\n🪝 Running %s hooks...\n\n", hookType)

	// Execute commands sequentially
	for i, cmd := range commands {
		fmt.Printf("[%d/%d] %s\n", i+1, len(commands), cmd)

		if err := executeCommand(cmd, worktreePath, repoRoot, hookType); err != nil {
			return &HookExecutionError{
				Command:  cmd,
				ExitCode: getExitCode(err),
				Err:      err,
			}
		}
	}

	fmt.Printf("\n✅ All %s hooks completed successfully\n\n", hookType)
	return nil
}

// executeCommand runs a single command in the worktree directory
func executeCommand(command, worktreePath, repoRoot string, hookType HookType) error {
	// Use sh -lc to execute the command (loads user's profile for proper PATH, etc.)
	cmd := exec.Command("sh", "-lc", command)
	cmd.Dir = worktreePath

	// Set environment variables
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env,
		fmt.Sprintf("SPROUT_REPO_ROOT=%s", repoRoot),
		fmt.Sprintf("SPROUT_WORKTREE_PATH=%s", worktreePath),
		fmt.Sprintf("SPROUT_HOOK_TYPE=%s", hookType),
	)

	// Pass through stdout and stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// getExitCode extracts exit code from an error
func getExitCode(err error) int {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return 1
}

// UntrustedError is returned when trying to run hooks for an untrusted repo
type UntrustedError struct {
	RepoRoot string
}

func (e *UntrustedError) Error() string {
	return fmt.Sprintf("hooks are not trusted for this repository")
}

// PrintUntrustedMessage prints a helpful message about trusting a repo
func PrintUntrustedMessage(repoRoot string) {
	configPath := filepath.Join(repoRoot, ".sprout.yml")

	fmt.Println()
	fmt.Println("🔒 Found .sprout.yml for this repository, but hooks are not trusted yet.")
	fmt.Printf("   Config: %s\n", configPath)
	fmt.Printf("   Repo:   %s\n", repoRoot)
	fmt.Println()
	fmt.Println("To enable and run hooks for this project, run:")
	fmt.Println()
	fmt.Println("    sprout trust")
	fmt.Println()
}

// HookExecutionError is returned when a hook command fails
type HookExecutionError struct {
	Command  string
	ExitCode int
	Err      error
}

func (e *HookExecutionError) Error() string {
	return fmt.Sprintf("hook command failed with exit code %d: %s", e.ExitCode, e.Command)
}

// CheckAndPrintUntrusted checks if repo has hooks but is not trusted, and prints message if so
// Returns true if hooks exist but are untrusted (message was printed)
func CheckAndPrintUntrusted(repoRoot string) (bool, error) {
	cfg, err := config.Load(repoRoot)
	if err != nil {
		// If config fails to load, don't treat it as untrusted error
		return false, err
	}

	if !cfg.HasHooks() {
		// No hooks defined, nothing to warn about
		return false, nil
	}

	trusted, err := trust.IsRepoTrusted(repoRoot)
	if err != nil {
		return false, err
	}

	if !trusted {
		PrintUntrustedMessage(repoRoot)
		return true, nil
	}

	return false, nil
}
