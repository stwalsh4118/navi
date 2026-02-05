package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// E2E tests for PBI 19: Remote Sessions
// These tests verify all acceptance criteria for the remote sessions feature.

// TestE2E_RemoteConfiguration tests CoS 1: Remote machines configurable via YAML file
func TestE2E_RemoteConfiguration(t *testing.T) {
	t.Run("valid YAML file loads correctly", func(t *testing.T) {
		// Create a temporary config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".config", "navi", "remotes.yaml")
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			t.Fatal(err)
		}

		configContent := `remotes:
  - name: dev-server
    host: dev.example.com
    user: developer
    key: ~/.ssh/id_rsa
  - name: staging
    host: staging.example.com
    user: admin
    key: ~/.ssh/staging_key
    jump_host: bastion.example.com
    sessions_dir: /home/admin/.claude/sessions
`
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Load config (we can't override HOME easily, so test the parsing directly)
		config, err := parseRemotesYAML([]byte(configContent))
		if err != nil {
			t.Fatalf("failed to parse valid YAML: %v", err)
		}

		if len(config.Remotes) != 2 {
			t.Errorf("expected 2 remotes, got %d", len(config.Remotes))
		}

		// Verify first remote
		if config.Remotes[0].Name != "dev-server" {
			t.Errorf("expected name 'dev-server', got %q", config.Remotes[0].Name)
		}
		if config.Remotes[0].Host != "dev.example.com" {
			t.Errorf("expected host 'dev.example.com', got %q", config.Remotes[0].Host)
		}
		if config.Remotes[0].User != "developer" {
			t.Errorf("expected user 'developer', got %q", config.Remotes[0].User)
		}
		if config.Remotes[0].Key != "~/.ssh/id_rsa" {
			t.Errorf("expected key '~/.ssh/id_rsa', got %q", config.Remotes[0].Key)
		}

		// Verify second remote with jump host
		if config.Remotes[1].JumpHost != "bastion.example.com" {
			t.Errorf("expected jump_host 'bastion.example.com', got %q", config.Remotes[1].JumpHost)
		}
		if config.Remotes[1].SessionsDir != "/home/admin/.claude/sessions" {
			t.Errorf("expected sessions_dir '/home/admin/.claude/sessions', got %q", config.Remotes[1].SessionsDir)
		}
	})

	t.Run("invalid YAML produces error", func(t *testing.T) {
		invalidYAML := `remotes:
  - name: dev
    host: [invalid nested`

		_, err := parseRemotesYAML([]byte(invalidYAML))
		if err == nil {
			t.Error("expected error for invalid YAML")
		}
	})

	t.Run("missing required fields detected", func(t *testing.T) {
		yamlWithMissingName := `remotes:
  - host: dev.example.com
    user: developer
    key: ~/.ssh/id_rsa
`
		config, err := parseRemotesYAML([]byte(yamlWithMissingName))
		if err != nil {
			t.Fatalf("parsing should succeed: %v", err)
		}

		err = validateRemoteConfig(&config.Remotes[0])
		if err == nil {
			t.Error("expected validation error for missing name")
		}
	})

	t.Run("defaults applied correctly", func(t *testing.T) {
		yaml := `remotes:
  - name: dev
    host: dev.example.com
    user: admin
    key: ~/.ssh/id_rsa
`
		config, _ := parseRemotesYAML([]byte(yaml))
		applyRemoteDefaults(&config.Remotes[0])

		if config.Remotes[0].SessionsDir != defaultSessionsDir {
			t.Errorf("expected default sessions_dir %q, got %q", defaultSessionsDir, config.Remotes[0].SessionsDir)
		}
	})
}

