package task

import (
	"testing"
)

func TestParseProviderOutput_GroupedFormat(t *testing.T) {
	input := `{
		"groups": [
			{
				"id": "PBI-13",
				"title": "Search & Filter",
				"status": "in_progress",
				"url": "https://github.com/owner/repo/milestone/3",
				"tasks": [
					{
						"id": "13-1",
						"title": "Implement fuzzy search",
						"status": "done",
						"assignee": "ai-agent",
						"labels": ["feat", "tui"],
						"priority": 1,
						"url": "https://github.com/owner/repo/issues/42",
						"created": "2025-05-19T15:02:00Z",
						"updated": "2025-05-20T10:30:00Z"
					},
					{
						"id": "13-2",
						"title": "Add status filter",
						"status": "open"
					}
				]
			}
		]
	}`

	result, err := ParseProviderOutput([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(result.Groups))
	}

	g := result.Groups[0]
	if g.ID != "PBI-13" {
		t.Errorf("expected group ID 'PBI-13', got %q", g.ID)
	}
	if g.Title != "Search & Filter" {
		t.Errorf("expected group title 'Search & Filter', got %q", g.Title)
	}
	if g.Status != "in_progress" {
		t.Errorf("expected group status 'in_progress', got %q", g.Status)
	}
	if g.URL != "https://github.com/owner/repo/milestone/3" {
		t.Errorf("expected group URL, got %q", g.URL)
	}
	if len(g.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(g.Tasks))
	}

	task := g.Tasks[0]
	if task.ID != "13-1" {
		t.Errorf("expected task ID '13-1', got %q", task.ID)
	}
	if task.Title != "Implement fuzzy search" {
		t.Errorf("expected task title 'Implement fuzzy search', got %q", task.Title)
	}
	if task.Status != "done" {
		t.Errorf("expected task status 'done', got %q", task.Status)
	}
	if task.Assignee != "ai-agent" {
		t.Errorf("expected assignee 'ai-agent', got %q", task.Assignee)
	}
	if len(task.Labels) != 2 || task.Labels[0] != "feat" {
		t.Errorf("expected labels [feat, tui], got %v", task.Labels)
	}
	if task.Priority != 1 {
		t.Errorf("expected priority 1, got %d", task.Priority)
	}
	if task.URL != "https://github.com/owner/repo/issues/42" {
		t.Errorf("expected task URL, got %q", task.URL)
	}
	if task.Created.IsZero() {
		t.Error("expected non-zero created time")
	}
	if task.Updated.IsZero() {
		t.Error("expected non-zero updated time")
	}
}

func TestParseProviderOutput_FlatFormat(t *testing.T) {
	input := `{
		"tasks": [
			{"id": "1", "title": "First task", "status": "open"},
			{"id": "2", "title": "Second task", "status": "closed"}
		]
	}`

	result, err := ParseProviderOutput([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(result.Groups))
	}
	if len(result.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(result.Tasks))
	}

	if result.Tasks[0].ID != "1" {
		t.Errorf("expected task ID '1', got %q", result.Tasks[0].ID)
	}
	if result.Tasks[1].Status != "closed" {
		t.Errorf("expected task status 'closed', got %q", result.Tasks[1].Status)
	}
}

func TestParseProviderOutput_OptionalFieldsMissing(t *testing.T) {
	input := `{
		"tasks": [
			{"id": "1", "title": "Minimal task", "status": "open"}
		]
	}`

	result, err := ParseProviderOutput([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	task := result.Tasks[0]
	if task.Assignee != "" {
		t.Errorf("expected empty assignee, got %q", task.Assignee)
	}
	if task.Labels != nil {
		t.Errorf("expected nil labels, got %v", task.Labels)
	}
	if task.Priority != 0 {
		t.Errorf("expected zero priority, got %d", task.Priority)
	}
	if task.URL != "" {
		t.Errorf("expected empty URL, got %q", task.URL)
	}
	if !task.Created.IsZero() {
		t.Error("expected zero created time")
	}
}

func TestParseProviderOutput_MalformedJSON(t *testing.T) {
	input := `{not valid json}`

	_, err := ParseProviderOutput([]byte(input))
	if err == nil {
		t.Error("expected error for malformed JSON, got nil")
	}
}

func TestParseProviderOutput_EmptyObject(t *testing.T) {
	input := `{}`

	result, err := ParseProviderOutput([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(result.Groups))
	}
	if len(result.Tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(result.Tasks))
	}
}

func TestProviderResult_AllTasks_Grouped(t *testing.T) {
	result := &ProviderResult{
		Groups: []TaskGroup{
			{
				ID: "g1",
				Tasks: []Task{
					{ID: "1", Title: "Task 1", Status: "open"},
					{ID: "2", Title: "Task 2", Status: "done"},
				},
			},
			{
				ID: "g2",
				Tasks: []Task{
					{ID: "3", Title: "Task 3", Status: "open"},
				},
			},
		},
	}

	all := result.AllTasks()
	if len(all) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(all))
	}
	if all[0].ID != "1" || all[1].ID != "2" || all[2].ID != "3" {
		t.Errorf("unexpected task order: %v", all)
	}
}

func TestProviderResult_AllTasks_Flat(t *testing.T) {
	result := &ProviderResult{
		Tasks: []Task{
			{ID: "1", Title: "Task 1", Status: "open"},
			{ID: "2", Title: "Task 2", Status: "done"},
		},
	}

	all := result.AllTasks()
	if len(all) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(all))
	}
}
