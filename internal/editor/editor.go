package editor

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Open opens the given path in an editor.
func Open(path string) error {
	// TODO: Check for configured editor (future)

	// Platform-aware defaults
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
