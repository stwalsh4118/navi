package main

import (
	"regexp"
	"time"
)

// PreviewLayout represents the layout mode for the preview pane.
type PreviewLayout int

// Preview layout constants
const (
	PreviewLayoutSide   PreviewLayout = iota // Side panel (default) - sessions left, preview right
	PreviewLayoutBottom                      // Bottom panel - sessions top, preview bottom
	PreviewLayoutInline                      // Inline expand - preview below selected session
)

// Preview dimension constants
const (
	// previewDefaultLines is the number of lines to capture from tmux (per PRD)
	previewDefaultLines = 50

	// previewMinWidth is the minimum width for the preview pane in columns
	previewMinWidth = 30

	// previewDefaultWidthPercent is the default preview width as percentage of terminal
	previewDefaultWidthPercent = 50

	// previewMinTerminalWidth is the minimum terminal width required to show preview
	previewMinTerminalWidth = 80

	// sessionListMinWidth is the minimum width for the session list when preview is shown
	sessionListMinWidth = 30

	// previewResizeStep is the number of columns/rows to resize by per keypress
	previewResizeStep = 5

	// previewMinHeight is the minimum height for the preview pane in rows (bottom layout)
	previewMinHeight = 5

	// previewDefaultHeightPercent is the default preview height as percentage of terminal (bottom layout)
	previewDefaultHeightPercent = 40

	// sessionListMinHeight is the minimum height for the session list when preview is shown (bottom layout)
	sessionListMinHeight = 5
)

// Preview timing constants
const (
	// previewPollInterval is how often to refresh preview content when visible
	previewPollInterval = 1500 * time.Millisecond

	// previewDebounceDelay is the delay before capturing after cursor movement
	previewDebounceDelay = 100 * time.Millisecond
)

// ANSI escape sequence patterns
var (
	// ansiEscapeRegex matches standard ANSI escape sequences (colors, formatting, cursor)
	ansiEscapeRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

	// oscSequenceRegex matches OSC (Operating System Command) sequences like title setting
	oscSequenceRegex = regexp.MustCompile(`\x1b\][^\x07]*\x07`)

	// controlCharRegex matches other control characters (except newline, tab)
	controlCharRegex = regexp.MustCompile(`[\x00-\x08\x0b\x0c\x0e-\x1a\x1c-\x1f]`)
)

// stripANSI removes ANSI escape sequences and other control characters from input.
// This ensures captured tmux output displays cleanly in the preview pane.
func stripANSI(input string) string {
	// Remove standard ANSI escape sequences (colors, cursor movement, etc.)
	result := ansiEscapeRegex.ReplaceAllString(input, "")

	// Remove OSC sequences (title setting, hyperlinks, etc.)
	result = oscSequenceRegex.ReplaceAllString(result, "")

	// Remove other control characters (except newline \n and tab \t)
	result = controlCharRegex.ReplaceAllString(result, "")

	return result
}
