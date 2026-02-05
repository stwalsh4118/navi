package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// SessionInfo represents the status data for a single Claude Code session.
// This struct matches the JSON format written by the navi hook scripts.
type SessionInfo struct {
	TmuxSession string `json:"tmux_session"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	CWD         string `json:"cwd"`
	Timestamp   int64  `json:"timestamp"`
}

// Model is the Bubble Tea application state for navi.
type Model struct {
	sessions []SessionInfo
	cursor   int
	width    int
	height   int
	err      error
}

// Message types for Bubble Tea communication.
type tickMsg time.Time
type sessionsMsg []SessionInfo
type attachDoneMsg struct{}

// Init implements tea.Model.
// Starts the tick command and performs initial poll.
func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), pollSessions)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case tickMsg:
		// On tick, poll sessions and schedule next tick
		return m, tea.Batch(pollSessions, tickCmd())

	case sessionsMsg:
		// Update sessions list
		m.sessions = msg
		// Clamp cursor if sessions list shrunk
		if m.cursor >= len(m.sessions) && len(m.sessions) > 0 {
			m.cursor = len(m.sessions) - 1
		} else if len(m.sessions) == 0 {
			m.cursor = 0
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

// Empty state message constant
const noSessionsMessage = "  No active sessions"

// View implements tea.Model.
func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
	}

	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")

	// Session list
	if len(m.sessions) == 0 {
		b.WriteString(dimStyle.Render(noSessionsMessage))
		b.WriteString("\n")
	} else {
		for i, session := range m.sessions {
			selected := i == m.cursor
			b.WriteString(m.renderSession(session, selected, m.width))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	// Footer
	b.WriteString(m.renderFooter())

	return b.String()
}
