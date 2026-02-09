package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/session"
	"github.com/stwalsh4118/navi/internal/task"
)

// newPreviewScrollTestModel creates a model with preview visible and content for testing.
func newPreviewScrollTestModel() Model {
	// Create content with many lines
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, strings.Repeat("x", 50))
	}
	content := strings.Join(lines, "\n")

	m := Model{
		width:  120,
		height: 40,
		sessions: []session.Info{
			{TmuxSession: "test", Status: session.StatusWorking, CWD: "/tmp", Timestamp: time.Now().Unix()},
		},
		cursor:              0,
		previewVisible:      true,
		previewUserEnabled:  true,
		previewContent:      content,
		previewLayout:       PreviewLayoutBottom,
		previewWrap:         true,
		previewAutoScroll:   true,
		previewScrollOffset: 0,
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

func TestPreviewFocusMode(t *testing.T) {
	t.Run("tab enters preview focus when preview visible", func(t *testing.T) {
		m := newPreviewScrollTestModel()

		msg := tea.KeyMsg{Type: tea.KeyTab}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !updated.previewFocused {
			t.Error("preview should be focused after pressing tab")
		}
	})

	t.Run("tab exits preview focus", func(t *testing.T) {
		m := newPreviewScrollTestModel()
		m.previewFocused = true

		msg := tea.KeyMsg{Type: tea.KeyTab}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewFocused {
			t.Error("preview should not be focused after pressing tab again")
		}
	})

	t.Run("esc exits preview focus", func(t *testing.T) {
		m := newPreviewScrollTestModel()
		m.previewFocused = true

		msg := tea.KeyMsg{Type: tea.KeyEscape}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewFocused {
			t.Error("preview should not be focused after pressing esc")
		}
	})
}

func TestPreviewScrollKeybindings(t *testing.T) {
	t.Run("j scrolls preview down", func(t *testing.T) {
		m := newPreviewScrollTestModel()
		m.previewFocused = true
		m.previewAutoScroll = false
		m.previewScrollOffset = 0

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewScrollOffset != 1 {
			t.Errorf("scroll offset should be 1, got %d", updated.previewScrollOffset)
		}
		if updated.previewAutoScroll {
			t.Error("autoScroll should be false after manual scroll")
		}
	})

	t.Run("k scrolls preview up", func(t *testing.T) {
		m := newPreviewScrollTestModel()
		m.previewFocused = true
		m.previewAutoScroll = false
		m.previewScrollOffset = 5

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewScrollOffset != 4 {
			t.Errorf("scroll offset should be 4, got %d", updated.previewScrollOffset)
		}
	})

	t.Run("k does not go below zero", func(t *testing.T) {
		m := newPreviewScrollTestModel()
		m.previewFocused = true
		m.previewAutoScroll = false
		m.previewScrollOffset = 0

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewScrollOffset != 0 {
			t.Errorf("scroll offset should stay 0, got %d", updated.previewScrollOffset)
		}
	})

	t.Run("PgDn scrolls by page amount", func(t *testing.T) {
		m := newPreviewScrollTestModel()
		m.previewFocused = true
		m.previewAutoScroll = false
		m.previewScrollOffset = 0

		msg := tea.KeyMsg{Type: tea.KeyPgDown}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewScrollOffset != previewPageScrollAmt {
			t.Errorf("scroll offset should be %d, got %d", previewPageScrollAmt, updated.previewScrollOffset)
		}
	})

	t.Run("PgUp scrolls back by page amount", func(t *testing.T) {
		m := newPreviewScrollTestModel()
		m.previewFocused = true
		m.previewAutoScroll = false
		m.previewScrollOffset = 20

		msg := tea.KeyMsg{Type: tea.KeyPgUp}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		expected := 20 - previewPageScrollAmt
		if updated.previewScrollOffset != expected {
			t.Errorf("scroll offset should be %d, got %d", expected, updated.previewScrollOffset)
		}
	})

	t.Run("g jumps to top", func(t *testing.T) {
		m := newPreviewScrollTestModel()
		m.previewFocused = true
		m.previewAutoScroll = false
		m.previewScrollOffset = 50

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewScrollOffset != 0 {
			t.Errorf("scroll offset should be 0, got %d", updated.previewScrollOffset)
		}
		if updated.previewAutoScroll {
			t.Error("autoScroll should be false after g")
		}
	})

	t.Run("G jumps to bottom and enables autoScroll", func(t *testing.T) {
		m := newPreviewScrollTestModel()
		m.previewFocused = true
		m.previewAutoScroll = false
		m.previewScrollOffset = 0

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !updated.previewAutoScroll {
			t.Error("autoScroll should be true after G")
		}
	})
}

func TestPreviewAutoScroll(t *testing.T) {
	t.Run("autoScroll true shows bottom of content", func(t *testing.T) {
		m := newPreviewScrollTestModel()
		m.previewAutoScroll = true
		m.previewScrollOffset = 0

		// Render preview - autoScroll should display bottom
		result := m.renderPreview(100, 20)

		// Result should contain content (not be empty)
		if result == "" {
			t.Error("preview should render content")
		}
	})

	t.Run("manual scroll disables autoScroll", func(t *testing.T) {
		m := newPreviewScrollTestModel()
		m.previewFocused = true
		m.previewAutoScroll = true

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewAutoScroll {
			t.Error("autoScroll should be false after manual scroll up")
		}
	})

	t.Run("cursor change resets autoScroll to true", func(t *testing.T) {
		m := newPreviewScrollTestModel()
		m.previewAutoScroll = false
		m.previewScrollOffset = 5
		m.previewFocused = true

		// Simulate previewDebounceMsg (cursor change)
		newModel, _ := m.Update(previewDebounceMsg{})
		updated := newModel.(Model)

		if !updated.previewAutoScroll {
			t.Error("autoScroll should be true after cursor change")
		}
		if updated.previewScrollOffset != 0 {
			t.Errorf("scroll offset should reset to 0, got %d", updated.previewScrollOffset)
		}
		if updated.previewFocused {
			t.Error("preview focus should be cleared on cursor change")
		}
	})
}

func TestPreviewScrollRender(t *testing.T) {
	t.Run("preview renders without panic when scroll offset is large", func(t *testing.T) {
		m := newPreviewScrollTestModel()
		m.previewAutoScroll = false
		m.previewScrollOffset = 999

		// Should not panic
		result := m.renderPreview(100, 20)
		_ = result
	})

	t.Run("preview renders without panic when content is empty", func(t *testing.T) {
		m := newPreviewScrollTestModel()
		m.previewContent = ""

		result := m.renderPreview(100, 20)
		if !strings.Contains(result, "No preview available") {
			t.Error("empty content should show 'No preview available'")
		}
	})
}

func TestPreviewToggleResetsScroll(t *testing.T) {
	t.Run("toggling preview off and on resets scroll state", func(t *testing.T) {
		m := newPreviewScrollTestModel()
		m.previewScrollOffset = 10
		m.previewAutoScroll = false
		m.previewFocused = false // Not focused, so 'p' goes through main handler

		// Toggle off
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewVisible {
			t.Error("preview should be hidden after pressing p")
		}
		if updated.previewFocused {
			t.Error("focus should be cleared when toggling preview off")
		}

		// Toggle back on
		newModel, _ = updated.Update(msg)
		updated = newModel.(Model)

		if updated.previewScrollOffset != 0 {
			t.Errorf("scroll offset should reset to 0, got %d", updated.previewScrollOffset)
		}
		if !updated.previewAutoScroll {
			t.Error("autoScroll should be true after re-enabling preview")
		}
	})
}

func TestPreviewFooterFocusHint(t *testing.T) {
	t.Run("footer shows Tab focus when preview visible", func(t *testing.T) {
		m := newPreviewScrollTestModel()

		result := m.renderFooter()

		if !strings.Contains(result, "Tab focus") {
			t.Error("footer should show Tab focus hint when preview is visible")
		}
	})

	t.Run("footer shows preview focus keybindings when focused", func(t *testing.T) {
		m := newPreviewScrollTestModel()
		m.previewFocused = true

		result := m.renderFooter()

		if !strings.Contains(result, "j/k scroll") {
			t.Error("footer should show scroll keybindings when preview is focused")
		}
	})
}
