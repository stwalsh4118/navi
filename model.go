package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	sessions            []SessionInfo
	cursor              int
	width               int
	height              int
	err                 error
	lastSelectedSession string // Used to preserve cursor position after attach/detach

	// Dialog state
	dialogMode  DialogMode // Which dialog is currently open (DialogNone if none)
	dialogError string     // Error message to display in dialog

	// Text inputs for dialogs
	nameInput       textinput.Model // Session name input
	dirInput        textinput.Model // Working directory input
	focusedInput    int             // Which input is focused (0 = name, 1 = dir, 2 = skipPerms)
	skipPermissions bool            // Whether to start claude with --dangerously-skip-permissions
	sessionToModify *SessionInfo    // Session being killed or renamed

	// Preview pane state
	previewVisible     bool          // Whether preview pane is shown
	previewUserEnabled bool          // User's intended state (for restore after terminal resize)
	previewContent     string        // Cached captured output
	previewLayout      PreviewLayout // Current layout mode (default: PreviewLayoutSide)
	previewWidth       int           // Width of preview pane in columns (side layout)
	previewHeight      int           // Height of preview pane in rows (bottom layout)
	previewLastCapture time.Time     // Last capture timestamp for debouncing
	previewLastCursor  int           // Last cursor position for detecting cursor changes
}

// Message types for Bubble Tea communication.
type tickMsg time.Time
type sessionsMsg []SessionInfo
type attachDoneMsg struct{}

// createSessionResultMsg is returned after attempting to create a new session.
type createSessionResultMsg struct {
	err error
}

// killSessionResultMsg is returned after attempting to kill a session.
type killSessionResultMsg struct {
	err error
}

// renameSessionResultMsg is returned after attempting to rename a session.
type renameSessionResultMsg struct {
	err     error
	newName string
}

// previewContentMsg is returned after capturing preview content.
type previewContentMsg struct {
	content string
	err     error
}

// previewTickMsg is sent to trigger periodic preview refresh.
type previewTickMsg time.Time

// previewDebounceMsg is sent after cursor movement debounce delay.
type previewDebounceMsg struct{}

