package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// E2E tests for PBI-11: Session Preview Pane
// These tests verify all acceptance criteria from the PRD.

// AC1: Press `p` or `Tab` toggles preview panel visibility
func TestE2E_AC1_PreviewToggle(t *testing.T) {
	t.Run("p key toggles preview visibility", func(t *testing.T) {
		m := Model{
			width:  120,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "test-session"},
			},
			cursor:         0,
			previewVisible: false,
		}

		// Press 'p' to enable
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)
		if !updated.previewVisible {
			t.Error("AC1: 'p' should enable preview")
		}

		// Press 'p' again to disable
		newModel, _ = updated.Update(msg)
		updated = newModel.(Model)
		if updated.previewVisible {
			t.Error("AC1: 'p' should disable preview")
		}
	})

	t.Run("Tab key toggles preview visibility outside dialog", func(t *testing.T) {
		m := Model{
			width:  120,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "test-session"},
			},
			cursor:         0,
			previewVisible: false,
			dialogMode:     DialogNone,
		}

		// Press Tab to enable
		msg := tea.KeyMsg{Type: tea.KeyTab}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)
		if !updated.previewVisible {
			t.Error("AC1: Tab should enable preview when no dialog open")
		}
	})
}

// AC2: Preview shows last N lines of selected session's tmux output
func TestE2E_AC2_PreviewContent(t *testing.T) {
	t.Run("preview content displays captured output", func(t *testing.T) {
		m := Model{
			width:  120,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "test-session"},
			},
			cursor:         0,
			previewVisible: true,
			previewContent: "Line 1\nLine 2\nLine 3",
		}

		result := m.renderPreview(50, 20)
		if !strings.Contains(result, "Line 1") {
			t.Error("AC2: Preview should display captured content")
		}
	})

	t.Run("previewContentMsg updates stored content", func(t *testing.T) {
		m := Model{
			width:          120,
			height:         24,
			previewVisible: true,
		}

		msg := previewContentMsg{content: "captured output from tmux", err: nil}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewContent != "captured output from tmux" {
			t.Error("AC2: previewContentMsg should update content")
		}
	})
}

// AC3: Preview updates when cursor moves to different session
func TestE2E_AC3_CursorMoveUpdate(t *testing.T) {
	t.Run("cursor movement triggers capture when preview visible", func(t *testing.T) {
		m := Model{
			width:  120,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "session-1"},
				{TmuxSession: "session-2"},
			},
			cursor:         0,
			previewVisible: true,
		}

		// Move cursor down
		msg := tea.KeyMsg{Type: tea.KeyDown}
		newModel, cmd := m.Update(msg)
		updated := newModel.(Model)

		if updated.cursor != 1 {
			t.Error("AC3: Cursor should move to session 2")
		}
		if cmd == nil {
			t.Error("AC3: Cursor movement should trigger debounce command when preview visible")
		}
	})
}

// AC4: Preview updates periodically while visible
func TestE2E_AC4_PeriodicUpdate(t *testing.T) {
	t.Run("previewTickMsg triggers capture when visible", func(t *testing.T) {
		m := Model{
			width:  120,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "test-session"},
			},
			cursor:         0,
			previewVisible: true,
		}

		msg := previewTickMsg(time.Now())
		_, cmd := m.Update(msg)

		if cmd == nil {
			t.Error("AC4: previewTickMsg should return commands when preview visible")
		}
	})

	t.Run("previewTickMsg does NOT trigger when preview hidden", func(t *testing.T) {
		m := Model{
			width:  120,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "test-session"},
			},
			cursor:         0,
			previewVisible: false,
		}

		msg := previewTickMsg(time.Now())
		_, cmd := m.Update(msg)

		if cmd != nil {
			t.Error("AC4: previewTickMsg should NOT trigger when preview hidden")
		}
	})
}

// AC5: ANSI escape codes are handled (stripped or rendered)
func TestE2E_AC5_ANSIHandling(t *testing.T) {
	t.Run("ANSI color codes are stripped", func(t *testing.T) {
		input := "\x1b[31mred text\x1b[0m"
		result := stripANSI(input)
		if strings.Contains(result, "\x1b") {
			t.Error("AC5: ANSI escape codes should be stripped")
		}
		if result != "red text" {
			t.Errorf("AC5: Plain text should be preserved, got %q", result)
		}
	})

	t.Run("cursor movement codes are stripped", func(t *testing.T) {
		input := "\x1b[2A\x1b[3Btext"
		result := stripANSI(input)
		if strings.Contains(result, "\x1b") {
			t.Error("AC5: Cursor codes should be stripped")
		}
	})

	t.Run("OSC sequences are stripped", func(t *testing.T) {
		input := "\x1b]0;Title\x07content"
		result := stripANSI(input)
		if result != "content" {
			t.Errorf("AC5: OSC sequences should be stripped, got %q", result)
		}
	})
}

