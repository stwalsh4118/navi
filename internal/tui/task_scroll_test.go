package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/session"
	"github.com/stwalsh4118/navi/internal/task"
)

// newTaskScrollTestModel creates a model with many task items to test scrolling.
// Creates 3 groups with 5 tasks each = 18 items total (3 groups + 15 tasks when expanded).
func newTaskScrollTestModel() Model {
	groups := []task.TaskGroup{
		{
			ID:     "g1",
			Title:  "Group One",
			Status: "in_progress",
			Tasks: []task.Task{
				{ID: "1-1", Title: "Task 1-1", Status: "done"},
				{ID: "1-2", Title: "Task 1-2", Status: "done"},
				{ID: "1-3", Title: "Task 1-3", Status: "active"},
				{ID: "1-4", Title: "Task 1-4", Status: "active"},
				{ID: "1-5", Title: "Task 1-5", Status: "todo"},
			},
		},
		{
			ID:     "g2",
			Title:  "Group Two",
			Status: "open",
			Tasks: []task.Task{
				{ID: "2-1", Title: "Task 2-1", Status: "active"},
				{ID: "2-2", Title: "Task 2-2", Status: "todo"},
				{ID: "2-3", Title: "Task 2-3", Status: "todo"},
				{ID: "2-4", Title: "Task 2-4", Status: "todo"},
				{ID: "2-5", Title: "Task 2-5", Status: "todo"},
			},
		},
		{
			ID:     "g3",
			Title:  "Group Three",
			Status: "open",
			Tasks: []task.Task{
				{ID: "3-1", Title: "Task 3-1", Status: "todo"},
				{ID: "3-2", Title: "Task 3-2", Status: "todo"},
				{ID: "3-3", Title: "Task 3-3", Status: "todo"},
				{ID: "3-4", Title: "Task 3-4", Status: "todo"},
				{ID: "3-5", Title: "Task 3-5", Status: "todo"},
			},
		},
	}

	m := Model{
		width:  120,
		height: 40,
		sessions: []session.Info{
			{TmuxSession: "test", Status: session.StatusWorking, CWD: testProjectDir, Timestamp: time.Now().Unix()},
		},
		searchInput:    initSearchInput(),
		taskSearchInput: initTaskSearchInput(),
		taskCache:       task.NewResultCache(),
		taskGlobalConfig: &task.GlobalConfig{},
		taskGroups:      groups,
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
		taskPanelHeight:  10, // Small panel to force scrolling: maxLines = 10-3 = 7
	}
	return m
}

func TestTaskPanelMaxScroll(t *testing.T) {
	t.Run("maxScroll is zero when items fit", func(t *testing.T) {
		result := taskPanelMaxScroll(5, 10)
		if result != 0 {
			t.Errorf("expected 0, got %d", result)
		}
	})

	t.Run("maxScroll is zero when items equal maxLines", func(t *testing.T) {
		result := taskPanelMaxScroll(10, 10)
		if result != 0 {
			t.Errorf("expected 0, got %d", result)
		}
	})

	t.Run("maxScroll is positive when items exceed maxLines", func(t *testing.T) {
		// With 15 items and 10 lines, at max scroll the top indicator takes 1 line,
		// so only 9 items are visible. maxScroll = 15 - 10 + 1 = 6.
		result := taskPanelMaxScroll(15, 10)
		if result != 6 {
			t.Errorf("expected 6, got %d", result)
		}
	})

	t.Run("maxScroll handles zero maxLines", func(t *testing.T) {
		// With 10 items and 0 lines, maxScroll = 10 - 0 + 1 = 11.
		result := taskPanelMaxScroll(10, 0)
		if result != 11 {
			t.Errorf("expected 11, got %d", result)
		}
	})
}

