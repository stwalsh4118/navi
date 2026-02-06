package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/session"
)

func TestDialogModeInitialState(t *testing.T) {
	m := Model{
		width:  80,
		height: 24,
	}

	if m.dialogMode != DialogNone {
		t.Errorf("dialogMode should be DialogNone initially, got %d", m.dialogMode)
	}
}

func TestDialogBlocksMainKeybindings(t *testing.T) {
	t.Run("navigation blocked when dialog open", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogNewSession,
			sessions: []session.Info{
				{TmuxSession: "1"},
				{TmuxSession: "2"},
			},
			cursor: 0,
		}

		// Try to navigate with j key
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.cursor != 0 {
			t.Errorf("cursor should remain at 0 when dialog is open, got %d", updated.cursor)
		}
	})

	t.Run("navigation works when no dialog", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogNone,
			sessions: []session.Info{
				{TmuxSession: "1"},
				{TmuxSession: "2"},
			},
			cursor: 0,
		}

		// Navigate with j key
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.cursor != 1 {
			t.Errorf("cursor should move to 1 when no dialog, got %d", updated.cursor)
		}
	})

	t.Run("dismiss key blocked when dialog open", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogKillConfirm,
			sessions: []session.Info{
				{TmuxSession: "test"},
			},
			cursor: 0,
		}

		// Try to dismiss with d key
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
		_, cmd := m.Update(msg)

		// d should not trigger poll when dialog is open
		if cmd != nil {
			t.Error("d key should not return command when dialog is open")
		}
	})
}

func TestEscapeClosesDialog(t *testing.T) {
	testCases := []struct {
		name       string
		dialogMode DialogMode
	}{
		{"closes new s dialog", DialogNewSession},
		{"closes kill confirm dialog", DialogKillConfirm},
		{"closes rename dialog", DialogRename},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := Model{
				width:       80,
				height:      24,
				dialogMode:  tc.dialogMode,
				dialogError: "some error",
			}

			msg := tea.KeyMsg{Type: tea.KeyEsc}
			newModel, _ := m.Update(msg)
			updated := newModel.(Model)

			if updated.dialogMode != DialogNone {
				t.Errorf("dialogMode should be DialogNone after Esc, got %d", updated.dialogMode)
			}
			if updated.dialogError != "" {
				t.Errorf("dialogError should be cleared after Esc, got '%s'", updated.dialogError)
			}
		})
	}
}

func TestDialogRender(t *testing.T) {
	t.Run("dialog renders when open", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogNewSession,
		}

		result := m.View()

		if !strings.Contains(result, "New Session") {
			t.Error("view should contain dialog title when dialog is open")
		}
	})

	t.Run("no dialog when DialogNone", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogNone,
		}

		result := m.renderDialog()

		if result != "" {
			t.Error("renderDialog should return empty string when DialogNone")
		}
	})

	t.Run("error message displays in dialog", func(t *testing.T) {
		m := Model{
			width:       80,
			height:      24,
			dialogMode:  DialogNewSession,
			dialogError: "test error message",
		}

		result := m.renderDialog()

		if !strings.Contains(result, "test error message") {
			t.Error("dialog should display error message")
		}
	})
}

func TestDialogTitle(t *testing.T) {
	testCases := []struct {
		mode     DialogMode
		expected string
	}{
		{DialogNone, ""},
		{DialogNewSession, "New Session"},
		{DialogKillConfirm, "Kill Session"},
		{DialogRename, "Rename Session"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := DialogTitle(tc.mode)
			if result != tc.expected {
				t.Errorf("DialogTitle(%d) = %s, want %s", tc.mode, result, tc.expected)
			}
		})
	}
}

func TestNewSessionKeyOpensDialog(t *testing.T) {
	t.Run("n opens new s dialog", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogNone,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogNewSession {
			t.Errorf("dialogMode should be DialogNewSession, got %d", updated.dialogMode)
		}
	})

	t.Run("n does nothing when new s dialog open", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogNewSession, // Already has a dialog open
			nameInput:  initNameInput(),
			dirInput:   initDirInput(),
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		// Should still be new s dialog (n just types into input)
		if updated.dialogMode != DialogNewSession {
			t.Errorf("dialogMode should remain DialogNewSession, got %d", updated.dialogMode)
		}
	})
}

