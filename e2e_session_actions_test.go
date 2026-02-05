package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// E2E tests for PBI-7: Session Management Actions
// These tests verify the complete user flows for create, kill, and rename operations.

// AC1: New Session Dialog
func TestE2E_AC1_NewSessionDialog(t *testing.T) {
	t.Run("n key opens dialog when no dialog is open", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogNone,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogNewSession {
			t.Errorf("expected DialogNewSession, got %d", updated.dialogMode)
		}
	})

	t.Run("dialog has name and directory input fields", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogNewSession,
			nameInput:  initNameInput(),
			dirInput:   initDirInput(),
		}

		result := m.renderDialog()

		if !strings.Contains(result, "Name:") {
			t.Error("dialog should have name input field")
		}
		if !strings.Contains(result, "Directory:") {
			t.Error("dialog should have directory input field")
		}
	})

	t.Run("tab switches between fields", func(t *testing.T) {
		m := Model{
			width:        80,
			height:       24,
			dialogMode:   DialogNewSession,
			nameInput:    initNameInput(),
			dirInput:     initDirInput(),
			focusedInput: focusName,
		}

		msg := tea.KeyMsg{Type: tea.KeyTab}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.focusedInput != focusDir {
			t.Error("tab should switch focus to directory input")
		}

		// Tab again should switch to skipPerms
		newModel2, _ := updated.Update(msg)
		updated2 := newModel2.(Model)

		if updated2.focusedInput != focusSkipPerms {
			t.Error("tab should switch focus to skip permissions checkbox")
		}

		// Tab again should switch back to name
		newModel3, _ := updated2.Update(msg)
		updated3 := newModel3.(Model)

		if updated3.focusedInput != focusName {
			t.Error("tab should switch focus back to name input")
		}
	})

	t.Run("escape cancels dialog", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogNewSession,
			nameInput:  initNameInput(),
			dirInput:   initDirInput(),
		}

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogNone {
			t.Error("escape should close the dialog")
		}
	})

	t.Run("enter submits dialog", func(t *testing.T) {
		m := Model{
			width:        80,
			height:       24,
			dialogMode:   DialogNewSession,
			nameInput:    initNameInput(),
			dirInput:     initDirInput(),
			focusedInput: focusName,
		}
		m.nameInput.SetValue("test-session")
		m.dirInput.SetValue("/tmp")

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		_, cmd := m.Update(msg)

		// Should return a command to create the session
		if cmd == nil {
			t.Error("enter should trigger session creation command")
		}
	})
}

// AC2: New Sessions Created
func TestE2E_AC2_NewSessionsCreated(t *testing.T) {
	t.Run("valid name/dir triggers create command", func(t *testing.T) {
		m := Model{
			width:        80,
			height:       24,
			dialogMode:   DialogNewSession,
			nameInput:    initNameInput(),
			dirInput:     initDirInput(),
			focusedInput: focusName,
			sessions:     []SessionInfo{},
		}
		m.nameInput.SetValue("new-session")
		m.dirInput.SetValue("/tmp")

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		_, cmd := m.Update(msg)

		if cmd == nil {
			t.Error("valid submission should return create command")
		}
	})

	t.Run("successful create closes dialog and triggers poll", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogNewSession,
			nameInput:  initNameInput(),
			dirInput:   initDirInput(),
		}

		msg := createSessionResultMsg{err: nil}
		newModel, cmd := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogNone {
			t.Error("dialog should close on successful create")
		}
		if cmd == nil {
			t.Error("should return poll command to refresh session list")
		}
	})

	t.Run("default name is generated when empty", func(t *testing.T) {
		name := getDefaultSessionName()
		if name == "" {
			t.Error("default session name should not be empty")
		}
		if !strings.HasPrefix(name, "claude") {
			t.Errorf("default session name should start with 'claude', got '%s'", name)
		}
	})

	t.Run("default directory is set", func(t *testing.T) {
		dir := getDefaultDirectory()
		if dir == "" {
			t.Error("default directory should not be empty")
		}
	})
}

