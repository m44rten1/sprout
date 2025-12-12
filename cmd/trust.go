package cmd

import (
	"fmt"
	"os"

	"github.com/m44rten1/sprout/internal/git"
	"github.com/m44rten1/sprout/internal/trust"

	"github.com/spf13/cobra"
)

var trustCmd = &cobra.Command{
	Use:   "trust [path]",
	Short: "Trust a repository to run hooks",
	Long: `Mark a repository as trusted to allow running hooks defined in .sprout.yml.

If no path is provided, the current repository is trusted.

WARNING: Only trust repositories you control or have reviewed the .sprout.yml file for.
Hooks can execute arbitrary commands on your system.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var repoRoot string
		var err error

		if len(args) > 0 {
			// Trust the specified path
			repoRoot = args[0]
			// Verify it's a git repo
			_, err := git.RunGitCommand(repoRoot, "rev-parse", "--show-toplevel")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s is not a git repository\n", repoRoot)
				os.Exit(1)
			}
		} else {
			// Trust current repo
			repoRoot, err = git.GetRepoRoot()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}

		// Check if already trusted
		isTrusted, err := trust.IsRepoTrusted(repoRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to check trust status: %v\n", err)
			os.Exit(1)
		}

		if isTrusted {
			fmt.Printf("✅ Repository is already trusted: %s\n", repoRoot)
			return
		}

		// Trust the repo
		if err := trust.TrustRepo(repoRoot); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to trust repository: %v\n", err)
			os.Exit(1)
		}

		fmt.Println()
		fmt.Printf("✅ Repository trusted: %s\n", repoRoot)
		fmt.Println()
		fmt.Println("Hooks defined in .sprout.yml will now run when you use:")
		fmt.Println("  - sprout add --init")
		fmt.Println("  - sprout open --sync")
		fmt.Println("  - sprout init")
		fmt.Println("  - sprout sync")
		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(trustCmd)
}
