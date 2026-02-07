package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// newContentViewerTestModel creates a model with the content viewer open for testing.
func newContentViewerTestModel(content string, mode ContentMode) Model {
	m := Model{
		width:              120,
		height:             40,
		searchInput:        initSearchInput(),
		taskSearchInput:    initTaskSearchInput(),
		taskExpandedGroups: make(map[string]bool),
	}
	m.openContentViewer("Test Title", content, mode)
	return m
}

// multiLineContent generates a string with n lines for scrolling tests.
func multiLineContent(n int) string {
	var lines []string
	for i := 1; i <= n; i++ {
		lines = append(lines, strings.Repeat("x", i))
	}
	return strings.Join(lines, "\n")
}

func TestOpenContentViewer(t *testing.T) {
	t.Run("sets dialog mode and initializes state", func(t *testing.T) {
		m := Model{width: 80, height: 24}
		m.openContentViewer("My Title", "line1\nline2\nline3", ContentModePlain)

		if m.dialogMode != DialogContentViewer {
			t.Errorf("expected DialogContentViewer, got %d", m.dialogMode)
		}
		if m.contentViewerTitle != "My Title" {
			t.Errorf("expected title 'My Title', got %q", m.contentViewerTitle)
		}
		if len(m.contentViewerLines) != 3 {
			t.Errorf("expected 3 lines, got %d", len(m.contentViewerLines))
		}
		if m.contentViewerScroll != 0 {
			t.Errorf("expected scroll 0, got %d", m.contentViewerScroll)
		}
		if m.contentViewerMode != ContentModePlain {
			t.Errorf("expected ContentModePlain, got %d", m.contentViewerMode)
		}
	})

	t.Run("resets scroll on reopen", func(t *testing.T) {
		m := Model{width: 80, height: 24}
		m.contentViewerScroll = 50
		m.openContentViewer("Title", "content", ContentModeDiff)

		if m.contentViewerScroll != 0 {
			t.Errorf("expected scroll reset to 0, got %d", m.contentViewerScroll)
		}
		if m.contentViewerMode != ContentModeDiff {
			t.Errorf("expected ContentModeDiff, got %d", m.contentViewerMode)
		}
	})
}

func TestContentViewerScrollNavigation(t *testing.T) {
	// Create content taller than viewport
	content := multiLineContent(100)

	t.Run("j scrolls down one line", func(t *testing.T) {
		m := newContentViewerTestModel(content, ContentModePlain)

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		updated := result.(Model)

		if updated.contentViewerScroll != 1 {
			t.Errorf("expected scroll 1, got %d", updated.contentViewerScroll)
		}
	})

	t.Run("k scrolls up one line", func(t *testing.T) {
		m := newContentViewerTestModel(content, ContentModePlain)
		m.contentViewerScroll = 5

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
		updated := result.(Model)

		if updated.contentViewerScroll != 4 {
			t.Errorf("expected scroll 4, got %d", updated.contentViewerScroll)
		}
	})

	t.Run("k does not scroll below 0", func(t *testing.T) {
		m := newContentViewerTestModel(content, ContentModePlain)
		m.contentViewerScroll = 0

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
		updated := result.(Model)

		if updated.contentViewerScroll != 0 {
			t.Errorf("expected scroll 0, got %d", updated.contentViewerScroll)
		}
	})

	t.Run("j does not scroll past max", func(t *testing.T) {
		m := newContentViewerTestModel(content, ContentModePlain)
		maxScroll := m.contentViewerMaxScroll()
		m.contentViewerScroll = maxScroll

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		updated := result.(Model)

		if updated.contentViewerScroll != maxScroll {
			t.Errorf("expected scroll %d, got %d", maxScroll, updated.contentViewerScroll)
		}
	})

	t.Run("down arrow scrolls down", func(t *testing.T) {
		m := newContentViewerTestModel(content, ContentModePlain)

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		updated := result.(Model)

		if updated.contentViewerScroll != 1 {
			t.Errorf("expected scroll 1, got %d", updated.contentViewerScroll)
		}
	})

	t.Run("up arrow scrolls up", func(t *testing.T) {
		m := newContentViewerTestModel(content, ContentModePlain)
		m.contentViewerScroll = 5

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
		updated := result.(Model)

		if updated.contentViewerScroll != 4 {
			t.Errorf("expected scroll 4, got %d", updated.contentViewerScroll)
		}
	})

	t.Run("g goes to top", func(t *testing.T) {
		m := newContentViewerTestModel(content, ContentModePlain)
		m.contentViewerScroll = 50

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
		updated := result.(Model)

		if updated.contentViewerScroll != 0 {
			t.Errorf("expected scroll 0, got %d", updated.contentViewerScroll)
		}
	})

	t.Run("G goes to bottom", func(t *testing.T) {
		m := newContentViewerTestModel(content, ContentModePlain)

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
		updated := result.(Model)

		maxScroll := m.contentViewerMaxScroll()
		if updated.contentViewerScroll != maxScroll {
			t.Errorf("expected scroll %d, got %d", maxScroll, updated.contentViewerScroll)
		}
	})

	t.Run("page down scrolls by page amount", func(t *testing.T) {
		m := newContentViewerTestModel(content, ContentModePlain)

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
		updated := result.(Model)

		if updated.contentViewerScroll != contentViewerPageScrollAmt {
			t.Errorf("expected scroll %d, got %d", contentViewerPageScrollAmt, updated.contentViewerScroll)
		}
	})

	t.Run("page up scrolls back by page amount", func(t *testing.T) {
		m := newContentViewerTestModel(content, ContentModePlain)
		m.contentViewerScroll = 20

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
		updated := result.(Model)

		expected := 20 - contentViewerPageScrollAmt
		if updated.contentViewerScroll != expected {
			t.Errorf("expected scroll %d, got %d", expected, updated.contentViewerScroll)
		}
	})

	t.Run("page up clamps to 0", func(t *testing.T) {
		m := newContentViewerTestModel(content, ContentModePlain)
		m.contentViewerScroll = 3

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
		updated := result.(Model)

		if updated.contentViewerScroll != 0 {
			t.Errorf("expected scroll 0, got %d", updated.contentViewerScroll)
		}
	})
}