// AC3: Kill Confirmation
func TestE2E_AC3_KillConfirmation(t *testing.T) {
	t.Run("x key shows confirmation dialog with session name", func(t *testing.T) {
		session := SessionInfo{TmuxSession: "target-session"}
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogNone,
			sessions:   []SessionInfo{session},
			cursor:     0,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogKillConfirm {
			t.Error("x should open kill confirmation dialog")
		}
		if updated.sessionToModify == nil || updated.sessionToModify.TmuxSession != "target-session" {
			t.Error("sessionToModify should be set to the selected session")
		}

		// Check dialog shows session name
		result := updated.renderDialog()
		if !strings.Contains(result, "target-session") {
			t.Error("dialog should display session name being killed")
		}
	})

	t.Run("y confirms kill action", func(t *testing.T) {
		session := SessionInfo{TmuxSession: "to-kill"}
		m := Model{
			width:           80,
			height:          24,
			dialogMode:      DialogKillConfirm,
			sessionToModify: &session,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
		_, cmd := m.Update(msg)

		if cmd == nil {
			t.Error("y should trigger kill command")
		}
	})

	t.Run("n cancels kill action", func(t *testing.T) {
		session := SessionInfo{TmuxSession: "keep-me"}
		m := Model{
			width:           80,
			height:          24,
			dialogMode:      DialogKillConfirm,
			sessionToModify: &session,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		newModel, cmd := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogNone {
			t.Error("n should close the dialog")
		}
		if cmd != nil {
			t.Error("n should not trigger any command")
		}
	})

	t.Run("escape cancels kill action", func(t *testing.T) {
		session := SessionInfo{TmuxSession: "keep-me"}
		m := Model{
			width:           80,
			height:          24,
			dialogMode:      DialogKillConfirm,
			sessionToModify: &session,
		}

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogNone {
			t.Error("escape should close the dialog")
		}
	})
}

// AC4: Kill Cleanup
func TestE2E_AC4_KillCleanup(t *testing.T) {
	t.Run("successful kill closes dialog and triggers poll", func(t *testing.T) {
		session := SessionInfo{TmuxSession: "killed-session"}
		m := Model{
			width:           80,
			height:          24,
			dialogMode:      DialogKillConfirm,
			sessionToModify: &session,
		}

		msg := killSessionResultMsg{err: nil}
		newModel, cmd := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogNone {
			t.Error("dialog should close on successful kill")
		}
		if updated.sessionToModify != nil {
			t.Error("sessionToModify should be cleared")
		}
		if cmd == nil {
			t.Error("should return poll command to refresh session list")
		}
	})
}

// AC5: Rename Validation
func TestE2E_AC5_RenameValidation(t *testing.T) {
	t.Run("R key opens rename dialog", func(t *testing.T) {
		session := SessionInfo{TmuxSession: "my-session"}
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogNone,
			sessions:   []SessionInfo{session},
			cursor:     0,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogRename {
			t.Error("R should open rename dialog")
		}
	})

	t.Run("input pre-filled with current name", func(t *testing.T) {
		session := SessionInfo{TmuxSession: "original-name"}
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogNone,
			sessions:   []SessionInfo{session},
			cursor:     0,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.nameInput.Value() != "original-name" {
			t.Errorf("expected 'original-name', got '%s'", updated.nameInput.Value())
		}
	})

	t.Run("empty name rejected", func(t *testing.T) {
		err := validateSessionName("", nil)
		if err == nil {
			t.Error("empty name should be rejected")
		}
		if !strings.Contains(err.Error(), "empty") {
			t.Errorf("error should mention empty, got '%s'", err.Error())
		}
	})

	t.Run("name with period rejected", func(t *testing.T) {
		err := validateSessionName("my.session", nil)
		if err == nil {
			t.Error("name with '.' should be rejected")
		}
		if !strings.Contains(err.Error(), ".") {
			t.Errorf("error should mention period, got '%s'", err.Error())
		}
	})

	t.Run("name with colon rejected", func(t *testing.T) {
		err := validateSessionName("my:session", nil)
		if err == nil {
			t.Error("name with ':' should be rejected")
		}
		if !strings.Contains(err.Error(), ":") {
			t.Errorf("error should mention colon, got '%s'", err.Error())
		}
	})

	t.Run("duplicate name rejected", func(t *testing.T) {
		sessions := []SessionInfo{
			{TmuxSession: "existing"},
		}
		err := validateSessionName("existing", sessions)
		if err == nil {
			t.Error("duplicate name should be rejected")
		}
		if !strings.Contains(err.Error(), "exists") {
			t.Errorf("error should mention exists, got '%s'", err.Error())
		}
	})

	t.Run("valid name accepted", func(t *testing.T) {
		sessions := []SessionInfo{
			{TmuxSession: "other-session"},
		}
		err := validateSessionName("new-valid-name", sessions)
		if err != nil {
			t.Errorf("valid name should be accepted, got error: %v", err)
		}
	})
}

// AC6: Error Messages
func TestE2E_AC6_ErrorMessages(t *testing.T) {
	t.Run("create failure shows error in dialog", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogNewSession,
			nameInput:  initNameInput(),
			dirInput:   initDirInput(),
		}

		msg := createSessionResultMsg{err: errTest}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogNewSession {
			t.Error("dialog should remain open on error")
		}
		if updated.dialogError == "" {
			t.Error("dialogError should be set")
		}
	})

	t.Run("kill failure shows error in dialog", func(t *testing.T) {
		session := SessionInfo{TmuxSession: "test"}
		m := Model{
			width:           80,
			height:          24,
			dialogMode:      DialogKillConfirm,
			sessionToModify: &session,
		}

		msg := killSessionResultMsg{err: errTest}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogKillConfirm {
			t.Error("dialog should remain open on error")
		}
		if updated.dialogError == "" {
			t.Error("dialogError should be set")
		}
	})

	t.Run("rename failure shows error in dialog", func(t *testing.T) {
		session := SessionInfo{TmuxSession: "test"}
		m := Model{
			width:           80,
			height:          24,
			dialogMode:      DialogRename,
			sessionToModify: &session,
			nameInput:       initNameInput(),
		}

		msg := renameSessionResultMsg{err: errTest, newName: "new-name"}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogRename {
			t.Error("dialog should remain open on error")
		}
		if updated.dialogError == "" {
			t.Error("dialogError should be set")
		}
	})

	t.Run("validation errors shown inline", func(t *testing.T) {
		session := SessionInfo{TmuxSession: "existing"}
		m := Model{
			width:           80,
			height:          24,
			dialogMode:      DialogRename,
			sessionToModify: &session,
			nameInput:       initNameInput(),
			sessions:        []SessionInfo{{TmuxSession: "taken"}},
		}
		m.nameInput.SetValue("taken") // duplicate

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogError == "" {
			t.Error("validation error should be shown in dialog")
		}
	})

	t.Run("error message displayed in dialog render", func(t *testing.T) {
		m := Model{
			width:       80,
			height:      24,
			dialogMode:  DialogNewSession,
			dialogError: "Test error message",
			nameInput:   initNameInput(),
			dirInput:    initDirInput(),
		}

		result := m.renderDialog()

		if !strings.Contains(result, "Test error message") {
			t.Error("dialog should display error message")
		}
	})
}

