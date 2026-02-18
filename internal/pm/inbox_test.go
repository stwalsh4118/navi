package pm

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestBuildInboxPopulatesFields(t *testing.T) {
	now := time.Now().UTC()

	events := []Event{{
		Type:        EventCommit,
		Timestamp:   now,
		ProjectName: "navi",
		ProjectDir:  "/tmp/navi",
	}}
	snapshots := []ProjectSnapshot{{
		ProjectName: "navi",
		ProjectDir:  "/tmp/navi",
	}}

	inbox, err := BuildInbox(TriggerTaskCompleted, snapshots, events)
	if err != nil {
		t.Fatalf("BuildInbox failed: %v", err)
	}
	if inbox.TriggerType != TriggerTaskCompleted {
		t.Fatalf("trigger_type = %q, want %q", inbox.TriggerType, TriggerTaskCompleted)
	}
	if len(inbox.Events) != 1 {
		t.Fatalf("events len = %d, want 1", len(inbox.Events))
	}
	if len(inbox.Snapshots) != 1 {
		t.Fatalf("snapshots len = %d, want 1", len(inbox.Snapshots))
	}
	if inbox.Timestamp.IsZero() {
		t.Fatal("timestamp must be populated")
	}
	if inbox.Timestamp.Location() != time.UTC {
		t.Fatalf("timestamp location = %v, want UTC", inbox.Timestamp.Location())
	}
	if inbox.Timestamp.Before(now.Add(-2*time.Second)) || inbox.Timestamp.After(time.Now().UTC().Add(2*time.Second)) {
		t.Fatalf("timestamp out of expected range: %s", inbox.Timestamp)
	}
}

func TestBuildInboxWithEmptyInputs(t *testing.T) {
	inbox, err := BuildInbox(TriggerOnDemand, nil, nil)
	if err != nil {
		t.Fatalf("BuildInbox failed: %v", err)
	}
	if inbox == nil {
		t.Fatal("inbox should not be nil")
	}
	if inbox.Events == nil {
		t.Fatal("events should be an empty array, not nil")
	}
	if inbox.Snapshots == nil {
		t.Fatal("snapshots should be an empty array, not nil")
	}
	if len(inbox.Events) != 0 || len(inbox.Snapshots) != 0 {
		t.Fatalf("expected empty arrays, got events=%d snapshots=%d", len(inbox.Events), len(inbox.Snapshots))
	}
}

func TestInboxToJSONProducesCompactJSON(t *testing.T) {
	inbox, err := BuildInbox(TriggerCommit, nil, nil)
	if err != nil {
		t.Fatalf("BuildInbox failed: %v", err)
	}

	encoded, err := InboxToJSON(inbox)
	if err != nil {
		t.Fatalf("InboxToJSON failed: %v", err)
	}
	if bytes.Contains(encoded, []byte("\n")) {
		t.Fatalf("expected compact JSON without newlines, got %q", string(encoded))
	}

	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json should be parseable: %v", err)
	}
	for _, key := range []string{"timestamp", "trigger_type", "events", "snapshots"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("missing key %q in serialized inbox", key)
		}
	}
}
