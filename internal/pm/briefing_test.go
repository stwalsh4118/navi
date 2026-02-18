package pm

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPMBriefingJSONRoundTrip(t *testing.T) {
	original := PMBriefing{
		Summary: "Two projects need attention.",
		Projects: []ProjectBriefing{
			{
				Name:           "navi",
				Status:         "working",
				CurrentWork:    "Implementing PBI-48",
				RecentActivity: "Committed invoker scaffold",
			},
		},
		AttentionItems: []AttentionItem{
			{
				Priority:    "high",
				Title:       "Finish E2E test",
				Description: "Task 48-8 remains open",
				ProjectName: "navi",
			},
		},
		Breadcrumbs: []Breadcrumb{
			{
				Timestamp: "2026-02-18T17:44:00Z",
				Summary:   "Recovered from stale session ID",
			},
		},
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal PMBriefing failed: %v", err)
	}

	var decoded PMBriefing
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal PMBriefing failed: %v", err)
	}

	if decoded.Summary != original.Summary {
		t.Fatalf("summary = %q, want %q", decoded.Summary, original.Summary)
	}
	if len(decoded.Projects) != 1 {
		t.Fatalf("projects len = %d, want 1", len(decoded.Projects))
	}
	if decoded.Projects[0].CurrentWork != original.Projects[0].CurrentWork {
		t.Fatalf("current_work = %q, want %q", decoded.Projects[0].CurrentWork, original.Projects[0].CurrentWork)
	}
	if len(decoded.AttentionItems) != 1 {
		t.Fatalf("attention_items len = %d, want 1", len(decoded.AttentionItems))
	}
	if decoded.AttentionItems[0].Priority != "high" {
		t.Fatalf("priority = %q, want high", decoded.AttentionItems[0].Priority)
	}
	if len(decoded.Breadcrumbs) != 1 {
		t.Fatalf("breadcrumbs len = %d, want 1", len(decoded.Breadcrumbs))
	}
}

func TestInboxPayloadJSONRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	original := InboxPayload{
		Timestamp:   now,
		TriggerType: TriggerCommit,
		Events: []Event{{
			Type:        EventCommit,
			Timestamp:   now,
			ProjectName: "navi",
			ProjectDir:  "/tmp/navi",
			Payload: map[string]string{
				"commits": "abc123 Add PM types",
			},
		}},
		Snapshots: []ProjectSnapshot{{
			ProjectName:   "navi",
			ProjectDir:    "/tmp/navi",
			SessionStatus: "working",
		}},
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal InboxPayload failed: %v", err)
	}

	var decoded InboxPayload
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal InboxPayload failed: %v", err)
	}

	if decoded.TriggerType != TriggerCommit {
		t.Fatalf("trigger_type = %q, want %q", decoded.TriggerType, TriggerCommit)
	}
	if len(decoded.Events) != 1 {
		t.Fatalf("events len = %d, want 1", len(decoded.Events))
	}
	if len(decoded.Snapshots) != 1 {
		t.Fatalf("snapshots len = %d, want 1", len(decoded.Snapshots))
	}
}

func TestTriggerTypeConstants(t *testing.T) {
	expected := map[TriggerType]string{
		TriggerTaskCompleted: "task_completed",
		TriggerCommit:        "commit",
		TriggerOnDemand:      "on_demand",
	}

	for trigger, want := range expected {
		if string(trigger) != want {
			t.Fatalf("trigger %q = %q, want %q", trigger, string(trigger), want)
		}
	}
}

func TestEmbeddedTemplates(t *testing.T) {
	if SystemPromptTemplate == "" {
		t.Fatal("SystemPromptTemplate should not be empty")
	}
	if OutputSchemaTemplate == "" {
		t.Fatal("OutputSchemaTemplate should not be empty")
	}

	var schema map[string]any
	if err := json.Unmarshal([]byte(OutputSchemaTemplate), &schema); err != nil {
		t.Fatalf("OutputSchemaTemplate must be valid JSON: %v", err)
	}

	if schema["type"] != "object" {
		t.Fatalf("schema type = %v, want object", schema["type"])
	}
	if _, ok := schema["properties"].(map[string]any); !ok {
		t.Fatal("schema properties must be an object")
	}
	requiredRaw, ok := schema["required"].([]any)
	if !ok {
		t.Fatal("schema required must be an array")
	}

	requiredSet := make(map[string]bool, len(requiredRaw))
	for _, item := range requiredRaw {
		s, ok := item.(string)
		if ok {
			requiredSet[s] = true
		}
	}
	for _, key := range []string{"summary", "projects", "attention_items", "breadcrumbs"} {
		if !requiredSet[key] {
			t.Fatalf("schema required missing %q", key)
		}
	}
}
