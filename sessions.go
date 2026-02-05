package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// readSessions reads all JSON status files from the specified directory
// and parses them into SessionInfo structs.
// Returns an empty slice if directory doesn't exist or on errors.
// Malformed JSON files are skipped silently.
func readSessions(dir string) ([]SessionInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []SessionInfo
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue // skip unreadable files
		}

		var session SessionInfo
		if err := json.Unmarshal(data, &session); err != nil {
			continue // skip malformed JSON
		}

		sessions = append(sessions, session)
	}
	return sessions, nil
}

// listTmuxSessions queries tmux for all active session names.
// Returns an empty slice if tmux is not running or has no sessions.
func listTmuxSessions() ([]string, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		// tmux returns error if no server running
		return nil, nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var sessions []string
	for _, line := range lines {
		if line != "" {
			sessions = append(sessions, line)
		}
	}
	return sessions, nil
}

// cleanStaleSessions removes status files for tmux sessions that no longer exist.
// liveSessions is the list of currently active tmux session names.
func cleanStaleSessions(dir string, liveSessions []string) {
	liveSet := make(map[string]bool)
	for _, name := range liveSessions {
		liveSet[name] = true
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return // directory doesn't exist, nothing to clean
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		// Extract session name from filename (remove .json)
		sessionName := strings.TrimSuffix(entry.Name(), ".json")

		if !liveSet[sessionName] {
			path := filepath.Join(dir, entry.Name())
			os.Remove(path) // ignore errors
		}
	}
}

// Polling and status constants
const (
	pollInterval     = 500 * time.Millisecond
	statusDir        = "~/.claude-sessions"
	statusWaiting    = "waiting"
	statusPermission = "permission"
)

// sortSessions sorts sessions with priority statuses (waiting, permission) first,
// then by timestamp descending (most recent first).
func sortSessions(sessions []SessionInfo) {
	sort.Slice(sessions, func(i, j int) bool {
		// Priority statuses come first
		iPriority := sessions[i].Status == statusWaiting || sessions[i].Status == statusPermission
		jPriority := sessions[j].Status == statusWaiting || sessions[j].Status == statusPermission

		if iPriority != jPriority {
			return iPriority // priority sessions first
		}

		// Within same priority, sort by timestamp descending
		return sessions[i].Timestamp > sessions[j].Timestamp
	})
}

// tickCmd returns a command that fires after pollInterval.
func tickCmd() tea.Cmd {
	return tea.Tick(pollInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// pollSessions orchestrates reading, cleaning stale, and sorting sessions.
// Returns a sessionsMsg with the current session list.
func pollSessions() tea.Msg {
	dir := expandPath(statusDir)

	// Get live tmux sessions
	liveSessions, _ := listTmuxSessions()

	// Clean stale sessions
	cleanStaleSessions(dir, liveSessions)

	// Read remaining sessions
	sessions, err := readSessions(dir)
	if err != nil {
		return sessionsMsg(nil)
	}

	// Sort sessions
	sortSessions(sessions)

	return sessionsMsg(sessions)
}

// expandPath expands ~ to the user's home directory.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
