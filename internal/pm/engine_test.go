package pm

import (
	"path/filepath"
	"testing"

	"github.com/stwalsh4118/navi/internal/git"
	"github.com/stwalsh4118/navi/internal/session"
	"github.com/stwalsh4118/navi/internal/task"
)

func TestEngineRunFirstAndSecondCycle(t *testing.T) {
	tempDir := t.TempDir()
	originalGitInfo := gitInfoFunc
	originalHeadSHA := getHeadSHAFunc
	originalCommitsBetween := commitsBetweenFunc
	originalEventLogPath := eventLogPath
	t.Cleanup(func() {
		gitInfoFunc = originalGitInfo
		getHeadSHAFunc = originalHeadSHA
		commitsBetweenFunc = originalCommitsBetween
		eventLogPath = originalEventLogPath
	})

	eventLogPath = filepath.Join(tempDir, "events.jsonl")

	headSHA := "oldsha"
	branch := "main"
	prNum := 0

	gitInfoFunc = func(_ string) *git.Info {
		return &git.Info{Branch: branch, Ahead: 1, Dirty: false, PRNum: prNum}
	}
	getHeadSHAFunc = func(_ string) string { return headSHA }
	commitsBetweenFunc = func(_, _, _ string) []string { return []string{"abc123 commit"} }

	engine := NewEngine()
	sessions := []session.Info{{TmuxSession: "s1", CWD: tempDir, Status: session.StatusWorking, Timestamp: 10}}
	firstTaskResults := map[string]*task.ProviderResult{
		tempDir: {
			Groups: []task.TaskGroup{{ID: "PBI-46", Title: "PM Engine", Tasks: []task.Task{{ID: "46-1", Status: "in_progress"}}}},
		},
	}

	firstOutput, err := engine.Run(sessions, firstTaskResults)
	if err != nil {
		t.Fatalf("first engine run failed: %v", err)
	}
	if len(firstOutput.Snapshots) != 1 {
		t.Fatalf("first snapshots = %d, want 1", len(firstOutput.Snapshots))
	}
	if len(firstOutput.Events) != 0 {
		t.Fatalf("first events = %d, want 0", len(firstOutput.Events))
	}

	headSHA = "newsha"
	branch = "feature/pm"
	prNum = 42
	secondTaskResults := map[string]*task.ProviderResult{
		tempDir: {
			Groups: []task.TaskGroup{{ID: "PBI-46", Title: "PM Engine", Tasks: []task.Task{{ID: "46-1", Status: "done"}}}},
		},
	}
	sessions[0].Status = session.StatusWaiting

	secondOutput, err := engine.Run(sessions, secondTaskResults)
	if err != nil {
		t.Fatalf("second engine run failed: %v", err)
	}
	if len(secondOutput.Events) == 0 {
		t.Fatal("expected events on second run")
	}
}

func TestEngineRunNoSessions(t *testing.T) {
	originalEventLogPath := eventLogPath
	tempDir := t.TempDir()
	t.Cleanup(func() {
		eventLogPath = originalEventLogPath
	})
	eventLogPath = filepath.Join(tempDir, "events.jsonl")

	engine := NewEngine()
	out, err := engine.Run(nil, nil)
	if err != nil {
		t.Fatalf("engine run failed: %v", err)
	}
	if len(out.Snapshots) != 0 {
		t.Fatalf("snapshots = %d, want 0", len(out.Snapshots))
	}
	if len(out.Events) != 0 {
		t.Fatalf("events = %d, want 0", len(out.Events))
	}
}
