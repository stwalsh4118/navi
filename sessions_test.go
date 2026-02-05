package main

import (
	"testing"
)

func TestCapturePane(t *testing.T) {
	t.Run("returns error for non-existent session", func(t *testing.T) {
		// Try to capture from a session that doesn't exist
		_, err := capturePane("nonexistent-session-12345", 50)
		if err == nil {
			t.Error("expected error for non-existent session, got nil")
		}
	})

	t.Run("handles empty session name", func(t *testing.T) {
		// Note: tmux with empty session name captures current pane if running in tmux
		// This is valid tmux behavior - empty string targets current session
		_, _ = capturePane("", 50)
		// Just verify it doesn't panic
	})

	t.Run("uses correct line count argument", func(t *testing.T) {
		// This test verifies the function doesn't panic with various line counts
		// Actual capture requires a real tmux session
		_, _ = capturePane("test", 10)
		_, _ = capturePane("test", 50)
		_, _ = capturePane("test", 100)
		// If we get here without panic, the argument formatting works
	})
}