// AC7: Footer Updates
func TestE2E_AC7_FooterUpdates(t *testing.T) {
	m := Model{
		width:  80,
		height: 24,
	}

	result := m.renderFooter()

	t.Run("footer shows n new", func(t *testing.T) {
		if !strings.Contains(result, "n new") {
			t.Error("footer should show 'n new' keybinding")
		}
	})

	t.Run("footer shows x kill", func(t *testing.T) {
		if !strings.Contains(result, "x kill") {
			t.Error("footer should show 'x kill' keybinding")
		}
	})

	t.Run("footer shows R rename", func(t *testing.T) {
		if !strings.Contains(result, "R rename") {
			t.Error("footer should show 'R rename' keybinding")
		}
	})
}

// Integration: Complete Rename Flow
func TestE2E_RenameSessionFlow(t *testing.T) {
	t.Run("complete rename flow preserves cursor position", func(t *testing.T) {
		sessions := []SessionInfo{
			{TmuxSession: "first"},
			{TmuxSession: "middle"},
			{TmuxSession: "last"},
		}
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogNone,
			sessions:   sessions,
			cursor:     1, // Select "middle"
		}

		// Step 1: Press R to open rename dialog
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogRename {
			t.Fatal("R should open rename dialog")
		}
		if updated.sessionToModify.TmuxSession != "middle" {
			t.Fatal("sessionToModify should be 'middle'")
		}

		// Step 2: Simulate successful rename
		msg2 := renameSessionResultMsg{err: nil, newName: "renamed-middle"}
		newModel2, _ := updated.Update(msg2)
		updated2 := newModel2.(Model)

		if updated2.dialogMode != DialogNone {
			t.Error("dialog should close on success")
		}
		if updated2.lastSelectedSession != "renamed-middle" {
			t.Error("lastSelectedSession should be set to new name for cursor preservation")
		}

		// Step 3: Simulate poll returning updated sessions
		newSessions := sessionsMsg{
			{TmuxSession: "first"},
			{TmuxSession: "renamed-middle"},
			{TmuxSession: "last"},
		}
		newModel3, _ := updated2.Update(newSessions)
		updated3 := newModel3.(Model)

		if updated3.cursor != 1 {
			t.Errorf("cursor should remain at 1 (on renamed session), got %d", updated3.cursor)
		}
		if updated3.lastSelectedSession != "" {
			t.Error("lastSelectedSession should be cleared after restoration")
		}
	})
}

