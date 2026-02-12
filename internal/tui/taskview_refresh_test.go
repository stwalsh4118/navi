package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestManualRefresh(t *testing.T) {
	t.Run("r sets taskRefreshing and returns refresh command", func(t *testing.T) {
		m := newNavTestModel()
		m.taskRefreshing = false

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
		newModel, cmd := m.Update(msg)
		updated := newModel.(Model)

		if !updated.taskRefreshing {
			t.Error("taskRefreshing should be true after pressing r")
		}
		if cmd == nil {
			t.Error("expected a refresh command to be returned")
		}
	})

	t.Run("r is ignored when already refreshing", func(t *testing.T) {
		m := newNavTestModel()
		m.taskRefreshing = true

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
		newModel, cmd := m.Update(msg)
		updated := newModel.(Model)

		if cmd != nil {
			t.Error("expected no command when already refreshing")
		}
		if !updated.taskRefreshing {
			t.Error("taskRefreshing should still be true")
		}
	})

	t.Run("tasksMsg clears taskRefreshing", func(t *testing.T) {
		m := newNavTestModel()
		m.taskRefreshing = true

		msg := tasksMsg{
			groupsByProject: m.taskGroupsByProject,
			errors:          nil,
		}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.taskRefreshing {
			t.Error("taskRefreshing should be false after receiving tasksMsg")
		}
	})

	t.Run("refreshing indicator shown in header", func(t *testing.T) {
		m := newNavTestModel()
		m.taskRefreshing = true
		m.taskPanelHeight = 20

		header := m.renderTaskPanelHeader(120)

		if !strings.Contains(header, "Refreshing...") {
			t.Errorf("expected 'Refreshing...' in header, got: %s", header)
		}
	})

	t.Run("refreshing indicator hidden when not refreshing", func(t *testing.T) {
		m := newNavTestModel()
		m.taskRefreshing = false
		m.taskPanelHeight = 20

		header := m.renderTaskPanelHeader(120)

		if strings.Contains(header, "Refreshing...") {
			t.Errorf("should not show 'Refreshing...' when not refreshing")
		}
	})

	t.Run("r invalidates cache before refresh", func(t *testing.T) {
		m := newNavTestModel()

		// Pre-fill cache
		m.taskCache.Set(testProjectDir, nil, nil)
		cached, ok := m.taskCache.Get(testProjectDir, 60_000_000_000)
		if !ok || cached == nil {
			t.Fatal("cache should have entry before refresh")
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
		newModel, _ := m.Update(msg)
		_ = newModel.(Model)

		// Cache should be invalidated
		_, ok = m.taskCache.Get(testProjectDir, 60_000_000_000)
		if ok {
			t.Error("cache should be invalidated after manual refresh")
		}
	})
}