// AC6: Preview panel can be resized
func TestE2E_AC6_Resizing(t *testing.T) {
	t.Run("[ key shrinks preview", func(t *testing.T) {
		m := Model{
			width:          120,
			height:         24,
			previewVisible: true,
			previewWidth:   50,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewWidth >= 50 {
			t.Error("AC6: '[' should shrink preview width")
		}
	})

	t.Run("] key expands preview", func(t *testing.T) {
		m := Model{
			width:          120,
			height:         24,
			previewVisible: true,
			previewWidth:   50,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewWidth <= 50 {
			t.Error("AC6: ']' should expand preview width")
		}
	})

	t.Run("minimum width is enforced", func(t *testing.T) {
		m := Model{
			width:          120,
			height:         24,
			previewVisible: true,
			previewWidth:   previewMinWidth,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewWidth < previewMinWidth {
			t.Error("AC6: Minimum width should be enforced")
		}
	})

	t.Run("maximum width is enforced", func(t *testing.T) {
		maxWidth := 100 - sessionListMinWidth - 1
		m := Model{
			width:          100,
			height:         24,
			previewVisible: true,
			previewWidth:   maxWidth,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewWidth > maxWidth {
			t.Error("AC6: Maximum width should be enforced")
		}
	})
}

// AC7: Layout adapts to terminal size
func TestE2E_AC7_LayoutAdaptation(t *testing.T) {
	t.Run("preview auto-hides on narrow terminal", func(t *testing.T) {
		m := Model{
			width:              120,
			height:             24,
			previewVisible:     true,
			previewUserEnabled: true,
		}

		msg := tea.WindowSizeMsg{Width: previewMinTerminalWidth - 10, Height: 24}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewVisible {
			t.Error("AC7: Preview should auto-hide on narrow terminal")
		}
	})

	t.Run("preview restores when terminal widens", func(t *testing.T) {
		m := Model{
			width:              previewMinTerminalWidth - 10,
			height:             24,
			previewVisible:     false,
			previewUserEnabled: true,
			sessions: []SessionInfo{
				{TmuxSession: "test-session"},
			},
			cursor: 0,
		}

		msg := tea.WindowSizeMsg{Width: 120, Height: 24}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !updated.previewVisible {
			t.Error("AC7: Preview should restore when terminal widens")
		}
	})
}

// AC8: Preview is disabled when terminal too narrow
func TestE2E_AC8_NarrowTerminal(t *testing.T) {
	t.Run("preview disabled on very narrow terminal", func(t *testing.T) {
		m := Model{
			width:          120,
			height:         24,
			previewVisible: true,
		}

		msg := tea.WindowSizeMsg{Width: 40, Height: 24}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewVisible {
			t.Error("AC8: Preview should be disabled when terminal too narrow")
		}
	})

	t.Run("no crash on very small terminal", func(t *testing.T) {
		m := Model{
			width:          120,
			height:         24,
			previewVisible: true,
		}

		// This should not panic
		msg := tea.WindowSizeMsg{Width: 10, Height: 5}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		// Just verify we didn't crash and dimensions are set
		if updated.width != 10 || updated.height != 5 {
			t.Error("AC8: Dimensions should be updated even on very small terminal")
		}
	})

	t.Run("View renders without panic on narrow terminal", func(t *testing.T) {
		m := Model{
			width:          40,
			height:         10,
			previewVisible: false,
			sessions: []SessionInfo{
				{TmuxSession: "test", Status: "working", CWD: "/tmp", Timestamp: time.Now().Unix()},
			},
		}

		// Should not panic
		result := m.View()
		if result == "" {
			t.Error("AC8: View should render something even on narrow terminal")
		}
	})
}

// AC9: Layout toggle (additional feature)
func TestE2E_LayoutToggle(t *testing.T) {
	t.Run("L key toggles between side and bottom layout", func(t *testing.T) {
		m := Model{
			width:          120,
			height:         24,
			previewVisible: true,
			previewLayout:  PreviewLayoutSide,
		}

		// Press 'L' to switch to bottom
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewLayout != PreviewLayoutBottom {
			t.Error("L should switch to bottom layout")
		}

		// Press 'L' again to switch back to side
		newModel, _ = updated.Update(msg)
		updated = newModel.(Model)

		if updated.previewLayout != PreviewLayoutSide {
			t.Error("L should switch back to side layout")
		}
	})

	t.Run("L key does nothing when preview hidden", func(t *testing.T) {
		m := Model{
			width:          120,
			height:         24,
			previewVisible: false,
			previewLayout:  PreviewLayoutSide,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewLayout != PreviewLayoutSide {
			t.Error("L should not change layout when preview hidden")
		}
	})

	t.Run("resize adjusts height in bottom layout", func(t *testing.T) {
		m := Model{
			width:          120,
			height:         30,
			previewVisible: true,
			previewLayout:  PreviewLayoutBottom,
			previewHeight:  10,
		}

		// Press ']' to expand height
		expandMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}}
		newModel, _ := m.Update(expandMsg)
		updated := newModel.(Model)

		if updated.previewHeight <= 10 {
			t.Error("']' should expand preview height in bottom layout")
		}

		// Press '[' to shrink height
		shrinkMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}}
		newModel, _ = updated.Update(shrinkMsg)
		updated = newModel.(Model)

		if updated.previewHeight >= 15 {
			t.Error("'[' should shrink preview height in bottom layout")
		}
	})

	t.Run("W key toggles wrap mode", func(t *testing.T) {
		m := Model{
			width:          120,
			height:         24,
			previewVisible: true,
			previewWrap:    true,
		}

		// Press 'W' to disable wrap (enable truncate)
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'W'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewWrap {
			t.Error("W should toggle wrap off")
		}

		// Press 'W' again to enable wrap
		newModel, _ = updated.Update(msg)
		updated = newModel.(Model)

		if !updated.previewWrap {
			t.Error("W should toggle wrap back on")
		}
	})

	t.Run("W key does nothing when preview hidden", func(t *testing.T) {
		m := Model{
			width:          120,
			height:         24,
			previewVisible: false,
			previewWrap:    true,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'W'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !updated.previewWrap {
			t.Error("W should not change wrap when preview hidden")
		}
	})

	t.Run("bottom layout renders correctly", func(t *testing.T) {
		m := Model{
			width:  120,
			height: 30,
			sessions: []SessionInfo{
				{TmuxSession: "test-session", Status: "working", CWD: "/tmp"},
			},
			cursor:         0,
			previewVisible: true,
			previewLayout:  PreviewLayoutBottom,
			previewContent: "some output",
		}

		result := m.View()

		// Should contain both session and preview content
		if !strings.Contains(result, "test-session") {
			t.Error("bottom layout should show session")
		}
		if !strings.Contains(result, "some output") {
			t.Error("bottom layout should show preview content")
		}
	})
}

// Integration test: Full preview workflow
func TestE2E_FullPreviewWorkflow(t *testing.T) {
	t.Run("complete preview enable/use/resize/disable cycle", func(t *testing.T) {
		// Start with preview disabled
		m := Model{
			width:  120,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "session-1"},
				{TmuxSession: "session-2"},
			},
			cursor:         0,
			previewVisible: false,
		}

		// Step 1: Enable preview with 'p'
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
		newModel, _ := m.Update(msg)
		m = newModel.(Model)
		if !m.previewVisible {
			t.Error("Step 1: Preview should be enabled")
		}

		// Step 2: Receive preview content
		contentMsg := previewContentMsg{content: "Session 1 output", err: nil}
		newModel, _ = m.Update(contentMsg)
		m = newModel.(Model)
		if m.previewContent != "Session 1 output" {
			t.Error("Step 2: Preview content should be set")
		}

		// Step 3: Move cursor to different session
		downMsg := tea.KeyMsg{Type: tea.KeyDown}
		newModel, cmd := m.Update(downMsg)
		m = newModel.(Model)
		if m.cursor != 1 {
			t.Error("Step 3: Cursor should move to session 2")
		}
		if cmd == nil {
			t.Error("Step 3: Should trigger debounce command")
		}

		// Step 4: Resize preview (height in bottom layout, which is now the default)
		expandMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}}
		oldHeight := m.getPreviewHeight()
		newModel, _ = m.Update(expandMsg)
		m = newModel.(Model)
		if m.previewHeight <= oldHeight {
			t.Error("Step 4: Preview should be taller after expansion")
		}

		// Step 5: Disable preview with 'p'
		newModel, _ = m.Update(msg)
		m = newModel.(Model)
		if m.previewVisible {
			t.Error("Step 5: Preview should be disabled")
		}
		if m.previewContent != "" {
			t.Error("Step 5: Preview content should be cleared")
		}
	})
}
