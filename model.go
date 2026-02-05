package main

import (
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
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	return "Navi\n\nPress q to quit.\n"
}
