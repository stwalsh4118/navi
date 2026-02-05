package main

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestView(t *testing.T) {
	t.Run("view with sessions", func(t *testing.T) {
		// Use wider terminal to fit all keybindings in footer
		m := Model{
			width:  120,
			height: 24,
			sessions: []SessionInfo{
				{
					TmuxSession: "test-session",
					Status:      "working",
					Message:     "Processing...",
					CWD:         "/tmp/test",
					Timestamp:   time.Now().Unix(),
				},
			},
			cursor: 0,
		}

		result := m.View()

		// Should contain header
		if !strings.Contains(result, headerTitle) {
			t.Error("view should contain header title")
		}

		// Should contain session name
		if !strings.Contains(result, "test-session") {
			t.Error("view should contain session name")
		}

		// Should contain footer help
		if !strings.Contains(result, "navigate") {
			t.Error("view should contain footer keybindings")
		}
	})

	t.Run("view with no sessions", func(t *testing.T) {
		m := Model{
			width:    80,
			height:   24,
			sessions: []SessionInfo{},
			cursor:   0,
		}

		result := m.View()

		// Should contain empty state message
		if !strings.Contains(result, "No active sessions") {
			t.Error("view should show 'No active sessions' when empty")
		}
	})

	t.Run("view with error", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			err:    errors.New("test error"),
		}

		result := m.View()

		// Should contain error message
		if !strings.Contains(result, "Error: test error") {
			t.Error("view should show error message")
		}

		// Should still have quit instruction
		if !strings.Contains(result, "Press q to quit") {
			t.Error("view should show quit instruction on error")
		}
	})

	t.Run("view with selection", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "first", Status: "working", CWD: "/tmp/1", Timestamp: time.Now().Unix()},
				{TmuxSession: "second", Status: "done", CWD: "/tmp/2", Timestamp: time.Now().Unix()},
			},
			cursor: 1, // Second session selected
		}

		result := m.View()

		// Should contain both sessions
		if !strings.Contains(result, "first") {
			t.Error("view should contain first session")
		}
		if !strings.Contains(result, "second") {
			t.Error("view should contain second session")
		}

		// Should show selection marker before second session
		// (checking that the marker exists somewhere in output)
		if !strings.Contains(result, selectedMarker) {
			t.Error("view should contain selection marker")
		}
	})
}

func TestWindowSizeMsg(t *testing.T) {
	m := Model{
		width:    80,
		height:   24,
		sessions: []SessionInfo{},
		cursor:   0,
	}

	// Simulate a window resize
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.width != 120 {
		t.Errorf("width should be 120, got %d", updated.width)
	}
	if updated.height != 40 {
		t.Errorf("height should be 40, got %d", updated.height)
	}
}

func TestCursorClamping(t *testing.T) {
	t.Run("clamp cursor when sessions shrink", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "1"},
				{TmuxSession: "2"},
				{TmuxSession: "3"},
			},
			cursor: 2, // Last session selected
		}

		// Simulate sessions list shrinking to 1 item
		msg := sessionsMsg{
			{TmuxSession: "only"},
		}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.cursor != 0 {
			t.Errorf("cursor should be clamped to 0, got %d", updated.cursor)
		}
	})

	t.Run("clamp cursor when sessions become empty", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "1"},
			},
			cursor: 0,
		}

		// Simulate sessions becoming empty
		msg := sessionsMsg{}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.cursor != 0 {
			t.Errorf("cursor should be 0 when empty, got %d", updated.cursor)
		}
	})
}

