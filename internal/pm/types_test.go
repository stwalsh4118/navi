package pm

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPMOutputJSONRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	original := PMOutput{
		Snapshots: []ProjectSnapshot{{
			ProjectName:      "navi",
			ProjectDir:       "/tmp/navi",
			HeadSHA:          "0123456789abcdef0123456789abcdef01234567",
			Branch:           "main",
			CommitsAhead:     2,
			Dirty:            true,
			CurrentPBIID:     "46",
			CurrentPBITitle:  "PM Engine",
			CurrentPBISource: "provider_hint",
			TaskCounts: TaskCounts{
				Total:      6,
				Done:       3,
				InProgress: 1,
			},
			SessionStatus: "working",
			LastActivity:  now,
			SessionCount:  2,
			PRNumber:      123,
		}},
		Events: []Event{{
			Type:        EventCommit,
			Timestamp:   now,
			ProjectName: "navi",
			ProjectDir:  "/tmp/navi",
			Payload: map[string]string{
				"commits": "abc123 Initial commit",
			},
		}},
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded PMOutput
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(decoded.Snapshots) != 1 {
		t.Fatalf("decoded snapshots len = %d, want 1", len(decoded.Snapshots))
	}
	if len(decoded.Events) != 1 {
		t.Fatalf("decoded events len = %d, want 1", len(decoded.Events))
	}

	s := decoded.Snapshots[0]
	if s.ProjectName != original.Snapshots[0].ProjectName {
		t.Fatalf("project name = %q, want %q", s.ProjectName, original.Snapshots[0].ProjectName)
	}
	if s.HeadSHA != original.Snapshots[0].HeadSHA {
		t.Fatalf("head sha = %q, want %q", s.HeadSHA, original.Snapshots[0].HeadSHA)
	}
	if s.TaskCounts.Done != original.Snapshots[0].TaskCounts.Done {
		t.Fatalf("done count = %d, want %d", s.TaskCounts.Done, original.Snapshots[0].TaskCounts.Done)
	}

	e := decoded.Events[0]
	if e.Type != EventCommit {
		t.Fatalf("event type = %q, want %q", e.Type, EventCommit)
	}
	if e.Payload["commits"] == "" {
		t.Fatal("expected event payload to contain commits")
	}
}

func TestEventTypeConstants(t *testing.T) {
	expected := map[EventType]string{
		EventTaskCompleted:       "task_completed",
		EventTaskStarted:         "task_started",
		EventCommit:              "commit",
		EventSessionStatusChange: "session_status_change",
		EventPBICompleted:        "pbi_completed",
		EventBranchCreated:       "branch_created",
		EventPRCreated:           "pr_created",
	}

	for eventType, want := range expected {
		if string(eventType) != want {
			t.Fatalf("event type %q = %q, want %q", eventType, string(eventType), want)
		}
	}
}
