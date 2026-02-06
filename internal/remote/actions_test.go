package remote

import (
	"strings"
	"testing"
)

func TestShellQuote(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple string",
			input: "my-session",
			want:  "'my-session'",
		},
		{
			name:  "string with spaces",
			input: "my session",
			want:  "'my session'",
		},
		{
			name:  "string with single quote",
			input: "it's",
			want:  "'it'\"'\"'s'",
		},
		{
			name:  "empty string",
			input: "",
			want:  "''",
		},
		{
			name:  "string with double quotes",
			input: `say "hello"`,
			want:  `'say "hello"'`,
		},
		{
			name:  "string with special chars",
			input: "test;rm -rf /",
			want:  "'test;rm -rf /'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shellQuote(tt.input)
			if got != tt.want {
				t.Errorf("shellQuote(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveSessionsDir(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty uses default with HOME expansion",
			input: "",
			want:  "$HOME/.claude-sessions",
		},
		{
			name:  "tilde prefix expanded",
			input: "~/.my-sessions",
			want:  "$HOME/.my-sessions",
		},
		{
			name:  "absolute path unchanged",
			input: "/opt/sessions",
			want:  "/opt/sessions",
		},
		{
			name:  "dollar HOME already set unchanged",
			input: "$HOME/.claude-sessions",
			want:  "$HOME/.claude-sessions",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveSessionsDir(tt.input)
			if got != tt.want {
				t.Errorf("resolveSessionsDir(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestKillSessionCommand verifies the structure of the kill command.
func TestKillSessionCommand(t *testing.T) {
	// We can't call KillSession without a real SSHPool, but we can verify
	// the command would be built correctly by testing the component parts.
	sessionName := "my-session"
	sessionsDir := resolveSessionsDir("~/.claude-sessions")

	cmd := buildKillCommand(sessionName, sessionsDir)

	// Should contain tmux kill-session
	if !strings.Contains(cmd, "tmux kill-session -t") {
		t.Error("kill command missing tmux kill-session")
	}
	// Should contain rm -f
	if !strings.Contains(cmd, "rm -f") {
		t.Error("kill command missing rm -f")
	}
	// Should use ; not &&
	if !strings.Contains(cmd, " ; ") {
		t.Error("kill command should use ; not &&")
	}
	// Should contain the session name (quoted)
	if !strings.Contains(cmd, shellQuote(sessionName)) {
		t.Error("kill command missing quoted session name")
	}
	// Should contain the JSON file path
	if !strings.Contains(cmd, sessionName+".json") {
		t.Error("kill command missing JSON file path")
	}
}

// TestRenameSessionCommand verifies the structure of the rename command.
func TestRenameSessionCommand(t *testing.T) {
	oldName := "old-session"
	newName := "new-session"
	sessionsDir := resolveSessionsDir("~/.claude-sessions")

	cmd := buildRenameCommand(oldName, newName, sessionsDir)

	// Should contain tmux rename-session
	if !strings.Contains(cmd, "tmux rename-session -t") {
		t.Error("rename command missing tmux rename-session")
	}
	// Should contain sed for updating tmux_session field
	if !strings.Contains(cmd, "sed -i") {
		t.Error("rename command missing sed")
	}
	if !strings.Contains(cmd, "tmux_session") {
		t.Error("rename command missing tmux_session field update")
	}
	// Should contain mv for renaming the file
	if !strings.Contains(cmd, "mv") {
		t.Error("rename command missing mv")
	}
	// Should contain both old and new names
	if !strings.Contains(cmd, oldName) {
		t.Error("rename command missing old name")
	}
	if !strings.Contains(cmd, newName) {
		t.Error("rename command missing new name")
	}
	// Should use && to chain (all must succeed)
	if !strings.Contains(cmd, " && ") {
		t.Error("rename command should use && to chain operations")
	}
}

// TestDismissSessionCommand verifies the structure of the dismiss command.
func TestDismissSessionCommand(t *testing.T) {
	sessionName := "my-session"
	sessionsDir := resolveSessionsDir("~/.claude-sessions")

	cmd := buildDismissCommand(sessionName, sessionsDir)

	// Should contain sed
	if !strings.Contains(cmd, "sed -i") {
		t.Error("dismiss command missing sed")
	}
	// Should set status to working
	if !strings.Contains(cmd, `"working"`) {
		t.Error("dismiss command should set status to working")
	}
	// Should clear message
	if !strings.Contains(cmd, `"message"`) {
		t.Error("dismiss command should update message field")
	}
	// Should update timestamp with date command
	if !strings.Contains(cmd, "date +") || !strings.Contains(cmd, "timestamp") {
		t.Error("dismiss command should update timestamp with date command")
	}
	// Should contain the file path
	if !strings.Contains(cmd, sessionName+".json") {
		t.Error("dismiss command missing JSON file path")
	}
}

// TestKillSessionCommandQuoting verifies that special characters in session names are properly quoted.
func TestKillSessionCommandQuoting(t *testing.T) {
	// Session name with shell injection attempt
	sessionName := "test;rm -rf /"
	sessionsDir := resolveSessionsDir("/opt/sessions")

	cmd := buildKillCommand(sessionName, sessionsDir)

	// The session name should be single-quoted, preventing injection
	if !strings.Contains(cmd, shellQuote(sessionName)) {
		t.Error("kill command should properly quote dangerous session names")
	}
}

// TestRenameSessionCommandQuoting verifies quoting in rename commands.
func TestRenameSessionCommandQuoting(t *testing.T) {
	oldName := "old session"
	newName := "new'session"
	sessionsDir := resolveSessionsDir("~/.claude-sessions")

	cmd := buildRenameCommand(oldName, newName, sessionsDir)

	// Should have quoted the old name
	if !strings.Contains(cmd, shellQuote(oldName)) {
		t.Error("rename command should quote old name with spaces")
	}
	// Should have quoted the new name (which contains a single quote)
	if !strings.Contains(cmd, shellQuote(newName)) {
		t.Error("rename command should quote new name with single quote")
	}
}

// TestDismissSessionWithAbsolutePath verifies dismiss with an absolute sessions dir.
func TestDismissSessionWithAbsolutePath(t *testing.T) {
	sessionName := "dev-session"
	sessionsDir := resolveSessionsDir("/var/lib/sessions")

	cmd := buildDismissCommand(sessionName, sessionsDir)

	if !strings.Contains(cmd, "/var/lib/sessions/dev-session.json") {
		t.Error("dismiss command should use absolute path when provided")
	}
}