// Init implements tea.Model.
// Starts the tick command and performs initial poll.
func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), pollSessions)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle dialog mode first - block main keybindings when dialog is open
		if m.dialogMode != DialogNone {
			return m.updateDialog(msg)
		}

		// Main keybindings (only when no dialog is open)
		switch msg.String() {
		case "up", "k":
			if len(m.sessions) > 0 {
				oldCursor := m.cursor
				if m.cursor > 0 {
					m.cursor--
				} else {
					m.cursor = len(m.sessions) - 1 // wrap to bottom
				}
				// Trigger debounced preview capture if cursor changed and preview visible
				if m.previewVisible && m.cursor != oldCursor {
					m.previewLastCursor = m.cursor
					return m, previewDebounceCmd()
				}
			}
			return m, nil

		case "down", "j":
			if len(m.sessions) > 0 {
				oldCursor := m.cursor
				if m.cursor < len(m.sessions)-1 {
					m.cursor++
				} else {
					m.cursor = 0 // wrap to top
				}
				// Trigger debounced preview capture if cursor changed and preview visible
				if m.previewVisible && m.cursor != oldCursor {
					m.previewLastCursor = m.cursor
					return m, previewDebounceCmd()
				}
			}
			return m, nil

		case "enter":
			if len(m.sessions) > 0 && m.cursor < len(m.sessions) {
				session := m.sessions[m.cursor]
				m.lastSelectedSession = session.TmuxSession
				return m, attachSession(session.TmuxSession)
			}
			return m, nil

		case "d":
			if len(m.sessions) > 0 && m.cursor < len(m.sessions) {
				session := m.sessions[m.cursor]
				_ = dismissSession(session) // Ignore error, poll will update view
				return m, pollSessions
			}
			return m, nil

		case "r":
			return m, pollSessions

		case "n":
			// Open new session dialog
			m.dialogMode = DialogNewSession
			m.dialogError = ""
			m.nameInput = initNameInput()
			m.dirInput = initDirInput()
			m.dirInput.SetValue(getDefaultDirectory())
			m.focusedInput = focusName
			m.skipPermissions = false
			return m, nil

		case "x":
			// Open kill confirmation dialog
			if len(m.sessions) > 0 && m.cursor < len(m.sessions) {
				session := m.sessions[m.cursor]
				m.sessionToModify = &session
				m.dialogMode = DialogKillConfirm
				m.dialogError = ""
				return m, nil
			}
			return m, nil

		case "R":
			// Open rename dialog
			if len(m.sessions) > 0 && m.cursor < len(m.sessions) {
				session := m.sessions[m.cursor]
				m.sessionToModify = &session
				m.dialogMode = DialogRename
				m.dialogError = ""
				m.nameInput = initNameInput()
				m.nameInput.SetValue(session.TmuxSession)
				m.nameInput.CursorEnd() // Put cursor at end
				return m, nil
			}
			return m, nil

		case "p", "tab":
			// Toggle preview pane visibility
			m.previewVisible = !m.previewVisible
			m.previewUserEnabled = m.previewVisible
			if m.previewVisible && len(m.sessions) > 0 && m.cursor < len(m.sessions) {
				// Trigger immediate capture and start polling when showing
				m.previewLastCursor = m.cursor
				return m, tea.Batch(
					capturePreviewCmd(m.sessions[m.cursor].TmuxSession),
					previewTickCmd(),
				)
			}
			// Clear content when hiding
			m.previewContent = ""
			return m, nil

		case "[":
			// Shrink preview pane
			if m.previewVisible {
				if m.previewLayout == PreviewLayoutBottom {
					// Shrink height in bottom layout
					currentHeight := m.getPreviewHeight()
					newHeight := currentHeight - previewResizeStep
					if newHeight < previewMinHeight {
						newHeight = previewMinHeight
					}
					m.previewHeight = newHeight
				} else {
					// Shrink width in side layout
					currentWidth := m.getPreviewWidth()
					newWidth := currentWidth - previewResizeStep
					if newWidth < previewMinWidth {
						newWidth = previewMinWidth
					}
					m.previewWidth = newWidth
				}
			}
			return m, nil

		case "]":
			// Expand preview pane
			if m.previewVisible {
				if m.previewLayout == PreviewLayoutBottom {
					// Expand height in bottom layout
					contentHeight := m.height - 8 // Same calculation as View()
					maxHeight := contentHeight - sessionListMinHeight
					currentHeight := m.getPreviewHeight()
					newHeight := currentHeight + previewResizeStep
					if newHeight > maxHeight {
						newHeight = maxHeight
					}
					m.previewHeight = newHeight
				} else {
					// Expand width in side layout
					currentWidth := m.getPreviewWidth()
					maxWidth := m.width - sessionListMinWidth - 1 // -1 for gap
					newWidth := currentWidth + previewResizeStep
					if newWidth > maxWidth {
						newWidth = maxWidth
					}
					m.previewWidth = newWidth
				}
			}
			return m, nil

		case "L":
			// Toggle preview layout between side and bottom
			if m.previewVisible {
				if m.previewLayout == PreviewLayoutSide {
					m.previewLayout = PreviewLayoutBottom
				} else {
					m.previewLayout = PreviewLayoutSide
				}
			}
			return m, nil

		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case tickMsg:
		// On tick, poll sessions and schedule next tick
		return m, tea.Batch(pollSessions, tickCmd())

	case sessionsMsg:
		// Update sessions list
		m.sessions = msg

		// Try to restore cursor to last selected session if set
		if m.lastSelectedSession != "" {
			for i, s := range m.sessions {
				if s.TmuxSession == m.lastSelectedSession {
					m.cursor = i
					m.lastSelectedSession = "" // Clear after restoring
					return m, nil
				}
			}
			m.lastSelectedSession = "" // Session no longer exists, clear it
		}

		// Clamp cursor if sessions list shrunk
		if m.cursor >= len(m.sessions) && len(m.sessions) > 0 {
			m.cursor = len(m.sessions) - 1
		} else if len(m.sessions) == 0 {
			m.cursor = 0
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Handle preview visibility based on terminal width
		if m.width < previewMinTerminalWidth {
			// Auto-hide preview when terminal too narrow
			m.previewVisible = false
		} else if m.previewUserEnabled && !m.previewVisible {
			// Restore preview if user had it enabled and space now available
			m.previewVisible = true
			// Trigger capture if we have sessions
			if len(m.sessions) > 0 && m.cursor < len(m.sessions) {
				return m, tea.Batch(
					capturePreviewCmd(m.sessions[m.cursor].TmuxSession),
					previewTickCmd(),
				)
			}
		}

	case attachDoneMsg:
		// After returning from tmux, trigger immediate refresh
		return m, pollSessions

	case createSessionResultMsg:
		if msg.err != nil {
			// Show error in dialog
			m.dialogError = "Failed to create session: " + msg.err.Error()
			return m, nil
		}
		// Success - close dialog and refresh
		m.dialogMode = DialogNone
		m.dialogError = ""
		return m, pollSessions

	case killSessionResultMsg:
		if msg.err != nil {
			// Show error in dialog
			m.dialogError = "Failed to kill session: " + msg.err.Error()
			return m, nil
		}
		// Success - close dialog and refresh
		m.dialogMode = DialogNone
		m.dialogError = ""
		m.sessionToModify = nil
		return m, pollSessions

	case renameSessionResultMsg:
		if msg.err != nil {
			// Show error in dialog
			m.dialogError = "Failed to rename session: " + msg.err.Error()
			return m, nil
		}
		// Success - close dialog and set lastSelectedSession for cursor preservation
		m.dialogMode = DialogNone
		m.dialogError = ""
		m.sessionToModify = nil
		m.lastSelectedSession = msg.newName // Preserve cursor position on renamed session
		return m, pollSessions

	case previewContentMsg:
		if msg.err == nil {
			m.previewContent = msg.content
			m.previewLastCapture = time.Now()
		}
		// Silently ignore errors - preview just won't update
		return m, nil

	case previewTickMsg:
		// Periodic preview refresh
		if !m.previewVisible || len(m.sessions) == 0 {
			// Don't continue polling if preview hidden or no sessions
			return m, nil
		}
		// Capture current session and schedule next tick
		if m.cursor < len(m.sessions) {
			return m, tea.Batch(
				capturePreviewCmd(m.sessions[m.cursor].TmuxSession),
				previewTickCmd(),
			)
		}
		return m, previewTickCmd()

	case previewDebounceMsg:
		// Debounced capture after cursor movement
		if !m.previewVisible || len(m.sessions) == 0 {
			return m, nil
		}
		if m.cursor < len(m.sessions) {
			return m, capturePreviewCmd(m.sessions[m.cursor].TmuxSession)
		}
		return m, nil
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

	// Calculate available height for content area
	// Header (3 lines) + Footer (3 lines) + spacing
	contentHeight := m.height - 8
	if contentHeight < 5 {
		contentHeight = 5
	}

	if m.previewVisible && m.width >= previewMinTerminalWidth {
		if m.previewLayout == PreviewLayoutBottom {
			// Bottom layout: sessions on top, preview on bottom
			previewHeight := m.getPreviewHeight()
			sessionListHeight := contentHeight - previewHeight - 1 // -1 for gap

			sessionList := m.renderSessionList(m.width)
			preview := m.renderPreview(m.width, previewHeight)

			// Limit session list height
			sessionLines := strings.Split(sessionList, "\n")
			if len(sessionLines) > sessionListHeight {
				sessionLines = sessionLines[:sessionListHeight]
			}
			sessionList = strings.Join(sessionLines, "\n")

			// Join vertically
			b.WriteString(sessionList)
			b.WriteString("\n")
			b.WriteString(preview)
		} else {
			// Side layout: sessions on left, preview on right
			previewWidth := m.getPreviewWidth()
			sessionListWidth := m.width - previewWidth - 1 // -1 for gap

			sessionList := m.renderSessionList(sessionListWidth)
			preview := m.renderPreview(previewWidth, contentHeight)

			// Join horizontally with a gap
			combined := lipgloss.JoinHorizontal(lipgloss.Top, sessionList, " ", preview)
			b.WriteString(combined)
		}
	} else {
		// Standard layout: just session list
		b.WriteString(m.renderSessionList(m.width))
	}

	b.WriteString("\n")

	// Dialog overlay (if open)
	if m.dialogMode != DialogNone {
		b.WriteString(m.renderDialog())
		b.WriteString("\n\n")
	}

	// Footer
	b.WriteString(m.renderFooter())

	return b.String()
}

// getPreviewWidth returns the width to use for the preview pane (side layout).
func (m Model) getPreviewWidth() int {
	if m.previewWidth > 0 {
		return m.previewWidth
	}
	// Default to percentage of terminal width
	return m.width * previewDefaultWidthPercent / 100
}

// getPreviewHeight returns the height to use for the preview pane (bottom layout).
func (m Model) getPreviewHeight() int {
	if m.previewHeight > 0 {
		return m.previewHeight
	}
	// Default to percentage of available content height
	contentHeight := m.height - 8
	return contentHeight * previewDefaultHeightPercent / 100
}

// attachSession returns a command that attaches to a tmux session.
// Uses tea.ExecProcess to hand off terminal control to tmux.
func attachSession(name string) tea.Cmd {
	c := exec.Command("tmux", "attach-session", "-t", name)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return attachDoneMsg{}
	})
}