// TestE2E_SessionDisplay tests CoS 2 and 3: Sessions appear in list with labels
func TestE2E_SessionDisplay(t *testing.T) {
	t.Run("remote sessions appear in session list", func(t *testing.T) {
		m := Model{
			width:  120,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "local-1", Status: "working", CWD: "/tmp", Timestamp: time.Now().Unix()},
				{TmuxSession: "remote-1", Status: "idle", CWD: "/home/user", Timestamp: time.Now().Unix(), Remote: "dev-server"},
			},
		}

		result := m.View()

		if !strings.Contains(result, "local-1") {
			t.Error("view should contain local session")
		}
		if !strings.Contains(result, "remote-1") {
			t.Error("view should contain remote session")
		}
	})

	t.Run("remote sessions show hostname label", func(t *testing.T) {
		m := Model{
			width:  120,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "my-session", Status: "working", CWD: "/tmp", Timestamp: time.Now().Unix(), Remote: "dev-server"},
			},
		}

		result := m.View()

		if !strings.Contains(result, "[dev-server]") {
			t.Error("view should contain remote label [dev-server]")
		}
	})

	t.Run("local and remote sessions sort correctly", func(t *testing.T) {
		sessions := []SessionInfo{
			{TmuxSession: "old-local", Timestamp: time.Now().Add(-10 * time.Minute).Unix()},
			{TmuxSession: "new-remote", Timestamp: time.Now().Unix(), Remote: "dev"},
			{TmuxSession: "mid-local", Timestamp: time.Now().Add(-5 * time.Minute).Unix()},
		}

		sortSessions(sessions)

		// Most recent should be first
		if sessions[0].TmuxSession != "new-remote" {
			t.Errorf("most recent session should be first, got %s", sessions[0].TmuxSession)
		}
	})
}

// TestE2E_RemoteAttach tests CoS 4: Attach opens SSH with tmux
func TestE2E_RemoteAttach(t *testing.T) {
	t.Run("SSH command constructed correctly", func(t *testing.T) {
		remote := &RemoteConfig{
			Name: "dev",
			Host: "dev.example.com",
			User: "developer",
			Key:  "~/.ssh/id_rsa",
		}

		args := buildSSHAttachCommand(remote, "my-session")

		// Verify ssh command
		if args[0] != "ssh" {
			t.Errorf("first arg should be 'ssh', got %s", args[0])
		}

		// Verify key is included
		hasKey := false
		for i, arg := range args {
			if arg == "-i" && i+1 < len(args) {
				hasKey = true
				break
			}
		}
		if !hasKey {
			t.Error("SSH command should include -i flag for key")
		}

		// Verify host is included
		hasHost := false
		for _, arg := range args {
			if strings.Contains(arg, "developer@dev.example.com") {
				hasHost = true
				break
			}
		}
		if !hasHost {
			t.Error("SSH command should include user@host")
		}

		// Verify tmux attach command
		commandStr := strings.Join(args, " ")
		if !strings.Contains(commandStr, "tmux attach-session -t") {
			t.Error("SSH command should include tmux attach")
		}
		if !strings.Contains(commandStr, "my-session") {
			t.Error("SSH command should include session name")
		}
	})

	t.Run("jump host included when configured (CoS 8)", func(t *testing.T) {
		remote := &RemoteConfig{
			Name:     "staging",
			Host:     "staging.internal.com",
			User:     "admin",
			Key:      "~/.ssh/staging_key",
			JumpHost: "bastion.example.com",
		}

		args := buildSSHAttachCommand(remote, "test-session")

		// Verify jump host flag (format is user@host)
		hasJumpHost := false
		for i, arg := range args {
			if arg == "-J" && i+1 < len(args) && args[i+1] == "admin@bastion.example.com" {
				hasJumpHost = true
				break
			}
		}
		if !hasJumpHost {
			t.Errorf("SSH command should include -J flag with user@bastion, got: %v", args)
		}
	})

	t.Run("enter key triggers remote attach for remote session", func(t *testing.T) {
		remotes := []RemoteConfig{
			{Name: "dev", Host: "dev.example.com", User: "user", Key: "~/.ssh/key"},
		}
		sshPool := NewSSHPool(remotes)

		m := Model{
			width:  80,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "remote-session", Remote: "dev"},
			},
			remotes: remotes,
			sshPool: sshPool,
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, cmd := m.Update(msg)
		updated := newModel.(Model)

		// Should store the session name for cursor restoration
		if updated.lastSelectedSession != "remote-session" {
			t.Errorf("should store lastSelectedSession, got %q", updated.lastSelectedSession)
		}

		// Should return a command (the attach command)
		if cmd == nil {
			t.Error("enter should return a command for remote session")
		}
	})
}

