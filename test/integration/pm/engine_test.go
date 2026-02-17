package pm_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pm "github.com/stwalsh4118/navi/internal/pm"
	"github.com/stwalsh4118/navi/internal/session"
	"github.com/stwalsh4118/navi/internal/task"
)

func TestEngineIntegration(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	traceFile := installGitTracer(t)

	repoA := createRepo(t, home, "repo-a")
	repoB := createRepo(t, home, "repo-b")

	if err := os.WriteFile(filepath.Join(repoA, "README.md"), []byte("first\n"), 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	runGit(t, repoA, "add", "README.md")
	runGit(t, repoA, "commit", "-m", "initial")
	if err := os.WriteFile(filepath.Join(repoA, "README.md"), []byte("first\ndirty\n"), 0644); err != nil {
		t.Fatalf("write dirty file failed: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repoB, "README.md"), []byte("other\n"), 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	runGit(t, repoB, "add", "README.md")
	runGit(t, repoB, "commit", "-m", "initial")

	now := time.Now().Unix()
	sessions := []session.Info{
		{TmuxSession: "s1", CWD: repoA, Status: session.StatusWorking, Timestamp: now - 20},
		{TmuxSession: "s2", CWD: repoA, Status: session.StatusWaiting, Timestamp: now - 5},
		{TmuxSession: "s3", CWD: repoB, Status: session.StatusIdle, Timestamp: now - 1},
	}

	taskResults := map[string]*task.ProviderResult{
		repoA: {
			Groups: []task.TaskGroup{{
				ID:    "PBI-46",
				Title: "PM Engine",
				Tasks: []task.Task{{
					ID: "46-1", Status: "done",
				}, {
					ID: "46-2", Status: "in_progress",
				}, {
					ID: "46-3", Status: "todo",
				}},
			}},
		},
		repoB: {
			Groups: []task.TaskGroup{{ID: "PBI-47", Title: "TUI PM View", Tasks: []task.Task{{ID: "47-1", Status: "todo"}}}},
		},
	}

	engine := pm.NewEngine()
	clearTraceFile(t, traceFile)
	first, err := engine.Run(sessions, taskResults)
	if err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	if len(first.Snapshots) != 2 {
		t.Fatalf("snapshot count = %d, want 2", len(first.Snapshots))
	}
	if len(first.Events) != 0 {
		t.Fatalf("first run event count = %d, want 0", len(first.Events))
	}
	if !traceContains(traceFile, "rev-parse HEAD") {
		t.Fatal("expected engine to call git rev-parse HEAD")
	}

	snapshot, found := findSnapshotByDir(first.Snapshots, repoA)
	if !found {
		t.Fatalf("missing snapshot for %q", repoA)
	}
	if snapshot.HeadSHA == "" || snapshot.Branch == "" {
		t.Fatalf("expected head sha and branch populated, got sha=%q branch=%q", snapshot.HeadSHA, snapshot.Branch)
	}
	if snapshot.CommitsAhead != 0 {
		t.Fatalf("commits ahead = %d, want 0", snapshot.CommitsAhead)
	}
	if !snapshot.Dirty {
		t.Fatal("expected dirty snapshot for repoA")
	}
	if snapshot.CurrentPBIID != "PBI-46" || snapshot.CurrentPBITitle != "PM Engine" {
		t.Fatalf("unexpected pbi data: id=%q title=%q", snapshot.CurrentPBIID, snapshot.CurrentPBITitle)
	}
	if snapshot.TaskCounts.Total != 3 || snapshot.TaskCounts.Done != 1 || snapshot.TaskCounts.InProgress != 1 {
		t.Fatalf("unexpected task counts: %+v", snapshot.TaskCounts)
	}
	if snapshot.SessionStatus != session.StatusWaiting {
		t.Fatalf("session status = %q, want %q", snapshot.SessionStatus, session.StatusWaiting)
	}
	if snapshot.LastActivity.Unix() != now-5 {
		t.Fatalf("last activity = %d, want %d", snapshot.LastActivity.Unix(), now-5)
	}
	if snapshot.SessionCount != 2 {
		t.Fatalf("session count = %d, want 2", snapshot.SessionCount)
	}

	repoBSnapshot, found := findSnapshotByDir(first.Snapshots, repoB)
	if !found {
		t.Fatalf("missing snapshot for %q", repoB)
	}
	if repoBSnapshot.SessionCount != 1 {
		t.Fatalf("repoB session count = %d, want 1", repoBSnapshot.SessionCount)
	}

	clearTraceFile(t, traceFile)
	unchanged, err := engine.Run(sessions, taskResults)
	if err != nil {
		t.Fatalf("unchanged run failed: %v", err)
	}
	if hasEventType(unchanged.Events, pm.EventCommit) {
		t.Fatal("did not expect commit event when HEAD is unchanged")
	}
	if traceContains(traceFile, "log --oneline") {
		t.Fatal("did not expect git log --oneline call when HEAD is unchanged")
	}

	if err := os.WriteFile(filepath.Join(repoA, "README.md"), []byte("first\nsecond\n"), 0644); err != nil {
		t.Fatalf("update file failed: %v", err)
	}
	runGit(t, repoA, "add", "README.md")
	runGit(t, repoA, "commit", "-m", "second")
	runGit(t, repoA, "checkout", "-b", "feature/pm")
	sessions[1].Status = session.StatusIdle
	taskResults[repoA] = &task.ProviderResult{
		Groups: []task.TaskGroup{{
			ID:    "PBI-46",
			Title: "PM Engine",
			Tasks: []task.Task{{ID: "46-1", Status: "done"}, {ID: "46-2", Status: "done"}, {ID: "46-3", Status: "done"}},
		}},
	}

	clearTraceFile(t, traceFile)
	second, err := engine.Run(sessions, taskResults)
	if err != nil {
		t.Fatalf("second run failed: %v", err)
	}
	if !hasEventType(second.Events, pm.EventCommit) {
		t.Fatal("expected commit event on second run")
	}
	if !hasEventType(second.Events, pm.EventTaskCompleted) {
		t.Fatal("expected task_completed event on second run")
	}
	if !hasEventType(second.Events, pm.EventPBICompleted) {
		t.Fatal("expected pbi_completed event on second run")
	}
	if !hasEventType(second.Events, pm.EventBranchCreated) {
		t.Fatal("expected branch_created event on second run")
	}
	if !hasEventType(second.Events, pm.EventSessionStatusChange) {
		t.Fatal("expected session_status_change event on second run")
	}
	if !traceContains(traceFile, "log --oneline") {
		t.Fatal("expected git log --oneline call when HEAD changes")
	}

	stored, err := pm.ReadEvents()
	if err != nil {
		t.Fatalf("read events failed: %v", err)
	}
	if len(stored) == 0 {
		t.Fatal("expected persisted events")
	}
	for _, event := range stored {
		if event.Timestamp.IsZero() {
			t.Fatal("expected persisted event timestamp")
		}
		if event.ProjectName == "" {
			t.Fatal("expected persisted event project name")
		}
	}
}

func TestDiffSnapshotsAllEventTypes(t *testing.T) {
	oldSnapshot := pm.ProjectSnapshot{
		ProjectName:     "repo",
		ProjectDir:      "/tmp/repo",
		HeadSHA:         "old",
		Branch:          "main",
		CurrentPBIID:    "PBI-46",
		CurrentPBITitle: "PM Engine",
		TaskCounts: pm.TaskCounts{
			Total:      3,
			Done:       1,
			InProgress: 0,
		},
		SessionStatus: "working",
		PRNumber:      0,
	}
	newSnapshot := oldSnapshot
	newSnapshot.HeadSHA = "new"
	newSnapshot.Branch = "feature/pm"
	newSnapshot.TaskCounts.Done = 3
	newSnapshot.TaskCounts.InProgress = 1
	newSnapshot.SessionStatus = "waiting"
	newSnapshot.PRNumber = 99

	events := pm.DiffSnapshots(oldSnapshot, newSnapshot)
	wantTypes := []pm.EventType{
		pm.EventCommit,
		pm.EventTaskCompleted,
		pm.EventTaskStarted,
		pm.EventSessionStatusChange,
		pm.EventPBICompleted,
		pm.EventBranchCreated,
		pm.EventPRCreated,
	}
	for _, eventType := range wantTypes {
		if !hasEventType(events, eventType) {
			t.Fatalf("missing event type %q", eventType)
		}
	}
}

func TestEventPruning(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	oldEvent := pm.Event{
		Type:        pm.EventCommit,
		Timestamp:   time.Now().UTC().Add(-25 * time.Hour),
		ProjectName: "old",
		ProjectDir:  "/tmp/old",
	}
	newEvent := pm.Event{
		Type:        pm.EventTaskCompleted,
		Timestamp:   time.Now().UTC(),
		ProjectName: "new",
		ProjectDir:  "/tmp/new",
	}

	if err := pm.AppendEvents([]pm.Event{oldEvent}); err != nil {
		t.Fatalf("append old event failed: %v", err)
	}
	if err := pm.AppendEvents([]pm.Event{newEvent}); err != nil {
		t.Fatalf("append new event failed: %v", err)
	}

	events, err := pm.ReadEvents()
	if err != nil {
		t.Fatalf("read events failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
	if events[0].ProjectName != "new" {
		t.Fatalf("retained project = %q, want new", events[0].ProjectName)
	}
}

func TestEnginePerformanceBudget(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	engine := pm.NewEngine()
	sessions := make([]session.Info, 0, 5)
	for i := 0; i < 5; i++ {
		dir := filepath.Join(home, fmt.Sprintf("proj-%d", i))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}
		sessions = append(sessions, session.Info{
			TmuxSession: fmt.Sprintf("s-%d", i),
			CWD:         dir,
			Status:      session.StatusWorking,
			Timestamp:   time.Now().Unix(),
		})
	}

	start := time.Now()
	if _, err := engine.Run(sessions, nil); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("performance budget exceeded: %v", elapsed)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}
}

func createRepo(t *testing.T, parentDir, name string) string {
	t.Helper()

	repo := filepath.Join(parentDir, name)
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatalf("mkdir repo failed: %v", err)
	}

	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.email", "pm@example.com")
	runGit(t, repo, "config", "user.name", "PM Tester")
	runGit(t, repo, "config", "commit.gpgsign", "false")

	return repo
}

func findSnapshotByDir(snapshots []pm.ProjectSnapshot, dir string) (pm.ProjectSnapshot, bool) {
	for _, snapshot := range snapshots {
		if snapshot.ProjectDir == dir {
			return snapshot, true
		}
	}
	return pm.ProjectSnapshot{}, false
}

func hasEventType(events []pm.Event, eventType pm.EventType) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}

func installGitTracer(t *testing.T) string {
	t.Helper()

	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("locate git failed: %v", err)
	}

	binDir := t.TempDir()
	traceFile := filepath.Join(t.TempDir(), "git-trace.log")
	scriptPath := filepath.Join(binDir, "git")
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\n' \"$*\" >> \"$PM_GIT_TRACE\"\nexec %s \"$@\"\n", realGit)
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("write git tracer failed: %v", err)
	}

	t.Setenv("PM_GIT_TRACE", traceFile)
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	return traceFile
}

func clearTraceFile(t *testing.T, traceFile string) {
	t.Helper()
	if err := os.WriteFile(traceFile, nil, 0644); err != nil {
		t.Fatalf("clear trace file failed: %v", err)
	}
}

func traceContains(traceFile, pattern string) bool {
	data, err := os.ReadFile(traceFile)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, pattern) {
			return true
		}
	}
	return false
}