func TestKeyboardNavigation(t *testing.T) {
	t.Run("down key moves cursor down", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "1"},
				{TmuxSession: "2"},
				{TmuxSession: "3"},
			},
			cursor: 0,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.cursor != 1 {
			t.Errorf("cursor should be 1 after down, got %d", updated.cursor)
		}
	})

	t.Run("up key moves cursor up", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "1"},
				{TmuxSession: "2"},
				{TmuxSession: "3"},
			},
			cursor: 2,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.cursor != 1 {
			t.Errorf("cursor should be 1 after up, got %d", updated.cursor)
		}
	})

	t.Run("down key wraps to top", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "1"},
				{TmuxSession: "2"},
			},
			cursor: 1, // At the end
		}

		msg := tea.KeyMsg{Type: tea.KeyDown}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.cursor != 0 {
			t.Errorf("cursor should wrap to 0, got %d", updated.cursor)
		}
	})

	t.Run("up key wraps to bottom", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "1"},
				{TmuxSession: "2"},
				{TmuxSession: "3"},
			},
			cursor: 0, // At the start
		}

		msg := tea.KeyMsg{Type: tea.KeyUp}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.cursor != 2 {
			t.Errorf("cursor should wrap to 2, got %d", updated.cursor)
		}
	})

	t.Run("navigation on empty list does nothing", func(t *testing.T) {
		m := Model{
			width:    80,
			height:   24,
			sessions: []SessionInfo{},
			cursor:   0,
		}

		msg := tea.KeyMsg{Type: tea.KeyDown}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.cursor != 0 {
			t.Errorf("cursor should stay at 0, got %d", updated.cursor)
		}
	})

	t.Run("j key works like down", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "1"},
				{TmuxSession: "2"},
			},
			cursor: 0,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.cursor != 1 {
			t.Errorf("cursor should be 1 after j, got %d", updated.cursor)
		}
	})

	t.Run("k key works like up", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "1"},
				{TmuxSession: "2"},
			},
			cursor: 1,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.cursor != 0 {
			t.Errorf("cursor should be 0 after k, got %d", updated.cursor)
		}
	})
}

func TestEnterKey(t *testing.T) {
	t.Run("enter returns command when sessions exist", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "test-session"},
			},
			cursor: 0,
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		_, cmd := m.Update(msg)

		// A command should be returned (attachSession)
		if cmd == nil {
			t.Error("enter should return a command when sessions exist")
		}
	})

	t.Run("enter returns nil when no sessions", func(t *testing.T) {
		m := Model{
			width:    80,
			height:   24,
			sessions: []SessionInfo{},
			cursor:   0,
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		_, cmd := m.Update(msg)

		if cmd != nil {
			t.Error("enter should return nil when no sessions")
		}
	})

	t.Run("enter stores lastSelectedSession", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "my-session"},
			},
			cursor: 0,
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.lastSelectedSession != "my-session" {
			t.Errorf("lastSelectedSession should be 'my-session', got '%s'", updated.lastSelectedSession)
		}
	})
}

func TestAttachDoneMsg(t *testing.T) {
	t.Run("attachDoneMsg triggers poll command", func(t *testing.T) {
		m := Model{
			width:    80,
			height:   24,
			sessions: []SessionInfo{},
			cursor:   0,
		}

		msg := attachDoneMsg{}
		_, cmd := m.Update(msg)

		if cmd == nil {
			t.Error("attachDoneMsg should return a poll command")
		}
	})
}

func TestRefreshKey(t *testing.T) {
	t.Run("r returns poll command", func(t *testing.T) {
		m := Model{
			width:    80,
			height:   24,
			sessions: []SessionInfo{},
			cursor:   0,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
		_, cmd := m.Update(msg)

		if cmd == nil {
			t.Error("r should return a poll command")
		}
	})
}

func TestQuitKey(t *testing.T) {
	t.Run("q returns quit command", func(t *testing.T) {
		m := Model{
			width:    80,
			height:   24,
			sessions: []SessionInfo{},
			cursor:   0,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		_, cmd := m.Update(msg)

		// Execute the command to verify it's a quit
		if cmd == nil {
			t.Error("q should return a command")
		}
	})

	t.Run("ctrl+c returns quit command", func(t *testing.T) {
		m := Model{
			width:    80,
			height:   24,
			sessions: []SessionInfo{},
			cursor:   0,
		}

		msg := tea.KeyMsg{Type: tea.KeyCtrlC}
		_, cmd := m.Update(msg)

		if cmd == nil {
			t.Error("ctrl+c should return a command")
		}
	})
}

func TestDismissKey(t *testing.T) {
	t.Run("d returns poll command when sessions exist", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "test-session", Status: "waiting"},
			},
			cursor: 0,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
		_, cmd := m.Update(msg)

		if cmd == nil {
			t.Error("d should return a poll command when sessions exist")
		}
	})

	t.Run("d returns nil when no sessions", func(t *testing.T) {
		m := Model{
			width:    80,
			height:   24,
			sessions: []SessionInfo{},
			cursor:   0,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
		_, cmd := m.Update(msg)

		if cmd != nil {
			t.Error("d should return nil when no sessions")
		}
	})
}