// capturePreviewCmd returns a command that captures preview content from a tmux session.
func capturePreviewCmd(sessionName string) tea.Cmd {
	return func() tea.Msg {
		content, err := capturePane(sessionName, previewDefaultLines)
		return previewContentMsg{content: content, err: err}
	}
}

// updateDialog handles key messages when a dialog is open.
func (m Model) updateDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Close any dialog and reset state
		m.dialogMode = DialogNone
		m.dialogError = ""
		m.sessionToModify = nil
		return m, nil

	case "tab":
		// Switch focus between inputs in new session dialog
		if m.dialogMode == DialogNewSession {
			m.focusedInput = (m.focusedInput + 1) % 3
			m.nameInput.Blur()
			m.dirInput.Blur()
			switch m.focusedInput {
			case focusName:
				m.nameInput.Focus()
			case focusDir:
				m.dirInput.Focus()
			// focusSkipPerms - no text input to focus
			}
			return m, nil
		}

	case " ":
		// Toggle skip permissions checkbox
		if m.dialogMode == DialogNewSession && m.focusedInput == focusSkipPerms {
			m.skipPermissions = !m.skipPermissions
			return m, nil
		}

	case "enter":
		// Handle submission based on dialog type
		switch m.dialogMode {
		case DialogNewSession:
			return m.submitNewSession()
		case DialogRename:
			return m.submitRename()
		}

	case "y":
		// Confirm kill
		if m.dialogMode == DialogKillConfirm && m.sessionToModify != nil {
			return m, killSessionCmd(m.sessionToModify.TmuxSession)
		}

	case "n":
		// Cancel kill (same as escape)
		if m.dialogMode == DialogKillConfirm {
			m.dialogMode = DialogNone
			m.dialogError = ""
			m.sessionToModify = nil
			return m, nil
		}
	}

	// Update the focused text input for dialogs that use text input
	var cmd tea.Cmd
	switch m.dialogMode {
	case DialogNewSession:
		if m.focusedInput == focusName {
			m.nameInput, cmd = m.nameInput.Update(msg)
		} else {
			m.dirInput, cmd = m.dirInput.Update(msg)
		}
	case DialogRename:
		m.nameInput, cmd = m.nameInput.Update(msg)
	}

	return m, cmd
}

