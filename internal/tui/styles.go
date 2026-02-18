package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Status icon color styles
var (
	yellowStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("226")) // waiting
	greenStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))  // done
	magentaStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("201")) // permission
	cyanStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("51"))  // working
	redStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // error
	grayStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245")) // offline
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241")) // unknown
)

// Text styles
var (
	boldStyle      = lipgloss.NewStyle().Bold(true)
	italicStyle    = lipgloss.NewStyle().Italic(true)
	highlightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99")) // focused input highlight
)

// Row styles
var (
	selectedStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Bold(true)
)

// Box styles for header/footer
var (
	boxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)
)

// Dialog styles
var (
	dialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("99")).
			Padding(1, 2)

	dialogTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("99"))

	dialogErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196"))
)

// Preview pane styles
var (
	previewBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("99")).
			Padding(0, 1)

	previewHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("99"))

	previewEmptyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Italic(true)

	previewFocusedBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("46")).
				Padding(0, 1)

	pmBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	pmFocusedBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("46")).
				Padding(0, 1)

	pmHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("46"))
)

// Search bar style
var searchBarStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("99")).
	Padding(0, 1)

// Search match styles
var (
	// searchMatchIndicatorStyle is used for the left-side gutter indicator on non-current matches.
	searchMatchIndicatorStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("226")) // Yellow indicator bar

	// searchCurrentMatchIndicatorStyle is used for the left-side gutter indicator on the current match.
	searchCurrentMatchIndicatorStyle = lipgloss.NewStyle().
						Foreground(lipgloss.Color("226")). // Bright yellow indicator bar
						Bold(true)

	searchMatchCountStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("99")).
				Bold(true)

	searchNoMatchStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true)
)

// Filter indicator style
var filterActiveStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("99")).
	Bold(true)

// Task panel styles
var (
	taskPanelBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(0, 1)

	taskPanelFocusedBoxStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("99")).
					Padding(0, 1)

	taskGroupStyle = lipgloss.NewStyle().
			Bold(true)

	taskGroupChevronStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("99"))

	taskIDStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	taskEmptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	taskErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Italic(true)
)

// TaskStatusBadge returns a styled status badge for a task status string.
func TaskStatusBadge(status string) string {
	lower := strings.ToLower(status)
	switch {
	case lower == "done" || lower == "closed" || lower == "completed":
		return greenStyle.Render(status)
	case lower == "active" || lower == "in_progress" || lower == "inprogress" || lower == "working":
		return cyanStyle.Render(status)
	case lower == "todo" || lower == "open" || lower == "proposed" || lower == "agreed":
		return yellowStyle.Render(status)
	case lower == "blocked":
		return redStyle.Render(status)
	case lower == "review" || lower == "inreview":
		return magentaStyle.Render(status)
	default:
		return dimStyle.Render(status)
	}
}

// Task view chevron constants
const (
	taskChevronExpanded  = "▾"
	taskChevronCollapsed = "▸"
)

// Status icon constants
const (
	iconWaiting    = "⏳"
	iconDone       = "✅"
	iconPermission = "❓"
	iconWorking    = "⚙️"
	iconError      = "❌"
	iconOffline    = "⏹️"
	iconIdle       = "⏸"
	iconStopped    = "⏹"
	iconUnknown    = "○"
)

// StatusIcon returns the appropriate colored icon based on session status.
func StatusIcon(status string) string {
	switch status {
	case "waiting":
		return yellowStyle.Render(iconWaiting)
	case "done":
		return greenStyle.Render(iconDone)
	case "permission":
		return magentaStyle.Render(iconPermission)
	case "working":
		return cyanStyle.Render(iconWorking)
	case "error":
		return redStyle.Render(iconError)
	case "offline":
		return grayStyle.Render(iconOffline)
	case "idle":
		return grayStyle.Render(iconIdle)
	case "stopped":
		return grayStyle.Render(iconStopped)
	default:
		return dimStyle.Render(iconUnknown)
	}
}
