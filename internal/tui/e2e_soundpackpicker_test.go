package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/audio"
	"github.com/stwalsh4118/navi/internal/session"
)

// newSoundPackPickerTestModel creates a model for sound pack picker testing.
func newSoundPackPickerTestModel(packs []audio.PackInfo, activePack string) Model {
	cfg := audio.DefaultConfig()
	cfg.Pack = activePack
	notifier := audio.NewNotifier(cfg)

	return Model{
		sessions:          []session.Info{{TmuxSession: "test", Status: "working"}},
		width:             80,
		height:            24,
		audioNotifier:     notifier,
		activeSoundPack:   activePack,
		soundPacks:        packs,
		lastSessionStates: make(map[string]string),
	}
}

// AC1: Pressing S in session view opens a scrollable pack picker overlay
func TestSoundPackPickerOpenWithS(t *testing.T) {
	m := newSoundPackPickerTestModel(nil, "")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	model := updated.(Model)

	if model.dialogMode != DialogSoundPackPicker {
		t.Fatalf("expected DialogSoundPackPicker, got %d", model.dialogMode)
	}
	if model.soundPackCursor != 0 {
		t.Fatalf("expected cursor reset to 0, got %d", model.soundPackCursor)
	}
	if model.soundPackScrollOffset != 0 {
		t.Fatalf("expected scroll reset to 0, got %d", model.soundPackScrollOffset)
	}
	if cmd == nil {
		t.Fatalf("expected a command to load packs")
	}
}

// AC2: Each pack shows name, event count, and file count
func TestSoundPackPickerShowsPackInfo(t *testing.T) {
	packs := []audio.PackInfo{
		{Name: "starcraft", EventCount: 3, FileCount: 7},
		{Name: "zelda", EventCount: 2, FileCount: 4},
	}
	m := newSoundPackPickerTestModel(packs, "")
	m.dialogMode = DialogSoundPackPicker

	view := m.View()

	if !strings.Contains(view, "starcraft") {
		t.Fatalf("expected pack name 'starcraft' in view")
	}
	if !strings.Contains(view, "3 events") {
		t.Fatalf("expected event count in view")
	}
	if !strings.Contains(view, "7 files") {
		t.Fatalf("expected file count in view")
	}
	if !strings.Contains(view, "zelda") {
		t.Fatalf("expected pack name 'zelda' in view")
	}
	if !strings.Contains(view, "2 events") {
		t.Fatalf("expected event count for zelda")
	}
}

// AC3: The currently active pack is visually marked
func TestSoundPackPickerActiveMarker(t *testing.T) {
	packs := []audio.PackInfo{
		{Name: "starcraft", EventCount: 3, FileCount: 7},
		{Name: "zelda", EventCount: 2, FileCount: 4},
	}
	m := newSoundPackPickerTestModel(packs, "zelda")
	m.dialogMode = DialogSoundPackPicker

	view := m.View()

	if !strings.Contains(view, "✓") {
		t.Fatalf("expected checkmark for active pack in view")
	}
}

