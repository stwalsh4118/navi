package remote

import (
	"fmt"
	"strings"
)

// resolveSessionsDir handles tilde expansion for the remote sessions directory.
// Uses $HOME instead of ~ so it works inside shell quotes on the remote.
func resolveSessionsDir(sessionsDir string) string {
	if sessionsDir == "" {
		sessionsDir = DefaultSessionsDir
	}
	if strings.HasPrefix(sessionsDir, "~/") {
		sessionsDir = "$HOME" + sessionsDir[1:]
	}
	return sessionsDir
}

// shellQuote wraps a string in single quotes, escaping any embedded single quotes.
// This prevents shell injection when embedding user-provided values in commands.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

// buildKillCommand builds the shell command to kill a tmux session and remove its status file.
// Uses ; so cleanup runs even if tmux kill fails.
func buildKillCommand(sessionName, sessionsDir string) string {
	return fmt.Sprintf(
		"tmux kill-session -t %s ; rm -f %s",
		shellQuote(sessionName),
		shellQuote(sessionsDir+"/"+sessionName+".json"),
	)
}

// buildRenameCommand builds the shell command to rename a tmux session and update its status file.
func buildRenameCommand(oldName, newName, sessionsDir string) string {
	oldFile := sessionsDir + "/" + oldName + ".json"
	newFile := sessionsDir + "/" + newName + ".json"
	// Escape sed metacharacters in newName to prevent injection in the sed pattern.
	safeName := strings.ReplaceAll(newName, `\`, `\\`)
	safeName = strings.ReplaceAll(safeName, `/`, `\/`)
	safeName = strings.ReplaceAll(safeName, `&`, `\&`)
	return fmt.Sprintf(
		"tmux rename-session -t %s %s && sed -i 's/\"tmux_session\"[[:space:]]*:[[:space:]]*\"[^\"]*\"/\"tmux_session\": \"%s\"/' %s && mv %s %s",
		shellQuote(oldName),
		shellQuote(newName),
		safeName,
		shellQuote(oldFile),
		shellQuote(oldFile),
		shellQuote(newFile),
	)
}

// buildDismissCommand builds the shell command to dismiss a session's notification.
func buildDismissCommand(sessionName, sessionsDir string) string {
	file := sessionsDir + "/" + sessionName + ".json"
	return fmt.Sprintf(
		`sed -i -e 's/"status"[[:space:]]*:[[:space:]]*"[^"]*"/"status": "working"/' -e 's/"message"[[:space:]]*:[[:space:]]*"[^"]*"/"message": ""/' -e "s/\"timestamp\"[[:space:]]*:[[:space:]]*[0-9]*/\"timestamp\": $(date +%%s)/" %s`,
		shellQuote(file),
	)
}

// KillSession kills a remote tmux session and removes its status file.
// Uses ; instead of && so the file cleanup runs even if tmux kill fails
// (e.g., the session was already gone).
func KillSession(pool *SSHPool, remoteName, sessionName, sessionsDir string) error {
	sessionsDir = resolveSessionsDir(sessionsDir)
	cmd := buildKillCommand(sessionName, sessionsDir)
	_, err := pool.Execute(remoteName, cmd)
	return err
}

// RenameSession renames a remote tmux session and updates its status file.
// Performs three operations in a single SSH call:
// 1. Rename the tmux session
// 2. Update the tmux_session field inside the JSON file
// 3. Move the JSON file to match the new session name
func RenameSession(pool *SSHPool, remoteName, oldName, newName, sessionsDir string) error {
	sessionsDir = resolveSessionsDir(sessionsDir)
	cmd := buildRenameCommand(oldName, newName, sessionsDir)
	_, err := pool.Execute(remoteName, cmd)
	return err
}

// DismissSession clears a remote session's notification by updating its status file.
// Sets status to "working", clears the message, and updates the timestamp.
func DismissSession(pool *SSHPool, remoteName, sessionName, sessionsDir string) error {
	sessionsDir = resolveSessionsDir(sessionsDir)
	cmd := buildDismissCommand(sessionName, sessionsDir)
	_, err := pool.Execute(remoteName, cmd)
	return err
}
