package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/task"
)

func TestSortTaskGroups(t *testing.T) {
	groups := []task.TaskGroup{
		{ID: "1", Title: "Zebra", Status: "done", Tasks: []task.Task{
			{ID: "1-1", Status: "done"},
			{ID: "1-2", Status: "done"},
		}},
		{ID: "2", Title: "Alpha", Status: "active", Tasks: []task.Task{
			{ID: "2-1", Status: "active"},
			{ID: "2-2", Status: "todo"},
		}},
		{ID: "3", Title: "Beta", Status: "todo", Tasks: []task.Task{
			{ID: "3-1", Status: "todo"},
		}},
		{ID: "4", Title: "Gamma", Status: "review"},
	}

	t.Run("source sort preserves original order", func(t *testing.T) {
		result := sortTaskGroups(groups, taskSortSource)
		assertGroupOrder(t, result, []string{"1", "2", "3", "4"})
	})

	t.Run("status sort orders by priority", func(t *testing.T) {
		result := sortTaskGroups(groups, taskSortStatus)
		// active > review > todo > done
		assertGroupOrder(t, result, []string{"2", "4", "3", "1"})
	})

	t.Run("name sort is alphabetical case-insensitive", func(t *testing.T) {
		result := sortTaskGroups(groups, taskSortName)
		assertGroupOrder(t, result, []string{"2", "3", "4", "1"}) // Alpha, Beta, Gamma, Zebra
	})

	t.Run("progress sort orders by lowest completion first", func(t *testing.T) {
		result := sortTaskGroups(groups, taskSortProgress)
		// Group 4: 0/0 (0%), Group 3: 0/1 (0%), Group 2: 0/2 (0%), Group 1: 2/2 (100%)
		// Ties broken by source order, so 4, 3, 2, 1
		// But wait: 4 has 0 tasks (0/0 = 0%), 3 has 0/1, 2 has 0/2, 1 has 2/2
		// All zeros are equal so source order applies: 2, 3, 4, then 1
		assertGroupOrder(t, result, []string{"2", "3", "4", "1"})
	})
}

func TestSortTasksByStatus(t *testing.T) {
	tasks := []task.Task{
		{ID: "1", Status: "done"},
		{ID: "2", Status: "todo"},
		{ID: "3", Status: "active"},
		{ID: "4", Status: "review"},
		{ID: "5", Status: "blocked"},
	}

	result := sortTasksByStatus(tasks)

	// active > review > blocked > todo > done
	expected := []string{"3", "4", "5", "2", "1"}
	for i, id := range expected {
		if result[i].ID != id {
			t.Errorf("position %d: got %s, want %s", i, result[i].ID, id)
		}
	}
}

func TestStatusPriority(t *testing.T) {
	tests := []struct {
		status string
		want   int
	}{
		{"active", 0},
		{"in_progress", 0},
		{"review", 1},
		{"blocked", 2},
		{"todo", 3},
		{"open", 3},
		{"done", 4},
		{"closed", 4},
	}
	for _, tt := range tests {
		got := statusPriority(tt.status)
		if got != tt.want {
			t.Errorf("statusPriority(%q) = %d, want %d", tt.status, got, tt.want)
		}
	}
}

func TestNextTaskSortMode(t *testing.T) {
	tests := []struct {
		current taskSortMode
		want    taskSortMode
	}{
		{taskSortSource, taskSortStatus},
		{taskSortStatus, taskSortName},
		{taskSortName, taskSortProgress},
		{taskSortProgress, taskSortSource},
	}
	for _, tt := range tests {
		got := nextTaskSortMode(tt.current)
		if got != tt.want {
			t.Errorf("nextTaskSortMode(%s) = %s, want %s", tt.current, got, tt.want)
		}
	}
}

func TestSortKeybinding(t *testing.T) {
	t.Run("s key cycles sort mode in task panel", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true
		m.taskSortMode = taskSortSource

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}

		newModel, _ := m.Update(msg)
		updated := newModel.(Model)
		if updated.taskSortMode != taskSortStatus {
			t.Errorf("sort mode should be status after first s, got %s", updated.taskSortMode)
		}

		newModel, _ = updated.Update(msg)
		updated = newModel.(Model)
		if updated.taskSortMode != taskSortName {
			t.Errorf("sort mode should be name after second s, got %s", updated.taskSortMode)
		}

		newModel, _ = updated.Update(msg)
		updated = newModel.(Model)
		if updated.taskSortMode != taskSortProgress {
			t.Errorf("sort mode should be progress after third s, got %s", updated.taskSortMode)
		}

		newModel, _ = updated.Update(msg)
		updated = newModel.(Model)
		if updated.taskSortMode != taskSortSource {
			t.Errorf("sort mode should be source after fourth s, got %s", updated.taskSortMode)
		}
	})
}