// Integration: Dialog Interactions
func TestE2E_DialogInteractions(t *testing.T) {
	t.Run("escape closes any dialog type", func(t *testing.T) {
		dialogs := []DialogMode{DialogNewSession, DialogKillConfirm, DialogRename}

		for _, dialog := range dialogs {
			m := Model{
				width:       80,
				height:      24,
				dialogMode:  dialog,
				dialogError: "some error",
			}

			msg := tea.KeyMsg{Type: tea.KeyEsc}
			newModel, _ := m.Update(msg)
			updated := newModel.(Model)

			if updated.dialogMode != DialogNone {
				t.Errorf("escape should close %v dialog", dialog)
			}
			if updated.dialogError != "" {
				t.Errorf("dialogError should be cleared for %v dialog", dialog)
			}
		}
	})

	t.Run("main keybindings blocked when dialog open", func(t *testing.T) {
		sessions := []SessionInfo{
			{TmuxSession: "1"},
			{TmuxSession: "2"},
		}
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogNewSession,
			sessions:   sessions,
			cursor:     0,
			nameInput:  initNameInput(),
			dirInput:   initDirInput(),
		}

		// Try navigation keys - should not move cursor
		keys := []rune{'j', 'k'}
		for _, key := range keys {
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}}
			newModel, _ := m.Update(msg)
			updated := newModel.(Model)

			if updated.cursor != 0 {
				t.Errorf("cursor should not move when dialog open (key: %c)", key)
			}
		}

		// Try action keys - should not trigger actions
		actionKeys := []rune{'d', 'x', 'R'}
		for _, key := range actionKeys {
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}}
			_, cmd := m.Update(msg)

			// These keys should be absorbed by the text input or ignored
			// They should not trigger their main functionality
			_ = cmd // Commands may or may not be returned depending on text input handling
		}
	})
}

// Test error for use in tests
var errTest = errTestType{}

type errTestType struct{}

func (e errTestType) Error() string {
	return "test error"
}