// AC4: Selecting a pack with Enter hot-swaps it on the running Notifier
func TestSoundPackPickerEnterSelectsPack(t *testing.T) {
	tmpDir := t.TempDir()
	packDir := filepath.Join(tmpDir, "testpack")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packDir, "done.wav"), []byte("fake"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Override pack base dir
	audio.SetPackBaseDirForTest(tmpDir)
	defer audio.ResetPackBaseDirForTest()

	packs := []audio.PackInfo{
		{Name: "testpack", EventCount: 1, FileCount: 1},
	}

	// Create a temp config path for SavePackSelection
	configPath := filepath.Join(tmpDir, "sounds.yaml")
	if err := os.WriteFile(configPath, []byte("enabled: true\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	m := newSoundPackPickerTestModel(packs, "")
	m.dialogMode = DialogSoundPackPicker

	// Override default config path by pre-creating the file
	// The update handler uses audio.DefaultConfigPath - we test via SavePackSelection separately

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)

	if model.audioNotifier.ActivePack() != "testpack" {
		t.Fatalf("expected notifier pack=testpack, got %q", model.audioNotifier.ActivePack())
	}
	if model.activeSoundPack != "testpack" {
		t.Fatalf("expected activeSoundPack=testpack, got %q", model.activeSoundPack)
	}
	if model.dialogMode != DialogNone {
		t.Fatalf("expected dialog closed after selection, got %d", model.dialogMode)
	}
}

// AC5: The selection is persisted to sounds.yaml (tested via SavePackSelection unit tests)
func TestSavePackSelectionPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "sounds.yaml")

	if err := os.WriteFile(configPath, []byte("enabled: true\ncooldown_seconds: 10\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := audio.SavePackSelection(configPath, "zelda"); err != nil {
		t.Fatalf("SavePackSelection error: %v", err)
	}

	// Reload and verify
	cfg, err := audio.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.Pack != "zelda" {
		t.Fatalf("expected pack=zelda after reload, got %q", cfg.Pack)
	}
	if !cfg.Enabled {
		t.Fatalf("expected enabled preserved after save")
	}
	if cfg.CooldownSeconds != 10 {
		t.Fatalf("expected cooldown preserved after save, got %d", cfg.CooldownSeconds)
	}
}

// AC6: Pressing Esc closes the picker without changing the pack
func TestSoundPackPickerEscCancels(t *testing.T) {
	packs := []audio.PackInfo{
		{Name: "starcraft", EventCount: 3, FileCount: 7},
		{Name: "zelda", EventCount: 2, FileCount: 4},
	}
	m := newSoundPackPickerTestModel(packs, "starcraft")
	m.dialogMode = DialogSoundPackPicker

	// Navigate to second pack
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	model := updated.(Model)
	if model.soundPackCursor != 1 {
		t.Fatalf("expected cursor at 1, got %d", model.soundPackCursor)
	}

	// Press Esc
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)

	if model.dialogMode != DialogNone {
		t.Fatalf("expected dialog closed, got %d", model.dialogMode)
	}
	if model.activeSoundPack != "starcraft" {
		t.Fatalf("expected active pack unchanged, got %q", model.activeSoundPack)
	}
}

// AC7: When no packs are installed, the picker shows a helpful message
func TestSoundPackPickerEmptyState(t *testing.T) {
	m := newSoundPackPickerTestModel(nil, "")
	m.dialogMode = DialogSoundPackPicker
	m.soundPacks = []audio.PackInfo{} // explicit empty

	view := m.View()

	if !strings.Contains(view, "No sound packs installed") {
		t.Fatalf("expected empty state message in view")
	}
	if !strings.Contains(view, "soundpacks") {
		t.Fatalf("expected directory hint in view")
	}
}

// AC7 continued: Only Esc works in empty state
func TestSoundPackPickerEmptyStateOnlyEsc(t *testing.T) {
	m := newSoundPackPickerTestModel(nil, "")
	m.dialogMode = DialogSoundPackPicker
	m.soundPacks = []audio.PackInfo{}

	// Enter should not crash or change state
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)
	if model.dialogMode != DialogSoundPackPicker {
		t.Fatalf("expected dialog still open after Enter in empty state")
	}

	// Up should not crash
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(Model)

	// Down should not crash
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)

	// Esc should close
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if model.dialogMode != DialogNone {
		t.Fatalf("expected dialog closed on Esc")
	}
}

// AC8: Existing keybinds are unaffected
func TestSoundPackPickerExistingKeybindsWork(t *testing.T) {
	m := newSoundPackPickerTestModel(nil, "")

	// 'm' should still toggle mute
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	model := updated.(Model)
	if !model.audioNotifier.IsMuted() {
		t.Fatalf("expected mute toggled via 'm' key")
	}

	// 'q' should still quit
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatalf("expected quit command from 'q' key")
	}
}

// S keybind should not open picker when another dialog is open
func TestSoundPackPickerBlockedByOtherDialog(t *testing.T) {
	m := newSoundPackPickerTestModel(nil, "")
	m.dialogMode = DialogKillConfirm

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	model := updated.(Model)

	// Should remain in kill confirm dialog, not switch to picker
	if model.dialogMode != DialogKillConfirm {
		t.Fatalf("expected dialog unchanged, got %d", model.dialogMode)
	}
}