func TestNewSessionDialogRender(t *testing.T) {
	m := Model{
		width:      80,
		height:     24,
		dialogMode: DialogNewSession,
		nameInput:  initNameInput(),
		dirInput:   initDirInput(),
	}

	result := m.renderDialog()

	if !strings.Contains(result, "New Session") {
		t.Error("dialog should contain 'New Session' title")
	}
	if !strings.Contains(result, "Name:") {
		t.Error("dialog should contain 'Name:' label")
	}
	if !strings.Contains(result, "Directory:") {
		t.Error("dialog should contain 'Directory:' label")
	}
	if !strings.Contains(result, "Tab: switch") {
		t.Error("dialog should contain help text")
	}
}

func TestTabSwitchesFocusInNewSessionDialog(t *testing.T) {
	m := Model{
		width:        80,
		height:       24,
		dialogMode:   DialogNewSession,
		nameInput:    initNameInput(),
		dirInput:     initDirInput(),
		focusedInput: focusName,
	}

	// First tab should switch to dir
	msg := tea.KeyMsg{Type: tea.KeyTab}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.focusedInput != focusDir {
		t.Errorf("focusedInput should be focusDir (1), got %d", updated.focusedInput)
	}

	// Second tab should switch to skipPerms
	newModel2, _ := updated.Update(msg)
	updated2 := newModel2.(Model)

	if updated2.focusedInput != focusSkipPerms {
		t.Errorf("focusedInput should be focusSkipPerms (2), got %d", updated2.focusedInput)
	}

	// Third tab should switch back to name
	newModel3, _ := updated2.Update(msg)
	updated3 := newModel3.(Model)

	if updated3.focusedInput != focusName {
		t.Errorf("focusedInput should be focusName (0), got %d", updated3.focusedInput)
	}
}

func TestSpaceTogglesSkipPermissions(t *testing.T) {
	m := Model{
		width:           80,
		height:          24,
		dialogMode:      DialogNewSession,
		nameInput:       initNameInput(),
		dirInput:        initDirInput(),
		focusedInput:    focusSkipPerms,
		skipPermissions: false,
	}

	// Space should toggle skipPermissions to true
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if !updated.skipPermissions {
		t.Error("space should toggle skipPermissions to true")
	}

	// Space again should toggle it back to false
	newModel2, _ := updated.Update(msg)
	updated2 := newModel2.(Model)

	if updated2.skipPermissions {
		t.Error("space should toggle skipPermissions back to false")
	}
}

func TestSpaceOnlyWorksOnSkipPermsField(t *testing.T) {
	m := Model{
		width:           80,
		height:          24,
		dialogMode:      DialogNewSession,
		nameInput:       initNameInput(),
		dirInput:        initDirInput(),
		focusedInput:    focusName, // Not on skipPerms field
		skipPermissions: false,
	}

	// Space should not toggle when focused on name field
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.skipPermissions {
		t.Error("space should not toggle skipPermissions when not focused on that field")
	}
}

func TestFooterShowsNewKey(t *testing.T) {
	m := Model{
		width:  80,
		height: 24,
	}

	result := m.renderFooter()

	if !strings.Contains(result, "n new") {
		t.Error("footer should contain 'n new' keybinding")
	}
}

func TestCreateSessionResultSuccess(t *testing.T) {
	m := Model{
		width:      80,
		height:     24,
		dialogMode: DialogNewSession,
		nameInput:  initNameInput(),
		dirInput:   initDirInput(),
	}

	// Simulate successful create
	msg := createSessionResultMsg{err: nil}
	newModel, cmd := m.Update(msg)
	updated := newModel.(Model)

	if updated.dialogMode != DialogNone {
		t.Error("dialog should be closed on success")
	}
	if cmd == nil {
		t.Error("should return poll command on success")
	}
}

func TestCreateSessionResultError(t *testing.T) {
	m := Model{
		width:      80,
		height:     24,
		dialogMode: DialogNewSession,
		nameInput:  initNameInput(),
		dirInput:   initDirInput(),
	}

	// Simulate failed create
	msg := createSessionResultMsg{err: errors.New("test error")}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.dialogMode != DialogNewSession {
		t.Error("dialog should remain open on error")
	}
	if !strings.Contains(updated.dialogError, "test error") {
		t.Errorf("dialogError should contain error message, got '%s'", updated.dialogError)
	}
}

