package cmd

import (
	"fmt"
	"os"

	"github.com/m44rten1/sprout/internal/core"
	"github.com/m44rten1/sprout/internal/effects"
)

// runPlan executes a plan, or prints it in dry-run mode.
func runPlan(plan core.Plan, fx effects.Effects) {
	if dryRunFlag {
		fmt.Println(core.FormatPlan(plan))
		return
	}
	if err := effects.ExecutePlan(plan, fx); err != nil {
		if code, ok := effects.IsExit(err); ok {
			os.Exit(code)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