func TestCursorRestoration(t *testing.T) {
	t.Run("cursor restored to last selected session", func(t *testing.T) {
		m := Model{
			width:               80,
			height:              24,
			sessions:            []SessionInfo{},
			cursor:              0,
			lastSelectedSession: "session-2",
		}

		// Simulate sessions arriving after detach
		msg := sessionsMsg{
			{TmuxSession: "session-1"},
			{TmuxSession: "session-2"},
			{TmuxSession: "session-3"},
		}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.cursor != 1 {
			t.Errorf("cursor should be restored to 1, got %d", updated.cursor)
		}
		if updated.lastSelectedSession != "" {
			t.Error("lastSelectedSession should be cleared after restoration")
		}
	})

	t.Run("cursor clamped when lastSelectedSession not found", func(t *testing.T) {
		m := Model{
			width:               80,
			height:              24,
			sessions:            []SessionInfo{},
			cursor:              5, // Out of range for new list
			lastSelectedSession: "nonexistent",
		}

		// Simulate sessions arriving
		msg := sessionsMsg{
			{TmuxSession: "session-1"},
			{TmuxSession: "session-2"},
		}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.cursor != 1 {
			t.Errorf("cursor should be clamped to 1, got %d", updated.cursor)
		}
		if updated.lastSelectedSession != "" {
			t.Error("lastSelectedSession should be cleared")
		}
	})
}

func TestPreviewToggle(t *testing.T) {
	t.Run("p key toggles preview visibility", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "test-session"},
			},
			cursor:         0,
			previewVisible: false,
		}

		// Press 'p' to enable preview
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !updated.previewVisible {
			t.Error("preview should be visible after pressing 'p'")
		}
		if !updated.previewUserEnabled {
			t.Error("previewUserEnabled should be true after enabling preview")
		}

		// Press 'p' again to disable preview
		newModel, _ = updated.Update(msg)
		updated = newModel.(Model)

		if updated.previewVisible {
			t.Error("preview should be hidden after pressing 'p' again")
		}
		if updated.previewUserEnabled {
			t.Error("previewUserEnabled should be false after disabling preview")
		}
	})

	t.Run("tab key toggles preview outside dialog", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "test-session"},
			},
			cursor:         0,
			previewVisible: false,
			dialogMode:     DialogNone,
		}

		// Press Tab to enable preview
		msg := tea.KeyMsg{Type: tea.KeyTab}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !updated.previewVisible {
			t.Error("preview should be visible after pressing Tab")
		}
	})

	t.Run("preview content cleared when hiding", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "test-session"},
			},
			cursor:         0,
			previewVisible: true,
			previewContent: "some content",
		}

		// Press 'p' to disable preview
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewContent != "" {
			t.Error("preview content should be cleared when hiding")
		}
	})

	t.Run("previewContentMsg updates content", func(t *testing.T) {
		m := Model{
			width:          80,
			height:         24,
			previewVisible: true,
		}

		msg := previewContentMsg{content: "captured output", err: nil}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewContent != "captured output" {
			t.Errorf("preview content should be 'captured output', got %q", updated.previewContent)
		}
	})
}

func TestPreviewPolling(t *testing.T) {
	t.Run("previewTickMsg triggers capture when preview visible", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "test-session"},
			},
			cursor:         0,
			previewVisible: true,
		}

		msg := previewTickMsg(time.Now())
		_, cmd := m.Update(msg)

		// Should return a batch command (capture + next tick)
		if cmd == nil {
			t.Error("previewTickMsg should return commands when preview visible")
		}
	})

	t.Run("previewTickMsg returns nil when preview hidden", func(t *testing.T) {
		m := Model{
			width:  80,
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
			t.Error("previewTickMsg should return nil when preview hidden")
		}
	})

	t.Run("previewTickMsg returns nil when no sessions", func(t *testing.T) {
		m := Model{
			width:          80,
			height:         24,
			sessions:       []SessionInfo{},
			previewVisible: true,
		}

		msg := previewTickMsg(time.Now())
		_, cmd := m.Update(msg)

		if cmd != nil {
			t.Error("previewTickMsg should return nil when no sessions")
		}
	})

	t.Run("cursor movement triggers debounced capture when preview visible", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "session-1"},
				{TmuxSession: "session-2"},
			},
			cursor:         0,
			previewVisible: true,
		}

		// Press down to move cursor
		msg := tea.KeyMsg{Type: tea.KeyDown}
		newModel, cmd := m.Update(msg)
		updated := newModel.(Model)

		if updated.cursor != 1 {
			t.Error("cursor should move down")
		}
		if cmd == nil {
			t.Error("cursor movement should trigger debounce command when preview visible")
		}
	})

	t.Run("previewDebounceMsg triggers capture when preview visible", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "test-session"},
			},
			cursor:         0,
			previewVisible: true,
		}

		msg := previewDebounceMsg{}
		_, cmd := m.Update(msg)

		if cmd == nil {
			t.Error("previewDebounceMsg should return capture command when preview visible")
		}
	})
}