func TestCursorStabilityAfterSort(t *testing.T) {
	t.Run("cursor stays on same item after sort change", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true
		m.taskExpandedGroups["g1"] = true

		// Move cursor to a specific task
		m.taskCursor = 2 // Should be task 13-2 in g1

		items := m.getVisibleTaskItems()
		targetItem := items[m.taskCursor]

		// Change sort mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		// Verify cursor is still on the same item
		newItems := updated.getVisibleTaskItems()
		if updated.taskCursor < len(newItems) {
			cursorItem := newItems[updated.taskCursor]
			if cursorItem.taskID != targetItem.taskID || cursorItem.groupID != targetItem.groupID {
				t.Errorf("cursor moved to different item: was %s/%s, now %s/%s",
					targetItem.groupID, targetItem.taskID, cursorItem.groupID, cursorItem.taskID)
			}
		}
	})
}

func TestReverseTaskGroups(t *testing.T) {
	groups := []task.TaskGroup{
		{ID: "1", Title: "First"},
		{ID: "2", Title: "Second"},
		{ID: "3", Title: "Third"},
	}

	t.Run("reverses group order", func(t *testing.T) {
		g := make([]task.TaskGroup, len(groups))
		copy(g, groups)
		reverseTaskGroups(g)
		assertGroupOrder(t, g, []string{"3", "2", "1"})
	})

	t.Run("empty slice is no-op", func(t *testing.T) {
		var g []task.TaskGroup
		reverseTaskGroups(g) // should not panic
	})

	t.Run("single element is no-op", func(t *testing.T) {
		g := []task.TaskGroup{{ID: "1"}}
		reverseTaskGroups(g)
		assertGroupOrder(t, g, []string{"1"})
	})
}

func TestReverseSortKeybinding(t *testing.T) {
	t.Run("S key toggles sort direction", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true

		if m.taskSortReversed {
			t.Error("sort should not be reversed initially")
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !updated.taskSortReversed {
			t.Error("sort should be reversed after S")
		}

		newModel, _ = updated.Update(msg)
		updated = newModel.(Model)

		if updated.taskSortReversed {
			t.Error("sort should not be reversed after second S")
		}
	})

	t.Run("reversed source sort shows groups in reverse order", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskSortMode = taskSortSource
		m.taskSortReversed = true

		groups := m.getSortedAndFilteredTaskGroups()
		if len(groups) < 2 {
			t.Skip("need at least 2 groups")
		}
		// Original order is g1, g2; reversed should be g2, g1
		assertGroupOrder(t, groups, []string{"g2", "g1"})
	})
}

func TestReverseSortHeaderIndicator(t *testing.T) {
	t.Run("ascending sort shows up arrow", func(t *testing.T) {
		m := newNavTestModel()
		m.taskSortMode = taskSortName
		m.taskSortReversed = false
		m.taskPanelHeight = 20

		header := m.renderTaskPanelHeader(120)
		if !strings.Contains(header, "sort:name↑") {
			t.Errorf("expected 'sort:name↑' in header, got: %s", header)
		}
	})

	t.Run("descending sort shows down arrow", func(t *testing.T) {
		m := newNavTestModel()
		m.taskSortMode = taskSortName
		m.taskSortReversed = true
		m.taskPanelHeight = 20

		header := m.renderTaskPanelHeader(120)
		if !strings.Contains(header, "sort:name↓") {
			t.Errorf("expected 'sort:name↓' in header, got: %s", header)
		}
	})

	t.Run("reversed source sort shows indicator", func(t *testing.T) {
		m := newNavTestModel()
		m.taskSortMode = taskSortSource
		m.taskSortReversed = true
		m.taskPanelHeight = 20

		header := m.renderTaskPanelHeader(120)
		if !strings.Contains(header, "sort:source↓") {
			t.Errorf("expected 'sort:source↓' in header, got: %s", header)
		}
	})

	t.Run("source ascending hides sort indicator", func(t *testing.T) {
		m := newNavTestModel()
		m.taskSortMode = taskSortSource
		m.taskSortReversed = false
		m.taskPanelHeight = 20

		header := m.renderTaskPanelHeader(120)
		if strings.Contains(header, "sort:") {
			t.Errorf("should not show sort indicator for source ascending, got: %s", header)
		}
	})
}

func assertGroupOrder(t *testing.T, groups []task.TaskGroup, expectedIDs []string) {
	t.Helper()
	if len(groups) != len(expectedIDs) {
		t.Fatalf("expected %d groups, got %d", len(expectedIDs), len(groups))
	}
	for i, id := range expectedIDs {
		if groups[i].ID != id {
			t.Errorf("position %d: got %s, want %s", i, groups[i].ID, id)
		}
	}
}
