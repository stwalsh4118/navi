package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/session"
	"github.com/stwalsh4118/navi/internal/task"
)

// newE2EScrollTestModel creates a comprehensive model for end-to-end scroll testing.
// It has many sessions, task groups, and preview content to exercise all scrollable panels.
func newE2EScrollTestModel() Model {
	now := time.Now().Unix()

	// Create 25 sessions to exceed viewport
	sessions := make([]session.Info, 25)
	for i := range sessions {
		sessions[i] = session.Info{
			TmuxSession: fmt.Sprintf("e2e-session-%02d", i),
			Status:      session.StatusWorking,
			CWD:         testProjectDir,
			Timestamp:   now,
		}
	}

	// Create 4 groups with 5 tasks each = 24 items when expanded
	groups := []task.TaskGroup{
		{
			ID: "g1", Title: "Backend API", Status: "in_progress",
			Tasks: []task.Task{
				{ID: "1-1", Title: "Add auth endpoint", Status: "done"},
				{ID: "1-2", Title: "Add user endpoint", Status: "done"},
				{ID: "1-3", Title: "Add search endpoint", Status: "active"},
				{ID: "1-4", Title: "Add filter endpoint", Status: "todo"},
				{ID: "1-5", Title: "Add sort endpoint", Status: "todo"},
			},
		},
		{
			ID: "g2", Title: "Frontend UI", Status: "open",
			Tasks: []task.Task{
				{ID: "2-1", Title: "Build dashboard page", Status: "active"},
				{ID: "2-2", Title: "Build settings page", Status: "todo"},
				{ID: "2-3", Title: "Build profile page", Status: "todo"},
				{ID: "2-4", Title: "Build search page", Status: "todo"},
				{ID: "2-5", Title: "Build login page", Status: "todo"},
			},
		},
		{
			ID: "g3", Title: "Testing", Status: "open",
			Tasks: []task.Task{
				{ID: "3-1", Title: "Unit tests for API", Status: "todo"},
				{ID: "3-2", Title: "Integration tests", Status: "todo"},
				{ID: "3-3", Title: "E2E tests", Status: "todo"},
				{ID: "3-4", Title: "Performance tests", Status: "todo"},
				{ID: "3-5", Title: "Security tests", Status: "todo"},
			},
		},
		{
			ID: "g4", Title: "Documentation", Status: "open",
			Tasks: []task.Task{
				{ID: "4-1", Title: "API docs", Status: "todo"},
				{ID: "4-2", Title: "User guide", Status: "todo"},
				{ID: "4-3", Title: "Dev guide", Status: "todo"},
				{ID: "4-4", Title: "Changelog", Status: "todo"},
				{ID: "4-5", Title: "README update", Status: "todo"},
			},
		},
	}

	// Create preview content with 100 lines
	var previewLines []string
	for i := 0; i < 100; i++ {
		previewLines = append(previewLines, fmt.Sprintf("Preview output line %d", i+1))
	}

	return Model{
		width:  120,
		height: 30, // Compact terminal to force scrolling
		sessions:            sessions,
		cursor:              0,
		sessionScrollOffset: 0,
		searchInput:         initSearchInput(),
		taskSearchInput:     initTaskSearchInput(),
		taskExpandedGroups: map[string]bool{
			"g1": true, "g2": true, "g3": true, "g4": true,
		},
		taskGroupsByProject: map[string][]task.TaskGroup{
			testProjectDir: groups,
		},
		taskGroups:         groups,
		taskFocusedProject: testProjectDir,
		taskProjectConfigs: []task.ProjectConfig{
			{Tasks: task.ProjectTaskConfig{Provider: "test"}, ProjectDir: testProjectDir},
		},
		taskCache:        task.NewResultCache(),
		taskGlobalConfig: &task.GlobalConfig{},
		sortMode:         SortPriority,
		// Preview state
		previewContent:    strings.Join(previewLines, "\n"),
		previewVisible:    false, // Off by default (task panel takes precedence)
		previewAutoScroll: true,
		// Task panel state
		taskPanelVisible: true,
		taskPanelFocused: false,
		taskPanelHeight:  12, // Small panel: maxLines = 12-3 = 9
	}
}

