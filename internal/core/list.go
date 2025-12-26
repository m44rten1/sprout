package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/m44rten1/sprout/internal/git"
)

// ANSI color codes for terminal output
const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorYellow  = "\033[33m"
	colorCyan    = "\033[36m"
	colorMagenta = "\033[35m"
	colorGreen   = "\033[32m"
	colorGray    = "\033[90m"
)

func colorize(s, color string) string {
	return color + s + colorReset
}

// BuildStatusEmojis builds a string of status emoji indicators.
// Returns empty string for clean worktrees.
func BuildStatusEmojis(status git.WorktreeStatus) string {
	emojis := make([]string, 0, 4)
	if status.Dirty {
		emojis = append(emojis, colorize("✗", colorRed)) // Red - urgent
	}
	if status.Ahead > 0 {
		emojis = append(emojis, colorize("↑", colorYellow)) // Yellow - warning
	}
	if status.Behind > 0 {
		emojis = append(emojis, colorize("↓", colorCyan)) // Cyan - informational
	}
	if status.Unmerged {
		emojis = append(emojis, colorize("↕", colorMagenta)) // Magenta - special state
	}
	return strings.Join(emojis, " ")
}

// ShortenPath replaces the home directory with ~ for shorter display.
// Returns original path if home directory cannot be determined or path is not under home.
func ShortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return ShortenPathWithHome(path, home)
}

// ShortenPathWithHome is the pure version of ShortenPath that takes home as a parameter.
// This allows testing without depending on the environment.
func ShortenPathWithHome(path, home string) string {
	// Handle empty home
	if home == "" {
		return path
	}

	if path == home {
		return "~"
	}

	// Ensure it's really a subpath by checking for path separator boundary.
	// This prevents "/home/user2" matching "/home/user" prefix incorrectly.
	sep := string(filepath.Separator)
	prefix := home + sep
	if strings.HasPrefix(path, prefix) {
		return "~" + strings.TrimPrefix(path, home)
	}

	return path
}

// WorktreeDisplay holds display configuration for a single worktree line.
type WorktreeDisplay struct {
	Branch       string
	Path         string
	StatusEmojis string
	IsMain       bool
	IsLast       bool
	UseTreeLines bool
}

// FormatWorktree formats a single worktree for display.
// Returns two lines: branch line with optional status, and path line.
func FormatWorktree(display WorktreeDisplay) string {
	icon := "🌱 "
	if display.IsMain {
		icon = ""
	}

	branch := display.Branch
	if branch == "" {
		branch = "(detached)"
	}

	// Tree line characters
	branchPrefix := ""
	pathPrefix := ""
	if display.UseTreeLines {
		if display.IsLast {
			branchPrefix = "└── "
			pathPrefix = "    "
		} else {
			branchPrefix = "├── "
			pathPrefix = "│   "
		}
	}

	label := fmt.Sprintf("%s%s%s", branchPrefix, icon, colorize(branch, colorGreen))

	// Build branch line
	branchLine := label
	if display.StatusEmojis != "" {
		branchLine += " " + display.StatusEmojis
	}

	// Build path line
	var pathLine string
	if pathPrefix != "" {
		pathLine = pathPrefix + " " + colorize(display.Path, colorGray)
	} else {
		pathLine = colorize(display.Path, colorGray)
	}

	return branchLine + "\n" + pathLine
}