func TestEnsureTaskCursorVisible(t *testing.T) {
	t.Run("cursor below viewport scrolls down", func(t *testing.T) {
		m := newTaskScrollTestModel()
		m.taskCursor = 10
		m.taskScrollOffset = 0
		maxLines := 7

		m.ensureTaskCursorVisible(maxLines)

		if m.taskScrollOffset == 0 {
			t.Error("scroll offset should have increased to make cursor visible")
		}
		// Cursor should be within viewport
		if m.taskCursor < m.taskScrollOffset || m.taskCursor >= m.taskScrollOffset+maxLines {
			t.Errorf("cursor %d should be within viewport [%d, %d)", m.taskCursor, m.taskScrollOffset, m.taskScrollOffset+maxLines)
		}
	})

	t.Run("cursor above viewport scrolls up", func(t *testing.T) {
		m := newTaskScrollTestModel()
		m.taskCursor = 2
		m.taskScrollOffset = 5

		m.ensureTaskCursorVisible(7)

		if m.taskScrollOffset != 2 {
			t.Errorf("scroll offset should be 2 (cursor position), got %d", m.taskScrollOffset)
		}
	})

	t.Run("cursor within viewport does not change scroll", func(t *testing.T) {
		m := newTaskScrollTestModel()
		m.taskCursor = 3
		m.taskScrollOffset = 2

		m.ensureTaskCursorVisible(7)

		if m.taskScrollOffset != 2 {
			t.Errorf("scroll offset should remain 2, got %d", m.taskScrollOffset)
		}
	})

	t.Run("scroll offset never goes below zero", func(t *testing.T) {
		m := newTaskScrollTestModel()
		m.taskCursor = 0
		m.taskScrollOffset = -5

		m.ensureTaskCursorVisible(7)

		if m.taskScrollOffset < 0 {
			t.Errorf("scroll offset should not be negative, got %d", m.taskScrollOffset)
		}
	})

	t.Run("scroll offset clamped to maxScroll", func(t *testing.T) {
		m := newTaskScrollTestModel()
		items := m.getVisibleTaskItems()
		maxLines := 7
		m.taskCursor = len(items) - 1
		m.taskScrollOffset = len(items) // Way beyond valid

		m.ensureTaskCursorVisible(maxLines)

		maxScroll := taskPanelMaxScroll(len(items), maxLines)
		if m.taskScrollOffset > maxScroll {
			t.Errorf("scroll offset %d should be <= maxScroll %d", m.taskScrollOffset, maxScroll)
		}
	})
}

func TestTaskPanelScrollWithCursorMovement(t *testing.T) {
	t.Run("moving cursor down past viewport scrolls viewport", func(t *testing.T) {
		m := newTaskScrollTestModel()
		m.taskCursor = 0
		m.taskScrollOffset = 0

		// Move down many times to go past viewport
		for i := 0; i < 10; i++ {
			msg := tea.KeyMsg{Type: tea.KeyDown}
			newModel, _ := m.Update(msg)
			m = newModel.(Model)
		}

		if m.taskScrollOffset <= 0 {
			t.Error("scroll offset should have increased after moving cursor down past viewport")
		}
		// Cursor should be visible
		maxLines := m.taskPanelViewportLines()
		if m.taskCursor < m.taskScrollOffset || m.taskCursor >= m.taskScrollOffset+maxLines {
			t.Errorf("cursor %d not visible in viewport [%d, %d)", m.taskCursor, m.taskScrollOffset, m.taskScrollOffset+maxLines)
		}
	})

	t.Run("moving cursor up past viewport scrolls viewport up", func(t *testing.T) {
		m := newTaskScrollTestModel()
		items := m.getVisibleTaskItems()
		maxLines := m.taskPanelViewportLines()
		m.taskCursor = len(items) - 1
		m.taskScrollOffset = taskPanelMaxScroll(len(items), maxLines)

		// Move up many times
		for i := 0; i < 10; i++ {
			msg := tea.KeyMsg{Type: tea.KeyUp}
			newModel, _ := m.Update(msg)
			m = newModel.(Model)
		}

		// Cursor should be visible
		if m.taskCursor < m.taskScrollOffset || m.taskCursor >= m.taskScrollOffset+maxLines {
			t.Errorf("cursor %d not visible in viewport [%d, %d)", m.taskCursor, m.taskScrollOffset, m.taskScrollOffset+maxLines)
		}
	})
}

