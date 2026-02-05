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
	statusWorking    = "working"
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

// dismissSession writes the session status as "working" to dismiss its notification.
// This clears the message and updates the timestamp.
func dismissSession(session SessionInfo) error {
	dir := expandPath(statusDir)
	path := filepath.Join(dir, session.TmuxSession+".json")

	// Update status to working
	session.Status = statusWorking
	session.Message = ""
	session.Timestamp = time.Now().Unix()

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// createSessionCmd returns a command that creates a new tmux session.
// The session starts a shell and runs 'claude' via send-keys, so when claude
// exits the session remains open with a shell prompt.
// It also creates an initial status file so the session appears immediately in the UI.
// If skipPermissions is true, claude is started with --dangerously-skip-permissions.
func createSessionCmd(name, dir string, skipPermissions bool) tea.Cmd {
	return func() tea.Msg {
		// Create tmux session with a shell (no command)
		cmd := exec.Command("tmux", "new-session", "-d", "-s", name, "-c", dir)
		err := cmd.Run()
		if err != nil {
			return createSessionResultMsg{err: err}
		}

		// Send keys to start claude - session stays open when claude exits
		claudeCmd := "claude"
		if skipPermissions {
			claudeCmd = "claude --dangerously-skip-permissions"
		}
		sendKeys := exec.Command("tmux", "send-keys", "-t", name, claudeCmd, "Enter")
		sendKeys.Run() // Ignore error - session is created, claude just won't auto-start

		// Create initial status file so session appears immediately in UI
		statusDirPath := expandPath(statusDir)
		// Ensure directory exists
		os.MkdirAll(statusDirPath, 0755)

		session := SessionInfo{
			TmuxSession: name,
			Status:      statusWorking,
			Message:     "",
			CWD:         dir,
			Timestamp:   time.Now().Unix(),
		}

		data, err := json.MarshalIndent(session, "", "  ")
		if err != nil {
			// Session was created, but we couldn't marshal the status
			// Return success anyway - the hook will create it later
			return createSessionResultMsg{err: nil}
		}

		statusPath := filepath.Join(statusDirPath, name+".json")
		os.WriteFile(statusPath, data, 0644) // Ignore error - hook will create it later if needed

		return createSessionResultMsg{err: nil}
	}
}

// killSessionCmd returns a command that kills a tmux session and cleans up its status file.
func killSessionCmd(name string) tea.Cmd {
	return func() tea.Msg {
		// Kill tmux session: tmux kill-session -t <name>
		cmd := exec.Command("tmux", "kill-session", "-t", name)
		err := cmd.Run()
		if err != nil {
			return killSessionResultMsg{err: err}
		}

		// Delete status file
		dir := expandPath(statusDir)
		statusPath := filepath.Join(dir, name+".json")
		os.Remove(statusPath) // Ignore error - file may not exist

		return killSessionResultMsg{err: nil}
	}
}

// renameSessionCmd returns a command that renames a tmux session and its status file.
func renameSessionCmd(oldName, newName string) tea.Cmd {
	return func() tea.Msg {
		// Rename tmux session: tmux rename-session -t <old> <new>
		cmd := exec.Command("tmux", "rename-session", "-t", oldName, newName)
		err := cmd.Run()
		if err != nil {
			return renameSessionResultMsg{err: err, newName: newName}
		}

		// Rename status file
		dir := expandPath(statusDir)
		oldPath := filepath.Join(dir, oldName+".json")
		newPath := filepath.Join(dir, newName+".json")

		// Read old file, update session name, write to new file
		data, err := os.ReadFile(oldPath)
		if err == nil {
			var session SessionInfo
			if json.Unmarshal(data, &session) == nil {
				session.TmuxSession = newName
				if newData, err := json.MarshalIndent(session, "", "  "); err == nil {
					os.WriteFile(newPath, newData, 0644)
				}
			}
			os.Remove(oldPath) // Remove old file
		}

		return renameSessionResultMsg{err: nil, newName: newName}
	}
}
