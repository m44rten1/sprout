package core_test

import (
	"strings"
	"testing"

	"github.com/m44rten1/sprout/internal/core"
	"github.com/stretchr/testify/assert"
)

func TestFormatPlan_EmptyPlan(t *testing.T) {
	plan := core.Plan{Actions: nil}
	output := core.FormatPlan(plan)

	assert.Equal(t, "No actions to perform.", output)
}

func TestFormatPlan_SingleAction(t *testing.T) {
	plan := core.Plan{
		Actions: []core.Action{
			core.PrintMessage{Msg: "Hello, world!"},
		},
	}
	output := core.FormatPlan(plan)

	expected := `Planned actions:
  1. Print: "Hello, world!"`
	assert.Equal(t, expected, output)
}

func TestFormatPlan_MultipleActions(t *testing.T) {
	// This test also verifies numbering format
	plan := core.Plan{
		Actions: []core.Action{
			core.CreateDirectory{Path: "/test/dir", Perm: 0755},
			core.RunGitCommand{Dir: "/repo", Args: []string{"worktree", "add", "/path"}},
			core.OpenEditor{Path: "/test/worktree"},
		},
	}
	output := core.FormatPlan(plan)

	// Verify header and numbering
	assert.Contains(t, output, "Planned actions:")
	assert.Contains(t, output, "1. Create directory: /test/dir")
	assert.Contains(t, output, "2. Run git command in /repo: git worktree add /path")
	assert.Contains(t, output, "3. Open editor: /test/worktree")
}

func TestFormatPlan_AllActionTypes(t *testing.T) {
	plan := core.Plan{
		Actions: []core.Action{
			core.NoOp{},
			core.PrintMessage{Msg: "message"},
			core.PrintError{Msg: "error"},
			core.CreateDirectory{Path: "/dir", Perm: 0755},
			core.RunGitCommand{Dir: "/repo", Args: []string{"status"}},
			core.OpenEditor{Path: "/path"},
			core.RunHooks{
				Type:     core.HookTypeOnCreate,
				Commands: []string{"npm install", "npm build"},
				Path:     "/worktree",
			},
			core.TrustRepo{RepoRoot: "/repo"},
			core.Exit{Code: 1},
		},
	}
	output := core.FormatPlan(plan)

	// Verify each action type is formatted
	assert.Contains(t, output, "No operation")
	assert.Contains(t, output, "Print: \"message\"")
	assert.Contains(t, output, "Print error: \"error\"")
	assert.Contains(t, output, "Create directory: /dir")
	assert.Contains(t, output, "Run git command in /repo: git status")
	assert.Contains(t, output, "Open editor: /path")
	assert.Contains(t, output, "Run 2 on_create hook(s) in /worktree")
	assert.Contains(t, output, "Trust repository: /repo")
	assert.Contains(t, output, "Exit with code 1")
}

func TestFormatPlan_MessageTruncation(t *testing.T) {
	longMsg := strings.Repeat("a", 100)
	plan := core.Plan{
		Actions: []core.Action{
			core.PrintMessage{Msg: longMsg},
		},
	}
	output := core.FormatPlan(plan)

	// Should be truncated and not contain the full 100-char message
	assert.Contains(t, output, "...")
	assert.NotContains(t, output, strings.Repeat("a", 70))

	// The formatted line should have reasonable length (header + number + action description)
	lines := strings.Split(output, "\n")
	assert.LessOrEqual(t, len(lines[1]), 100, "formatted line should not be excessively long")
}

func TestFormatPlan_MultilineMessageTruncation(t *testing.T) {
	multilineMsg := "First line\nSecond line\nThird line"
	plan := core.Plan{
		Actions: []core.Action{
			core.PrintMessage{Msg: multilineMsg},
		},
	}
	output := core.FormatPlan(plan)

	// Should show only first line (no "..." added for multiline truncation)
	assert.Contains(t, output, "First line")
	assert.NotContains(t, output, "Second line")
	assert.NotContains(t, output, "Third line")
}

func TestFormatPlan_TruncateEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		want string
	}{
		{
			name: "short message not truncated",
			msg:  "Hello, world!",
			want: `Print: "Hello, world!"`,
		},
		{
			name: "exactly 60 chars not truncated",
			msg:  strings.Repeat("a", 60),
			want: strings.Repeat("a", 60),
		},
		{
			name: "61+ chars truncated to 60",
			msg:  strings.Repeat("a", 100),
			want: strings.Repeat("a", 57) + "...",
		},
		{
			name: "multiline shows only first line",
			msg:  "First\nSecond",
			want: `Print: "First"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := core.Plan{
				Actions: []core.Action{
					core.PrintMessage{Msg: tt.msg},
				},
			}
			output := core.FormatPlan(plan)
			assert.Contains(t, output, tt.want)
		})
	}
}

func TestFormatPlan_GitCommandFormatting(t *testing.T) {
	tests := []struct {
		name   string
		action core.RunGitCommand
		want   string
		avoid  string
	}{
		{
			name:   "with directory",
			action: core.RunGitCommand{Dir: "/repo", Args: []string{"status"}},
			want:   "Run git command in /repo: git status",
		},
		{
			name:   "without directory",
			action: core.RunGitCommand{Dir: "", Args: []string{"status"}},
			want:   "Run git command: git status",
			avoid:  "Run git command in",
		},
		{
			name:   "multiple args",
			action: core.RunGitCommand{Dir: "/repo", Args: []string{"worktree", "add", "/path"}},
			want:   "Run git command in /repo: git worktree add /path",
		},
		{
			name:   "empty args with directory",
			action: core.RunGitCommand{Dir: "/repo", Args: []string{}},
			want:   "Run git in /repo",
			avoid:  "git command",
		},
		{
			name:   "empty args without directory",
			action: core.RunGitCommand{Dir: "", Args: []string{}},
			want:   "Run git",
			avoid:  "git command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := core.Plan{Actions: []core.Action{tt.action}}
			output := core.FormatPlan(plan)

			assert.Contains(t, output, tt.want)
			if tt.avoid != "" {
				assert.NotContains(t, output, tt.avoid)
			}
		})
	}
}

func TestFormatPlan_HookFormatting(t *testing.T) {
	tests := []struct {
		name     string
		commands []string
		want     string
	}{
		{
			name:     "multiple hooks",
			commands: []string{"npm install", "npm build"},
			want:     "Run 2 on_create hook(s) in /worktree",
		},
		{
			name:     "single hook",
			commands: []string{"npm install"},
			want:     "Run 1 on_create hook(s) in /worktree",
		},
		{
			name:     "zero hooks (edge case)",
			commands: []string{},
			want:     "Run 0 on_create hook(s) in /worktree",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := core.RunHooks{
				Type:     core.HookTypeOnCreate,
				Commands: tt.commands,
				Path:     "/worktree",
			}
			plan := core.Plan{Actions: []core.Action{action}}
			output := core.FormatPlan(plan)

			assert.Contains(t, output, tt.want)
		})
	}
}

func TestFormatPlan_Deterministic(t *testing.T) {
	plan := core.Plan{
		Actions: []core.Action{
			core.PrintMessage{Msg: "test"},
			core.CreateDirectory{Path: "/dir", Perm: 0755},
		},
	}

	output1 := core.FormatPlan(plan)
	output2 := core.FormatPlan(plan)

	// Same input produces same output
	assert.Equal(t, output1, output2)
}
