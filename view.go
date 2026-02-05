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
	footerHelp  = "↑/↓ navigate  ⏎ attach  d dismiss  q quit  r refresh"
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
