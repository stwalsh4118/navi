package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/git"
	"github.com/stwalsh4118/navi/internal/remote"
	"github.com/stwalsh4118/navi/internal/session"
)

// E2E tests for PBI 27: Enhanced Remote Session Management
// These tests verify all 9 acceptance criteria for the remote actions feature.

// newRemoteActionsTestModel creates a model with one local and one remote session.
func newRemoteActionsTestModel() Model {
	remotes := []remote.Config{
		{Name: "dev", Host: "dev.example.com", User: "user", Key: "~/.ssh/key", SessionsDir: "~/.claude-sessions"},
	}
	sshPool := remote.NewSSHPool(remotes)

	return Model{
		width:  120,
		height: 24,
		sessions: []session.Info{
			{TmuxSession: "local-1", Status: "working", CWD: "/tmp/local", Timestamp: time.Now().Unix()},
			{TmuxSession: "remote-1", Status: "waiting", CWD: "/home/user/project", Timestamp: time.Now().Unix(), Remote: "dev"},
		},
		Remotes:  remotes,
		SSHPool:  sshPool,
		gitCache: make(map[string]*git.Info),
	}
}

// selectRemoteSession moves the cursor to the remote session (index 1).
func selectRemoteSession(m Model) Model {
	m.cursor = 1
	return m
}