// Navigation test: up/down cursor movement
func TestSoundPackPickerNavigation(t *testing.T) {
	packs := make([]audio.PackInfo, 15)
	for i := range packs {
		packs[i] = audio.PackInfo{Name: fmt.Sprintf("pack-%02d", i), EventCount: i + 1, FileCount: i * 2}
	}
	m := newSoundPackPickerTestModel(packs, "")
	m.dialogMode = DialogSoundPackPicker

	// Move down 5 times
	var updated tea.Model
	for i := 0; i < 5; i++ {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(Model)
	}
	if m.soundPackCursor != 5 {
		t.Fatalf("expected cursor=5, got %d", m.soundPackCursor)
	}

	// Move up once
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	if m.soundPackCursor != 4 {
		t.Fatalf("expected cursor=4, got %d", m.soundPackCursor)
	}

	// Move past viewport — verify scroll
	for i := 0; i < 10; i++ {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(Model)
	}
	if m.soundPackCursor != 14 {
		t.Fatalf("expected cursor at last item (14), got %d", m.soundPackCursor)
	}
	if m.soundPackScrollOffset <= 0 {
		t.Fatalf("expected scroll offset > 0 for scrolled view, got %d", m.soundPackScrollOffset)
	}

	// Cannot go past the end
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.soundPackCursor != 14 {
		t.Fatalf("expected cursor clamped at 14, got %d", m.soundPackCursor)
	}
}

// Navigation with j/k keys
func TestSoundPackPickerVimNavigation(t *testing.T) {
	packs := []audio.PackInfo{
		{Name: "a", EventCount: 1, FileCount: 1},
		{Name: "b", EventCount: 2, FileCount: 2},
		{Name: "c", EventCount: 3, FileCount: 3},
	}
	m := newSoundPackPickerTestModel(packs, "")
	m.dialogMode = DialogSoundPackPicker

	// j moves down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model := updated.(Model)
	if model.soundPackCursor != 1 {
		t.Fatalf("expected cursor=1 after j, got %d", model.soundPackCursor)
	}

	// k moves up
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	model = updated.(Model)
	if model.soundPackCursor != 0 {
		t.Fatalf("expected cursor=0 after k, got %d", model.soundPackCursor)
	}
}

// soundPacksMsg handler populates state
func TestSoundPacksMsgPopulatesState(t *testing.T) {
	m := newSoundPackPickerTestModel(nil, "active-pack")
	m.dialogMode = DialogSoundPackPicker

	packs := []audio.PackInfo{
		{Name: "pack-a", EventCount: 2, FileCount: 5},
	}
	updated, _ := m.Update(soundPacksMsg{packs: packs})
	model := updated.(Model)

	if len(model.soundPacks) != 1 {
		t.Fatalf("expected 1 pack, got %d", len(model.soundPacks))
	}
	if model.soundPacks[0].Name != "pack-a" {
		t.Fatalf("expected pack-a, got %q", model.soundPacks[0].Name)
	}
	if model.activeSoundPack != "active-pack" {
		t.Fatalf("expected activeSoundPack from notifier, got %q", model.activeSoundPack)
	}
}

// soundPacksMsg error handling
func TestSoundPacksMsgError(t *testing.T) {
	m := newSoundPackPickerTestModel(nil, "")
	m.dialogMode = DialogSoundPackPicker

	updated, _ := m.Update(soundPacksMsg{err: fmt.Errorf("read error")})
	model := updated.(Model)

	if model.dialogError != "read error" {
		t.Fatalf("expected dialog error set, got %q", model.dialogError)
	}
}

// DialogTitle returns correct title
func TestSoundPackPickerDialogTitle(t *testing.T) {
	title := DialogTitle(DialogSoundPackPicker)
	if title != "Sound Packs" {
		t.Fatalf("expected title 'Sound Packs', got %q", title)
	}
}