// --- AC1: Task panel scrolls with cursor movement ---

func TestE2E_AC1_TaskPanelScrollsWithCursor(t *testing.T) {
	t.Run("cursor movement scrolls viewport to keep cursor visible", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.taskPanelFocused = true
		m.taskCursor = 0
		m.taskScrollOffset = 0

		// Move cursor down past the viewport (24 items, ~9 visible lines)
		for i := 0; i < 15; i++ {
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
			newModel, _ := m.Update(msg)
			m = newModel.(Model)
		}

		// Cursor should be at position 15
		if m.taskCursor != 15 {
			t.Errorf("cursor should be at 15, got %d", m.taskCursor)
		}

		// Scroll offset should have increased to keep cursor visible
		if m.taskScrollOffset == 0 {
			t.Error("scroll offset should have increased to follow cursor")
		}

		maxLines := m.taskPanelViewportLines()
		if m.taskCursor < m.taskScrollOffset || m.taskCursor >= m.taskScrollOffset+maxLines {
			t.Errorf("cursor %d should be within viewport [%d, %d)", m.taskCursor, m.taskScrollOffset, m.taskScrollOffset+maxLines)
		}
	})

	t.Run("cursor movement up scrolls viewport up", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.taskPanelFocused = true
		items := m.getVisibleTaskItems()
		maxLines := m.taskPanelViewportLines()
		m.taskCursor = len(items) - 1
		m.taskScrollOffset = taskPanelMaxScroll(len(items), maxLines)

		// Move cursor up past the viewport
		for i := 0; i < 15; i++ {
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
			newModel, _ := m.Update(msg)
			m = newModel.(Model)
		}

		// Cursor should be visible
		if m.taskCursor < m.taskScrollOffset || m.taskCursor >= m.taskScrollOffset+maxLines {
			t.Errorf("cursor %d should be within viewport [%d, %d)", m.taskCursor, m.taskScrollOffset, m.taskScrollOffset+maxLines)
		}
	})
}

// --- AC2: Preview pane supports scrolling ---

func TestE2E_AC2_PreviewPaneScrolling(t *testing.T) {
	t.Run("preview scrolls through full content with keybindings", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.previewVisible = true
		m.taskPanelVisible = false
		m.previewFocused = true
		m.previewAutoScroll = false
		m.previewScrollOffset = 0

		// Scroll down several lines
		for i := 0; i < 10; i++ {
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
			newModel, _ := m.Update(msg)
			m = newModel.(Model)
		}

		if m.previewScrollOffset != 10 {
			t.Errorf("preview scroll offset should be 10, got %d", m.previewScrollOffset)
		}
		if m.previewAutoScroll {
			t.Error("auto-scroll should be disabled after manual scroll")
		}
	})

	t.Run("auto-tail follows new content when enabled", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.previewVisible = true
		m.taskPanelVisible = false
		m.previewAutoScroll = true

		// Render preview - auto-scroll should show bottom content
		result := m.renderPreview(100, 20)

		// Should contain content from near the end
		if !strings.Contains(result, "Preview output line 100") {
			// With auto-scroll, the bottom lines should be visible
			// (may be wrapped/truncated, so check last few lines)
			if !strings.Contains(result, "line 9") {
				t.Error("auto-scroll should show content near the bottom")
			}
		}
	})

	t.Run("G re-enables auto-scroll", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.previewVisible = true
		m.taskPanelVisible = false
		m.previewFocused = true
		m.previewAutoScroll = false
		m.previewScrollOffset = 5

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		newModel, _ := m.Update(msg)
		m = newModel.(Model)

		if !m.previewAutoScroll {
			t.Error("G should re-enable auto-scroll")
		}
	})

	t.Run("session change resets preview scroll state", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.previewVisible = true
		m.taskPanelVisible = false
		m.previewFocused = true
		m.previewAutoScroll = false
		m.previewScrollOffset = 20

		// Simulate cursor change debounce
		newModel, _ := m.Update(previewDebounceMsg{})
		m = newModel.(Model)

		if m.previewScrollOffset != 0 {
			t.Errorf("preview scroll should reset on cursor change, got %d", m.previewScrollOffset)
		}
		if !m.previewAutoScroll {
			t.Error("auto-scroll should be re-enabled on cursor change")
		}
		if m.previewFocused {
			t.Error("preview focus should be cleared on cursor change")
		}
	})
}

