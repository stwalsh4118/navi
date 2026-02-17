package pm

import "testing"

func TestDiffSnapshots(t *testing.T) {
	originalCommitsBetween := commitsBetweenFunc
	t.Cleanup(func() {
		commitsBetweenFunc = originalCommitsBetween
	})

	commitsBetweenFunc = func(_, _, _ string) []string {
		return []string{"abc123 commit one", "def456 commit two"}
	}

	oldSnapshot := ProjectSnapshot{
		ProjectName:     "navi",
		ProjectDir:      "/tmp/navi",
		HeadSHA:         "oldsha",
		Branch:          "main",
		CurrentPBIID:    "PBI-46",
		CurrentPBITitle: "PM Engine",
		TaskCounts: TaskCounts{
			Total:      4,
			Done:       1,
			InProgress: 0,
		},
		SessionStatus: "working",
	}

	newSnapshot := oldSnapshot
	newSnapshot.HeadSHA = "newsha"
	newSnapshot.Branch = "feature/pm"
	newSnapshot.SessionStatus = "waiting"
	newSnapshot.TaskCounts.Done = 4
	newSnapshot.TaskCounts.InProgress = 1
	newSnapshot.PRNumber = 12

	events := DiffSnapshots(oldSnapshot, newSnapshot)
	if len(events) != 7 {
		t.Fatalf("event count = %d, want 7", len(events))
	}

	eventSet := make(map[EventType]Event)
	for _, event := range events {
		eventSet[event.Type] = event
	}

	if eventSet[EventCommit].Payload["commits"] == "" {
		t.Fatal("expected commit payload")
	}
	if eventSet[EventTaskCompleted].Payload["new_done"] != "4" {
		t.Fatalf("new_done = %q, want 4", eventSet[EventTaskCompleted].Payload["new_done"])
	}
	if eventSet[EventTaskStarted].Payload["new_in_progress"] != "1" {
		t.Fatalf("new_in_progress = %q, want 1", eventSet[EventTaskStarted].Payload["new_in_progress"])
	}
	if eventSet[EventSessionStatusChange].Payload["new_status"] != "waiting" {
		t.Fatalf("new_status = %q, want waiting", eventSet[EventSessionStatusChange].Payload["new_status"])
	}
	if eventSet[EventPBICompleted].Payload["pbi_id"] != "PBI-46" {
		t.Fatalf("pbi_id = %q, want PBI-46", eventSet[EventPBICompleted].Payload["pbi_id"])
	}
	if eventSet[EventBranchCreated].Payload["new_branch"] != "feature/pm" {
		t.Fatalf("new_branch = %q, want feature/pm", eventSet[EventBranchCreated].Payload["new_branch"])
	}
	if eventSet[EventPRCreated].Payload["pr_number"] != "12" {
		t.Fatalf("pr_number = %q, want 12", eventSet[EventPRCreated].Payload["pr_number"])
	}
}

func TestDiffSnapshotsNoEventsOnFirstSnapshot(t *testing.T) {
	events := DiffSnapshots(ProjectSnapshot{}, ProjectSnapshot{ProjectDir: "/tmp/navi"})
	if len(events) != 0 {
		t.Fatalf("event count = %d, want 0", len(events))
	}
}

func TestDiffSnapshotsNoCommitLookupWhenHEADUnchanged(t *testing.T) {
	called := false
	originalCommitsBetween := commitsBetweenFunc
	t.Cleanup(func() {
		commitsBetweenFunc = originalCommitsBetween
	})

	commitsBetweenFunc = func(_, _, _ string) []string {
		called = true
		return nil
	}

	oldSnapshot := ProjectSnapshot{ProjectDir: "/tmp/navi", HeadSHA: "same"}
	newSnapshot := ProjectSnapshot{ProjectDir: "/tmp/navi", HeadSHA: "same"}

	_ = DiffSnapshots(oldSnapshot, newSnapshot)
	if called {
		t.Fatal("expected commit lookup not to be called")
	}
}
