package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/task"
)

func newNavTestModel() Model {
	groups := []task.TaskGroup{
		{
			ID: "g1", Title: "Group One", Status: "active",
			Tasks: []task.Task{
				{ID: "1-1", Title: "Task 1-1", Status: "done"},
				{ID: "1-2", Title: "Task 1-2", Status: "active"},
			},
		},
		{
			ID: "g2", Title: "Group Two", Status: "todo",
			Tasks: []task.Task{
				{ID: "2-1", Title: "Task 2-1", Status: "todo"},
			},
		},
		{
			ID: "g3", Title: "Group Three", Status: "done",
			Tasks: []task.Task{
				{ID: "3-1", Title: "Task 3-1", Status: "done"},
			},
		},
	}

	return Model{
		width:            120,
		height:           40,
		searchInput:      initSearchInput(),
		taskSearchInput:  initTaskSearchInput(),
		taskCache:        task.NewResultCache(),
		taskGlobalConfig: &task.GlobalConfig{},
		taskGroups:       groups,
		taskExpandedGroups: map[string]bool{
			"g1": true,
			"g2": true,
			"g3": true,
		},
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

func TestJumpToNextGroup(t *testing.T) {
	t.Run("J jumps to next group header", func(t *testing.T) {
		m := newNavTestModel()
		m.taskCursor = 0 // On g1 header

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		items := updated.getVisibleTaskItems()
		if !items[updated.taskCursor].isGroup || items[updated.taskCursor].groupID != "g2" {
			t.Errorf("expected cursor on g2 group header, got cursor=%d, groupID=%s", updated.taskCursor, items[updated.taskCursor].groupID)
		}
	})

	t.Run("J skips tasks within group", func(t *testing.T) {
		m := newNavTestModel()
		m.taskCursor = 1 // On task 1-1 within g1

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		items := updated.getVisibleTaskItems()
		if !items[updated.taskCursor].isGroup || items[updated.taskCursor].groupID != "g2" {
			t.Errorf("expected cursor on g2 header, got cursor=%d", updated.taskCursor)
		}
	})

	t.Run("J wraps to first group from last", func(t *testing.T) {
		m := newNavTestModel()
		// Find the last group header
		items := m.getVisibleTaskItems()
		lastGroupIdx := 0
		for i, item := range items {
			if item.isGroup && item.groupID == "g3" {
				lastGroupIdx = i
			}
		}
		m.taskCursor = lastGroupIdx

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		updatedItems := updated.getVisibleTaskItems()
		if !updatedItems[updated.taskCursor].isGroup || updatedItems[updated.taskCursor].groupID != "g1" {
			t.Errorf("expected wrap to g1, got groupID=%s", updatedItems[updated.taskCursor].groupID)
		}
	})
}

func TestJumpToPrevGroup(t *testing.T) {
	t.Run("K jumps to previous group header", func(t *testing.T) {
		m := newNavTestModel()
		// Put cursor on g2
		items := m.getVisibleTaskItems()
		for i, item := range items {
			if item.isGroup && item.groupID == "g2" {
				m.taskCursor = i
				break
			}
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		updatedItems := updated.getVisibleTaskItems()
		if !updatedItems[updated.taskCursor].isGroup || updatedItems[updated.taskCursor].groupID != "g1" {
			t.Errorf("expected cursor on g1, got groupID=%s", updatedItems[updated.taskCursor].groupID)
		}
	})

	t.Run("K wraps to last group from first", func(t *testing.T) {
		m := newNavTestModel()
		m.taskCursor = 0 // On g1

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		updatedItems := updated.getVisibleTaskItems()
		if !updatedItems[updated.taskCursor].isGroup || updatedItems[updated.taskCursor].groupID != "g3" {
			t.Errorf("expected wrap to g3, got groupID=%s", updatedItems[updated.taskCursor].groupID)
		}
	})
}

func TestExpandCollapseAll(t *testing.T) {
	t.Run("e expands all when any are collapsed", func(t *testing.T) {
		m := newNavTestModel()
		delete(m.taskExpandedGroups, "g2") // Collapse g2

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !updated.taskExpandedGroups["g1"] || !updated.taskExpandedGroups["g2"] || !updated.taskExpandedGroups["g3"] {
			t.Error("all groups should be expanded")
		}
	})

	t.Run("e collapses all when all are expanded", func(t *testing.T) {
		m := newNavTestModel()
		// All are already expanded

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.taskExpandedGroups["g1"] || updated.taskExpandedGroups["g2"] || updated.taskExpandedGroups["g3"] {
			t.Error("all groups should be collapsed")
		}
	})
}

func TestAccordionMode(t *testing.T) {
	t.Run("a toggles accordion mode", func(t *testing.T) {
		m := newNavTestModel()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !updated.taskAccordionMode {
			t.Error("accordion mode should be on after pressing a")
		}

		newModel, _ = updated.Update(msg)
		updated = newModel.(Model)

		if updated.taskAccordionMode {
			t.Error("accordion mode should be off after pressing a again")
		}
	})

	t.Run("expanding group collapses others in accordion mode", func(t *testing.T) {
		m := newNavTestModel()
		m.taskAccordionMode = true
		// Collapse all first
		m.taskExpandedGroups = make(map[string]bool)

		// Move to g1 and expand
		m.taskCursor = 0
		msg := tea.KeyMsg{Type: tea.KeySpace}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !updated.taskExpandedGroups["g1"] {
			t.Error("g1 should be expanded")
		}

		// Move to g2 and expand — g1 should collapse
		items := updated.getVisibleTaskItems()
		for i, item := range items {
			if item.isGroup && item.groupID == "g2" {
				updated.taskCursor = i
				break
			}
		}

		newModel, _ = updated.Update(msg)
		updated = newModel.(Model)

		if !updated.taskExpandedGroups["g2"] {
			t.Error("g2 should be expanded")
		}
		if updated.taskExpandedGroups["g1"] {
			t.Error("g1 should be collapsed in accordion mode")
		}
	})

	t.Run("accordion indicator shown in header when active", func(t *testing.T) {
		m := newNavTestModel()
		m.taskAccordionMode = true
		m.taskPanelHeight = 20

		header := m.renderTaskPanelHeader(120)

		if !strings.Contains(header, "accordion") {
			t.Errorf("expected 'accordion' in header when active, got: %s", header)
		}
	})

	t.Run("accordion indicator hidden when inactive", func(t *testing.T) {
		m := newNavTestModel()
		m.taskAccordionMode = false
		m.taskPanelHeight = 20

		header := m.renderTaskPanelHeader(120)

		if strings.Contains(header, "accordion") {
			t.Errorf("should not contain 'accordion' when inactive, got: %s", header)
		}
	})
}
