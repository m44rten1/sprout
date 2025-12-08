package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/m44rten1/sprout/internal/git"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List worktrees",
	Run: func(cmd *cobra.Command, args []string) {
		repoRoot, err := git.GetRepoRoot()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		worktrees, err := git.ListWorktrees(repoRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list worktrees: %v\n", err)
			os.Exit(1)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		for _, wt := range worktrees {
			branch := wt.Branch
			if branch == "" {
				branch = "(detached)"
			}
			fmt.Fprintf(w, "%s\t%s\n", branch, wt.Path)
		}
		w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
