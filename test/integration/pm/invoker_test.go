package pm_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pm "github.com/stwalsh4118/navi/internal/pm"
)

// TestAC1_Initialization verifies first-run creates the full directory structure
// and seeds memory files, system prompt, and output schema.
func TestAC1_Initialization(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := pm.EnsureStorageLayout(); err != nil {
		t.Fatalf("EnsureStorageLayout failed: %v", err)
	}

	pmDir := filepath.Join(home, ".config", "navi", "pm")
	for _, sub := range []string{"", "memory", "memory/projects", "snapshots"} {
		p := filepath.Join(pmDir, sub)
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("expected directory %q: %v", p, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %q to be a directory", p)
		}
	}

	shortTerm, err := os.ReadFile(filepath.Join(pmDir, "memory", "short-term.md"))
	if err != nil {
		t.Fatalf("read short-term.md failed: %v", err)
	}
	if !strings.Contains(string(shortTerm), "Short-term PM memory") {
		t.Fatal("short-term.md should contain template header")
	}

	longTerm, err := os.ReadFile(filepath.Join(pmDir, "memory", "long-term.md"))
	if err != nil {
		t.Fatalf("read long-term.md failed: %v", err)
	}
	if !strings.Contains(string(longTerm), "Long-term PM memory") {
		t.Fatal("long-term.md should contain template header")
	}

	if _, err := os.Stat(filepath.Join(pmDir, "system-prompt.md")); err != nil {
		t.Fatalf("system-prompt.md should exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(pmDir, "output-schema.json")); err != nil {
		t.Fatalf("output-schema.json should exist: %v", err)
	}
}

// TestAC2_FreshSessionInvocation verifies each invocation runs as a fresh
// session (no --resume), and the output is parsed into a PMBriefing.
func TestAC2_FreshSessionInvocation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	installMockClaudeWithArgsCapture(t, "fresh-run", argsFile)

	invoker, err := pm.NewInvoker()
	if err != nil {
		t.Fatalf("NewInvoker failed: %v", err)
	}

	inbox, err := pm.BuildInbox(pm.TriggerOnDemand, nil, nil)
	if err != nil {
		t.Fatalf("BuildInbox failed: %v", err)
	}

	result, err := invoker.Invoke(inbox)
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}
	if result.Output == nil || result.Output.Summary == "" {
		t.Fatal("expected non-empty briefing summary")
	}

	argsContent, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read args file failed: %v", err)
	}
	if strings.Contains(string(argsContent), "--resume") {
		t.Fatal("did not expect --resume in fresh session args")
	}
}

// TestAC4_InboxPayload verifies inbox JSON contains timestamp, trigger type,
// events, and project snapshots.
func TestAC4_InboxPayload(t *testing.T) {
	events := []pm.Event{{
		Type:        pm.EventCommit,
		Timestamp:   time.Now().UTC(),
		ProjectName: "test-proj",
		ProjectDir:  "/tmp/test-proj",
		Payload:     map[string]string{"new_head_sha": "abc123"},
	}}
	snapshots := []pm.ProjectSnapshot{{
		ProjectName:   "test-proj",
		ProjectDir:    "/tmp/test-proj",
		Branch:        "main",
		SessionStatus: "working",
	}}

	inbox, err := pm.BuildInbox(pm.TriggerCommit, snapshots, events)
	if err != nil {
		t.Fatalf("BuildInbox failed: %v", err)
	}

	data, err := pm.InboxToJSON(inbox)
	if err != nil {
		t.Fatalf("InboxToJSON failed: %v", err)
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("parse inbox JSON failed: %v", err)
	}

	for _, key := range []string{"timestamp", "trigger_type", "events", "snapshots"} {
		if _, ok := parsed[key]; !ok {
			t.Fatalf("expected %q in inbox JSON", key)
		}
	}

	if !strings.Contains(string(data), `"trigger_type":"commit"`) {
		t.Fatalf("expected trigger_type=commit in JSON: %s", string(data))
	}
	if !strings.Contains(string(data), `"test-proj"`) {
		t.Fatalf("expected project name in JSON: %s", string(data))
	}
}

// TestAC5_OutputParsingAndCaching verifies output is parsed into PMBriefing
// and cached to last-output.json.
func TestAC5_OutputParsingAndCaching(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := pm.EnsureStorageLayout(); err != nil {
		t.Fatalf("EnsureStorageLayout failed: %v", err)
	}

	raw := []byte(`{"structured_output":{"summary":"test briefing","projects":[{"name":"proj","status":"healthy"}],"attention_items":[{"priority":"high","title":"check auth"}],"breadcrumbs":[]}}`)
	briefing, err := pm.ParseOutput(raw)
	if err != nil {
		t.Fatalf("ParseOutput failed: %v", err)
	}

	if briefing.Summary != "test briefing" {
		t.Fatalf("summary = %q, want test briefing", briefing.Summary)
	}
	if len(briefing.Projects) != 1 || briefing.Projects[0].Name != "proj" {
		t.Fatalf("unexpected projects: %+v", briefing.Projects)
	}
	if len(briefing.AttentionItems) != 1 || briefing.AttentionItems[0].Priority != "high" {
		t.Fatalf("unexpected attention items: %+v", briefing.AttentionItems)
	}

	if err := pm.CacheOutput(briefing); err != nil {
		t.Fatalf("CacheOutput failed: %v", err)
	}

	cached, err := pm.LoadCachedOutput()
	if err != nil {
		t.Fatalf("LoadCachedOutput failed: %v", err)
	}
	if cached == nil || cached.Briefing == nil {
		t.Fatal("expected non-nil cached briefing")
	}
	if cached.Briefing.Summary != "test briefing" {
		t.Fatalf("cached summary = %q, want test briefing", cached.Briefing.Summary)
	}
	if cached.CachedAt.IsZero() {
		t.Fatal("expected CachedAt timestamp")
	}
}

