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

// newSessionScrollTestModel creates a model with many sessions to test scrolling.
func newSessionScrollTestModel() Model {
	now := time.Now().Unix()
	sessions := make([]session.Info, 20)
	for i := range sessions {
		sessions[i] = session.Info{
			TmuxSession: fmt.Sprintf("session-%02d", i),
			Status:      session.StatusWorking,
			CWD:         "/home/user/project",
			Timestamp:   now,
		}
	}

	m := Model{
		width:               120,
		height:              30, // Small terminal to force session scrolling
		sessions:            sessions,
		cursor:              0,
		sessionScrollOffset: 0,
		searchInput:         initSearchInput(),
		taskSearchInput:     initTaskSearchInput(),
		taskExpandedGroups:  make(map[string]bool),
		taskGroupsByProject: make(map[string][]task.TaskGroup),
		taskCache:           task.NewResultCache(),
		taskGlobalConfig:    &task.GlobalConfig{},
		sortMode:            SortPriority,
	}
	return m
}

func TestEnsureSessionCursorVisible(t *testing.T) {
	t.Run("cursor below viewport scrolls down", func(t *testing.T) {
		m := newSessionScrollTestModel()
		m.cursor = 10
		m.sessionScrollOffset = 0

		m.ensureSessionCursorVisible(5)

		if m.sessionScrollOffset == 0 {
			t.Error("scroll offset should have increased")
		}
		if m.cursor < m.sessionScrollOffset || m.cursor >= m.sessionScrollOffset+5 {
			t.Errorf("cursor %d should be within viewport [%d, %d)", m.cursor, m.sessionScrollOffset, m.sessionScrollOffset+5)
		}
	})

	t.Run("cursor above viewport scrolls up", func(t *testing.T) {
		m := newSessionScrollTestModel()
		m.cursor = 2
		m.sessionScrollOffset = 5

		m.ensureSessionCursorVisible(5)

		if m.sessionScrollOffset != 2 {
			t.Errorf("scroll offset should be 2, got %d", m.sessionScrollOffset)
		}
	})

	t.Run("cursor within viewport does not change scroll", func(t *testing.T) {
		m := newSessionScrollTestModel()
		m.cursor = 3
		m.sessionScrollOffset = 2

		m.ensureSessionCursorVisible(5)

		if m.sessionScrollOffset != 2 {
			t.Errorf("scroll offset should remain 2, got %d", m.sessionScrollOffset)
		}
	})

	t.Run("scroll offset never negative", func(t *testing.T) {
		m := newSessionScrollTestModel()
		m.cursor = 0
		m.sessionScrollOffset = -3

		m.ensureSessionCursorVisible(5)

		if m.sessionScrollOffset < 0 {
			t.Errorf("scroll offset should not be negative, got %d", m.sessionScrollOffset)
		}
	})

	t.Run("scroll offset clamped to max", func(t *testing.T) {
		m := newSessionScrollTestModel()
		maxSessions := 5
		m.cursor = len(m.sessions) - 1
		m.sessionScrollOffset = 100

		m.ensureSessionCursorVisible(maxSessions)

		maxScroll := len(m.sessions) - maxSessions
		if m.sessionScrollOffset > maxScroll {
			t.Errorf("scroll offset %d should be <= %d", m.sessionScrollOffset, maxScroll)
		}
	})

	t.Run("empty session list resets to zero", func(t *testing.T) {
		m := newSessionScrollTestModel()
		m.sessions = nil
		m.sessionScrollOffset = 5

		m.ensureSessionCursorVisible(5)

		if m.sessionScrollOffset != 0 {
			t.Errorf("scroll offset should be 0 for empty list, got %d", m.sessionScrollOffset)
		}
	})
}

func TestSessionScrollWithCursorMovement(t *testing.T) {
	t.Run("moving cursor down eventually scrolls viewport", func(t *testing.T) {
		m := newSessionScrollTestModel()

		for i := 0; i < 15; i++ {
			msg := tea.KeyMsg{Type: tea.KeyDown}
			newModel, _ := m.Update(msg)
			m = newModel.(Model)
		}

		if m.sessionScrollOffset <= 0 {
			t.Error("scroll offset should have increased after many down presses")
		}
	})

	t.Run("wrapping from bottom to top resets scroll", func(t *testing.T) {
		m := newSessionScrollTestModel()
		m.cursor = len(m.sessions) - 1
		m.sessionScrollOffset = 10

		// Move down one more to wrap to top
		msg := tea.KeyMsg{Type: tea.KeyDown}
		newModel, _ := m.Update(msg)
		m = newModel.(Model)

		if m.cursor != 0 {
			t.Errorf("cursor should wrap to 0, got %d", m.cursor)
		}
		if m.sessionScrollOffset != 0 {
			t.Errorf("scroll offset should be 0 after wrapping to top, got %d", m.sessionScrollOffset)
		}
	})

	t.Run("wrapping from top to bottom scrolls to end", func(t *testing.T) {
		m := newSessionScrollTestModel()
		m.cursor = 0
		m.sessionScrollOffset = 0

		// Move up to wrap to bottom
		msg := tea.KeyMsg{Type: tea.KeyUp}
		newModel, _ := m.Update(msg)
		m = newModel.(Model)

		if m.cursor != len(m.sessions)-1 {
			t.Errorf("cursor should wrap to %d, got %d", len(m.sessions)-1, m.cursor)
		}
		if m.sessionScrollOffset == 0 {
			t.Error("scroll offset should increase to show bottom item")
		}
	})
}

func TestSessionScrollWithFilterChange(t *testing.T) {
	t.Run("filter change resets scroll via preserveCursor", func(t *testing.T) {
		m := newSessionScrollTestModel()
		m.sessionScrollOffset = 5
		m.cursor = 5

		// Call preserveCursor with empty name (simulates filter clear)
		m.preserveCursor("")

		if m.sessionScrollOffset != 0 {
			t.Errorf("scroll offset should reset to 0, got %d", m.sessionScrollOffset)
		}
	})
}

func TestSessionScrollRenderSlicing(t *testing.T) {
	t.Run("renderSessionList starts from scroll offset", func(t *testing.T) {
		m := newSessionScrollTestModel()
		m.sessionScrollOffset = 5

		result := m.renderSessionList(100)

		// First session (session-00) should NOT appear since we're scrolled past it
		if strings.Contains(result, "session-00") {
			t.Error("session-00 should not appear when scrolled past it")
		}
		// Session at offset (session-05) should appear
		if !strings.Contains(result, "session-05") {
			t.Error("session-05 should appear at scroll offset 5")
		}
	})
}

func TestSessionListMaxVisible(t *testing.T) {
	t.Run("returns positive value", func(t *testing.T) {
		m := newSessionScrollTestModel()
		maxVisible := m.sessionListMaxVisible()
		if maxVisible < 1 {
			t.Errorf("maxVisible should be at least 1, got %d", maxVisible)
		}
	})

	t.Run("accounts for task panel height", func(t *testing.T) {
		m := newSessionScrollTestModel()
		normalMax := m.sessionListMaxVisible()

		m.taskPanelVisible = true
		m.taskPanelHeight = 10
		withPanel := m.sessionListMaxVisible()

		if withPanel >= normalMax {
			t.Errorf("maxVisible with task panel (%d) should be less than without (%d)", withPanel, normalMax)
		}
	})
}