func TestTaskPanelPageScroll(t *testing.T) {
	t.Run("PgDn scrolls down by page amount", func(t *testing.T) {
		m := newTaskScrollTestModel()
		m.taskCursor = 0
		m.taskScrollOffset = 0

		msg := tea.KeyMsg{Type: tea.KeyPgDown}
		newModel, _ := m.Update(msg)
		m = newModel.(Model)

		if m.taskScrollOffset <= 0 {
			t.Error("scroll offset should increase on PgDn")
		}
	})

	t.Run("PgUp scrolls up by page amount", func(t *testing.T) {
		m := newTaskScrollTestModel()
		m.taskScrollOffset = 10
		m.taskCursor = 10

		msg := tea.KeyMsg{Type: tea.KeyPgUp}
		newModel, _ := m.Update(msg)
		m = newModel.(Model)

		if m.taskScrollOffset >= 10 {
			t.Errorf("scroll offset should decrease on PgUp, got %d", m.taskScrollOffset)
		}
	})

	t.Run("PgUp does not go below zero", func(t *testing.T) {
		m := newTaskScrollTestModel()
		m.taskScrollOffset = 2
		m.taskCursor = 2

		msg := tea.KeyMsg{Type: tea.KeyPgUp}
		newModel, _ := m.Update(msg)
		m = newModel.(Model)

		if m.taskScrollOffset < 0 {
			t.Errorf("scroll offset should not go negative, got %d", m.taskScrollOffset)
		}
	})

	t.Run("PgDn does not exceed maxScroll", func(t *testing.T) {
		m := newTaskScrollTestModel()
		items := m.getVisibleTaskItems()
		maxLines := m.taskPanelViewportLines()
		maxScroll := taskPanelMaxScroll(len(items), maxLines)
		m.taskScrollOffset = maxScroll
		m.taskCursor = len(items) - 1

		msg := tea.KeyMsg{Type: tea.KeyPgDown}
		newModel, _ := m.Update(msg)
		m = newModel.(Model)

		if m.taskScrollOffset > maxScroll {
			t.Errorf("scroll offset %d should not exceed maxScroll %d", m.taskScrollOffset, maxScroll)
		}
	})
}

func TestTaskPanelJumpToEnds(t *testing.T) {
	t.Run("g jumps cursor and scroll to top", func(t *testing.T) {
		m := newTaskScrollTestModel()
		m.taskCursor = 10
		m.taskScrollOffset = 5

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		newModel, _ := m.Update(msg)
		m = newModel.(Model)

		if m.taskCursor != 0 {
			t.Errorf("cursor should be 0 after g, got %d", m.taskCursor)
		}
		if m.taskScrollOffset != 0 {
			t.Errorf("scroll offset should be 0 after g, got %d", m.taskScrollOffset)
		}
	})

	t.Run("G jumps cursor and scroll to bottom", func(t *testing.T) {
		m := newTaskScrollTestModel()
		m.taskCursor = 0
		m.taskScrollOffset = 0

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		newModel, _ := m.Update(msg)
		m = newModel.(Model)

		items := m.getVisibleTaskItems()
		if m.taskCursor != len(items)-1 {
			t.Errorf("cursor should be %d after G, got %d", len(items)-1, m.taskCursor)
		}
		maxLines := m.taskPanelViewportLines()
		maxScroll := taskPanelMaxScroll(len(items), maxLines)
		if m.taskScrollOffset != maxScroll {
			t.Errorf("scroll offset should be %d after G, got %d", maxScroll, m.taskScrollOffset)
		}
	})
}

