package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/m44rten1/sprout/internal/core"
	"github.com/m44rten1/sprout/internal/effects"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sprout",
	Short: "A Git worktree helper",
	Long:  `sprout is a lightweight Go CLI tool for managing Git worktrees.`,
}

var dryRunFlag bool

func init() {
	// Enable shell completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = false

	// Add global --dry-run flag
	rootCmd.PersistentFlags().BoolVar(&dryRunFlag, "dry-run", false, "Show what would be done without executing")

	// Auto-repair worktrees before any command
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		// Skip in tests or when explicitly disabled
		if flag.Lookup("test.v") != nil || os.Getenv("SPROUT_SKIP_AUTOREPAIR") == "1" {
			return
		}

		// Skip for commands that don't need worktree repair
		if cmd.Name() == "completion" || cmd.Name() == "help" {
			return
		}

		// Skip auto-repair in dry-run mode (no side effects)
		if dryRunFlag {
			return
		}

		autoRepairWorktrees()
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// autoRepairWorktrees runs silent worktree repair before each command.
// Pattern: gather context → plan → execute (functional core / imperative shell).
func autoRepairWorktrees() {
	defer func() {
		if r := recover(); r != nil {
			// In debug mode, show the panic; otherwise swallow silently
			if os.Getenv("SPROUT_DEBUG") == "1" {
				fmt.Fprintf(os.Stderr, "sprout: auto-repair panic: %v\n", r)
			}
		}
	}()

	fx := effects.NewRealEffects()

	// Imperative shell: discover repos using Effects
	repos, err := collectAllRepos()
	if err != nil || len(repos) == 0 {
		return // Silent failure - non-critical operation
	}

	// Extract repo paths for repair
	repoPaths := make([]string, 0, len(repos))
	for _, repo := range repos {
		repoPaths = append(repoPaths, repo.MainPath)
	}

	// Functional core: create pure repair plan
	ctx := core.RepairContext{Repos: repoPaths}
	plan := core.PlanRepair(ctx)

	// Execute plan silently (ignore errors - best effort)
	if err := effects.ExecutePlan(plan, fx); err != nil && os.Getenv("SPROUT_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "sprout: auto-repair failed: %v\n", err)
	}
}
