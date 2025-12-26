package editor

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Open opens the given path in an editor.
// Priority order:
//  1. $SPROUT_EDITOR - sprout-specific override
//  2. $EDITOR - standard Unix convention
//  3. Platform defaults (Cursor, Code, system defaults)
func Open(path string) error {
	// Check SPROUT_EDITOR first (highest priority)
	if editor := os.Getenv("SPROUT_EDITOR"); editor != "" {
		return openWithCommand(editor, path)
	}

	// Check EDITOR second (standard Unix convention)
	if editor := os.Getenv("EDITOR"); editor != "" {
		return openWithCommand(editor, path)
	}

	// Fall back to platform-aware defaults
	return openWithPlatformDefaults(path)
}

// openWithCommand executes the given editor command with the path.
// Supports space-separated arguments (e.g., "code --wait").
func openWithCommand(editor, path string) error {
	// Split command and args (simple split on spaces)
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return fmt.Errorf("empty editor command")
	}

	cmd := parts[0]
	args := append(parts[1:], path)

	// Check if command exists
	if _, err := exec.LookPath(cmd); err != nil {
		return fmt.Errorf("editor command not found: %s", cmd)
	}

	return exec.Command(cmd, args...).Run()
}

// openWithPlatformDefaults tries platform-specific editor defaults.
func openWithPlatformDefaults(path string) error {
	switch runtime.GOOS {
	case "darwin":
		// macOS: Try Cursor app, then cursor/code commands, then system open
		if err := exec.Command("open", "-a", "Cursor", path).Run(); err == nil {
			return nil
		}
		if _, err := exec.LookPath("cursor"); err == nil {
			if err := exec.Command("cursor", path).Run(); err == nil {
				return nil
			}
		}
		if _, err := exec.LookPath("code"); err == nil {
			if err := exec.Command("code", path).Run(); err == nil {
				return nil
			}
		}
		return exec.Command("open", path).Run()

	case "linux":
		// Linux: Try cursor/code commands, then xdg-open
		if _, err := exec.LookPath("cursor"); err == nil {
			if err := exec.Command("cursor", path).Run(); err == nil {
				return nil
			}
		}
		if _, err := exec.LookPath("code"); err == nil {
			if err := exec.Command("code", path).Run(); err == nil {
				return nil
			}
		}
		if _, err := exec.LookPath("xdg-open"); err == nil {
			return exec.Command("xdg-open", path).Run()
		}
		return fmt.Errorf("no suitable editor found (tried: cursor, code, xdg-open)")

	case "windows":
		// Windows: Try cursor/code commands, then cmd start
		if _, err := exec.LookPath("cursor"); err == nil {
			if err := exec.Command("cursor", path).Run(); err == nil {
				return nil
			}
		}
		if _, err := exec.LookPath("code"); err == nil {
			if err := exec.Command("code", path).Run(); err == nil {
				return nil
			}
		}
		// Use cmd.exe with /C start to open the path
		return exec.Command("cmd", "/C", "start", "", path).Run()

	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