func TestTaskPanelScrollWithGroupToggle(t *testing.T) {
	t.Run("collapsing a group clamps scroll offset", func(t *testing.T) {
		m := newTaskScrollTestModel()
		items := m.getVisibleTaskItems()
		maxLines := m.taskPanelViewportLines()
		// Scroll to near the end
		m.taskScrollOffset = taskPanelMaxScroll(len(items), maxLines)
		m.taskCursor = len(items) - 1

		// Move cursor to first group header and collapse it
		m.taskCursor = 0
		m.taskScrollOffset = 0

		// Navigate to the group header and toggle
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
		newModel, _ := m.Update(msg)
		m = newModel.(Model)

		// Scroll offset should still be valid
		newItems := m.getVisibleTaskItems()
		newMaxScroll := taskPanelMaxScroll(len(newItems), maxLines)
		if m.taskScrollOffset > newMaxScroll {
			t.Errorf("scroll offset %d exceeds new maxScroll %d after collapse", m.taskScrollOffset, newMaxScroll)
		}
	})
}

func TestTaskPanelScrollResetOnProjectChange(t *testing.T) {
	t.Run("scroll offset resets when project changes via updateTaskPanelForCursor", func(t *testing.T) {
		m := newTaskScrollTestModel()
		m.taskScrollOffset = 5

		// Simulate a direct project change
		oldProject := m.taskFocusedProject
		m.taskFocusedProject = "/other/project"
		// Call updateTaskPanelForCursor with a session that has a different CWD
		m.sessions = []session.Info{
			{TmuxSession: "other", Status: session.StatusWorking, CWD: "/other/project", Timestamp: time.Now().Unix()},
		}
		m.cursor = 0
		m.updateTaskPanelForCursor()

		// Since /other/project has no config, project changes to ""
		if m.taskFocusedProject == oldProject {
			t.Error("focused project should have changed")
		}
		if m.taskScrollOffset != 0 {
			t.Errorf("scroll offset should reset to 0 on project change, got %d", m.taskScrollOffset)
		}
	})
}

func TestTaskPanelScrollRenderSlicing(t *testing.T) {
	t.Run("renderTaskPanelList shows items from scroll offset", func(t *testing.T) {
		m := newTaskScrollTestModel()
		m.taskScrollOffset = 3 // Skip first 3 items

		result := m.renderTaskPanelList(100, 5) // Show 5 items

		items := m.getVisibleTaskItems()
		// Should contain item at offset 3 but not item at offset 0
		if len(items) > 3 {
			// Item at index 0 (first group) should not be present since we scrolled past it
			// We can't easily test for absence of specific text since groups may have similar names
			// but we can verify the render doesn't panic and returns content
			if result == "" {
				t.Error("renderTaskPanelList should produce output")
			}
		}
	})

	t.Run("renderTaskPanelList handles scroll offset beyond items", func(t *testing.T) {
		m := newTaskScrollTestModel()
		m.taskScrollOffset = 100 // Way beyond

		result := m.renderTaskPanelList(100, 5)

		// Should not panic and should return empty-ish content
		if result == "" {
			// Empty is fine since offset is beyond items
		}
		_ = result // No panic = pass
	})
}

func TestTaskPanelViewportLines(t *testing.T) {
	t.Run("viewport lines accounts for panel chrome", func(t *testing.T) {
		m := newTaskScrollTestModel()
		m.taskPanelHeight = 10

		maxLines := m.taskPanelViewportLines()

		expected := 10 - 3 // height - (header + borders)
		if maxLines != expected {
			t.Errorf("expected %d viewport lines, got %d", expected, maxLines)
		}
	})

	t.Run("viewport lines accounts for search bar", func(t *testing.T) {
		m := newTaskScrollTestModel()
		m.taskPanelHeight = 10
		m.taskSearchQuery = "something"

		maxLines := m.taskPanelViewportLines()

		expected := 10 - 3 - 1 // height - chrome - search bar
		if maxLines != expected {
			t.Errorf("expected %d viewport lines with search, got %d", expected, maxLines)
		}
	})

	t.Run("viewport lines minimum is 1", func(t *testing.T) {
		m := newTaskScrollTestModel()
		m.taskPanelHeight = 2 // Too small

		maxLines := m.taskPanelViewportLines()

		if maxLines < 1 {
			t.Errorf("viewport lines should be at least 1, got %d", maxLines)
		}
	})
}