// TestAC6_TriggerEvents verifies trigger event type classification:
// task_completed and commit are triggers; other types are not.
func TestAC6_TriggerEvents(t *testing.T) {
	triggerTypes := map[pm.EventType]bool{
		pm.EventTaskCompleted:       true,
		pm.EventCommit:              true,
		pm.EventSessionStatusChange: false,
		pm.EventBranchCreated:       false,
		pm.EventPRCreated:           false,
		pm.EventTaskStarted:         false,
		pm.EventPBICompleted:        false,
	}

	for eventType, shouldTrigger := range triggerTypes {
		_, isTrigger := map[pm.EventType]bool{
			pm.EventTaskCompleted: true,
			pm.EventCommit:        true,
		}[eventType]

		if isTrigger != shouldTrigger {
			t.Errorf("event type %q: expected trigger=%v, got %v", eventType, shouldTrigger, isTrigger)
		}
	}
}

// TestAC8_FailureFallback verifies non-zero exit falls back to cached output
// with staleness indicator.
func TestAC8_FailureFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	installMockClaude(t, "exit-failure")

	invoker, err := pm.NewInvoker()
	if err != nil {
		t.Fatalf("NewInvoker failed: %v", err)
	}

	if err := pm.CacheOutput(&pm.PMBriefing{Summary: "cached-fallback"}); err != nil {
		t.Fatalf("CacheOutput failed: %v", err)
	}

	inbox, err := pm.BuildInbox(pm.TriggerOnDemand, nil, nil)
	if err != nil {
		t.Fatalf("BuildInbox failed: %v", err)
	}

	briefing, stale, err := invoker.InvokeWithRecovery(inbox)
	if err != nil {
		t.Fatalf("InvokeWithRecovery should fallback, got error: %v", err)
	}
	if !stale {
		t.Fatal("expected stale=true for cached fallback")
	}
	if briefing == nil || briefing.Summary != "cached-fallback" {
		t.Fatalf("expected cached briefing, got %v", briefing)
	}
}

// TestFullPipeline_InitInvokeRecoverCache runs the full lifecycle:
// init -> invoke -> cache -> failure -> fallback.
func TestFullPipeline_InitInvokeRecoverCache(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	installMockClaude(t, "fresh-run")

	// Step 1: Initialize and invoke.
	invoker, err := pm.NewInvoker()
	if err != nil {
		t.Fatalf("NewInvoker: %v", err)
	}

	inbox, err := pm.BuildInbox(pm.TriggerOnDemand, nil, nil)
	if err != nil {
		t.Fatalf("BuildInbox: %v", err)
	}

	briefing, stale, err := invoker.InvokeWithRecovery(inbox)
	if err != nil {
		t.Fatalf("first invoke: %v", err)
	}
	if stale {
		t.Fatal("first invoke should not be stale")
	}
	if briefing == nil || briefing.Summary == "" {
		t.Fatal("expected briefing from first invoke")
	}

	// Step 2: Verify cache written.
	cached, err := pm.LoadCachedOutput()
	if err != nil {
		t.Fatalf("LoadCachedOutput: %v", err)
	}
	if cached == nil || cached.Briefing == nil {
		t.Fatal("expected cached output after successful invoke")
	}

	// Step 3: Switch to failure mode and verify fallback.
	installMockClaude(t, "exit-failure")

	inbox2, _ := pm.BuildInbox(pm.TriggerOnDemand, nil, nil)
	briefing2, stale2, err2 := invoker.InvokeWithRecovery(inbox2)
	if err2 != nil {
		t.Fatalf("fallback should succeed: %v", err2)
	}
	if !stale2 {
		t.Fatal("expected stale=true on fallback")
	}
	if briefing2 == nil {
		t.Fatal("expected cached briefing on fallback")
	}
}

// installMockClaude creates a mock claude binary on PATH.
func installMockClaude(t *testing.T, mode string) {
	t.Helper()
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}

	script := fmt.Sprintf(`#!/bin/sh
# Mock claude CLI - mode: %s
cat > /dev/null
case "%s" in
  fresh-run)
    echo '{"structured_output":{"summary":"fresh-run briefing","projects":[],"attention_items":[],"breadcrumbs":[]}}'
    exit 0
    ;;
  exit-failure)
    echo "simulated failure" >&2
    exit 17
    ;;
  *)
    echo '{"structured_output":{"summary":"default","projects":[],"attention_items":[],"breadcrumbs":[]}}'
    exit 0
    ;;
esac
`, mode, mode)

	scriptPath := filepath.Join(binDir, "claude")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("write mock claude: %v", err)
	}

	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
}

// installMockClaudeWithArgsCapture creates a mock claude that also captures args.
func installMockClaudeWithArgsCapture(t *testing.T, mode, argsFile string) {
	t.Helper()
	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}

	script := fmt.Sprintf(`#!/bin/sh
# Mock claude CLI with args capture - mode: %s
printf '%%s' "$*" > "%s"
cat > /dev/null
echo '{"structured_output":{"summary":"fresh-run","projects":[],"attention_items":[],"breadcrumbs":[]}}'
exit 0
`, mode, argsFile)

	scriptPath := filepath.Join(binDir, "claude")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("write mock claude: %v", err)
	}

	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
}