// submitNewSession validates and creates a new tmux session.
func (m Model) submitNewSession() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.nameInput.Value())
	dir := strings.TrimSpace(m.dirInput.Value())

	// Use default name if empty
	if name == "" {
		name = getDefaultSessionName()
	}

	// Validate session name
	if err := validateSessionName(name, m.sessions); err != nil {
		m.dialogError = err.Error()
		return m, nil
	}

	// Validate directory
	if err := validateDirectory(dir); err != nil {
		m.dialogError = err.Error()
		return m, nil
	}

	// Use default directory if empty
	if dir == "" {
		dir = getDefaultDirectory()
	}

	// Expand home directory
	dir = expandPath(dir)

	// Create the session
	return m, createSessionCmd(name, dir, m.skipPermissions)
}

// submitRename validates and renames a tmux session.
func (m Model) submitRename() (tea.Model, tea.Cmd) {
	if m.sessionToModify == nil {
		m.dialogError = "No session selected"
		return m, nil
	}

	oldName := m.sessionToModify.TmuxSession
	newName := strings.TrimSpace(m.nameInput.Value())

	// If same name, just close dialog
	if newName == oldName {
		m.dialogMode = DialogNone
		m.dialogError = ""
		m.sessionToModify = nil
		return m, nil
	}

	// Validate session name (exclude current session from duplicate check)
	sessionsWithoutCurrent := make([]SessionInfo, 0, len(m.sessions)-1)
	for _, s := range m.sessions {
		if s.TmuxSession != oldName {
			sessionsWithoutCurrent = append(sessionsWithoutCurrent, s)
		}
	}

	if err := validateSessionName(newName, sessionsWithoutCurrent); err != nil {
		m.dialogError = err.Error()
		return m, nil
	}

	// Rename the session
	return m, renameSessionCmd(oldName, newName)
}

// closeDialog resets dialog state.
func (m *Model) closeDialog() {
	m.dialogMode = DialogNone
	m.dialogError = ""
}
