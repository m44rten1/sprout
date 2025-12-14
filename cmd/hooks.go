package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/m44rten1/sprout/internal/config"
	"github.com/m44rten1/sprout/internal/git"
	"github.com/m44rten1/sprout/internal/trust"

	"github.com/spf13/cobra"
)

var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Show hook configuration status",
	Long: `Display information about hooks for the current repository:
- Whether .sprout.yml exists
- Trust status
- Which hooks are defined`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get repo root
		repoRoot, err := git.GetRepoRoot()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println()
		fmt.Printf("Repository: %s\n", repoRoot)
		fmt.Println()

		// Get main worktree path for config fallback
		mainWorktreePath, err := git.GetMainWorktreePath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get main worktree path: %v\n", err)
			os.Exit(1)
		}

		// Check if .sprout.yml exists in current or main worktree
		configPath := filepath.Join(repoRoot, ".sprout.yml")
		_, err = os.Stat(configPath)
		configExists := err == nil

		if !configExists && mainWorktreePath != repoRoot {
			configPath = filepath.Join(mainWorktreePath, ".sprout.yml")
			_, err = os.Stat(configPath)
			configExists = err == nil
		}

		if !configExists {
			fmt.Println("❌ No .sprout.yml found")
			fmt.Println()
			fmt.Println("To add hooks, create a .sprout.yml file in your repository root.")
			fmt.Println("Example:")
			fmt.Println()
			fmt.Println("  hooks:")
			fmt.Println("    on_create:")
			fmt.Println("      - npm ci")
			fmt.Println("      - npm run build")
			fmt.Println("    on_open:")
			fmt.Println("      - npm run lint:types")
			fmt.Println()
			return
		}

		fmt.Printf("✅ Config file: %s\n", configPath)
		fmt.Println()

		// Check trust status
		isTrusted, err := trust.IsRepoTrusted(repoRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to check trust status: %v\n", err)
			os.Exit(1)
		}

		if isTrusted {
			fmt.Println("✅ Repository is trusted")
		} else {
			fmt.Println("🔒 Repository is NOT trusted")
			fmt.Println()
			fmt.Println("Run 'sprout trust' to enable hooks for this repository.")
		}
		fmt.Println()

		// Load and display hooks
		cfg, err := config.Load(repoRoot, mainWorktreePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
			os.Exit(1)
		}

		if !cfg.HasHooks() {
			fmt.Println("ℹ️  No hooks defined")
			return
		}

		// Display on_create hooks
		if cfg.HasCreateHooks() {
			fmt.Println("on_create hooks:")
			for i, cmd := range cfg.Hooks.OnCreate {
				fmt.Printf("  %d. %s\n", i+1, cmd)
			}
			fmt.Println()
		}

		// Display on_open hooks
		if cfg.HasOpenHooks() {
			fmt.Println("on_open hooks:")
			for i, cmd := range cfg.Hooks.OnOpen {
				fmt.Printf("  %d. %s\n", i+1, cmd)
			}
			fmt.Println()
		}

		// Show how to run hooks
		if isTrusted {
			fmt.Println("Run hooks with:")
			if cfg.HasCreateHooks() {
				fmt.Println("  - sprout add --init    (runs on_create)")
				fmt.Println("  - sprout init          (runs on_create)")
			}
			if cfg.HasOpenHooks() {
				fmt.Println("  - sprout open --sync   (runs on_open)")
				fmt.Println("  - sprout sync          (runs on_open)")
			}
			fmt.Println()
		}
	},
}

func init() {
	rootCmd.AddCommand(hooksCmd)
}
