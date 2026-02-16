package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadStatusFiles(t *testing.T) {
	t.Run("non-existent directory returns empty without error", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "does-not-exist")
		sessions, err := ReadStatusFiles(dir)
		if err != nil {
			t.Fatalf("ReadStatusFiles() error = %v, want nil", err)
		}
		if len(sessions) != 0 {
			t.Fatalf("len(sessions) = %d, want 0", len(sessions))
		}
	})

	t.Run("empty directory returns empty slice", func(t *testing.T) {
		dir := t.TempDir()
		sessions, err := ReadStatusFiles(dir)
		if err != nil {
			t.Fatalf("ReadStatusFiles() error = %v, want nil", err)
		}
		if len(sessions) != 0 {
			t.Fatalf("len(sessions) = %d, want 0", len(sessions))
		}
	})

	t.Run("reads valid JSON and skips malformed/non-json files", func(t *testing.T) {
		dir := t.TempDir()

		validJSON := `{"tmux_session":"a","status":"waiting","message":"hi","cwd":"/tmp","timestamp":123}`
		if err := os.WriteFile(filepath.Join(dir, "a.json"), []byte(validJSON), 0644); err != nil {
			t.Fatalf("WriteFile valid JSON failed: %v", err)
		}

		if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte("not-json"), 0644); err != nil {
			t.Fatalf("WriteFile malformed JSON failed: %v", err)
		}

		if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignored"), 0644); err != nil {
			t.Fatalf("WriteFile non-json failed: %v", err)
		}

		sessions, err := ReadStatusFiles(dir)
		if err != nil {
			t.Fatalf("ReadStatusFiles() error = %v, want nil", err)
		}

		if len(sessions) != 1 {
			t.Fatalf("len(sessions) = %d, want 1", len(sessions))
		}

		if sessions[0].TmuxSession != "a" {
			t.Fatalf("TmuxSession = %q, want %q", sessions[0].TmuxSession, "a")
		}
		if sessions[0].Status != StatusWaiting {
			t.Fatalf("Status = %q, want %q", sessions[0].Status, StatusWaiting)
		}
	})
}
