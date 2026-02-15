package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stwalsh4118/navi/internal/session"
)

func TestFormatSummaryDefaultPriorityOnly(t *testing.T) {
	counts := map[string]int{
		session.StatusWorking:    3,
		session.StatusWaiting:    1,
		session.StatusPermission: 2,
		session.StatusIdle:       1,
	}

	got := formatSummary(counts, false)
	want := "1 waiting, 2 permission"
	if got != want {
		t.Fatalf("formatSummary(default) = %q, want %q", got, want)
	}
}

func TestFormatSummaryDefaultNoPriority(t *testing.T) {
	counts := map[string]int{session.StatusWorking: 3}
	if got := formatSummary(counts, false); got != "" {
		t.Fatalf("formatSummary(default) = %q, want empty", got)
	}
}

func TestFormatSummaryVerbose(t *testing.T) {
	counts := map[string]int{
		session.StatusWorking:    3,
		session.StatusWaiting:    1,
		session.StatusPermission: 2,
		session.StatusIdle:       1,
	}

	got := formatSummary(counts, true)
	want := "3 working, 1 waiting, 2 permission, 1 idle"
	if got != want {
		t.Fatalf("formatSummary(verbose) = %q, want %q", got, want)
	}
}

func TestRunStatusIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	if err := writeStatus(tmpDir, session.Info{TmuxSession: "a", Status: session.StatusWorking}); err != nil {
		t.Fatalf("writeStatus failed: %v", err)
	}
	if err := writeStatus(tmpDir, session.Info{TmuxSession: "b", Status: session.StatusWaiting}); err != nil {
		t.Fatalf("writeStatus failed: %v", err)
	}
	if err := writeStatus(tmpDir, session.Info{TmuxSession: "c", Status: session.StatusPermission}); err != nil {
		t.Fatalf("writeStatus failed: %v", err)
	}

	origStatusDir := session.StatusDir
	session.StatusDir = tmpDir
	t.Cleanup(func() { session.StatusDir = origStatusDir })

	stdout, stderr := captureOutput(t, func() {
		code := RunStatus(nil)
		if code != 0 {
			t.Fatalf("RunStatus(default) code = %d, want 0", code)
		}
	})

	if stdout != "1 waiting, 1 permission\n" {
		t.Fatalf("RunStatus(default) stdout = %q, want %q", stdout, "1 waiting, 1 permission\n")
	}
	if stderr != "" {
		t.Fatalf("RunStatus(default) stderr = %q, want empty", stderr)
	}

	stdout, stderr = captureOutput(t, func() {
		code := RunStatus([]string{"--verbose"})
		if code != 0 {
			t.Fatalf("RunStatus(verbose) code = %d, want 0", code)
		}
	})

	if stdout != "1 working, 1 waiting, 1 permission\n" {
		t.Fatalf("RunStatus(verbose) stdout = %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("RunStatus(verbose) stderr = %q, want empty", stderr)
	}

	stdout, stderr = captureOutput(t, func() {
		code := RunStatus([]string{"--format=tmux"})
		if code != 0 {
			t.Fatalf("RunStatus(format=tmux) code = %d, want 0", code)
		}
	})

	if stdout != "1 waiting, 1 permission\n" {
		t.Fatalf("RunStatus(format=tmux) stdout = %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("RunStatus(format=tmux) stderr = %q, want empty", stderr)
	}
}

func TestRunStatusNoPriorityPrintsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	if err := writeStatus(tmpDir, session.Info{TmuxSession: "a", Status: session.StatusWorking}); err != nil {
		t.Fatalf("writeStatus failed: %v", err)
	}

	origStatusDir := session.StatusDir
	session.StatusDir = tmpDir
	t.Cleanup(func() { session.StatusDir = origStatusDir })

	stdout, _ := captureOutput(t, func() {
		code := RunStatus(nil)
		if code != 0 {
			t.Fatalf("RunStatus(default) code = %d, want 0", code)
		}
	})

	if stdout != "" {
		t.Fatalf("RunStatus(default no-priority) stdout = %q, want empty", stdout)
	}
}

func writeStatus(dir string, info session.Info) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, info.TmuxSession+".json"), data, 0644)
}

func captureOutput(t *testing.T, fn func()) (string, string) {
	t.Helper()
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdout pipe failed: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("stderr pipe failed: %v", err)
	}

	os.Stdout = wOut
	os.Stderr = wErr

	fn()

	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var outBuf bytes.Buffer
	_, _ = outBuf.ReadFrom(rOut)
	var errBuf bytes.Buffer
	_, _ = errBuf.ReadFrom(rErr)

	return outBuf.String(), errBuf.String()
}
