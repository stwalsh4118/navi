package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Remote polling constants
const (
	// remotePollTimeout is the maximum time to wait for a remote poll
	remotePollTimeout = 10 * time.Second
)

// RemoteSessionsResult contains the polling results from a single remote.
type RemoteSessionsResult struct {
	RemoteName string
	Sessions   []SessionInfo
	Error      error
}

// pollRemoteSessions polls all configured remotes for session status.
// It runs polls in parallel and returns combined results with the Remote field set.
func pollRemoteSessions(pool *SSHPool, remotes []RemoteConfig) []SessionInfo {
	if pool == nil || len(remotes) == 0 {
		return nil
	}

	// Create channel for results
	results := make(chan RemoteSessionsResult, len(remotes))

	// Poll all remotes concurrently
	var wg sync.WaitGroup
	for _, remote := range remotes {
		wg.Add(1)
		go func(r RemoteConfig) {
			defer wg.Done()
			sessions, err := pollSingleRemote(pool, r)
			results <- RemoteSessionsResult{
				RemoteName: r.Name,
				Sessions:   sessions,
				Error:      err,
			}
		}(remote)
	}

	// Close results channel when all polls complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect all sessions
	var allSessions []SessionInfo
	for result := range results {
		if result.Error != nil {
			// Log error but continue - one remote failing shouldn't stop others
			continue
		}
		allSessions = append(allSessions, result.Sessions...)
	}

	return allSessions
}

// pollSingleRemote polls a single remote for session status.
// Returns sessions with the Remote field set to the remote's name.
func pollSingleRemote(pool *SSHPool, remote RemoteConfig) ([]SessionInfo, error) {
	// Build the command to cat all session files
	sessionsDir := remote.SessionsDir
	if sessionsDir == "" {
		sessionsDir = defaultSessionsDir
	}

	// Use a shell command to handle glob expansion and missing files gracefully
	// The 2>/dev/null suppresses "No such file" errors when no sessions exist
	cmd := fmt.Sprintf("cat %s/*.json 2>/dev/null || true", sessionsDir)

	// Execute the command
	output, err := pool.Execute(remote.Name, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to execute remote command: %w", err)
	}

	// Parse the output - it may contain multiple JSON objects concatenated
	sessions := parseRemoteSessionOutput(string(output), remote.Name)

	return sessions, nil
}

// parseRemoteSessionOutput parses concatenated JSON session data from remote output.
// Sets the Remote field on each session to remoteName.
// Malformed JSON entries are skipped silently.
func parseRemoteSessionOutput(output string, remoteName string) []SessionInfo {
	if strings.TrimSpace(output) == "" {
		return nil
	}

	var sessions []SessionInfo

	// Try to parse as individual JSON objects
	// The output from cat may have multiple JSON objects concatenated
	// We try to split by looking for object boundaries

	// Strategy: Use json.Decoder which handles stream of objects
	decoder := json.NewDecoder(strings.NewReader(output))

	for decoder.More() {
		var session SessionInfo
		if err := decoder.Decode(&session); err != nil {
			// Try to recover by skipping to next potential object
			// This handles cases where there's garbage between valid JSON
			break
		}

		// Set the remote name
		session.Remote = remoteName

		sessions = append(sessions, session)
	}

	// If decoder approach failed (common with concatenated files),
	// try splitting by common patterns
	if len(sessions) == 0 {
		sessions = parseMultipleJSONObjects(output, remoteName)
	}

	return sessions
}

// parseMultipleJSONObjects handles the case where multiple JSON files
// are concatenated without delimiters (common from `cat *.json`).
func parseMultipleJSONObjects(output string, remoteName string) []SessionInfo {
	var sessions []SessionInfo

	// Look for JSON object boundaries
	// Each session file should start with { and end with }
	depth := 0
	start := -1

	for i, ch := range output {
		switch ch {
		case '{':
			if depth == 0 {
				start = i
			}
			depth++
		case '}':
			depth--
			if depth == 0 && start >= 0 {
				// Found a complete object
				jsonStr := output[start : i+1]
				var session SessionInfo
				if err := json.Unmarshal([]byte(jsonStr), &session); err == nil {
					session.Remote = remoteName
					sessions = append(sessions, session)
				}
				start = -1
			}
		}
	}

	return sessions
}

// pollRemoteSessionsCmd returns a tea.Cmd that polls remote sessions.
// This is meant to be called from the Bubble Tea update loop.
func pollRemoteSessionsCmd(pool *SSHPool, remotes []RemoteConfig) func() remoteSessionsMsg {
	return func() remoteSessionsMsg {
		sessions := pollRemoteSessions(pool, remotes)
		return remoteSessionsMsg{sessions: sessions}
	}
}

// remoteSessionsMsg is the Bubble Tea message for remote session polling results.
type remoteSessionsMsg struct {
	sessions []SessionInfo
}
