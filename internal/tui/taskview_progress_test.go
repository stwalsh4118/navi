package tui

import (
	"strings"
	"testing"

	"github.com/stwalsh4118/navi/internal/task"
)

func TestGroupProgress(t *testing.T) {
	tests := []struct {
		name      string
		group     task.TaskGroup
		wantDone  int
		wantTotal int
	}{
		{
			name:      "empty group",
			group:     task.TaskGroup{ID: "g1", Tasks: nil},
			wantDone:  0,
			wantTotal: 0,
		},
		{
			name: "all done",
			group: task.TaskGroup{
				ID: "g1",
				Tasks: []task.Task{
					{ID: "1", Status: "done"},
					{ID: "2", Status: "Done"},
					{ID: "3", Status: "closed"},
				},
			},
			wantDone:  3,
			wantTotal: 3,
		},
		{
			name: "none done",
			group: task.TaskGroup{
				ID: "g1",
				Tasks: []task.Task{
					{ID: "1", Status: "todo"},
					{ID: "2", Status: "active"},
				},
			},
			wantDone:  0,
			wantTotal: 2,
		},
		{
			name: "mixed statuses",
			group: task.TaskGroup{
				ID: "g1",
				Tasks: []task.Task{
					{ID: "1", Status: "done"},
					{ID: "2", Status: "active"},
					{ID: "3", Status: "completed"},
					{ID: "4", Status: "todo"},
					{ID: "5", Status: "Closed"},
				},
			},
			wantDone:  3,
			wantTotal: 5,
		},
		{
			name: "review and blocked not counted as done",
			group: task.TaskGroup{
				ID: "g1",
				Tasks: []task.Task{
					{ID: "1", Status: "review"},
					{ID: "2", Status: "blocked"},
				},
			},
			wantDone:  0,
			wantTotal: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			done, total := groupProgress(tt.group)
			if done != tt.wantDone {
				t.Errorf("done = %d, want %d", done, tt.wantDone)
			}
			if total != tt.wantTotal {
				t.Errorf("total = %d, want %d", total, tt.wantTotal)
			}
		})
	}
}

func TestRenderProgressBar(t *testing.T) {
	tests := []struct {
		name  string
		done  int
		total int
		want  string
	}{
		{
			name:  "0 percent",
			done:  0,
			total: 4,
			want:  "░░░░",
		},
		{
			name:  "25 percent",
			done:  1,
			total: 4,
			want:  "█░░░",
		},
		{
			name:  "50 percent",
			done:  2,
			total: 4,
			want:  "██░░",
		},
		{
			name:  "75 percent",
			done:  3,
			total: 4,
			want:  "███░",
		},
		{
			name:  "100 percent",
			done:  4,
			total: 4,
			want:  "████",
		},
		{
			name:  "zero total returns empty",
			done:  0,
			total: 0,
			want:  "",
		},
		{
			name:  "partial progress gets at least 1 filled",
			done:  1,
			total: 100,
			want:  "█░░░",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderProgressBar(tt.done, tt.total)
			if got != tt.want {
				t.Errorf("renderProgressBar(%d, %d) = %q, want %q", tt.done, tt.total, got, tt.want)
			}
		})
	}
}

func TestIsDoneStatus(t *testing.T) {
	doneStatuses := []string{"done", "Done", "DONE", "closed", "Closed", "completed", "Completed"}
	for _, s := range doneStatuses {
		if !isDoneStatus(s) {
			t.Errorf("isDoneStatus(%q) = false, want true", s)
		}
	}

	notDoneStatuses := []string{"todo", "active", "review", "blocked", "open", "in_progress", ""}
	for _, s := range notDoneStatuses {
		if isDoneStatus(s) {
			t.Errorf("isDoneStatus(%q) = true, want false", s)
		}
	}
}

func TestGroupHeaderShowsProgress(t *testing.T) {
	t.Run("group with tasks shows progress display", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true

		// Get items and render the first group header
		items := m.getVisibleTaskItems()
		if len(items) == 0 {
			t.Fatal("expected at least one item")
		}

		result := m.renderTaskPanelGroupHeader(items[0], false, 120)

		// Group g1 has 3 tasks, 2 done → [2/3]
		if !strings.Contains(result, "[2/3]") {
			t.Errorf("expected progress [2/3] in header, got: %s", result)
		}
		// Should have progress bar characters
		if !strings.Contains(result, "██") {
			t.Errorf("expected progress bar in header, got: %s", result)
		}
	})

	t.Run("empty group shows (0)", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskGroups = append(m.taskGroups, task.TaskGroup{
			ID:    "g3",
			Title: "Empty Group",
			Tasks: nil,
		})

		items := m.getVisibleTaskItems()
		// Find the empty group item
		var emptyItem taskItem
		for _, item := range items {
			if item.groupID == "g3" {
				emptyItem = item
				break
			}
		}

		result := m.renderTaskPanelGroupHeader(emptyItem, false, 120)

		if !strings.Contains(result, "(0)") {
			t.Errorf("expected (0) for empty group, got: %s", result)
		}
	})
}
