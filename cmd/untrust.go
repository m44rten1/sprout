package cmd

import (
	"fmt"
	"os"

	"github.com/m44rten1/sprout/internal/core"
	"github.com/m44rten1/sprout/internal/effects"

	"github.com/spf13/cobra"
)

var untrustCmd = &cobra.Command{
	Use:   "untrust [path]",
	Short: "Untrust a repository to prevent hooks",
	Long: `Remove trust from a repository to prevent hooks from running automatically.

If no path is provided, the current repository is untrusted.

After untrusting, hooks defined in .sprout.yml will not run automatically.
You can still create worktrees with --no-hooks or run 'sprout trust' to re-enable hooks.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Create effects
		fx := effects.NewRealEffects()

		// Determine path argument
		var pathArg string
		if len(args) > 0 {
			pathArg = args[0]
		}

		// Build context from effects (reuse BuildTrustContext)
		ctx, err := BuildTrustContext(fx, pathArg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Plan and execute
		plan := core.PlanUntrustCommand(ctx)

		// Dry-run mode: print plan instead of executing
		if dryRunFlag {
			fmt.Println(core.FormatPlan(plan))
			return
		}

		// Execute plan
		if err := effects.ExecutePlan(plan, fx); err != nil {
			if code, ok := effects.IsExit(err); ok {
				os.Exit(code)
			}
			fmt.Fprintf(os.Stderr, "Execution failed: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(untrustCmd)
}
