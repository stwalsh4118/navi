package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Selection marker constants
const (
	selectedMarker   = "▸ "
	unselectedMarker = "  "
	rowIndent        = "      " // 6 spaces for indented lines
)

// renderSession renders a single session row with icon, name, age, cwd, and message.
func (m Model) renderSession(s SessionInfo, selected bool, width int) string {
	var b strings.Builder

	// Selection marker
	marker := unselectedMarker
	if selected {
		marker = selectedMarker
	}

	// First line: marker + icon + name + age
	icon := statusIcon(s.Status)
	name := boldStyle.Render(s.TmuxSession)
	age := formatAge(s.Timestamp)

	// Calculate padding for right-aligned age
	firstLine := fmt.Sprintf("%s%s  %s", marker, icon, name)
	padding := width - lipgloss.Width(firstLine) - len(age) - 2
	if padding < 1 {
		padding = 1
	}

	b.WriteString(firstLine)
	b.WriteString(strings.Repeat(" ", padding))
	b.WriteString(age)
	b.WriteString("\n")

	// Second line: working directory (indented, dimmed)
	cwd := shortenPath(s.CWD)
	b.WriteString(rowIndent)
	b.WriteString(dimStyle.Render(cwd))

	// Third line: message if present (indented, dimmed/italic)
	if s.Message != "" {
		b.WriteString("\n")
		msg := truncate(s.Message, width-len(rowIndent))
		b.WriteString(rowIndent)
		b.WriteString(dimStyle.Render(italicStyle.Render(fmt.Sprintf("\"%s\"", msg))))
	}

	return b.String()
}

// formatAge formats a Unix timestamp as a human-readable age string.
func formatAge(timestamp int64) string {
	elapsed := time.Since(time.Unix(timestamp, 0))
	if elapsed < time.Minute {
		return fmt.Sprintf("%ds ago", int(elapsed.Seconds()))
	}
	if elapsed < time.Hour {
		return fmt.Sprintf("%dm ago", int(elapsed.Minutes()))
	}
	return fmt.Sprintf("%dh ago", int(elapsed.Hours()))
}

// shortenPath replaces the home directory prefix with ~.
func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// truncate truncates a string to maxLen characters, adding ellipsis if needed.
func truncate(s string, maxLen int) string {
	if maxLen < 4 {
		maxLen = 4
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Header and footer constants
const (
	headerTitle = "Claude Sessions"
	footerHelp  = "↑/↓ navigate  ⏎ attach  d dismiss  n new  x kill  R rename  r refresh  q quit"
)

// renderHeader renders the header box with title and session count.
func (m Model) renderHeader() string {
	count := fmt.Sprintf("%d active", len(m.sessions))

	// Calculate padding for count on right
	// Account for box border padding (1 on each side)
	contentWidth := m.width - 4
	padding := contentWidth - lipgloss.Width(headerTitle) - lipgloss.Width(count)
	if padding < 1 {
		padding = 1
	}

	content := headerTitle + strings.Repeat(" ", padding) + count
	return boxStyle.Width(m.width - 2).Render(content)
}

// renderFooter renders the footer box with keybinding help.
func (m Model) renderFooter() string {
	return boxStyle.Width(m.width - 2).Render(footerHelp)
}

// Dialog width constant
const dialogWidth = 56

// renderDialog renders the dialog overlay when dialogMode is set.
// Returns empty string if no dialog is open.
func (m Model) renderDialog() string {
	if m.dialogMode == DialogNone {
		return ""
	}

	var b strings.Builder

	// Title
	title := dialogTitle(m.dialogMode)
	b.WriteString(dialogTitleStyle.Render(title))
	b.WriteString("\n\n")

	// Dialog-specific content
	switch m.dialogMode {
	case DialogNewSession:
		b.WriteString("Name:      ")
		b.WriteString(m.nameInput.View())
		b.WriteString("\n")
		b.WriteString("Directory: ")
		b.WriteString(m.dirInput.View())
		b.WriteString("\n")
		// Checkbox for skip permissions
		checkbox := "[ ]"
		if m.skipPermissions {
			checkbox = "[x]"
		}
		checkboxLine := checkbox + " Skip permissions"
		if m.focusedInput == focusSkipPerms {
			checkboxLine = highlightStyle.Render(checkboxLine)
		}
		b.WriteString(checkboxLine)
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("Tab: switch  Space: toggle  Enter: create  Esc: cancel"))
	case DialogKillConfirm:
		if m.sessionToModify != nil {
			b.WriteString(fmt.Sprintf("Kill session '%s'?\n\n", m.sessionToModify.TmuxSession))
		}
		b.WriteString(dimStyle.Render("y: yes  n: no  Esc: cancel"))
	case DialogRename:
		if m.sessionToModify != nil {
			b.WriteString(fmt.Sprintf("Current: %s\n\n", dimStyle.Render(m.sessionToModify.TmuxSession)))
		}
		b.WriteString("New name: ")
		b.WriteString(m.nameInput.View())
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("Enter: rename  Esc: cancel"))
	}

	// Error message if present
	if m.dialogError != "" {
		b.WriteString("\n\n")
		b.WriteString(dialogErrorStyle.Render(m.dialogError))
	}

	// Render the dialog box
	content := b.String()
	dialog := dialogBoxStyle.Width(dialogWidth).Render(content)

	// Center the dialog horizontally using lipgloss.Place for proper multi-line handling
	return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, dialog)
}
