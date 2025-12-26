package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	completionDryRunFlag bool
)

var completionInstallCmd = &cobra.Command{
	Use:   "install-completion",
	Short: "Install shell completion automatically",
	Long:  `Detects your shell and automatically configures completion by adding the necessary lines to your shell config file.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := installCompletion(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionInstallCmd)
	completionInstallCmd.Flags().BoolVar(&completionDryRunFlag, "dry-run", false, "Show what would be added without modifying files")
}

func installCompletion() error {
	shell := detectShell()
	if shell == "" {
		return fmt.Errorf("could not detect shell. Supported shells: zsh, bash, fish")
	}

	fmt.Printf("Detected shell: %s\n", shell)

	configFile, err := getShellConfigFile(shell)
	if err != nil {
		return err
	}

	fmt.Printf("Config file: %s\n", configFile)

	// Check if already configured
	if isAlreadyConfigured(configFile, shell) {
		fmt.Println("✓ Completion is already configured!")
		fmt.Println("If completion isn't working, try restarting your shell:")
		fmt.Printf("  exec %s\n", shell)
		return nil
	}

	// Generate the completion setup lines
	setupLines := generateSetupLines(shell)

	if completionDryRunFlag {
		fmt.Println("Dry run mode - would add the following to", configFile)
		fmt.Println(strings.Repeat("-", 60))
		fmt.Println(setupLines)
		fmt.Println(strings.Repeat("-", 60))
		return nil
	}

	// Backup the config file
	if err := backupConfigFile(configFile); err != nil {
		return fmt.Errorf("failed to backup config file: %w", err)
	}

	// Append the setup lines
	f, err := os.OpenFile(configFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString("\n" + setupLines + "\n"); err != nil {
		return fmt.Errorf("failed to write to config file: %w", err)
	}

	fmt.Println("✓ Completion configured successfully!")
	fmt.Printf("✓ Backup saved to: %s.backup-sprout\n", configFile)
	fmt.Println("\nTo activate, restart your shell:")
	fmt.Printf("  exec %s\n", shell)
	fmt.Println("\nOr source your config file:")
	fmt.Printf("  source %s\n", configFile)

	return nil
}

func detectShell() string {
	// Try $SHELL environment variable
	shell := os.Getenv("SHELL")
	if shell != "" {
		shell = filepath.Base(shell)
		if shell == "zsh" || shell == "bash" || shell == "fish" {
			return shell
		}
	}

	// Fallback: try to detect from parent process
	return ""
}

func getShellConfigFile(shell string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}

	var configFile string
	switch shell {
	case "zsh":
		configFile = filepath.Join(home, ".zshrc")
	case "bash":
		// Prefer .bashrc, fallback to .bash_profile
		bashrc := filepath.Join(home, ".bashrc")
		bashProfile := filepath.Join(home, ".bash_profile")
		if _, err := os.Stat(bashrc); err == nil {
			configFile = bashrc
		} else {
			configFile = bashProfile
		}
	case "fish":
		configFile = filepath.Join(home, ".config", "fish", "config.fish")
		// Ensure fish config directory exists
		if err := os.MkdirAll(filepath.Dir(configFile), 0755); err != nil {
			return "", fmt.Errorf("failed to create fish config directory: %w", err)
		}
	default:
		return "", fmt.Errorf("unsupported shell: %s", shell)
	}

	return configFile, nil
}

func isAlreadyConfigured(configFile, shell string) bool {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return false
	}

	content := string(data)

	// Check for various completion markers
	markers := []string{
		"sprout completion",
		"_sprout",
	}

	for _, marker := range markers {
		if strings.Contains(content, marker) {
			return true
		}
	}

	return false
}

func generateSetupLines(shell string) string {
	var lines strings.Builder

	lines.WriteString("# Sprout completion setup (added by 'sprout completion install')\n")

	switch shell {
	case "zsh":
		// Check if Homebrew is installed and add fpath setup
		if isHomebrewInstalled() {
			lines.WriteString("if type brew &>/dev/null; then\n")
			lines.WriteString("  FPATH=\"$(brew --prefix)/share/zsh/site-functions:${FPATH}\"\n")
			lines.WriteString("fi\n")
		}
		lines.WriteString("autoload -Uz compinit\n")
		lines.WriteString("compinit\n")

	case "bash":
		if isHomebrewInstalled() {
			lines.WriteString("if type brew &>/dev/null; then\n")
			lines.WriteString("  HOMEBREW_PREFIX=\"$(brew --prefix)\"\n")
			lines.WriteString("  if [[ -r \"${HOMEBREW_PREFIX}/etc/profile.d/bash_completion.sh\" ]]; then\n")
			lines.WriteString("    source \"${HOMEBREW_PREFIX}/etc/profile.d/bash_completion.sh\"\n")
			lines.WriteString("  fi\n")
			lines.WriteString("fi\n")
		} else {
			lines.WriteString("source <(sprout completion bash)\n")
		}

	case "fish":
		if isHomebrewInstalled() {
			lines.WriteString("if type -q brew\n")
			lines.WriteString("  set -gx fish_complete_path (brew --prefix)/share/fish/vendor_completions.d $fish_complete_path\n")
			lines.WriteString("end\n")
		} else {
			lines.WriteString("sprout completion fish | source\n")
		}
	}

	return lines.String()
}

func isHomebrewInstalled() bool {
	_, err := exec.LookPath("brew")
	return err == nil
}

func backupConfigFile(configFile string) error {
	// Only backup if file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return nil
	}

	backupFile := configFile + ".backup-sprout"
	data, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}

	return os.WriteFile(backupFile, data, 0644)
}