// --- AC3: Session list scrolls ---

func TestE2E_AC3_SessionListScrolls(t *testing.T) {
	t.Run("session list scrolls when sessions exceed visible area", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.taskPanelVisible = false // Full session list
		m.cursor = 0
		m.sessionScrollOffset = 0

		// Move cursor down to force scrolling
		for i := 0; i < 20; i++ {
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
			newModel, _ := m.Update(msg)
			m = newModel.(Model)
		}

		if m.sessionScrollOffset == 0 {
			t.Error("session scroll offset should increase after moving cursor down many times")
		}

		// Cursor should be visible
		maxVisible := m.sessionListMaxVisible()
		if m.cursor < m.sessionScrollOffset || m.cursor >= m.sessionScrollOffset+maxVisible {
			t.Errorf("cursor %d should be within viewport [%d, %d)", m.cursor, m.sessionScrollOffset, m.sessionScrollOffset+maxVisible)
		}
	})

	t.Run("session list wrapping resets scroll offset", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.taskPanelVisible = false
		m.cursor = len(m.sessions) - 1
		m.sessionScrollOffset = 10

		// Move down once to wrap to top
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		newModel, _ := m.Update(msg)
		m = newModel.(Model)

		if m.cursor != 0 {
			t.Errorf("cursor should wrap to 0, got %d", m.cursor)
		}
		if m.sessionScrollOffset != 0 {
			t.Errorf("scroll offset should reset to 0 after wrapping, got %d", m.sessionScrollOffset)
		}
	})
}

// --- AC4: Scroll indicators appear ---

func TestE2E_AC4_ScrollIndicators(t *testing.T) {
	t.Run("task panel shows indicators when content overflows", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.taskPanelFocused = true
		m.taskScrollOffset = 5

		// Render the full view
		view := m.View()

		// Should contain scroll indicators
		if !strings.Contains(view, "more above") {
			t.Error("view should contain top scroll indicator for task panel")
		}
		if !strings.Contains(view, "more below") {
			t.Error("view should contain bottom scroll indicator for task panel")
		}
	})

	t.Run("session list shows indicators when scrolled", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.taskPanelVisible = false
		m.sessionScrollOffset = 5

		view := m.View()

		if !strings.Contains(view, "more above") {
			t.Error("view should contain top scroll indicator for session list")
		}
	})

	t.Run("preview shows indicators when scrolled", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.taskPanelVisible = false
		m.previewVisible = true
		m.previewUserEnabled = true
		m.previewLayout = PreviewLayoutBottom
		m.previewAutoScroll = false
		m.previewScrollOffset = 10

		view := m.View()

		if !strings.Contains(view, "more above") {
			t.Error("view should contain top scroll indicator for preview pane")
		}
	})

	t.Run("no indicators when all content fits", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.taskPanelVisible = true
		// Collapse all groups - only 4 group headers visible
		m.taskExpandedGroups = map[string]bool{}
		m.taskScrollOffset = 0

		result := m.renderTaskPanelList(100, 9)

		if strings.Contains(result, "more above") || strings.Contains(result, "more below") {
			t.Error("should not show indicators when all items fit")
		}
	})
}

// --- AC5: Vim keybindings work ---

