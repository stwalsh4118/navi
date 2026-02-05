package main

import (
	"errors"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// Text input configuration constants
const (
	inputNameCharLimit = 50
	inputDirCharLimit  = 256
	inputWidth         = 40
)

// Input focus indices
const (
	focusName = iota
	focusDir
	focusSkipPerms
)

// Validation error messages
var (
	errEmptyName    = errors.New("session name cannot be empty")
	errInvalidChars = errors.New("session name cannot contain '.' or ':'")
	errNameExists   = errors.New("session name already exists")
	errInvalidDir   = errors.New("directory does not exist")
)

// initNameInput creates and configures a text input for session names.
func initNameInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "Session name"
	ti.CharLimit = inputNameCharLimit
	ti.Width = inputWidth
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))
	ti.TextStyle = lipgloss.NewStyle()
	ti.Focus()
	return ti
}

// initDirInput creates and configures a text input for directory paths.
func initDirInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "Working directory"
	ti.CharLimit = inputDirCharLimit
	ti.Width = inputWidth
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))
	ti.TextStyle = lipgloss.NewStyle()
	return ti
}

// validateSessionName validates a session name for tmux compatibility.
// Returns an error if the name is invalid.
func validateSessionName(name string, existingSessions []SessionInfo) error {
	// Check for empty name
	name = strings.TrimSpace(name)
	if name == "" {
		return errEmptyName
	}

	// Check for invalid characters (tmux restrictions)
	if strings.ContainsAny(name, ".:") {
		return errInvalidChars
	}

	// Check for name conflicts
	for _, s := range existingSessions {
		if s.TmuxSession == name {
			return errNameExists
		}
	}

	return nil
}

// validateDirectory checks if a directory path exists.
// Returns an error if the directory does not exist.
func validateDirectory(path string) error {
	if path == "" {
		return nil // Empty path is allowed (will use default)
	}

	// Expand home directory
	expandedPath := expandPath(path)

	info, err := os.Stat(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return errInvalidDir
		}
		return err
	}

	if !info.IsDir() {
		return errInvalidDir
	}

	return nil
}

// getDefaultSessionName generates a default session name based on timestamp.
func getDefaultSessionName() string {
	// Use a simple format: claude-<timestamp suffix>
	return "claude"
}

// getDefaultDirectory returns the current working directory or home.
func getDefaultDirectory() string {
	cwd, err := os.Getwd()
	if err != nil {
		home, _ := os.UserHomeDir()
		return home
	}
	return cwd
}
