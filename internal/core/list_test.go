package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m44rten1/sprout/internal/git"
	"github.com/stretchr/testify/assert"
)

func TestBuildStatusEmojis(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		status   git.WorktreeStatus
		expected string
	}{
		{
			name:     "clean worktree",
			status:   git.WorktreeStatus{},
			expected: "",
		},
		{
			name:     "dirty only",
			status:   git.WorktreeStatus{Dirty: true},
			expected: "\033[31m✗\033[0m",
		},
		{
			name:     "ahead only",
			status:   git.WorktreeStatus{Ahead: 3},
			expected: "\033[33m↑\033[0m",
		},
		{
			name:     "behind only",
			status:   git.WorktreeStatus{Behind: 2},
			expected: "\033[36m↓\033[0m",
		},
		{
			name:     "unmerged only",
			status:   git.WorktreeStatus{Unmerged: true},
			expected: "\033[35m↕\033[0m",
		},
		{
			name:     "dirty and ahead",
			status:   git.WorktreeStatus{Dirty: true, Ahead: 1},
			expected: "\033[31m✗\033[0m \033[33m↑\033[0m",
		},
		{
			name:     "all statuses",
			status:   git.WorktreeStatus{Dirty: true, Ahead: 1, Behind: 2, Unmerged: true},
			expected: "\033[31m✗\033[0m \033[33m↑\033[0m \033[36m↓\033[0m \033[35m↕\033[0m",
		},
		{
			name:     "ahead and behind only",
			status:   git.WorktreeStatus{Ahead: 5, Behind: 3},
			expected: "\033[33m↑\033[0m \033[36m↓\033[0m",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := BuildStatusEmojis(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShortenPath(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot determine home directory")
	}

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "path under home",
			path:     filepath.Join(home, "Documents", "project"),
			expected: filepath.Join("~", "Documents", "project"),
		},
		{
			name:     "home directory itself",
			path:     home,
			expected: "~",
		},
		{
			name:     "path not under home",
			path:     filepath.Join(string(filepath.Separator), "usr", "local", "bin"),
			expected: filepath.Join(string(filepath.Separator), "usr", "local", "bin"),
		},
		{
			name:     "empty path",
			path:     "",
			expected: "",
		},
		{
			name:     "relative path",
			path:     filepath.Join("relative", "path"),
			expected: filepath.Join("relative", "path"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Use the pure function with home as parameter
			result := ShortenPathWithHome(tt.path, home)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShortenPathWithHome(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		home     string
		expected string
	}{
		{
			name:     "path under home",
			path:     "/home/user/Documents/project",
			home:     "/home/user",
			expected: "~/Documents/project",
		},
		{
			name:     "home directory itself",
			path:     "/home/user",
			home:     "/home/user",
			expected: "~",
		},
		{
			name:     "sibling path should not match (bug fix)",
			path:     "/home/user2/project",
			home:     "/home/user",
			expected: "/home/user2/project",
		},
		{
			name:     "prefix match but not subpath",
			path:     "/home/username-long/docs",
			home:     "/home/username",
			expected: "/home/username-long/docs",
		},
		{
			name:     "path not under home",
			path:     "/usr/local/bin",
			home:     "/home/user",
			expected: "/usr/local/bin",
		},
		{
			name:     "empty path",
			path:     "",
			home:     "/home/user",
			expected: "",
		},
		{
			name:     "empty home",
			path:     "/home/user/docs",
			home:     "",
			expected: "/home/user/docs",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ShortenPathWithHome(tt.path, tt.home)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatWorktree(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		display        WorktreeDisplay
		expectedHas    []string
		expectedNotHas []string
	}{
		{
			name: "sprout worktree with status",
			display: WorktreeDisplay{
				Branch:       "feature-branch",
				Path:         "~/sprout/repo/feature",
				StatusEmojis: colorize("✗", colorRed),
				IsMain:       false,
				IsLast:       false,
				UseTreeLines: false,
			},
			expectedHas:    []string{"🌱", "feature-branch", colorize("✗", colorRed), "~/sprout/repo/feature"},
			expectedNotHas: []string{"├──", "└──", "│"},
		},
		{
			name: "main worktree without status",
			display: WorktreeDisplay{
				Branch:       "main",
				Path:         "~/projects/myrepo",
				StatusEmojis: "",
				IsMain:       true,
				IsLast:       false,
				UseTreeLines: false,
			},
			expectedHas:    []string{"main", "~/projects/myrepo"},
			expectedNotHas: []string{"🌱", "✗", "↑", "├──"},
		},
		{
			name: "detached HEAD",
			display: WorktreeDisplay{
				Branch:       "",
				Path:         "~/sprout/repo/detached",
				StatusEmojis: "",
				IsMain:       false,
				IsLast:       false,
				UseTreeLines: false,
			},
			expectedHas:    []string{"🌱", "(detached)", "~/sprout/repo/detached"},
			expectedNotHas: []string{"✗"},
		},
		{
			name: "with tree lines - not last",
			display: WorktreeDisplay{
				Branch:       "feature",
				Path:         "~/sprout/repo/feature",
				StatusEmojis: "",
				IsMain:       false,
				IsLast:       false,
				UseTreeLines: true,
			},
			expectedHas:    []string{"├──", "│", "feature"},
			expectedNotHas: []string{"└──"},
		},
		{
			name: "with tree lines - last",
			display: WorktreeDisplay{
				Branch:       "hotfix",
				Path:         "~/sprout/repo/hotfix",
				StatusEmojis: "",
				IsMain:       false,
				IsLast:       true,
				UseTreeLines: true,
			},
			expectedHas:    []string{"└──", "hotfix"},
			expectedNotHas: []string{"├──", "│"},
		},
		{
			name: "all status indicators",
			display: WorktreeDisplay{
				Branch: "busy-branch",
				Path:   "~/sprout/repo/busy",
				StatusEmojis: strings.Join([]string{
					colorize("✗", colorRed),
					colorize("↑", colorYellow),
					colorize("↓", colorCyan),
					colorize("↕", colorMagenta),
				}, " "),
				IsMain:       false,
				IsLast:       false,
				UseTreeLines: false,
			},
			expectedHas: []string{
				"busy-branch",
				colorize("✗", colorRed),
				colorize("↑", colorYellow),
				colorize("↓", colorCyan),
				colorize("↕", colorMagenta),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := FormatWorktree(tt.display)

			// Check for expected content
			for _, expected := range tt.expectedHas {
				assert.Contains(t, result, expected, "output should contain %q", expected)
			}

			// Check for unexpected content
			for _, unexpected := range tt.expectedNotHas {
				assert.NotContains(t, result, unexpected, "output should not contain %q", unexpected)
			}

			// Verify structure: should have exactly one newline (two lines)
			assert.Equal(t, 1, strings.Count(result, "\n"), "output should have exactly one newline")

			// Verify both lines have content
			lines := strings.Split(result, "\n")
			assert.Len(t, lines, 2, "output should split into exactly 2 lines")
			assert.NotEmpty(t, lines[0], "first line (branch) should not be empty")
			assert.NotEmpty(t, lines[1], "second line (path) should not be empty")

			// Branch line should contain branch name (or "(detached)")
			if tt.display.Branch != "" {
				assert.Contains(t, lines[0], tt.display.Branch, "first line should contain branch name")
			} else {
				assert.Contains(t, lines[0], "(detached)", "first line should show (detached) for empty branch")
			}

			// Path line should contain the path
			assert.Contains(t, lines[1], tt.display.Path, "second line should contain path")
		})
	}
}