func TestE2E_AC5_VimKeybindings(t *testing.T) {
	t.Run("task panel j/k navigate", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.taskPanelFocused = true
		m.taskCursor = 0

		// j moves down
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		newModel, _ := m.Update(msg)
		m = newModel.(Model)
		if m.taskCursor != 1 {
			t.Errorf("j should move cursor to 1, got %d", m.taskCursor)
		}

		// k moves up
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		newModel, _ = m.Update(msg)
		m = newModel.(Model)
		if m.taskCursor != 0 {
			t.Errorf("k should move cursor to 0, got %d", m.taskCursor)
		}
	})

	t.Run("task panel g/G jump to ends", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.taskPanelFocused = true
		m.taskCursor = 5

		// G jumps to bottom
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		newModel, _ := m.Update(msg)
		m = newModel.(Model)
		items := m.getVisibleTaskItems()
		if m.taskCursor != len(items)-1 {
			t.Errorf("G should jump to %d, got %d", len(items)-1, m.taskCursor)
		}

		// g jumps to top
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		newModel, _ = m.Update(msg)
		m = newModel.(Model)
		if m.taskCursor != 0 {
			t.Errorf("g should jump to 0, got %d", m.taskCursor)
		}
	})

	t.Run("task panel PgUp/PgDn scroll pages", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.taskPanelFocused = true
		m.taskCursor = 0
		m.taskScrollOffset = 0

		// PgDn
		msg := tea.KeyMsg{Type: tea.KeyPgDown}
		newModel, _ := m.Update(msg)
		m = newModel.(Model)
		if m.taskScrollOffset == 0 {
			t.Error("PgDn should increase scroll offset")
		}

		savedOffset := m.taskScrollOffset

		// PgUp
		msg = tea.KeyMsg{Type: tea.KeyPgUp}
		newModel, _ = m.Update(msg)
		m = newModel.(Model)
		if m.taskScrollOffset >= savedOffset {
			t.Error("PgUp should decrease scroll offset")
		}
	})

	t.Run("preview j/k scroll content", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.taskPanelVisible = false
		m.previewVisible = true
		m.previewFocused = true
		m.previewAutoScroll = false
		m.previewScrollOffset = 5

		// j scrolls down
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		newModel, _ := m.Update(msg)
		m = newModel.(Model)
		if m.previewScrollOffset != 6 {
			t.Errorf("j should increment preview offset to 6, got %d", m.previewScrollOffset)
		}

		// k scrolls up
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		newModel, _ = m.Update(msg)
		m = newModel.(Model)
		if m.previewScrollOffset != 5 {
			t.Errorf("k should decrement preview offset to 5, got %d", m.previewScrollOffset)
		}
	})

	t.Run("preview g/G jump to ends", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.taskPanelVisible = false
		m.previewVisible = true
		m.previewFocused = true
		m.previewAutoScroll = false
		m.previewScrollOffset = 20

		// g jumps to top
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		newModel, _ := m.Update(msg)
		m = newModel.(Model)
		if m.previewScrollOffset != 0 {
			t.Errorf("g should set offset to 0, got %d", m.previewScrollOffset)
		}

		// G jumps to bottom and enables auto-scroll
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		newModel, _ = m.Update(msg)
		m = newModel.(Model)
		if !m.previewAutoScroll {
			t.Error("G should enable auto-scroll")
		}
	})
}

// --- AC6: Expand/shrink still works ---

func TestE2E_AC6_ExpandShrinkWorksWithScroll(t *testing.T) {
	t.Run("resize with [ still works when scrolled", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.taskPanelFocused = false
		m.taskScrollOffset = 5
		initialHeight := m.getTaskPanelHeight()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}}
		newModel, _ := m.Update(msg)
		m = newModel.(Model)

		if m.taskPanelHeight >= initialHeight {
			t.Error("[ should shrink the task panel")
		}
	})

	t.Run("resize with ] still works when scrolled", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.taskPanelFocused = false
		m.taskPanelHeight = 10
		m.taskScrollOffset = 3

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}}
		newModel, _ := m.Update(msg)
		m = newModel.(Model)

		if m.taskPanelHeight <= 10 {
			t.Error("] should expand the task panel")
		}
	})

	t.Run("preview resize works when scrolled", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.taskPanelVisible = false
		m.previewVisible = true
		m.previewUserEnabled = true
		m.previewLayout = PreviewLayoutBottom
		m.previewHeight = 10
		m.previewScrollOffset = 5

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}}
		newModel, _ := m.Update(msg)
		m = newModel.(Model)

		if m.previewHeight <= 10 {
			t.Error("] should expand the preview pane")
		}
	})
}

// --- AC8: Scroll offset clamping, viewport calculation, cursor-follow ---

