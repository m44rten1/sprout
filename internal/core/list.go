package core

import (
	"fmt"
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

// ListContext contains all inputs needed for list formatting.
// This is the context struct passed from the imperative shell to the pure formatter.
type ListContext struct {
	Repos   []RepoDisplay
	Home    string // User's home directory for path shortening
	ShowAll bool   // Whether --all flag was used (affects headers and empty message)
}

// RepoDisplay holds display data for a repository (pure data, no I/O).
type RepoDisplay struct {
	Name      string
	MainPath  string
	Worktrees []WorktreeDisplayItem
}

// WorktreeDisplayItem holds display data for a worktree.
type WorktreeDisplayItem struct {
	Branch string
	Path   string
	Status git.WorktreeStatus
	IsMain bool
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

// FormatListOutput formats the list command output.
// Pure function that handles both empty and non-empty cases.
// This is the single entry point for list formatting from the command layer.
func FormatListOutput(ctx ListContext) string {
	if len(ctx.Repos) == 0 {
		if ctx.ShowAll {
			return "\nNo sprout worktrees found."
		}
		return "\nNo sprout worktrees found for this repository."
	}

	return FormatRepoList(ctx.Repos, ctx.Home, ctx.ShowAll)
}

// FormatRepoList formats a list of repositories for display.
// Pure function - takes home dir as parameter instead of calling os.UserHomeDir().
// Returns empty string if repos is empty.
func FormatRepoList(repos []RepoDisplay, home string, showHeaders bool) string {
	if len(repos) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, "") // Add spacing from prompt

	for i, repo := range repos {
		if showHeaders {
			if i > 0 {
				lines = append(lines, "") // Blank line between repos
			}
			lines = append(lines, fmt.Sprintf("\033[1m%s\033[0m", repo.Name))
		}

		for j, wt := range repo.Worktrees {
			isLast := j == len(repo.Worktrees)-1
			display := WorktreeDisplay{
				Branch:       wt.Branch,
				Path:         ShortenPathWithHome(wt.Path, home),
				StatusEmojis: BuildStatusEmojis(wt.Status),
				IsMain:       wt.IsMain,
				IsLast:       isLast,
				UseTreeLines: showHeaders,
			}
			lines = append(lines, FormatWorktree(display))
		}
	}

	return strings.Join(lines, "\n")
}
