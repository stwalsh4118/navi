package pm

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAppendAndReadEvents(t *testing.T) {
	tempDir := t.TempDir()
	originalPath := eventLogPath
	t.Cleanup(func() {
		eventLogPath = originalPath
	})

	eventLogPath = filepath.Join(tempDir, "events.jsonl")

	events := []Event{{
		Type:        EventCommit,
		Timestamp:   time.Now().UTC(),
		ProjectName: "navi",
		ProjectDir:  "/tmp/navi",
		Payload: map[string]string{
			"commits": "abc123 test",
		},
	}}

	if err := AppendEvents(events); err != nil {
		t.Fatalf("AppendEvents failed: %v", err)
	}

	readBack, err := ReadEvents()
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}
	if len(readBack) != 1 {
		t.Fatalf("event count = %d, want 1", len(readBack))
	}
	if readBack[0].Type != EventCommit {
		t.Fatalf("event type = %q, want %q", readBack[0].Type, EventCommit)
	}
}

func TestReadEventsSkipsMalformedLines(t *testing.T) {
	tempDir := t.TempDir()
	originalPath := eventLogPath
	t.Cleanup(func() {
		eventLogPath = originalPath
	})

	eventLogPath = filepath.Join(tempDir, "events.jsonl")
	content := "{bad json}\n{\"type\":\"commit\",\"timestamp\":\"2026-02-17T00:00:00Z\",\"project_name\":\"navi\",\"project_dir\":\"/tmp/navi\"}\n"
	if err := os.WriteFile(resolveEventLogPath(), []byte(content), 0644); err != nil {
		t.Fatalf("write fixture failed: %v", err)
	}

	events, err := ReadEvents()
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
}

func TestPruneEvents(t *testing.T) {
	tempDir := t.TempDir()
	originalPath := eventLogPath
	t.Cleanup(func() {
		eventLogPath = originalPath
	})

	eventLogPath = filepath.Join(tempDir, "events.jsonl")
	oldEvent := Event{Type: EventCommit, Timestamp: time.Now().UTC().Add(-25 * time.Hour), ProjectName: "old", ProjectDir: "/tmp/old"}
	recentEvent := Event{Type: EventTaskCompleted, Timestamp: time.Now().UTC().Add(-1 * time.Hour), ProjectName: "new", ProjectDir: "/tmp/new"}
	if err := AppendEvents([]Event{oldEvent, recentEvent}); err != nil {
		t.Fatalf("append fixture events failed: %v", err)
	}

	if err := PruneEvents(); err != nil {
		t.Fatalf("PruneEvents failed: %v", err)
	}

	events, err := ReadEvents()
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
	if events[0].ProjectName != "new" {
		t.Fatalf("retained project = %q, want new", events[0].ProjectName)
	}
}

func TestReadEventsMissingFile(t *testing.T) {
	tempDir := t.TempDir()
	originalPath := eventLogPath
	t.Cleanup(func() {
		eventLogPath = originalPath
	})

	eventLogPath = filepath.Join(tempDir, "does-not-exist.jsonl")
	events, err := ReadEvents()
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("event count = %d, want 0", len(events))
	}
}