// Kill s tests

func TestKillSessionKeyOpensDialog(t *testing.T) {
	t.Run("x opens kill confirmation dialog", func(t *testing.T) {
		s := session.Info{TmuxSession: "test-session"}
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogNone,
			sessions:   []session.Info{s},
			cursor:     0,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogKillConfirm {
			t.Errorf("dialogMode should be DialogKillConfirm, got %d", updated.dialogMode)
		}
		if updated.sessionToModify == nil {
			t.Error("sessionToModify should be set")
		} else if updated.sessionToModify.TmuxSession != "test-session" {
			t.Errorf("sessionToModify should be 'test-session', got '%s'", updated.sessionToModify.TmuxSession)
		}
	})

	t.Run("x does nothing with no sessions", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogNone,
			sessions:   []session.Info{},
			cursor:     0,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogNone {
			t.Errorf("dialogMode should remain DialogNone, got %d", updated.dialogMode)
		}
	})
}

func TestKillConfirmDialogRender(t *testing.T) {
	s := session.Info{TmuxSession: "my-session"}
	m := Model{
		width:           80,
		height:          24,
		dialogMode:      DialogKillConfirm,
		sessionToModify: &s,
	}

	result := m.renderDialog()

	if !strings.Contains(result, "Kill Session") {
		t.Error("dialog should contain 'Kill Session' title")
	}
	if !strings.Contains(result, "my-session") {
		t.Error("dialog should contain s name")
	}
	if !strings.Contains(result, "y: yes") {
		t.Error("dialog should contain 'y: yes' help")
	}
}

func TestKillConfirmYesKey(t *testing.T) {
	s := session.Info{TmuxSession: "test-session"}
	m := Model{
		width:           80,
		height:          24,
		dialogMode:      DialogKillConfirm,
		sessionToModify: &s,
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Error("y key should trigger kill command")
	}
}

func TestKillConfirmNoKey(t *testing.T) {
	s := session.Info{TmuxSession: "test-session"}
	m := Model{
		width:           80,
		height:          24,
		dialogMode:      DialogKillConfirm,
		sessionToModify: &s,
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.dialogMode != DialogNone {
		t.Errorf("n key should close dialog, got dialogMode %d", updated.dialogMode)
	}
}

func TestKillSessionResultSuccess(t *testing.T) {
	s := session.Info{TmuxSession: "test-session"}
	m := Model{
		width:           80,
		height:          24,
		dialogMode:      DialogKillConfirm,
		sessionToModify: &s,
	}

	msg := killSessionResultMsg{err: nil}
	newModel, cmd := m.Update(msg)
	updated := newModel.(Model)

	if updated.dialogMode != DialogNone {
		t.Error("dialog should be closed on success")
	}
	if updated.sessionToModify != nil {
		t.Error("sessionToModify should be cleared")
	}
	if cmd == nil {
		t.Error("should return poll command on success")
	}
}

func TestKillSessionResultError(t *testing.T) {
	s := session.Info{TmuxSession: "test-session"}
	m := Model{
		width:           80,
		height:          24,
		dialogMode:      DialogKillConfirm,
		sessionToModify: &s,
	}

	msg := killSessionResultMsg{err: errors.New("kill failed")}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.dialogMode != DialogKillConfirm {
		t.Error("dialog should remain open on error")
	}
	if !strings.Contains(updated.dialogError, "kill failed") {
		t.Errorf("dialogError should contain error message, got '%s'", updated.dialogError)
	}
}

func TestFooterShowsKillKey(t *testing.T) {
	m := Model{
		width:  80,
		height: 24,
	}

	result := m.renderFooter()

	if !strings.Contains(result, "x kill") {
		t.Error("footer should contain 'x kill' keybinding")
	}
}

// Rename s tests

func TestRenameSessionKeyOpensDialog(t *testing.T) {
	t.Run("R opens rename dialog", func(t *testing.T) {
		s := session.Info{TmuxSession: "test-session"}
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogNone,
			sessions:   []session.Info{s},
			cursor:     0,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogRename {
			t.Errorf("dialogMode should be DialogRename, got %d", updated.dialogMode)
		}
		if updated.sessionToModify == nil {
			t.Error("sessionToModify should be set")
		} else if updated.sessionToModify.TmuxSession != "test-session" {
			t.Errorf("sessionToModify should be 'test-session', got '%s'", updated.sessionToModify.TmuxSession)
		}
	})

	t.Run("R does nothing with no sessions", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogNone,
			sessions:   []session.Info{},
			cursor:     0,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogNone {
			t.Errorf("dialogMode should remain DialogNone, got %d", updated.dialogMode)
		}
	})

	t.Run("R pre-fills name input with current s name", func(t *testing.T) {
		s := session.Info{TmuxSession: "my-session"}
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogNone,
			sessions:   []session.Info{s},
			cursor:     0,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.nameInput.Value() != "my-session" {
			t.Errorf("nameInput should be pre-filled with 'my-session', got '%s'", updated.nameInput.Value())
		}
	})
}

