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
	if runtime.GOOS == "darwin" {
		// Try opening with "Cursor"
		if err := exec.Command("open", "-a", "Cursor", path).Run(); err == nil {
			return nil
		}
		// Fallback: cursor command
		if _, err := exec.LookPath("cursor"); err == nil {
			if err := exec.Command("cursor", path).Run(); err == nil {
				return nil
			}
		}
		// Fallback: code command
		if _, err := exec.LookPath("code"); err == nil {
			if err := exec.Command("code", path).Run(); err == nil {
				return nil
			}
		}
		// Fallback: open command
		return exec.Command("open", path).Run()
	}

	// Linux/Windows fallbacks could be added here
	return fmt.Errorf("unsupported platform for default editor opening")
}
