package editor

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpenWithCommand(t *testing.T) {
	t.Run("simple command", func(t *testing.T) {
		// Use 'true' which exists on Unix systems and always succeeds
		err := openWithCommand("true", "/test/path")
		assert.NoError(t, err)
	})

	t.Run("command with arguments", func(t *testing.T) {
		// Use 'echo' with arguments - should succeed
		err := openWithCommand("echo test", "/test/path")
		assert.NoError(t, err)
	})

	t.Run("empty command", func(t *testing.T) {
		err := openWithCommand("", "/test/path")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty editor command")
	})

	t.Run("nonexistent command", func(t *testing.T) {
		err := openWithCommand("nonexistent-editor-12345", "/test/path")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestOpen_EnvironmentVariables(t *testing.T) {
	// Save original environment
	origSproutEditor := os.Getenv("SPROUT_EDITOR")
	origEditor := os.Getenv("EDITOR")
	defer func() {
		os.Setenv("SPROUT_EDITOR", origSproutEditor)
		os.Setenv("EDITOR", origEditor)
	}()

	t.Run("SPROUT_EDITOR takes precedence", func(t *testing.T) {
		os.Setenv("SPROUT_EDITOR", "true")
		os.Setenv("EDITOR", "false") // would fail if used

		err := Open("/test/path")
		assert.NoError(t, err, "Should use SPROUT_EDITOR (true) not EDITOR (false)")
	})

	t.Run("EDITOR is used if SPROUT_EDITOR not set", func(t *testing.T) {
		os.Unsetenv("SPROUT_EDITOR")
		os.Setenv("EDITOR", "true")

		err := Open("/test/path")
		assert.NoError(t, err, "Should use EDITOR")
	})

	t.Run("falls back to platform defaults if neither set", func(t *testing.T) {
		os.Unsetenv("SPROUT_EDITOR")
		os.Unsetenv("EDITOR")

		// This will try platform defaults - may succeed or fail depending on system
		// We just verify it doesn't panic
		_ = Open("/test/path")
	})
}

func TestOpen_CommandParsing(t *testing.T) {
	// Save original environment
	origSproutEditor := os.Getenv("SPROUT_EDITOR")
	defer os.Setenv("SPROUT_EDITOR", origSproutEditor)

	t.Run("handles commands with spaces", func(t *testing.T) {
		// Use echo with multiple args
		os.Setenv("SPROUT_EDITOR", "echo hello world")
		err := Open("/test/path")
		assert.NoError(t, err)
	})

	t.Run("handles single-word command", func(t *testing.T) {
		os.Setenv("SPROUT_EDITOR", "true")
		err := Open("/test/path")
		assert.NoError(t, err)
	})
}