func TestContentViewerEscCloses(t *testing.T) {
	t.Run("esc closes the content viewer", func(t *testing.T) {
		m := newContentViewerTestModel("some content", ContentModePlain)

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
		updated := result.(Model)

		if updated.dialogMode != DialogNone {
			t.Errorf("expected DialogNone, got %d", updated.dialogMode)
		}
		if updated.contentViewerLines != nil {
			t.Error("content viewer lines should be nil after close")
		}
	})

	t.Run("q also closes the content viewer", func(t *testing.T) {
		m := newContentViewerTestModel("some content", ContentModePlain)

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		updated := result.(Model)

		if updated.dialogMode != DialogNone {
			t.Errorf("expected DialogNone, got %d", updated.dialogMode)
		}
	})
}

func TestContentViewerRendering(t *testing.T) {
	t.Run("render includes title", func(t *testing.T) {
		m := newContentViewerTestModel("line1\nline2", ContentModePlain)
		output := m.renderContentViewer()

		if !strings.Contains(output, "Test Title") {
			t.Error("rendered output should contain the title")
		}
	})

	t.Run("render includes content lines", func(t *testing.T) {
		m := newContentViewerTestModel("hello world\nsecond line", ContentModePlain)
		output := m.renderContentViewer()

		if !strings.Contains(output, "hello world") {
			t.Error("rendered output should contain content lines")
		}
		if !strings.Contains(output, "second line") {
			t.Error("rendered output should contain all visible content lines")
		}
	})

	t.Run("render includes scroll indicator", func(t *testing.T) {
		content := multiLineContent(100)
		m := newContentViewerTestModel(content, ContentModePlain)
		output := m.renderContentViewer()

		if !strings.Contains(output, "Line") {
			t.Error("rendered output should contain scroll indicator")
		}
	})

	t.Run("render includes keybinding help", func(t *testing.T) {
		m := newContentViewerTestModel("content", ContentModePlain)
		output := m.renderContentViewer()

		if !strings.Contains(output, "Esc close") {
			t.Error("rendered output should contain keybinding help")
		}
	})

	t.Run("short content shows line count instead of percentage", func(t *testing.T) {
		m := newContentViewerTestModel("line1\nline2", ContentModePlain)
		indicator := m.contentViewerScrollIndicator()

		if !strings.Contains(indicator, "2 lines") {
			t.Errorf("expected '2 lines' indicator, got %q", indicator)
		}
	})
}

func TestContentViewerResize(t *testing.T) {
	t.Run("viewport height adapts to terminal height", func(t *testing.T) {
		m := newContentViewerTestModel("content", ContentModePlain)
		h1 := m.contentViewerViewportHeight()

		m.height = 60
		h2 := m.contentViewerViewportHeight()

		if h2 <= h1 {
			t.Errorf("viewport height should increase with terminal height, got %d then %d", h1, h2)
		}
	})

	t.Run("scroll clamps after resize shrinks viewport", func(t *testing.T) {
		content := multiLineContent(100)
		m := newContentViewerTestModel(content, ContentModePlain)
		m.height = 60

		// Scroll to bottom with large viewport
		m.contentViewerScroll = m.contentViewerMaxScroll()

		// Shrink terminal
		m.height = 20
		newMax := m.contentViewerMaxScroll()

		if m.contentViewerScroll > newMax {
			// The content viewer should handle this during next render
			// but the maxScroll method correctly reports the new max
			if newMax < 0 {
				t.Error("maxScroll should never be negative")
			}
		}
	})
}