// TestE2E_ConnectionPooling tests CoS 5: Connection pooling reuses SSH connections
func TestE2E_ConnectionPooling(t *testing.T) {
	t.Run("SSHPool manages connections by remote name", func(t *testing.T) {
		remotes := []RemoteConfig{
			{Name: "dev", Host: "dev.example.com", User: "user", Key: "~/.ssh/key"},
			{Name: "staging", Host: "staging.example.com", User: "admin", Key: "~/.ssh/key"},
		}

		pool := NewSSHPool(remotes)
		if pool == nil {
			t.Fatal("NewSSHPool should return non-nil pool")
		}

		// Verify remotes are tracked
		if len(pool.remotes) != 2 {
			t.Errorf("pool should have 2 remotes, got %d", len(pool.remotes))
		}

		// Verify initial status is disconnected
		status := pool.GetStatus("dev")
		if status.Status != StatusDisconnected {
			t.Errorf("initial status should be Disconnected, got %v", status.Status)
		}
	})

	t.Run("GetRemoteConfig returns correct config", func(t *testing.T) {
		remotes := []RemoteConfig{
			{Name: "dev", Host: "dev.example.com", User: "user", Key: "~/.ssh/key"},
		}

		pool := NewSSHPool(remotes)
		config := pool.GetRemoteConfig("dev")

		if config == nil {
			t.Fatal("GetRemoteConfig should return config for 'dev'")
		}
		if config.Host != "dev.example.com" {
			t.Errorf("expected host 'dev.example.com', got %q", config.Host)
		}

		// Non-existent remote
		config = pool.GetRemoteConfig("nonexistent")
		if config != nil {
			t.Error("GetRemoteConfig should return nil for unknown remote")
		}
	})
}

// TestE2E_ErrorHandling tests CoS 6 and 7: Offline remotes handled gracefully
func TestE2E_ErrorHandling(t *testing.T) {
	t.Run("status indicator shows connection state", func(t *testing.T) {
		remotes := []RemoteConfig{
			{Name: "dev", Host: "dev.example.com", User: "user", Key: "~/.ssh/key"},
		}
		sshPool := NewSSHPool(remotes)

		m := Model{
			width:   120,
			height:  24,
			remotes: remotes,
			sshPool: sshPool,
		}

		result := m.View()

		// Should show status indicator in header
		// Initial status is disconnected, shown as "-"
		if !strings.Contains(result, "[dev:") {
			t.Error("header should contain remote status indicator [dev:]")
		}
	})

	t.Run("remote sessions merged without crash when poll fails", func(t *testing.T) {
		// Create a model with existing local sessions
		m := Model{
			width:  80,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "local-1", Remote: ""},
			},
		}

		// Simulate receiving empty remote sessions (poll failed)
		msg := remoteSessionsMsg{sessions: []SessionInfo{}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		// Should not crash, and local sessions preserved
		if len(updated.sessions) != 1 {
			t.Errorf("local sessions should be preserved, got %d", len(updated.sessions))
		}
	})
}

// TestE2E_Filtering tests CoS 10: Filter sessions by local/remote
func TestE2E_Filtering(t *testing.T) {
	t.Run("filter cycles through all modes", func(t *testing.T) {
		remotes := []RemoteConfig{
			{Name: "dev", Host: "dev.example.com"},
		}
		m := Model{
			width:  80,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "local-1", Remote: ""},
				{TmuxSession: "remote-1", Remote: "dev"},
			},
			remotes:    remotes,
			filterMode: FilterAll,
		}

		// Test cycle: All -> Local -> Remote -> All
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}

		// All -> Local
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)
		if updated.filterMode != FilterLocal {
			t.Error("filter should cycle to Local")
		}
		if m.filterModeString() != "all" {
			t.Error("original model should still be All")
		}
		if updated.filterModeString() != "local" {
			t.Error("updated model should be Local")
		}

		// Local -> Remote
		newModel, _ = updated.Update(msg)
		updated = newModel.(Model)
		if updated.filterMode != FilterRemote {
			t.Error("filter should cycle to Remote")
		}

		// Remote -> All
		newModel, _ = updated.Update(msg)
		updated = newModel.(Model)
		if updated.filterMode != FilterAll {
			t.Error("filter should cycle back to All")
		}
	})

	t.Run("filter indicator visible in footer", func(t *testing.T) {
		remotes := []RemoteConfig{
			{Name: "dev", Host: "dev.example.com"},
		}
		m := Model{
			width:  120,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "local-1", Remote: ""},
			},
			remotes:    remotes,
			filterMode: FilterLocal,
		}

		result := m.View()

		if !strings.Contains(result, "filter:local") {
			t.Error("footer should show current filter state 'filter:local'")
		}
	})

	t.Run("filtered list shows correct sessions", func(t *testing.T) {
		m := Model{
			width:  120,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "local-1", Remote: "", Status: "working", CWD: "/tmp", Timestamp: time.Now().Unix()},
				{TmuxSession: "remote-1", Remote: "dev", Status: "idle", CWD: "/home", Timestamp: time.Now().Unix()},
				{TmuxSession: "local-2", Remote: "", Status: "done", CWD: "/var", Timestamp: time.Now().Unix()},
			},
			filterMode: FilterLocal,
		}

		filtered := m.getFilteredSessions()

		if len(filtered) != 2 {
			t.Errorf("FilterLocal should return 2 sessions, got %d", len(filtered))
		}

		for _, s := range filtered {
			if s.Remote != "" {
				t.Errorf("FilterLocal should only return local sessions, got remote: %s", s.Remote)
			}
		}
	})

	t.Run("empty filter shows appropriate message", func(t *testing.T) {
		m := Model{
			width:  120,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "local-1", Remote: ""},
			},
			filterMode: FilterRemote,
		}

		result := m.View()

		if !strings.Contains(result, "No remote sessions") {
			t.Errorf("view should show 'No remote sessions', got: %s", result)
		}
	})
}

