package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ContentMode distinguishes plain text from diff content for rendering.
type ContentMode int

// Content mode constants
const (
	ContentModePlain ContentMode = iota // Plain text, no special coloring
	ContentModeDiff                     // Git diff with line-level coloring
)

// Content viewer layout constants
const (
	contentViewerHorizChrome   = 6 // Left border(1) + right border(1) + left padding(2) + right padding(2)
	contentViewerVertChrome    = 4 // Top border(1) + bottom border(1) + top padding(1) + bottom padding(1)
	contentViewerHeaderLines   = 2 // Title line + blank separator
	contentViewerFooterLines   = 2 // Blank separator + keybindings line
	contentViewerMinWidth      = 20
	contentViewerPageScrollAmt = 10
)

// openContentViewer initializes the content viewer state and opens the dialog overlay.
func (m *Model) openContentViewer(title, content string, mode ContentMode) {
	m.contentViewerTitle = title
	m.contentViewerMode = mode
	m.contentViewerScroll = 0
	m.contentViewerLines = strings.Split(content, "\n")
	m.contentViewerPrevDialog = DialogNone
	m.dialogMode = DialogContentViewer
	m.dialogError = ""
}

// openContentViewerFrom opens the content viewer with a return-to dialog.
// When the viewer is closed with Esc, it returns to prevDialog instead of DialogNone.
func (m *Model) openContentViewerFrom(title, content string, mode ContentMode, prevDialog DialogMode) {
	m.openContentViewer(title, content, mode)
	m.contentViewerPrevDialog = prevDialog
}

// contentViewerViewportHeight returns the number of visible content lines.
func (m Model) contentViewerViewportHeight() int {
	// Full screen overlay: total height minus vertical box chrome, header, and footer
	available := m.height - contentViewerVertChrome - contentViewerHeaderLines - contentViewerFooterLines
	if available < 1 {
		return 1
	}
	return available
}

// updateContentViewer handles keyboard input when the content viewer is active.
func (m Model) updateContentViewer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	maxScroll := m.contentViewerMaxScroll()

	switch msg.String() {
	case "esc", "q":
		prevDialog := m.contentViewerPrevDialog
		m.dialogMode = prevDialog
		m.contentViewerLines = nil
		m.contentViewerTitle = ""
		m.contentViewerScroll = 0
		m.contentViewerPrevDialog = DialogNone
		return m, nil

	case "down", "j":
		if m.contentViewerScroll < maxScroll {
			m.contentViewerScroll++
		}
		return m, nil

	case "up", "k":
		if m.contentViewerScroll > 0 {
			m.contentViewerScroll--
		}
		return m, nil

	case "pgdown", " ":
		m.contentViewerScroll += contentViewerPageScrollAmt
		if m.contentViewerScroll > maxScroll {
			m.contentViewerScroll = maxScroll
		}
		return m, nil

	case "pgup":
		m.contentViewerScroll -= contentViewerPageScrollAmt
		if m.contentViewerScroll < 0 {
			m.contentViewerScroll = 0
		}
		return m, nil

	case "g", "home":
		m.contentViewerScroll = 0
		return m, nil

	case "G", "end":
		m.contentViewerScroll = maxScroll
		return m, nil
	}

	return m, nil
}

// contentViewerMaxScroll returns the maximum scroll offset.
func (m Model) contentViewerMaxScroll() int {
	viewportHeight := m.contentViewerViewportHeight()
	maxScroll := len(m.contentViewerLines) - viewportHeight
	if maxScroll < 0 {
		return 0
	}
	return maxScroll
}

// renderContentViewer renders the content viewer as a full-screen overlay.
func (m Model) renderContentViewer() string {
	contentWidth := m.width - contentViewerHorizChrome
	if contentWidth < contentViewerMinWidth {
		contentWidth = contentViewerMinWidth
	}

	var b strings.Builder

	// Title
	b.WriteString(dialogTitleStyle.Render(m.contentViewerTitle))
	b.WriteString("\n\n")

	// Visible content lines
	viewportHeight := m.contentViewerViewportHeight()
	start := m.contentViewerScroll
	end := start + viewportHeight
	if end > len(m.contentViewerLines) {
		end = len(m.contentViewerLines)
	}

	visibleLines := m.contentViewerLines[start:end]
	for _, line := range visibleLines {
		rendered := renderContentLine(line, m.contentViewerMode, contentWidth)
		b.WriteString(rendered)
		b.WriteString("\n")
	}

	// Pad remaining lines if content is shorter than viewport
	for i := len(visibleLines); i < viewportHeight; i++ {
		b.WriteString("\n")
	}

	// Scroll indicator + keybindings
	b.WriteString("\n")
	scrollInfo := m.contentViewerScrollIndicator()
	keys := dimStyle.Render("j/k scroll  g/G top/bottom  PgUp/PgDn page  Esc close")
	b.WriteString(scrollInfo + "  " + keys)

	content := b.String()
	dialog := dialogBoxStyle.Width(m.width - 2).Render(content)
	return dialog
}

// contentViewerScrollIndicator returns a scroll position indicator string.
func (m Model) contentViewerScrollIndicator() string {
	totalLines := len(m.contentViewerLines)
	if totalLines == 0 {
		return dimStyle.Render("(empty)")
	}

	viewportHeight := m.contentViewerViewportHeight()
	if totalLines <= viewportHeight {
		return dimStyle.Render(fmt.Sprintf("%d lines", totalLines))
	}

	// Show percentage
	percentage := 0
	maxScroll := m.contentViewerMaxScroll()
	if maxScroll > 0 {
		percentage = m.contentViewerScroll * 100 / maxScroll
	}
	return dimStyle.Render(fmt.Sprintf("Line %d/%d (%d%%)", m.contentViewerScroll+1, totalLines, percentage))
}

// renderContentLine renders a single content line with optional mode-specific styling.
func renderContentLine(line string, mode ContentMode, maxWidth int) string {
	if mode == ContentModeDiff {
		return renderDiffLine(line, maxWidth)
	}
	// Plain text: truncate to width
	if lipgloss.Width(line) > maxWidth {
		return truncate(line, maxWidth)
	}
	return line
}

// renderDiffLine applies diff-specific coloring to a single line.
func renderDiffLine(line string, maxWidth int) string {
	if lipgloss.Width(line) > maxWidth {
		line = truncate(line, maxWidth)
	}

	switch {
	case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
		return dimStyle.Render(line)
	case strings.HasPrefix(line, "+"):
		return greenStyle.Render(line)
	case strings.HasPrefix(line, "-"):
		return redStyle.Render(line)
	case strings.HasPrefix(line, "@@"):
		return cyanStyle.Render(line)
	case strings.HasPrefix(line, "diff "), strings.HasPrefix(line, "index "):
		return dimStyle.Render(line)
	default:
		return line
	}
}
