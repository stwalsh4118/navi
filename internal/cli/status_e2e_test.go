package cli

import (
	"testing"

	"github.com/stwalsh4118/navi/internal/session"
)

func TestStatusDefaultOutput(t *testing.T) {
	tmpDir := t.TempDir()
	if err := writeStatus(tmpDir, session.Info{TmuxSession: "a", Status: session.StatusWorking}); err != nil {
		t.Fatalf("writeStatus failed: %v", err)
	}
	if err := writeStatus(tmpDir, session.Info{TmuxSession: "b", Status: session.StatusWaiting}); err != nil {
		t.Fatalf("writeStatus failed: %v", err)
	}

	origDir := session.StatusDir
	session.StatusDir = tmpDir
	t.Cleanup(func() { session.StatusDir = origDir })

	stdout, _ := captureOutput(t, func() {
		if code := RunStatus(nil); code != 0 {
			t.Fatalf("RunStatus code = %d, want 0", code)
		}
	})

	if stdout != "1 waiting\n" {
		t.Fatalf("stdout = %q, want %q", stdout, "1 waiting\n")
	}
}

func TestStatusVerboseOutput(t *testing.T) {
	tmpDir := t.TempDir()
	if err := writeStatus(tmpDir, session.Info{TmuxSession: "a", Status: session.StatusWorking}); err != nil {
		t.Fatalf("writeStatus failed: %v", err)
	}
	if err := writeStatus(tmpDir, session.Info{TmuxSession: "b", Status: session.StatusIdle}); err != nil {
		t.Fatalf("writeStatus failed: %v", err)
	}

	origDir := session.StatusDir
	session.StatusDir = tmpDir
	t.Cleanup(func() { session.StatusDir = origDir })

	stdout, _ := captureOutput(t, func() {
		if code := RunStatus([]string{"--verbose"}); code != 0 {
			t.Fatalf("RunStatus verbose code = %d, want 0", code)
		}
	})

	if stdout != "1 working, 1 idle\n" {
		t.Fatalf("stdout = %q, want %q", stdout, "1 working, 1 idle\n")
	}
}

func TestStatusTmuxFormat(t *testing.T) {
	tmpDir := t.TempDir()
	if err := writeStatus(tmpDir, session.Info{TmuxSession: "a", Status: session.StatusPermission}); err != nil {
		t.Fatalf("writeStatus failed: %v", err)
	}

	origDir := session.StatusDir
	session.StatusDir = tmpDir
	t.Cleanup(func() { session.StatusDir = origDir })

	stdout, _ := captureOutput(t, func() {
		if code := RunStatus([]string{"--format=tmux"}); code != 0 {
			t.Fatalf("RunStatus format=tmux code = %d, want 0", code)
		}
	})

	if stdout != "1 permission\n" {
		t.Fatalf("stdout = %q, want %q", stdout, "1 permission\n")
	}
}

func TestStatusEmptyOutput(t *testing.T) {
	tmpDir := t.TempDir()
	origDir := session.StatusDir
	session.StatusDir = tmpDir
	t.Cleanup(func() { session.StatusDir = origDir })

	stdout, _ := captureOutput(t, func() {
		if code := RunStatus(nil); code != 0 {
			t.Fatalf("RunStatus code = %d, want 0", code)
		}
	})

	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
}
