package editor

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/google/shlex"
)

// Open opens the given path in an editor.
func Open(path string) error {
	// Allow explicit override. Unlike $EDITOR, this is intended to be a "launcher"
	// (typically non-blocking) so sprout can continue running hooks afterwards.
	if cmdline := os.Getenv("SPROUT_EDITOR"); cmdline != "" {
		return runConfigured(cmdline, path, false)
	}

	// Platform-aware defaults
	switch runtime.GOOS {
	case "darwin":
		// Try opening with "Cursor" (macOS app)
		if err := startIfExists("open", "-a", "Cursor", path); err == nil {
			return nil
		}
		// Fallback: cursor command
		if err := startIfExists("cursor", path); err == nil {
			return nil
		}
		// Fallback: VS Code command
		if err := startIfExists("code", path); err == nil {
			return nil
		}
		// Fallback: open command (file manager / default handler)
		if err := startIfExists("open", path); err == nil {
			return nil
		}

	case "linux":
		// Respect conventional terminal editor vars (these may be blocking, by design).
		if cmdline := os.Getenv("VISUAL"); cmdline != "" {
			return runConfigured(cmdline, path, true)
		}
		if cmdline := os.Getenv("EDITOR"); cmdline != "" {
			return runConfigured(cmdline, path, true)
		}

		// Prefer GUI editors/launchers (non-blocking).
		if err := startIfExists("cursor", path); err == nil {
			return nil
		}
		if err := startIfExists("code", path); err == nil {
			return nil
		}
		if err := startIfExists("code-insiders", path); err == nil {
			return nil
		}
		if err := startIfExists("codium", path); err == nil {
			return nil
		}

		// Last resort: open the folder via desktop handler.
		if err := startIfExists("xdg-open", path); err == nil {
			return nil
		}
	}

	return fmt.Errorf("no supported editor opener found (set $SPROUT_EDITOR, or $VISUAL/$EDITOR on linux)")
}

func startIfExists(name string, args ...string) error {
	if _, err := exec.LookPath(name); err != nil {
		return err
	}
	return exec.Command(name, args...).Start()
}

func runConfigured(cmdline string, path string, foreground bool) error {
	parts, err := shlex.Split(cmdline)
	if err != nil {
		return fmt.Errorf("failed to parse editor command %q: %w", cmdline, err)
	}
	if len(parts) == 0 {
		return fmt.Errorf("editor command is empty")
	}

	name := parts[0]
	args := append(parts[1:], path)
	cmd := exec.Command(name, args...)

	if foreground {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	return cmd.Start()
}
