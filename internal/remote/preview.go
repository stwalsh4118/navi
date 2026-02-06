package remote

import (
	"fmt"
	"regexp"
	"strings"
)

// ANSI escape sequence patterns for stripping terminal control codes.
// These are duplicated from internal/tui/preview.go to avoid a circular
// dependency (tui already imports remote).
var (
	ansiEscapeRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	oscSequenceRegex = regexp.MustCompile(`\x1b\][^\x07]*\x07`)
	controlCharRegex = regexp.MustCompile(`[\x00-\x08\x0b\x0c\x0e-\x1a\x1c-\x1f]`)
)

// stripANSI removes ANSI escape sequences and other control characters from input.
func stripANSI(input string) string {
	result := ansiEscapeRegex.ReplaceAllString(input, "")
	result = oscSequenceRegex.ReplaceAllString(result, "")
	result = controlCharRegex.ReplaceAllString(result, "")
	return result
}

// CapturePane captures the recent output from a remote tmux session pane via SSH.
// It executes tmux capture-pane on the remote, strips ANSI escape codes, and
// returns the cleaned output matching the local preview format.
func CapturePane(pool *SSHPool, remoteName, sessionName string, lines int) (string, error) {
	lineArg := fmt.Sprintf("-%d", lines)
	cmd := fmt.Sprintf("tmux capture-pane -t %q -p -S %s",
		sessionName, lineArg)

	output, err := pool.Execute(remoteName, cmd)
	if err != nil {
		return "", fmt.Errorf("capture-pane failed for %s/%s: %w", remoteName, sessionName, err)
	}

	cleaned := stripANSI(string(output))
	return strings.TrimRight(cleaned, "\n\t "), nil
}
