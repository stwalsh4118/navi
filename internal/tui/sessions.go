package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/git"
	"github.com/stwalsh4118/navi/internal/metrics"
	"github.com/stwalsh4118/navi/internal/pathutil"
	"github.com/stwalsh4118/navi/internal/remote"
	"github.com/stwalsh4118/navi/internal/session"
	"github.com/stwalsh4118/navi/internal/tokens"
)

// readSessions reads all JSON status files from the specified directory
// and parses them into session.Info structs.
// Returns an empty slice if directory doesn't exist or on errors.
// Malformed JSON files are skipped silently.
func readSessions(dir string) ([]session.Info, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []session.Info
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue // skip unreadable files
		}

		var s session.Info
		if err := json.Unmarshal(data, &s); err != nil {
			continue // skip malformed JSON
		}

		sessions = append(sessions, s)
	}
	return sessions, nil
}

// capturePane captures the recent output from a tmux session pane.
// Uses tmux capture-pane to retrieve the last N lines of output.
// ANSI escape sequences are stripped for clean display.
// Returns an empty string if the session doesn't exist or tmux is not running.
func capturePane(sessionName string, lines int) (string, error) {
	// Build the -S argument for number of lines (negative value captures from end)
	lineArg := fmt.Sprintf("-%d", lines)

	cmd := exec.Command("tmux", "capture-pane", "-t", sessionName, "-p", "-S", lineArg)
	output, err := cmd.Output()
	if err != nil {
		// tmux returns error if session doesn't exist or server not running
		return "", err
	}

	// Strip ANSI escape sequences for clean display
	cleaned := StripANSI(string(output))

	// Trim trailing whitespace but preserve internal structure
	return strings.TrimRight(cleaned, "\n\t "), nil
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

// tickCmd returns a command that fires after pollInterval.
func tickCmd() tea.Cmd {
	return tea.Tick(session.PollInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// previewTickCmd returns a command that fires after previewPollInterval.
func previewTickCmd() tea.Cmd {
	return tea.Tick(previewPollInterval, func(t time.Time) tea.Msg {
		return previewTickMsg(t)
	})
}

// previewDebounceCmd returns a command that fires after previewDebounceDelay.
func previewDebounceCmd() tea.Cmd {
	return tea.Tick(previewDebounceDelay, func(t time.Time) tea.Msg {
		return previewDebounceMsg{}
	})
}

// gitTickCmd returns a command that fires after gitPollInterval.
func gitTickCmd() tea.Cmd {
	return tea.Tick(git.PollInterval, func(t time.Time) tea.Msg {
		return gitTickMsg(t)
	})
}

// prAutoRefreshTickCmd returns a command that fires after prAutoRefreshInterval.
func prAutoRefreshTickCmd() tea.Cmd {
	return tea.Tick(prAutoRefreshInterval, func(t time.Time) tea.Msg {
		return prAutoRefreshTickMsg(t)
	})
}

// pollGitInfoCmd returns a command that polls git info for local session working directories.
// Git info is fetched concurrently for all local sessions to minimize latency.
// Remote sessions are handled separately by pollRemoteGitInfoCmd.
func pollGitInfoCmd(sessions []session.Info) tea.Cmd {
	return func() tea.Msg {
		// Collect unique CWDs from local sessions only
		cwds := make(map[string]bool)
		for _, s := range sessions {
			if s.CWD != "" && s.Remote == "" {
				cwds[s.CWD] = true
			}
		}

		if len(cwds) == 0 {
			return gitInfoMsg{cache: make(map[string]*git.Info)}
		}

		// Fetch git info concurrently
		type result struct {
			cwd  string
			info *git.Info
		}
		results := make(chan result, len(cwds))

		for cwd := range cwds {
			go func(cwd string) {
				dir := pathutil.ExpandPath(cwd)
				info := git.GetInfo(dir)
				results <- result{cwd: cwd, info: info}
			}(cwd)
		}

		// Collect results
		cache := make(map[string]*git.Info)
		for i := 0; i < len(cwds); i++ {
			r := <-results
			if r.info != nil {
				cache[r.cwd] = r.info
			}
		}

		return gitInfoMsg{cache: cache}
	}
}

// pollRemoteGitInfoCmd returns a command that polls git info for remote sessions via SSH.
// It fetches git info concurrently for all unique remote+CWD combinations.
func pollRemoteGitInfoCmd(pool *remote.SSHPool, sessions []session.Info) tea.Cmd {
	// Collect unique remote+CWD pairs
	type remoteKey struct {
		remote string
		cwd    string
	}
	seen := make(map[remoteKey]bool)
	var keys []remoteKey
	for _, s := range sessions {
		if s.Remote != "" && s.CWD != "" {
			k := remoteKey{remote: s.Remote, cwd: s.CWD}
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}

	if len(keys) == 0 {
		return nil
	}

	return func() tea.Msg {
		type result struct {
			cwd  string
			info *git.Info
		}
		results := make(chan result, len(keys))

		for _, k := range keys {
			go func(remoteName, cwd string) {
				info, _ := remote.FetchGitInfo(pool, remoteName, cwd)
				results <- result{cwd: cwd, info: info}
			}(k.remote, k.cwd)
		}

		cache := make(map[string]*git.Info)
		for i := 0; i < len(keys); i++ {
			r := <-results
			if r.info != nil {
				cache[r.cwd] = r.info
			}
		}

		return gitInfoMsg{cache: cache}
	}
}

// killRemoteSessionCmd returns a command that kills a remote tmux session via SSH.
func killRemoteSessionCmd(pool *remote.SSHPool, remoteName, sessionName string) tea.Cmd {
	return func() tea.Msg {
		config := pool.GetRemoteConfig(remoteName)
		if config == nil {
			return killSessionResultMsg{err: fmt.Errorf("remote %q not found", remoteName)}
		}
		err := remote.KillSession(pool, remoteName, sessionName, config.SessionsDir)
		return killSessionResultMsg{err: err}
	}
}

// renameRemoteSessionCmd returns a command that renames a remote tmux session via SSH.
func renameRemoteSessionCmd(pool *remote.SSHPool, remoteName, oldName, newName string) tea.Cmd {
	return func() tea.Msg {
		config := pool.GetRemoteConfig(remoteName)
		if config == nil {
			return renameSessionResultMsg{err: fmt.Errorf("remote %q not found", remoteName), newName: newName}
		}
		err := remote.RenameSession(pool, remoteName, oldName, newName, config.SessionsDir)
		return renameSessionResultMsg{err: err, newName: newName}
	}
}

// dismissRemoteSessionCmd returns a command that dismisses a remote session via SSH.
func dismissRemoteSessionCmd(pool *remote.SSHPool, remoteName, sessionName string) tea.Cmd {
	return func() tea.Msg {
		config := pool.GetRemoteConfig(remoteName)
		if config == nil {
			return remoteDismissResultMsg{err: fmt.Errorf("remote %q not found", remoteName)}
		}
		err := remote.DismissSession(pool, remoteName, sessionName, config.SessionsDir)
		return remoteDismissResultMsg{err: err}
	}
}

// captureRemotePreviewCmd returns a command that captures preview content from a remote tmux session via SSH.
func captureRemotePreviewCmd(pool *remote.SSHPool, remoteName, sessionName string) tea.Cmd {
	return func() tea.Msg {
		content, err := remote.CapturePane(pool, remoteName, sessionName, previewDefaultLines)
		if err != nil {
			return previewContentMsg{content: "Failed to fetch remote preview: " + err.Error()}
		}
		return previewContentMsg{content: content}
	}
}

// capturePreviewForSession returns the appropriate capture command for the given session.
// Uses SSH for remote sessions and local tmux for local sessions.
func (m Model) capturePreviewForSession(s session.Info) tea.Cmd {
	if s.Remote != "" && m.SSHPool != nil {
		return captureRemotePreviewCmd(m.SSHPool, s.Remote, s.TmuxSession)
	}
	return capturePreviewCmd(s.TmuxSession)
}

// fetchRemoteGitCmd returns a command that fetches git info from a remote session via SSH.
func fetchRemoteGitCmd(pool *remote.SSHPool, remoteName, cwd string) tea.Cmd {
	return func() tea.Msg {
		info, err := remote.FetchGitInfo(pool, remoteName, cwd)
		return remoteGitInfoMsg{cwd: cwd, info: info, err: err}
	}
}

// fetchPRCmd returns a command that fetches PR detail for a specific directory.
// This is called lazily (e.g., when opening git detail view) to avoid slow gh CLI calls on every poll.
func fetchPRCmd(cwd string) tea.Cmd {
	return func() tea.Msg {
		dir := pathutil.ExpandPath(cwd)
		prDetail := git.GetPRDetail(dir)
		prNum := 0
		if prDetail != nil {
			prNum = prDetail.Number
		}
		return gitPRMsg{cwd: cwd, prNum: prNum, prDetail: prDetail}
	}
}

// fetchPRCommentsCmd fetches PR comments asynchronously.
func fetchPRCommentsCmd(cwd string, prNum int) tea.Cmd {
	return func() tea.Msg {
		dir := pathutil.ExpandPath(cwd)
		comments, err := git.GetPRComments(dir, prNum)
		return gitPRCommentsMsg{comments: comments, err: err}
	}
}

// fetchRemotePRCommentsCmd fetches PR comments for remote sessions.
func fetchRemotePRCommentsCmd(owner, repo string, prNum int) tea.Cmd {
	return func() tea.Msg {
		comments, err := git.GetPRCommentsByRepo(owner, repo, prNum)
		return gitPRCommentsMsg{comments: comments, err: err}
	}
}

// fetchRemotePRCmd returns a command that fetches PR detail using branch and remote URL
// instead of a local directory. Used for remote sessions where the CWD doesn't exist locally.
func fetchRemotePRCmd(cwd, branch, remoteURL string) tea.Cmd {
	return func() tea.Msg {
		prDetail := git.GetPRDetailByRepo(branch, remoteURL)
		prNum := 0
		if prDetail != nil {
			prNum = prDetail.Number
		}
		return gitPRMsg{cwd: cwd, prNum: prNum, prDetail: prDetail}
	}
}

// pollSessions orchestrates reading, cleaning stale, and sorting sessions.
// Returns a sessionsMsg with the current session list.
func pollSessions() tea.Msg {
	dir := pathutil.ExpandPath(session.StatusDir)

	// Get live tmux sessions
	liveSessions, _ := listTmuxSessions()

	// Clean stale sessions
	cleanStaleSessions(dir, liveSessions)

	// Read remaining sessions
	sessions, err := readSessions(dir)
	if err != nil {
		return sessionsMsg(nil)
	}

	// Enrich sessions with token data from transcripts
	enrichSessionsWithTokens(sessions)

	// Sort sessions
	session.SortSessions(sessions)

	return sessionsMsg(sessions)
}

// enrichSessionsWithTokens adds token metrics to sessions by parsing their transcript files.
func enrichSessionsWithTokens(sessions []session.Info) {
	for i := range sessions {
		if sessions[i].CWD == "" {
			continue
		}

		toks := tokens.GetSessionTokens(sessions[i].CWD)
		if toks == nil {
			continue
		}

		// Initialize Metrics if nil
		if sessions[i].Metrics == nil {
			sessions[i].Metrics = &metrics.Metrics{}
		}

		sessions[i].Metrics.Tokens = toks
	}
}

// dismissSession writes the session status as "working" to dismiss its notification.
// This clears the message and updates the timestamp.
func dismissSession(s session.Info) error {
	dir := pathutil.ExpandPath(session.StatusDir)
	path := filepath.Join(dir, s.TmuxSession+".json")

	// Update status to working
	s.Status = session.StatusWorking
	s.Message = ""
	s.Timestamp = time.Now().Unix()

	data, err := json.MarshalIndent(s, "", "  ")
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
		statusDirPath := pathutil.ExpandPath(session.StatusDir)
		// Ensure directory exists
		os.MkdirAll(statusDirPath, 0755)

		s := session.Info{
			TmuxSession: name,
			Status:      session.StatusWorking,
			Message:     "",
			CWD:         dir,
			Timestamp:   time.Now().Unix(),
		}

		data, err := json.MarshalIndent(s, "", "  ")
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
		dir := pathutil.ExpandPath(session.StatusDir)
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
		dir := pathutil.ExpandPath(session.StatusDir)
		oldPath := filepath.Join(dir, oldName+".json")
		newPath := filepath.Join(dir, newName+".json")

		// Read old file, update session name, write to new file
		data, err := os.ReadFile(oldPath)
		if err == nil {
			var s session.Info
			if json.Unmarshal(data, &s) == nil {
				s.TmuxSession = newName
				if newData, err := json.MarshalIndent(s, "", "  "); err == nil {
					os.WriteFile(newPath, newData, 0644)
				}
			}
			os.Remove(oldPath) // Remove old file
		}

		return renameSessionResultMsg{err: nil, newName: newName}
	}
}
