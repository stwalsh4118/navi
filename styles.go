package main

import "github.com/charmbracelet/lipgloss"

// Status icon color styles
var (
	yellowStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))  // waiting
	greenStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))   // done
	magentaStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("201"))  // permission
	cyanStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("51"))   // working
	redStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))  // error
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))  // unknown
)

// Text styles
var (
	boldStyle   = lipgloss.NewStyle().Bold(true)
	italicStyle = lipgloss.NewStyle().Italic(true)
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

// Status icon constants
const (
	iconWaiting    = "⏳"
	iconDone       = "✅"
	iconPermission = "❓"
	iconWorking    = "⚙️"
	iconError      = "❌"
	iconUnknown    = "○"
)

// statusIcon returns the appropriate colored icon based on session status.
func statusIcon(status string) string {
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
	default:
		return dimStyle.Render(iconUnknown)
	}
}