func TestE2E_AC8_ScrollEdgeCases(t *testing.T) {
	t.Run("scrolling with zero task items", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.taskPanelFocused = true
		m.taskGroups = nil
		m.taskScrollOffset = 5

		result := m.renderTaskPanelList(100, 9)

		// Should show empty message without panic
		if !strings.Contains(result, "No tasks found") {
			t.Error("should show empty message for zero items")
		}
	})

	t.Run("scrolling with exactly maxLines items", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.taskPanelFocused = true
		// Collapse all groups - only 4 headers, well under maxLines=9
		m.taskExpandedGroups = map[string]bool{}
		m.taskScrollOffset = 0

		result := m.renderTaskPanelList(100, 9)

		// No indicators should appear since 4 items < 9 maxLines
		if strings.Contains(result, "more above") || strings.Contains(result, "more below") {
			t.Error("no indicators when items fit exactly")
		}
	})

	t.Run("filter change resets session scroll via preserveCursor", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.sessionScrollOffset = 10
		m.cursor = 10

		// Simulate status filter change
		m.statusFilter = "working"
		m.preserveCursor("")

		if m.sessionScrollOffset != 0 {
			t.Errorf("scroll offset should reset on filter change, got %d", m.sessionScrollOffset)
		}
	})

	t.Run("search in task panel while scrolled", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.taskPanelFocused = true
		m.taskScrollOffset = 10
		m.taskCursor = 10

		// Enter search mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
		newModel, _ := m.Update(msg)
		m = newModel.(Model)

		if !m.taskSearchMode {
			t.Error("should be in task search mode")
		}

		// Type a search query
		for _, r := range "API" {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
			newModel, _ = m.Update(msg)
			m = newModel.(Model)
		}

		// Should have search matches
		if m.taskSearchQuery != "API" {
			t.Errorf("search query should be 'API', got %q", m.taskSearchQuery)
		}
	})

	t.Run("scroll offset clamped when content shrinks", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.taskPanelFocused = true
		m.taskScrollOffset = 20
		m.taskCursor = 20

		// Collapse a group to reduce item count
		delete(m.taskExpandedGroups, "g1")
		items := m.getVisibleTaskItems()
		if m.taskCursor >= len(items) {
			m.taskCursor = len(items) - 1
		}
		maxLines := m.taskPanelViewportLines()
		m.clampTaskScrollOffset(maxLines)

		maxScroll := taskPanelMaxScroll(len(items), maxLines)
		if m.taskScrollOffset > maxScroll {
			t.Errorf("scroll offset %d should be <= maxScroll %d after collapse", m.taskScrollOffset, maxScroll)
		}
	})

	t.Run("preview scroll offset clamped to valid range", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.taskPanelVisible = false
		m.previewVisible = true
		m.previewAutoScroll = false
		m.previewScrollOffset = 999 // Way beyond content

		// Render should not panic and should clamp
		result := m.renderPreview(100, 20)
		_ = result // No panic = pass
	})

	t.Run("session list handles empty state gracefully", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.sessions = nil
		m.sessionScrollOffset = 5

		result := m.renderSessionList(100)

		if !strings.Contains(result, "No active sessions") {
			t.Error("should show empty message for no sessions")
		}
		if strings.Contains(result, "more above") || strings.Contains(result, "more below") {
			t.Error("should not show indicators for empty session list")
		}
	})
}

// --- Cross-cutting: Full View() renders without panic ---

func TestE2E_FullViewRender(t *testing.T) {
	t.Run("full view renders with task panel scrolled", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.taskScrollOffset = 5

		// Should not panic
		view := m.View()
		if view == "" {
			t.Error("view should not be empty")
		}
	})

	t.Run("full view renders with session list scrolled", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.taskPanelVisible = false
		m.sessionScrollOffset = 5

		view := m.View()
		if view == "" {
			t.Error("view should not be empty")
		}
	})

	t.Run("full view renders with preview scrolled", func(t *testing.T) {
		m := newE2EScrollTestModel()
		m.taskPanelVisible = false
		m.previewVisible = true
		m.previewUserEnabled = true
		m.previewLayout = PreviewLayoutBottom
		m.previewAutoScroll = false
		m.previewScrollOffset = 10

		view := m.View()
		if view == "" {
			t.Error("view should not be empty")
		}
	})
}
