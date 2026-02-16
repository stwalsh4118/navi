package session

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNotifyHookSessionIDRollover(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is required for notify hook tests")
	}

	home := t.TempDir()
	statusDir := filepath.Join(home, ".claude-sessions")
	if err := os.MkdirAll(statusDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	sessionName := resolveNotifySessionName(t, home)

	statusPath := filepath.Join(statusDir, sessionName+".json")
	initial := map[string]any{
		"tmux_session": sessionName,
		"session_id":   "old-main-sid",
		"status":       "idle",
		"message":      "",
		"cwd":          "/tmp",
		"timestamp":    1,
	}
	writeJSONFile(t, statusPath, initial)

	payload := map[string]any{
		"hook_event_name": "UserPromptSubmit",
		"session_id":      "new-main-sid",
	}
	out := runNotifyHook(t, home, statusPath, "working", payload)

	if out["session_id"] != "new-main-sid" {
		t.Fatalf("session_id = %v, want %q", out["session_id"], "new-main-sid")
	}
	if out["status"] != "working" {
		t.Fatalf("status = %v, want %q", out["status"], "working")
	}
}

func TestNotifyHookSuppressesMismatchedPostToolUse(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is required for notify hook tests")
	}

	home := t.TempDir()
	statusDir := filepath.Join(home, ".claude-sessions")
	if err := os.MkdirAll(statusDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	sessionName := resolveNotifySessionName(t, home)

	statusPath := filepath.Join(statusDir, sessionName+".json")
	initial := map[string]any{
		"tmux_session": sessionName,
		"session_id":   "main-sid",
		"status":       "idle",
		"message":      "baseline",
		"cwd":          "/tmp",
		"timestamp":    1,
	}
	writeJSONFile(t, statusPath, initial)

	payload := map[string]any{
		"hook_event_name": "PostToolUse",
		"session_id":      "teammate-sid",
	}
	out := runNotifyHook(t, home, statusPath, "working", payload)

	if out["session_id"] != "main-sid" {
		t.Fatalf("session_id = %v, want %q", out["session_id"], "main-sid")
	}
	if out["status"] != "idle" {
		t.Fatalf("status = %v, want %q", out["status"], "idle")
	}
	if out["message"] != "baseline" {
		t.Fatalf("message = %v, want %q", out["message"], "baseline")
	}
}

func runNotifyHook(t *testing.T, home, statusPath, status string, payload map[string]any) map[string]any {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	scriptPath := filepath.Join(repoRoot, "hooks", "notify.sh")

	stdinPayload, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	cmd := exec.Command("bash", scriptPath, status)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "HOME="+home)
	cmd.Stdin = bytes.NewReader(stdinPayload)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("notify.sh failed: %v, output: %s", err, string(out))
	}

	data, err := os.ReadFile(statusPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	return parsed
}

func writeJSONFile(t *testing.T, path string, data map[string]any) {
	t.Helper()

	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func resolveNotifySessionName(t *testing.T, home string) string {
	t.Helper()

	cmd := exec.Command("bash", "-lc", "tmux display-message -p '#{session_name}' 2>/dev/null || echo unknown")
	cmd.Env = append(os.Environ(), "HOME="+home)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("resolve session name failed: %v", err)
	}
	name := string(bytes.TrimSpace(out))
	if name == "" {
		return "unknown"
	}
	return name
}
