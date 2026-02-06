package remote

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/stwalsh4118/navi/internal/debug"
	"github.com/stwalsh4118/navi/internal/session"
)

// Polling constants
const (
	PollTimeout = 10 * time.Second
)

// SessionsResult contains the polling results from a single remote.
type SessionsResult struct {
	RemoteName string
	Sessions   []session.Info
	Error      error
}

// PollSessions polls all configured remotes for session status.
func PollSessions(pool *SSHPool, remotes []Config) []session.Info {
	if pool == nil || len(remotes) == 0 {
		return nil
	}

	results := make(chan SessionsResult, len(remotes))

	var wg sync.WaitGroup
	for _, remote := range remotes {
		wg.Add(1)
		go func(r Config) {
			defer wg.Done()
			sessions, err := PollSingleRemote(pool, r)
			results <- SessionsResult{
				RemoteName: r.Name,
				Sessions:   sessions,
				Error:      err,
			}
		}(remote)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var allSessions []session.Info
	for result := range results {
		if result.Error != nil {
			debug.Log("remote[%s]: poll error: %v", result.RemoteName, result.Error)
			continue
		}
		debug.Log("remote[%s]: got %d sessions", result.RemoteName, len(result.Sessions))
		allSessions = append(allSessions, result.Sessions...)
	}

	debug.Log("PollSessions: total %d sessions from %d remotes", len(allSessions), len(remotes))
	return allSessions
}

// PollSingleRemote polls a single remote for session status.
func PollSingleRemote(pool *SSHPool, remote Config) ([]session.Info, error) {
	sessionsDir := remote.SessionsDir
	if sessionsDir == "" {
		sessionsDir = DefaultSessionsDir
	}

	cmd := fmt.Sprintf("cat '%s'/*.json 2>/dev/null || true", strings.ReplaceAll(sessionsDir, "'", "'\\''"))

	debug.Log("remote[%s]: executing command: %s", remote.Name, cmd)

	output, err := pool.Execute(remote.Name, cmd)
	if err != nil {
		debug.Log("remote[%s]: execute error: %v", remote.Name, err)
		return nil, fmt.Errorf("failed to execute remote command: %w", err)
	}

	debug.Log("remote[%s]: raw output (%d bytes): %q", remote.Name, len(output), string(output))

	sessions := ParseSessionOutput(string(output), remote.Name)

	debug.Log("remote[%s]: parsed %d sessions", remote.Name, len(sessions))

	return sessions, nil
}

// ParseSessionOutput parses concatenated JSON session data from remote output.
// The output from `cat *.json` produces concatenated top-level JSON objects
// (not an array), so we use Decoder.Decode in a loop until io.EOF.
func ParseSessionOutput(output string, remoteName string) []session.Info {
	if strings.TrimSpace(output) == "" {
		return nil
	}

	var sessions []session.Info

	decoder := json.NewDecoder(strings.NewReader(output))

	for {
		var s session.Info
		if err := decoder.Decode(&s); err != nil {
			if err == io.EOF {
				break
			}
			debug.Log("remote[%s]: skipping malformed JSON object: %v", remoteName, err)
			break
		}
		s.Remote = remoteName
		sessions = append(sessions, s)
	}

	return sessions
}
