package tui

import (
	"strings"
	"testing"

	"github.com/stwalsh4118/navi/internal/task"
)

func TestGroupStatusCategory(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"done", "done"},
		{"Done", "done"},
		{"closed", "done"},
		{"completed", "done"},
		{"active", "active"},
		{"in_progress", "active"},
		{"inprogress", "active"},
		{"working", "active"},
		{"review", "review"},
		{"inreview", "review"},
		{"blocked", "blocked"},
		{"todo", "todo"},
		{"open", "todo"},
		{"proposed", "todo"},
		{"", "todo"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := groupStatusCategory(tt.status)
			if got != tt.want {
				t.Errorf("groupStatusCategory(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestGroupStatusSummary(t *testing.T) {
	groups := []task.TaskGroup{
		{ID: "1", Status: "done"},
		{ID: "2", Status: "done"},
		{ID: "3", Status: "active"},
		{ID: "4", Status: "review"},
		{ID: "5", Status: "todo"},
		{ID: "6", Status: "open"}, // maps to "todo"
		{ID: "7", Status: "blocked"},
		{ID: "8", Status: "in_progress"}, // maps to "active"
	}

	counts := groupStatusSummary(groups)

	if counts["done"] != 2 {
		t.Errorf("done count = %d, want 2", counts["done"])
	}
	if counts["active"] != 2 {
		t.Errorf("active count = %d, want 2", counts["active"])
	}
	if counts["review"] != 1 {
		t.Errorf("review count = %d, want 1", counts["review"])
	}
	if counts["blocked"] != 1 {
		t.Errorf("blocked count = %d, want 1", counts["blocked"])
	}
	if counts["todo"] != 2 {
		t.Errorf("todo count = %d, want 2", counts["todo"])
	}
}

func TestRenderStatusSummary(t *testing.T) {
	t.Run("shows non-zero categories in order", func(t *testing.T) {
		counts := map[string]int{
			"done":   12,
			"active": 3,
			"todo":   18,
		}
		result := renderStatusSummary(counts)
		// Should contain these numbers
		if !strings.Contains(result, "12") {
			t.Errorf("expected done count 12 in summary: %s", result)
		}
		if !strings.Contains(result, "3") {
			t.Errorf("expected active count 3 in summary: %s", result)
		}
		if !strings.Contains(result, "18") {
			t.Errorf("expected todo count 18 in summary: %s", result)
		}
		// Should NOT contain review/blocked since they're zero
		if strings.Contains(result, "review") {
			t.Errorf("should not contain review when count is zero: %s", result)
		}
		if strings.Contains(result, "blocked") {
			t.Errorf("should not contain blocked when count is zero: %s", result)
		}
	})

	t.Run("empty counts returns empty string", func(t *testing.T) {
		counts := map[string]int{}
		result := renderStatusSummary(counts)
		if result != "" {
			t.Errorf("expected empty string for no counts, got: %q", result)
		}
	})
}

func TestHeaderSummaryInOutput(t *testing.T) {
	t.Run("header includes summary line when panel is tall enough", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskPanelHeight = 20 // Tall enough for summary

		header := m.renderTaskPanelHeader(120)

		// Should have newline (indicating second line)
		if !strings.Contains(header, "\n") {
			t.Error("expected two-line header with summary stats")
		}
	})

	t.Run("header omits summary line when panel is too short", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskPanelHeight = 5 // Very short — contentLines = 5-3 = 2 < 4

		header := m.renderTaskPanelHeader(120)

		if strings.Contains(header, "\n") {
			t.Error("expected single-line header when panel is short")
		}
	})
}
