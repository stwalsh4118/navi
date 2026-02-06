package tui

import (
	"testing"

	"github.com/stwalsh4118/navi/internal/remote"
	"github.com/stwalsh4118/navi/internal/session"
)

func TestParseRemoteSessionOutput(t *testing.T) {
	t.Run("parses single s JSON", func(t *testing.T) {
		output := `{"tmux_session":"test","status":"working","message":"","cwd":"/home/user","timestamp":1234567890}`

		sessions := remote.ParseSessionOutput(output, "remote1")

		if len(sessions) != 1 {
			t.Fatalf("expected 1 s, got %d", len(sessions))
		}
		if sessions[0].TmuxSession != "test" {
			t.Errorf("session name = %v, want test", sessions[0].TmuxSession)
		}
		if sessions[0].Remote != "remote1" {
			t.Errorf("remote = %v, want remote1", sessions[0].Remote)
		}
	})

	t.Run("parses multiple concatenated s JSONs", func(t *testing.T) {
		output := `{"tmux_session":"session1","status":"working","message":"","cwd":"/tmp/1","timestamp":1234567890}{"tmux_session":"session2","status":"waiting","message":"Need input","cwd":"/tmp/2","timestamp":1234567891}`

		sessions := remote.ParseSessionOutput(output, "dev-server")

		if len(sessions) != 2 {
			t.Fatalf("expected 2 sessions, got %d", len(sessions))
		}
		if sessions[0].TmuxSession != "session1" {
			t.Errorf("first s name = %v, want session1", sessions[0].TmuxSession)
		}
		if sessions[1].TmuxSession != "session2" {
			t.Errorf("second s name = %v, want session2", sessions[1].TmuxSession)
		}
		// Both should have the remote set
		for i, s := range sessions {
			if s.Remote != "dev-server" {
				t.Errorf("session %d remote = %v, want dev-server", i, s.Remote)
			}
		}
	})

	t.Run("handles empty output", func(t *testing.T) {
		sessions := remote.ParseSessionOutput("", "remote1")

		if sessions != nil {
			t.Errorf("expected nil for empty output, got %v", sessions)
		}
	})

	t.Run("handles whitespace-only output", func(t *testing.T) {
		sessions := remote.ParseSessionOutput("   \n\t  ", "remote1")

		if sessions != nil {
			t.Errorf("expected nil for whitespace output, got %v", sessions)
		}
	})

	t.Run("handles malformed JSON gracefully", func(t *testing.T) {
		output := `{"tmux_session":"valid","status":"working"}{invalid json here}{"tmux_session":"also-valid","status":"done"}`

		sessions := remote.ParseSessionOutput(output, "remote1")

		// Should get at least the valid s before the malformed one
		if len(sessions) < 1 {
			t.Error("should parse at least one valid session")
		}
	})

	t.Run("parses newline-separated JSON objects", func(t *testing.T) {
		output := `{"tmux_session":"session1","status":"working","message":"","cwd":"/tmp/1","timestamp":1234567890}
{"tmux_session":"session2","status":"working","message":"","cwd":"/tmp/2","timestamp":1234567891}`

		sessions := remote.ParseSessionOutput(output, "remote1")

		if len(sessions) != 2 {
			t.Fatalf("expected 2 sessions, got %d", len(sessions))
		}
	})
}

func TestParseMultipleJSONObjects(t *testing.T) {
	t.Run("parses concatenated objects without delimiters", func(t *testing.T) {
		output := `{"name":"a"}{"name":"b"}{"name":"c"}`

		sessions := remote.ParseMultipleJSONObjects(output, "test")

		if len(sessions) != 3 {
			t.Errorf("expected 3 objects, got %d", len(sessions))
		}
	})

	t.Run("handles nested braces correctly", func(t *testing.T) {
		output := `{"tmux_session":"test","git":{"branch":"main","dirty":true}}`

		sessions := remote.ParseMultipleJSONObjects(output, "test")

		if len(sessions) != 1 {
			t.Errorf("expected 1 object, got %d", len(sessions))
		}
		if sessions[0].TmuxSession != "test" {
			t.Errorf("session name = %v, want test", sessions[0].TmuxSession)
		}
	})
}

func TestPollRemoteSessions(t *testing.T) {
	t.Run("returns nil with nil pool", func(t *testing.T) {
		sessions := remote.PollSessions(nil, []remote.Config{})

		if sessions != nil {
			t.Errorf("expected nil, got %v", sessions)
		}
	})

	t.Run("returns nil with empty remotes", func(t *testing.T) {
		pool := remote.NewSSHPool([]remote.Config{})
		sessions := remote.PollSessions(pool, []remote.Config{})

		if sessions != nil {
			t.Errorf("expected nil, got %v", sessions)
		}
	})
}

func TestRemoteSessionsMsg(t *testing.T) {
	t.Run("message carries sessions", func(t *testing.T) {
		sessions := []session.Info{
			{TmuxSession: "remote1", Remote: "server1"},
			{TmuxSession: "remote2", Remote: "server2"},
		}

		msg := remoteSessionsMsg{sessions: sessions}

		if len(msg.sessions) != 2 {
			t.Errorf("message should have 2 sessions, got %d", len(msg.sessions))
		}
	})
}