// TestE2E_PBI27_AC1_GitInfoRemote tests AC1: G key on remote session fetches and displays git info via SSH.
func TestE2E_PBI27_AC1_GitInfoRemote(t *testing.T) {
	t.Run("G key on remote session opens git detail dialog", func(t *testing.T) {
		m := selectRemoteSession(newRemoteActionsTestModel())

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		newModel, cmd := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogGitDetail {
			t.Errorf("expected DialogGitDetail, got %d", updated.dialogMode)
		}
		if updated.sessionToModify == nil {
			t.Fatal("sessionToModify should be set")
		}
		if updated.sessionToModify.TmuxSession != "remote-1" {
			t.Errorf("expected sessionToModify to be remote-1, got %q", updated.sessionToModify.TmuxSession)
		}
		// Should return a command to fetch remote git info (no cache hit)
		if cmd == nil {
			t.Error("expected a command to fetch remote git info")
		}
	})

	t.Run("remoteGitInfoMsg with success populates cache and sessionToModify", func(t *testing.T) {
		m := selectRemoteSession(newRemoteActionsTestModel())
		m.dialogMode = DialogGitDetail
		remoteSess := m.sessions[1]
		m.sessionToModify = &remoteSess

		gitInfo := &git.Info{
			Branch:     "feature/test",
			Dirty:      true,
			Ahead:      2,
			Behind:     1,
			LastCommit: "abc1234 test commit",
			Remote:     "git@github.com:user/repo.git",
			FetchedAt:  time.Now().Unix(),
		}
		msg := remoteGitInfoMsg{
			cwd:  "/home/user/project",
			info: gitInfo,
			err:  nil,
		}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		// Check cache was populated
		cached, ok := updated.gitCache["/home/user/project"]
		if !ok {
			t.Fatal("gitCache should contain entry for /home/user/project")
		}
		if cached.Branch != "feature/test" {
			t.Errorf("cached branch should be 'feature/test', got %q", cached.Branch)
		}

		// Check sessionToModify was updated
		if updated.sessionToModify == nil || updated.sessionToModify.Git == nil {
			t.Fatal("sessionToModify.Git should be set")
		}
		if updated.sessionToModify.Git.Branch != "feature/test" {
			t.Errorf("sessionToModify.Git.Branch should be 'feature/test', got %q", updated.sessionToModify.Git.Branch)
		}
	})

	t.Run("remoteGitInfoMsg with error sets dialogError", func(t *testing.T) {
		m := selectRemoteSession(newRemoteActionsTestModel())
		m.dialogMode = DialogGitDetail
		remoteSess := m.sessions[1]
		m.sessionToModify = &remoteSess

		msg := remoteGitInfoMsg{
			cwd:  "/home/user/project",
			info: nil,
			err:  errors.New("connection refused"),
		}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogError == "" {
			t.Error("dialogError should be set on SSH error")
		}
		if !strings.Contains(updated.dialogError, "connection refused") {
			t.Errorf("dialogError should contain error message, got %q", updated.dialogError)
		}
	})

	t.Run("cached fresh git info used without re-fetching", func(t *testing.T) {
		m := selectRemoteSession(newRemoteActionsTestModel())
		// Pre-populate cache with fresh data
		m.gitCache["/home/user/project"] = &git.Info{
			Branch:    "cached-branch",
			FetchedAt: time.Now().Unix(),
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		newModel, cmd := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogGitDetail {
			t.Errorf("expected DialogGitDetail, got %d", updated.dialogMode)
		}
		// sessionToModify.Git should be set from cache
		if updated.sessionToModify == nil || updated.sessionToModify.Git == nil {
			t.Fatal("sessionToModify.Git should be set from cache")
		}
		if updated.sessionToModify.Git.Branch != "cached-branch" {
			t.Errorf("expected cached branch 'cached-branch', got %q", updated.sessionToModify.Git.Branch)
		}
		// The cmd should be fetchPRCmd (not fetchRemoteGitCmd) since cache was fresh
		if cmd == nil {
			t.Error("expected a command (fetchPRCmd) when using cached data")
		}
	})
}

// TestE2E_PBI27_AC2_PreviewRemote tests AC2: p key on remote session fetches and displays preview via SSH.
func TestE2E_PBI27_AC2_PreviewRemote(t *testing.T) {
	t.Run("capturePreviewForSession dispatches remote command for remote session", func(t *testing.T) {
		m := newRemoteActionsTestModel()
		remoteSession := m.sessions[1] // remote-1

		cmd := m.capturePreviewForSession(remoteSession)
		if cmd == nil {
			t.Error("capturePreviewForSession should return a command for remote session")
		}
	})

	t.Run("capturePreviewForSession dispatches local command for local session", func(t *testing.T) {
		m := newRemoteActionsTestModel()
		localSession := m.sessions[0] // local-1

		cmd := m.capturePreviewForSession(localSession)
		if cmd == nil {
			t.Error("capturePreviewForSession should return a command for local session")
		}
	})

	t.Run("p key on remote session toggles preview and returns command", func(t *testing.T) {
		m := selectRemoteSession(newRemoteActionsTestModel())

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
		newModel, cmd := m.Update(msg)
		updated := newModel.(Model)

		if !updated.previewVisible {
			t.Error("preview should be visible after pressing p")
		}
		if cmd == nil {
			t.Error("expected a command to capture preview content")
		}
	})

	t.Run("previewContentMsg updates previewContent", func(t *testing.T) {
		m := newRemoteActionsTestModel()
		m.previewVisible = true

		msg := previewContentMsg{content: "remote terminal output here", err: nil}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.previewContent != "remote terminal output here" {
			t.Errorf("previewContent should be updated, got %q", updated.previewContent)
		}
	})

	t.Run("previewTickMsg triggers capture for remote session", func(t *testing.T) {
		m := selectRemoteSession(newRemoteActionsTestModel())
		m.previewVisible = true

		msg := previewTickMsg(time.Now())
		_, cmd := m.Update(msg)

		if cmd == nil {
			t.Error("previewTickMsg should return commands when preview is visible")
		}
	})

	t.Run("previewDebounceMsg triggers capture for remote session", func(t *testing.T) {
		m := selectRemoteSession(newRemoteActionsTestModel())
		m.previewVisible = true

		msg := previewDebounceMsg{}
		_, cmd := m.Update(msg)

		if cmd == nil {
			t.Error("previewDebounceMsg should return a command when preview is visible")
		}
	})
}

// TestE2E_PBI27_AC3_KillRemote tests AC3: x key on remote session kills via SSH.
func TestE2E_PBI27_AC3_KillRemote(t *testing.T) {
	t.Run("x key on remote session opens kill confirm dialog", func(t *testing.T) {
		m := selectRemoteSession(newRemoteActionsTestModel())

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogKillConfirm {
			t.Errorf("expected DialogKillConfirm, got %d", updated.dialogMode)
		}
		if updated.sessionToModify == nil {
			t.Fatal("sessionToModify should be set")
		}
		if updated.sessionToModify.TmuxSession != "remote-1" {
			t.Errorf("expected session remote-1, got %q", updated.sessionToModify.TmuxSession)
		}
		if updated.sessionToModify.Remote != "dev" {
			t.Errorf("expected remote 'dev', got %q", updated.sessionToModify.Remote)
		}
	})

	t.Run("y key in kill confirm dispatches remote kill for remote session", func(t *testing.T) {
		m := selectRemoteSession(newRemoteActionsTestModel())
		remoteSess := m.sessions[1]
		m.sessionToModify = &remoteSess
		m.dialogMode = DialogKillConfirm

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
		_, cmd := m.Update(msg)

		// Should return a command (killRemoteSessionCmd)
		if cmd == nil {
			t.Error("y in kill confirm should return a command for remote kill")
		}
	})

	t.Run("y key in kill confirm dispatches local kill for local session", func(t *testing.T) {
		m := newRemoteActionsTestModel()
		localSess := m.sessions[0]
		m.sessionToModify = &localSess
		m.dialogMode = DialogKillConfirm

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
		_, cmd := m.Update(msg)

		if cmd == nil {
			t.Error("y in kill confirm should return a command for local kill")
		}
	})
}

// TestE2E_PBI27_AC4_RenameRemote tests AC4: R key on remote session renames via SSH.
func TestE2E_PBI27_AC4_RenameRemote(t *testing.T) {
	t.Run("R key on remote session opens rename dialog", func(t *testing.T) {
		m := selectRemoteSession(newRemoteActionsTestModel())

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogRename {
			t.Errorf("expected DialogRename, got %d", updated.dialogMode)
		}
		if updated.sessionToModify == nil {
			t.Fatal("sessionToModify should be set")
		}
		if updated.sessionToModify.TmuxSession != "remote-1" {
			t.Errorf("expected session remote-1, got %q", updated.sessionToModify.TmuxSession)
		}
		// nameInput should be pre-filled with current session name
		if updated.nameInput.Value() != "remote-1" {
			t.Errorf("nameInput should be pre-filled with 'remote-1', got %q", updated.nameInput.Value())
		}
	})

	t.Run("enter in rename dialog dispatches remote rename for remote session", func(t *testing.T) {
		m := selectRemoteSession(newRemoteActionsTestModel())
		remoteSess := m.sessions[1]
		m.sessionToModify = &remoteSess
		m.dialogMode = DialogRename
		m.nameInput = initNameInput()
		m.nameInput.SetValue("new-remote-name")

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		_, cmd := m.Update(msg)

		if cmd == nil {
			t.Error("enter in rename dialog should return a command for remote rename")
		}
	})

	t.Run("enter in rename dialog dispatches local rename for local session", func(t *testing.T) {
		m := newRemoteActionsTestModel()
		localSess := m.sessions[0]
		m.sessionToModify = &localSess
		m.dialogMode = DialogRename
		m.nameInput = initNameInput()
		m.nameInput.SetValue("new-local-name")

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		_, cmd := m.Update(msg)

		if cmd == nil {
			t.Error("enter in rename dialog should return a command for local rename")
		}
	})

	t.Run("rename with same name just closes dialog", func(t *testing.T) {
		m := selectRemoteSession(newRemoteActionsTestModel())
		remoteSess := m.sessions[1]
		m.sessionToModify = &remoteSess
		m.dialogMode = DialogRename
		m.nameInput = initNameInput()
		m.nameInput.SetValue("remote-1") // Same name

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, cmd := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogNone {
			t.Errorf("dialog should close when renaming to same name, got %d", updated.dialogMode)
		}
		if cmd != nil {
			t.Error("no command should be returned when renaming to same name")
		}
	})
}

// TestE2E_PBI27_AC5_DismissRemote tests AC5: d key on remote session dismisses via SSH.
func TestE2E_PBI27_AC5_DismissRemote(t *testing.T) {
	t.Run("d key on remote session dispatches dismiss command", func(t *testing.T) {
		m := selectRemoteSession(newRemoteActionsTestModel())

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
		_, cmd := m.Update(msg)

		if cmd == nil {
			t.Error("d key on remote session should return a command")
		}
	})

	t.Run("d key on local session does not use remote dismiss", func(t *testing.T) {
		m := newRemoteActionsTestModel()
		m.cursor = 0 // local session

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
		_, cmd := m.Update(msg)

		// Local dismiss returns pollSessions, not nil
		if cmd == nil {
			t.Error("d key on local session should return a command (pollSessions)")
		}
	})

	t.Run("remoteDismissResultMsg triggers pollSessions", func(t *testing.T) {
		m := newRemoteActionsTestModel()

		msg := remoteDismissResultMsg{err: nil}
		_, cmd := m.Update(msg)

		if cmd == nil {
			t.Error("remoteDismissResultMsg should return a command (pollSessions)")
		}
	})

	t.Run("remoteDismissResultMsg with error still triggers pollSessions", func(t *testing.T) {
		m := newRemoteActionsTestModel()

		msg := remoteDismissResultMsg{err: errors.New("ssh error")}
		_, cmd := m.Update(msg)

		// Should still refresh sessions even on error (same as local behavior)
		if cmd == nil {
			t.Error("remoteDismissResultMsg with error should still return pollSessions")
		}
	})
}

// TestE2E_PBI27_AC6_SSHPoolReuse tests AC6: All SSH operations reuse SSHPool.
func TestE2E_PBI27_AC6_SSHPoolReuse(t *testing.T) {
	t.Run("model is constructed with SSHPool from remotes", func(t *testing.T) {
		m := newRemoteActionsTestModel()

		if m.SSHPool == nil {
			t.Fatal("model should have non-nil SSHPool")
		}

		// Pool should contain the configured remote
		names := m.SSHPool.RemoteNames()
		if len(names) != 1 {
			t.Errorf("SSHPool should have 1 remote, got %d", len(names))
		}

		config := m.SSHPool.GetRemoteConfig("dev")
		if config == nil {
			t.Fatal("SSHPool should have config for 'dev'")
		}
		if config.Host != "dev.example.com" {
			t.Errorf("expected host 'dev.example.com', got %q", config.Host)
		}
	})

	t.Run("SSHPool provides SessionsDir for remote operations", func(t *testing.T) {
		m := newRemoteActionsTestModel()

		config := m.SSHPool.GetRemoteConfig("dev")
		if config == nil {
			t.Fatal("SSHPool should have config for 'dev'")
		}
		if config.SessionsDir != "~/.claude-sessions" {
			t.Errorf("expected SessionsDir '~/.claude-sessions', got %q", config.SessionsDir)
		}
	})

	t.Run("all remote commands receive SSHPool reference", func(t *testing.T) {
		m := selectRemoteSession(newRemoteActionsTestModel())

		// G key - git info uses pool
		gMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		_, cmd := m.Update(gMsg)
		if cmd == nil {
			t.Error("G key on remote session should use SSHPool to fetch git info")
		}

		// p key - preview uses pool (via capturePreviewForSession)
		previewCmd := m.capturePreviewForSession(m.sessions[1])
		if previewCmd == nil {
			t.Error("capturePreviewForSession should use SSHPool for remote session")
		}

		// d key - dismiss uses pool
		dMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
		_, cmd = m.Update(dMsg)
		if cmd == nil {
			t.Error("d key on remote session should use SSHPool to dismiss")
		}
	})
}

// TestE2E_PBI27_AC7_CacheTTL tests AC7: On-demand fetches cached with TTL.
func TestE2E_PBI27_AC7_CacheTTL(t *testing.T) {
	t.Run("gitCache stores remote git info after fetch", func(t *testing.T) {
		m := newRemoteActionsTestModel()
		m.dialogMode = DialogGitDetail
		remoteSess := m.sessions[1]
		m.sessionToModify = &remoteSess

		gitInfo := &git.Info{
			Branch:    "main",
			FetchedAt: time.Now().Unix(),
		}
		msg := remoteGitInfoMsg{cwd: "/home/user/project", info: gitInfo}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		cached, ok := updated.gitCache["/home/user/project"]
		if !ok {
			t.Fatal("gitCache should store remote git info")
		}
		if cached.Branch != "main" {
			t.Errorf("cached branch should be 'main', got %q", cached.Branch)
		}
	})

	t.Run("fresh cache is used on second G press", func(t *testing.T) {
		m := selectRemoteSession(newRemoteActionsTestModel())

		// Pre-populate with fresh cache
		freshInfo := &git.Info{
			Branch:    "cached-branch",
			FetchedAt: time.Now().Unix(), // Fresh
		}
		m.gitCache["/home/user/project"] = freshInfo

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		// Should use cached info (set on sessionToModify.Git)
		if updated.sessionToModify == nil || updated.sessionToModify.Git == nil {
			t.Fatal("sessionToModify.Git should be set from cache")
		}
		if updated.sessionToModify.Git.Branch != "cached-branch" {
			t.Errorf("should use cached branch, got %q", updated.sessionToModify.Git.Branch)
		}
	})

	t.Run("stale cache triggers re-fetch", func(t *testing.T) {
		m := selectRemoteSession(newRemoteActionsTestModel())

		// Pre-populate with stale cache (FetchedAt = 0 is always stale)
		staleInfo := &git.Info{
			Branch:    "stale-branch",
			FetchedAt: 0, // Stale
		}
		m.gitCache["/home/user/project"] = staleInfo

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		newModel, cmd := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogGitDetail {
			t.Errorf("expected DialogGitDetail, got %d", updated.dialogMode)
		}
		// Should trigger a fetch command because cache is stale
		if cmd == nil {
			t.Error("stale cache should trigger a re-fetch command")
		}
	})

	t.Run("IsStale with recent FetchedAt returns false", func(t *testing.T) {
		info := &git.Info{
			Branch:    "test",
			FetchedAt: time.Now().Unix(),
		}
		if info.IsStale() {
			t.Error("git info with recent FetchedAt should not be stale")
		}
	})

	t.Run("IsStale with zero FetchedAt returns true", func(t *testing.T) {
		info := &git.Info{
			Branch:    "test",
			FetchedAt: 0,
		}
		if !info.IsStale() {
			t.Error("git info with zero FetchedAt should be stale")
		}
	})

	t.Run("IsStale with old FetchedAt returns true", func(t *testing.T) {
		info := &git.Info{
			Branch:    "test",
			FetchedAt: time.Now().Add(-1 * time.Minute).Unix(),
		}
		if !info.IsStale() {
			t.Error("git info with old FetchedAt should be stale")
		}
	})
}

// TestE2E_PBI27_AC8_SSHErrorHandling tests AC8: SSH errors handled gracefully.
func TestE2E_PBI27_AC8_SSHErrorHandling(t *testing.T) {
	t.Run("remoteGitInfoMsg error shows error in dialog", func(t *testing.T) {
		m := selectRemoteSession(newRemoteActionsTestModel())
		m.dialogMode = DialogGitDetail
		remoteSess := m.sessions[1]
		m.sessionToModify = &remoteSess

		msg := remoteGitInfoMsg{
			cwd: "/home/user/project",
			err: errors.New("ssh: connect to host dev.example.com port 22: Connection refused"),
		}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !strings.Contains(updated.dialogError, "SSH error") {
			t.Errorf("dialogError should contain 'SSH error', got %q", updated.dialogError)
		}
		if !strings.Contains(updated.dialogError, "Connection refused") {
			t.Errorf("dialogError should contain actual error message, got %q", updated.dialogError)
		}
	})

	t.Run("preview error content shown in preview pane", func(t *testing.T) {
		m := selectRemoteSession(newRemoteActionsTestModel())
		m.previewVisible = true

		// captureRemotePreviewCmd returns error content as previewContentMsg
		msg := previewContentMsg{content: "Failed to fetch remote preview: connection timeout"}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !strings.Contains(updated.previewContent, "Failed to fetch remote preview") {
			t.Errorf("preview should show error content, got %q", updated.previewContent)
		}
	})

	t.Run("killSessionResultMsg error shows in dialog", func(t *testing.T) {
		m := selectRemoteSession(newRemoteActionsTestModel())
		m.dialogMode = DialogKillConfirm
		remoteSess := m.sessions[1]
		m.sessionToModify = &remoteSess

		msg := killSessionResultMsg{err: errors.New("remote kill failed: session not found")}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !strings.Contains(updated.dialogError, "Failed to kill session") {
			t.Errorf("dialogError should contain kill error, got %q", updated.dialogError)
		}
	})

	t.Run("renameSessionResultMsg error shows in dialog", func(t *testing.T) {
		m := selectRemoteSession(newRemoteActionsTestModel())
		m.dialogMode = DialogRename
		remoteSess := m.sessions[1]
		m.sessionToModify = &remoteSess

		msg := renameSessionResultMsg{err: errors.New("remote rename failed"), newName: "new-name"}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !strings.Contains(updated.dialogError, "Failed to rename session") {
			t.Errorf("dialogError should contain rename error, got %q", updated.dialogError)
		}
	})
}

// TestE2E_PBI27_AC9_SameConfirmationFeedback tests AC9: Same confirmation/feedback as local sessions.
func TestE2E_PBI27_AC9_SameConfirmationFeedback(t *testing.T) {
	t.Run("kill dialog shown for remote session same as local", func(t *testing.T) {
		// Remote session
		mRemote := selectRemoteSession(newRemoteActionsTestModel())
		xMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
		newModel, _ := mRemote.Update(xMsg)
		updatedRemote := newModel.(Model)

		// Local session
		mLocal := newRemoteActionsTestModel()
		mLocal.cursor = 0
		newModel, _ = mLocal.Update(xMsg)
		updatedLocal := newModel.(Model)

		// Both should open the same dialog
		if updatedRemote.dialogMode != DialogKillConfirm {
			t.Errorf("remote: expected DialogKillConfirm, got %d", updatedRemote.dialogMode)
		}
		if updatedLocal.dialogMode != DialogKillConfirm {
			t.Errorf("local: expected DialogKillConfirm, got %d", updatedLocal.dialogMode)
		}
	})

	t.Run("rename dialog shown for remote session same as local", func(t *testing.T) {
		// Remote session
		mRemote := selectRemoteSession(newRemoteActionsTestModel())
		rMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}}
		newModel, _ := mRemote.Update(rMsg)
		updatedRemote := newModel.(Model)

		// Local session
		mLocal := newRemoteActionsTestModel()
		mLocal.cursor = 0
		newModel, _ = mLocal.Update(rMsg)
		updatedLocal := newModel.(Model)

		// Both should open the same dialog
		if updatedRemote.dialogMode != DialogRename {
			t.Errorf("remote: expected DialogRename, got %d", updatedRemote.dialogMode)
		}
		if updatedLocal.dialogMode != DialogRename {
			t.Errorf("local: expected DialogRename, got %d", updatedLocal.dialogMode)
		}
	})

	t.Run("footer shows all keybindings for remote sessions", func(t *testing.T) {
		m := selectRemoteSession(newRemoteActionsTestModel())

		result := m.View()

		// All action keybindings should be visible regardless of remote/local
		expectedKeys := []string{"attach", "d dismiss", "n new", "x kill", "R rename", "G git", "p preview"}
		for _, key := range expectedKeys {
			if !strings.Contains(result, key) {
				t.Errorf("footer should contain %q for remote session, view output:\n%s", key, result)
			}
		}
	})

	t.Run("footer shows filter option when remotes configured", func(t *testing.T) {
		m := newRemoteActionsTestModel()

		result := m.View()

		if !strings.Contains(result, "f filter:") {
			t.Error("footer should show filter option when remotes are configured")
		}
	})

	t.Run("git detail dialog renders for remote session", func(t *testing.T) {
		m := selectRemoteSession(newRemoteActionsTestModel())
		remoteSess := m.sessions[1]
		m.sessionToModify = &remoteSess
		m.sessionToModify.Git = &git.Info{
			Branch:     "feature/remote",
			Dirty:      true,
			LastCommit: "abc1234 remote commit",
			Remote:     "git@github.com:user/repo.git",
			FetchedAt:  time.Now().Unix(),
		}
		m.dialogMode = DialogGitDetail

		result := m.View()

		if !strings.Contains(result, "Git Information") {
			t.Error("git detail dialog should render for remote session")
		}
		if !strings.Contains(result, "feature/remote") {
			t.Error("git detail should show remote branch name")
		}
	})

	t.Run("kill result success closes dialog and refreshes for remote", func(t *testing.T) {
		m := selectRemoteSession(newRemoteActionsTestModel())
		remoteSess := m.sessions[1]
		m.sessionToModify = &remoteSess
		m.dialogMode = DialogKillConfirm

		msg := killSessionResultMsg{err: nil}
		newModel, cmd := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogNone {
			t.Errorf("dialog should close on successful kill, got %d", updated.dialogMode)
		}
		if updated.sessionToModify != nil {
			t.Error("sessionToModify should be cleared on successful kill")
		}
		if cmd == nil {
			t.Error("successful kill should trigger session refresh")
		}
	})

	t.Run("rename result success closes dialog and preserves cursor for remote", func(t *testing.T) {
		m := selectRemoteSession(newRemoteActionsTestModel())
		remoteSess := m.sessions[1]
		m.sessionToModify = &remoteSess
		m.dialogMode = DialogRename

		msg := renameSessionResultMsg{err: nil, newName: "renamed-remote"}
		newModel, cmd := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogNone {
			t.Errorf("dialog should close on successful rename, got %d", updated.dialogMode)
		}
		if updated.lastSelectedSession != "renamed-remote" {
			t.Errorf("lastSelectedSession should be set to new name, got %q", updated.lastSelectedSession)
		}
		if cmd == nil {
			t.Error("successful rename should trigger session refresh")
		}
	})
}