func TestRenameDialogRender(t *testing.T) {
	s := session.Info{TmuxSession: "old-name"}
	m := Model{
		width:           80,
		height:          24,
		dialogMode:      DialogRename,
		sessionToModify: &s,
		nameInput:       initNameInput(),
	}
	m.nameInput.SetValue("old-name")

	result := m.renderDialog()

	if !strings.Contains(result, "Rename Session") {
		t.Error("dialog should contain 'Rename Session' title")
	}
	if !strings.Contains(result, "Current:") {
		t.Error("dialog should contain 'Current:' label")
	}
	if !strings.Contains(result, "old-name") {
		t.Error("dialog should show current s name")
	}
	if !strings.Contains(result, "New name:") {
		t.Error("dialog should contain 'New name:' label")
	}
	if !strings.Contains(result, "Enter: rename") {
		t.Error("dialog should contain help text")
	}
}

func TestRenameSessionEnterSameName(t *testing.T) {
	s := session.Info{TmuxSession: "same-name"}
	m := Model{
		width:           80,
		height:          24,
		dialogMode:      DialogRename,
		sessionToModify: &s,
		nameInput:       initNameInput(),
		sessions:        []session.Info{s},
	}
	m.nameInput.SetValue("same-name")

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := m.Update(msg)
	updated := newModel.(Model)

	// Should close dialog without executing command
	if updated.dialogMode != DialogNone {
		t.Error("dialog should close when name unchanged")
	}
	if cmd != nil {
		t.Error("no command should be issued when name unchanged")
	}
}

func TestRenameSessionResultSuccess(t *testing.T) {
	s := session.Info{TmuxSession: "old-name"}
	m := Model{
		width:           80,
		height:          24,
		dialogMode:      DialogRename,
		sessionToModify: &s,
	}

	msg := renameSessionResultMsg{err: nil, newName: "new-name"}
	newModel, cmd := m.Update(msg)
	updated := newModel.(Model)

	if updated.dialogMode != DialogNone {
		t.Error("dialog should be closed on success")
	}
	if updated.sessionToModify != nil {
		t.Error("sessionToModify should be cleared")
	}
	if updated.lastSelectedSession != "new-name" {
		t.Errorf("lastSelectedSession should be 'new-name', got '%s'", updated.lastSelectedSession)
	}
	if cmd == nil {
		t.Error("should return poll command on success")
	}
}

func TestRenameSessionResultError(t *testing.T) {
	s := session.Info{TmuxSession: "test-session"}
	m := Model{
		width:           80,
		height:          24,
		dialogMode:      DialogRename,
		sessionToModify: &s,
	}

	msg := renameSessionResultMsg{err: errors.New("rename failed"), newName: "new-name"}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.dialogMode != DialogRename {
		t.Error("dialog should remain open on error")
	}
	if !strings.Contains(updated.dialogError, "rename failed") {
		t.Errorf("dialogError should contain error message, got '%s'", updated.dialogError)
	}
}

func TestFooterShowsRenameKey(t *testing.T) {
	// Use wider terminal to fit all keybindings
	m := Model{
		width:  120,
		height: 24,
	}

	result := m.renderFooter()

	if !strings.Contains(result, "R rename") {
		t.Error("footer should contain 'R rename' keybinding")
	}
}
