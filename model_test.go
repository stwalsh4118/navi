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
		m := Model{
			width:  80,
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
