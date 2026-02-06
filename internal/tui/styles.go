package tui

import "github.com/charmbracelet/lipgloss"

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
)

// Status icon constants
const (
	iconWaiting    = "⏳"
	iconDone       = "✅"
	iconPermission = "❓"
	iconWorking    = "⚙️"
	iconError      = "❌"
	iconOffline    = "⏹️"
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
	default:
		return dimStyle.Render(iconUnknown)
	}
}