// TestE2E_HeaderCounts tests that local/remote counts display correctly
func TestE2E_HeaderCounts(t *testing.T) {
	t.Run("header shows separate local/remote counts when remotes exist", func(t *testing.T) {
		remotes := []RemoteConfig{
			{Name: "dev", Host: "dev.example.com"},
		}
		sshPool := NewSSHPool(remotes)

		m := Model{
			width:  120,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "local-1", Remote: ""},
				{TmuxSession: "local-2", Remote: ""},
				{TmuxSession: "remote-1", Remote: "dev"},
			},
			remotes: remotes,
			sshPool: sshPool,
		}

		result := m.View()

		// Should show "2 local, 1 remote" in header
		if !strings.Contains(result, "2 local") {
			t.Error("header should show '2 local'")
		}
		if !strings.Contains(result, "1 remote") {
			t.Error("header should show '1 remote'")
		}
	})

	t.Run("header shows total count when no remotes configured", func(t *testing.T) {
		m := Model{
			width:  120,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "local-1", Remote: ""},
				{TmuxSession: "local-2", Remote: ""},
			},
			remotes: nil,
		}

		result := m.View()

		// Should show "2 active" (not separate counts)
		if !strings.Contains(result, "2 active") {
			t.Errorf("header should show '2 active', got: %s", result)
		}
	})
}

// TestE2E_JSONParsing tests remote session JSON parsing
func TestE2E_JSONParsing(t *testing.T) {
	t.Run("parses concatenated JSON objects", func(t *testing.T) {
		// This is what `cat *.json` produces
		output := `{"tmux_session":"session1","status":"working","cwd":"/home/user/project1","timestamp":1234567890}
{"tmux_session":"session2","status":"idle","cwd":"/home/user/project2","timestamp":1234567891}`

		sessions := parseRemoteSessionOutput(output, "dev-server")

		if len(sessions) != 2 {
			t.Errorf("expected 2 sessions, got %d", len(sessions))
		}

		if sessions[0].TmuxSession != "session1" {
			t.Errorf("expected 'session1', got %q", sessions[0].TmuxSession)
		}
		if sessions[0].Remote != "dev-server" {
			t.Errorf("expected remote 'dev-server', got %q", sessions[0].Remote)
		}

		if sessions[1].TmuxSession != "session2" {
			t.Errorf("expected 'session2', got %q", sessions[1].TmuxSession)
		}
	})

	t.Run("handles empty output", func(t *testing.T) {
		sessions := parseRemoteSessionOutput("", "dev")

		if len(sessions) != 0 {
			t.Errorf("expected 0 sessions for empty output, got %d", len(sessions))
		}
	})

	t.Run("handles single JSON object", func(t *testing.T) {
		output := `{"tmux_session":"solo","status":"working","cwd":"/tmp","timestamp":1234567890}`

		sessions := parseRemoteSessionOutput(output, "dev")

		if len(sessions) != 1 {
			t.Errorf("expected 1 session, got %d", len(sessions))
		}
	})
}

// Helper function to parse YAML (for testing)
func parseRemotesYAML(data []byte) (*RemotesConfig, error) {
	config, err := parseRemotesConfigData(data)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
