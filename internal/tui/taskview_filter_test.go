package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/task"
)

func TestFilterTaskGroups(t *testing.T) {
	groups := []task.TaskGroup{
		{ID: "1", Title: "Done Group", Status: "done"},
		{ID: "2", Title: "Active Group", Status: "active"},
		{ID: "3", Title: "Todo Group", Status: "todo"},
		{ID: "4", Title: "Review Group", Status: "review"},
		{ID: "5", Title: "Blocked Group", Status: "blocked"},
		{ID: "6", Title: "Another Done", Status: "closed"},
	}

	t.Run("filter all returns all groups", func(t *testing.T) {
		result := filterTaskGroups(groups, taskFilterAll)
		if len(result) != 6 {
			t.Errorf("expected 6 groups, got %d", len(result))
		}
	})

	t.Run("filter active shows only active/review/blocked", func(t *testing.T) {
		result := filterTaskGroups(groups, taskFilterActive)
		expected := []string{"2", "4", "5"} // active, review, blocked
		assertGroupOrder(t, result, expected)
	})

	t.Run("filter incomplete shows everything except done", func(t *testing.T) {
		result := filterTaskGroups(groups, taskFilterIncomplete)
		expected := []string{"2", "3", "4", "5"} // active, todo, review, blocked
		assertGroupOrder(t, result, expected)
	})

	t.Run("empty string mode treated as all", func(t *testing.T) {
		result := filterTaskGroups(groups, "")
		if len(result) != 6 {
			t.Errorf("expected 6 groups for empty filter mode, got %d", len(result))
		}
	})
}

func TestNextTaskFilterMode(t *testing.T) {
	tests := []struct {
		current taskFilterMode
		want    taskFilterMode
	}{
		{taskFilterAll, taskFilterActive},
		{taskFilterActive, taskFilterIncomplete},
		{taskFilterIncomplete, taskFilterAll},
	}
	for _, tt := range tests {
		got := nextTaskFilterMode(tt.current)
		if got != tt.want {
			t.Errorf("nextTaskFilterMode(%s) = %s, want %s", tt.current, got, tt.want)
		}
	}
}

func TestFilterKeybinding(t *testing.T) {
	t.Run("f key cycles filter mode in task panel", func(t *testing.T) {
		m := newFilterTestModel()
		m.taskFilterMode = taskFilterAll

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}

		newModel, _ := m.Update(msg)
		updated := newModel.(Model)
		if updated.taskFilterMode != taskFilterActive {
			t.Errorf("filter mode should be active after first f, got %s", updated.taskFilterMode)
		}

		newModel, _ = updated.Update(msg)
		updated = newModel.(Model)
		if updated.taskFilterMode != taskFilterIncomplete {
			t.Errorf("filter mode should be incomplete after second f, got %s", updated.taskFilterMode)
		}

		newModel, _ = updated.Update(msg)
		updated = newModel.(Model)
		if updated.taskFilterMode != taskFilterAll {
			t.Errorf("filter mode should be all after third f, got %s", updated.taskFilterMode)
		}
	})
}

func TestSearchWithinFilteredSet(t *testing.T) {
	t.Run("search only matches within filtered groups", func(t *testing.T) {
		m := newFilterTestModel()
		m.taskFilterMode = taskFilterActive // Only shows active/review/blocked
		m.taskSearchQuery = "Done"          // Group that is filtered out

		m.computeTaskSearchMatches()

		if len(m.taskSearchMatches) != 0 {
			t.Errorf("expected 0 matches for filtered-out group, got %d", len(m.taskSearchMatches))
		}
	})

	t.Run("search finds groups in filtered set", func(t *testing.T) {
		m := newFilterTestModel()
		m.taskFilterMode = taskFilterActive
		m.taskSearchQuery = "Active" // This group is visible

		m.computeTaskSearchMatches()

		if len(m.taskSearchMatches) == 0 {
			t.Error("expected matches for visible group")
		}
	})
}

func TestFilterAndSearchIndependence(t *testing.T) {
	t.Run("changing filter preserves search query", func(t *testing.T) {
		m := newFilterTestModel()
		m.taskSearchQuery = "something"

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.taskSearchQuery != "something" {
			t.Errorf("search query should be preserved after filter change, got %q", updated.taskSearchQuery)
		}
	})
}

func TestCursorStabilityAfterFilter(t *testing.T) {
	t.Run("cursor moves to valid position when item filtered out", func(t *testing.T) {
		m := newFilterTestModel()
		m.taskCursor = 0 // On first group (done)

		// Filter to active only — done group should disappear
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		items := updated.getVisibleTaskItems()
		if updated.taskCursor >= len(items) {
			t.Errorf("cursor %d is out of range for %d items", updated.taskCursor, len(items))
		}
	})
}

func TestSummaryUsesUnfilteredTotals(t *testing.T) {
	t.Run("summary counts reflect all groups regardless of filter", func(t *testing.T) {
		m := newFilterTestModel()
		m.taskFilterMode = taskFilterActive

		// groupStatusSummary should use m.taskGroups (unfiltered)
		counts := groupStatusSummary(m.taskGroups)

		// The model has 4 groups: done, active, todo, review
		total := 0
		for _, n := range counts {
			total += n
		}
		if total != 4 {
			t.Errorf("summary should count all 4 groups, got %d", total)
		}
	})
}

// newFilterTestModel creates a model with diverse group statuses for filter testing.
func newFilterTestModel() Model {
	groups := []task.TaskGroup{
		{
			ID: "done1", Title: "Done Group", Status: "done",
			Tasks: []task.Task{{ID: "d1-1", Title: "A done task", Status: "done"}},
		},
		{
			ID: "active1", Title: "Active Group", Status: "active",
			Tasks: []task.Task{{ID: "a1-1", Title: "An active task", Status: "active"}},
		},
		{
			ID: "todo1", Title: "Todo Group", Status: "todo",
			Tasks: []task.Task{{ID: "t1-1", Title: "A todo task", Status: "todo"}},
		},
		{
			ID: "review1", Title: "Review Group", Status: "review",
			Tasks: []task.Task{{ID: "r1-1", Title: "A review task", Status: "review"}},
		},
	}

	return Model{
		width:              120,
		height:             40,
		searchInput:        initSearchInput(),
		taskSearchInput:    initTaskSearchInput(),
		taskCache:          task.NewResultCache(),
		taskGlobalConfig:   &task.GlobalConfig{},
		taskGroups:         groups,
		taskExpandedGroups: make(map[string]bool),
		taskFocusedProject: testProjectDir,
		taskGroupsByProject: map[string][]task.TaskGroup{
			testProjectDir: groups,
		},
		taskProjectConfigs: []task.ProjectConfig{
			{Tasks: task.ProjectTaskConfig{Provider: "test"}, ProjectDir: testProjectDir},
		},
		taskPanelVisible: true,
		taskPanelFocused: true,
		taskSortMode:     taskSortSource,
		taskFilterMode:   taskFilterAll,
	}
}