func TestPreviewResize(t *testing.T) {
	t.Run("[ key shrinks preview pane", func(t *testing.T) {
		m := Model{
			width:          120,
			height:         24,
			previewVisible: true,
			previewWidth:   50,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		expectedWidth := 50 - previewResizeStep
		if updated.previewWidth != expectedWidth {
			t.Errorf("preview width should be %d after shrinking, got %d", expectedWidth, updated.previewWidth)
		}
	})

	t.Run("] key expands preview pane", func(t *testing.T) {
		m := Model{
			width:          120,
			height:         24,
			previewVisible: true,
			previewWidth:   50,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		expectedWidth := 50 + previewResizeStep
		if updated.previewWidth != expectedWidth {
			t.Errorf("preview width should be %d after expanding, got %d", expectedWidth, updated.previewWidth)
		}
	})

	t.Run("[ key respects minimum width", func(t *testing.T) {
		m := Model{
			width:          120,
			height:         24,
			previewVisible: true,
			previewWidth:   previewMinWidth, // Already at minimum
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewWidth != previewMinWidth {
			t.Errorf("preview width should stay at minimum %d, got %d", previewMinWidth, updated.previewWidth)
		}
	})

	t.Run("] key respects maximum width", func(t *testing.T) {
		m := Model{
			width:          100,
			height:         24,
			previewVisible: true,
			previewWidth:   100 - sessionListMinWidth - 1, // Max width
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		maxWidth := 100 - sessionListMinWidth - 1
		if updated.previewWidth > maxWidth {
			t.Errorf("preview width should not exceed max %d, got %d", maxWidth, updated.previewWidth)
		}
	})

	t.Run("resize keys ignored when preview hidden", func(t *testing.T) {
		m := Model{
			width:          120,
			height:         24,
			previewVisible: false,
			previewWidth:   50,
		}

		// Try to shrink
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewWidth != 50 {
			t.Error("resize keys should be ignored when preview hidden")
		}
	})
}

func TestTerminalSizeAdaptation(t *testing.T) {
	t.Run("preview auto-hides when terminal too narrow", func(t *testing.T) {
		m := Model{
			width:              120,
			height:             24,
			previewVisible:     true,
			previewUserEnabled: true,
		}

		// Resize to narrow terminal
		msg := tea.WindowSizeMsg{Width: previewMinTerminalWidth - 10, Height: 24}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewVisible {
			t.Error("preview should auto-hide when terminal too narrow")
		}
		// User preference should be preserved
		if !updated.previewUserEnabled {
			t.Error("previewUserEnabled should still be true")
		}
	})

	t.Run("preview restores when terminal widens", func(t *testing.T) {
		m := Model{
			width:              previewMinTerminalWidth - 10,
			height:             24,
			previewVisible:     false,
			previewUserEnabled: true, // User had it enabled
			sessions: []SessionInfo{
				{TmuxSession: "test-session"},
			},
			cursor: 0,
		}

		// Resize to wide enough terminal
		msg := tea.WindowSizeMsg{Width: 120, Height: 24}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !updated.previewVisible {
			t.Error("preview should restore when terminal widens and user had it enabled")
		}
	})

	t.Run("preview stays hidden if user disabled it", func(t *testing.T) {
		m := Model{
			width:              previewMinTerminalWidth - 10,
			height:             24,
			previewVisible:     false,
			previewUserEnabled: false, // User did NOT have it enabled
		}

		// Resize to wide enough terminal
		msg := tea.WindowSizeMsg{Width: 120, Height: 24}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewVisible {
			t.Error("preview should stay hidden if user didn't enable it")
		}
	})

	t.Run("handles very small terminal without panic", func(t *testing.T) {
		m := Model{
			width:              120,
			height:             24,
			previewVisible:     true,
			previewUserEnabled: true,
		}

		// Very small terminal
		msg := tea.WindowSizeMsg{Width: 20, Height: 5}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		// Should not panic, and preview should be hidden
		if updated.previewVisible {
			t.Error("preview should be hidden on very small terminal")
		}
		if updated.width != 20 || updated.height != 5 {
			t.Error("dimensions should be updated")
		}
	})
}
