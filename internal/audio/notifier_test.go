package audio

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type mockPlayer struct {
	available bool
	files     []string
	volumes   []int
}

func (m *mockPlayer) Play(filePath string, volume int) error {
	m.files = append(m.files, filePath)
	m.volumes = append(m.volumes, volume)
	return nil
}

func (m *mockPlayer) Available() bool { return m.available }
func (m *mockPlayer) Backend() string { return "mock-player" }

type mockTTS struct {
	available bool
	texts     []string
}

func (m *mockTTS) Speak(text string) error {
	m.texts = append(m.texts, text)
	return nil
}

func (m *mockTTS) Available() bool { return m.available }
func (m *mockTTS) Backend() string { return "mock-tts" }

func newTestNotifier(cfg *Config, player *mockPlayer, tts *mockTTS) *Notifier {
	n := &Notifier{
		cfg:       cfg,
		player:    player,
		tts:       tts,
		cooldowns: make(map[string]time.Time),
		now:       time.Now,
		runAsync: func(fn func()) {
			fn()
		},
		ttsDelay: 0,
		randIntn: func(n int) int { return 0 },
	}
	return n
}

func newTestNotifierWithPack(cfg *Config, player *mockPlayer, tts *mockTTS, packFiles map[string][]string) *Notifier {
	n := newTestNotifier(cfg, player, tts)
	n.packFiles = packFiles
	return n
}

func TestNotifyTriggerEnabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Triggers["permission"] = true
	cfg.Files["permission"] = "/tmp/permission.wav"
	cfg.TTS.Enabled = true
	cfg.TTS.Template = "{session} — {status}"

	player := &mockPlayer{available: true}
	tts := &mockTTS{available: true}
	n := newTestNotifier(cfg, player, tts)

	n.Notify("mysession", "permission")

	if len(player.files) != 1 || player.files[0] != "/tmp/permission.wav" {
		t.Fatalf("expected one player call with mapped file, got %#v", player.files)
	}
	if len(tts.texts) != 1 || tts.texts[0] != "mysession — permission" {
		t.Fatalf("expected one tts call with formatted announcement, got %#v", tts.texts)
	}
}

func TestNotifyTriggerDisabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Triggers["working"] = false
	cfg.Files["working"] = "/tmp/working.wav"

	player := &mockPlayer{available: true}
	tts := &mockTTS{available: true}
	n := newTestNotifier(cfg, player, tts)

	n.Notify("mysession", "working")

	if len(player.files) != 0 || len(tts.texts) != 0 {
		t.Fatalf("expected no calls when trigger disabled")
	}
}

func TestNotifyCooldownSameSession(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.CooldownSeconds = 5
	cfg.Triggers["done"] = true
	cfg.Files["done"] = "/tmp/done.wav"

	player := &mockPlayer{available: true}
	tts := &mockTTS{available: true}
	n := newTestNotifier(cfg, player, tts)

	current := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	n.now = func() time.Time { return current }

	n.Notify("mysession", "done")
	n.Notify("mysession", "done")

	if len(player.files) != 1 {
		t.Fatalf("expected one playback call during cooldown window, got %d", len(player.files))
	}

	current = current.Add(6 * time.Second)
	n.Notify("mysession", "done")
	if len(player.files) != 2 {
		t.Fatalf("expected second call after cooldown expiration, got %d", len(player.files))
	}
}

func TestNotifyCooldownPerSession(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.CooldownSeconds = 10
	cfg.Triggers["error"] = true
	cfg.Files["error"] = "/tmp/error.wav"

	player := &mockPlayer{available: true}
	tts := &mockTTS{available: true}
	n := newTestNotifier(cfg, player, tts)

	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	n.now = func() time.Time { return now }

	n.Notify("session-a", "error")
	n.Notify("session-b", "error")

	if len(player.files) != 2 {
		t.Fatalf("expected independent cooldowns per session, got %d calls", len(player.files))
	}
}

func TestNotifyNoSoundFileStillSpeaks(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Triggers["permission"] = true
	cfg.Files = map[string]string{}
	cfg.TTS.Enabled = true

	player := &mockPlayer{available: true}
	tts := &mockTTS{available: true}
	n := newTestNotifier(cfg, player, tts)

	n.Notify("mysession", "permission")

	if len(player.files) != 0 {
		t.Fatalf("expected no playback when no file mapping exists")
	}
	if len(tts.texts) != 1 {
		t.Fatalf("expected tts call when enabled")
	}
}

func TestNotifyDisabledConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false

	player := &mockPlayer{available: true}
	tts := &mockTTS{available: true}
	n := newTestNotifier(cfg, player, tts)

	n.Notify("mysession", "permission")
	if len(player.files) != 0 || len(tts.texts) != 0 {
		t.Fatalf("expected no calls when config disabled")
	}
}

func TestNotifyNoBackends(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Triggers["permission"] = true

	player := &mockPlayer{available: false}
	tts := &mockTTS{available: false}
	n := newTestNotifier(cfg, player, tts)

	n.Notify("mysession", "permission")
	if len(player.files) != 0 || len(tts.texts) != 0 {
		t.Fatalf("expected no calls with no available backends")
	}
	if n.Enabled() {
		t.Fatalf("expected notifier disabled with no backends")
	}
}

func TestMuteBlocksNotify(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Triggers["permission"] = true
	cfg.Files["permission"] = "/tmp/permission.wav"
	cfg.TTS.Enabled = true

	player := &mockPlayer{available: true}
	tts := &mockTTS{available: true}
	n := newTestNotifier(cfg, player, tts)

	n.SetMuted(true)
	n.Notify("mysession", "permission")

	if len(player.files) != 0 {
		t.Fatalf("expected no playback when muted, got %d", len(player.files))
	}
	if len(tts.texts) != 0 {
		t.Fatalf("expected no TTS when muted, got %d", len(tts.texts))
	}
}

func TestUnmuteRestoresNotify(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Triggers["done"] = true
	cfg.Files["done"] = "/tmp/done.wav"

	player := &mockPlayer{available: true}
	tts := &mockTTS{available: true}
	n := newTestNotifier(cfg, player, tts)

	n.SetMuted(true)
	n.Notify("s1", "done")
	if len(player.files) != 0 {
		t.Fatalf("expected no playback when muted")
	}

	n.SetMuted(false)
	n.Notify("s1", "done")
	if len(player.files) != 1 {
		t.Fatalf("expected playback after unmute, got %d", len(player.files))
	}
}

func TestSetMutedNilSafety(t *testing.T) {
	var n *Notifier
	n.SetMuted(true)  // should not panic
	n.SetMuted(false) // should not panic
	if n.IsMuted() {
		t.Fatalf("nil notifier should return false for IsMuted")
	}
}

func TestPackResolution(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Triggers["done"] = true

	packFiles := map[string][]string{
		"done": {"/pack/done.wav"},
	}

	player := &mockPlayer{available: true}
	tts := &mockTTS{available: true}
	n := newTestNotifierWithPack(cfg, player, tts, packFiles)

	n.Notify("s1", "done")

	if len(player.files) != 1 || player.files[0] != "/pack/done.wav" {
		t.Fatalf("expected pack file, got %v", player.files)
	}
}

func TestFilesOverridesPack(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Triggers["done"] = true
	cfg.Files["done"] = "/override/done.wav"

	packFiles := map[string][]string{
		"done": {"/pack/done.wav"},
	}

	player := &mockPlayer{available: true}
	tts := &mockTTS{available: true}
	n := newTestNotifierWithPack(cfg, player, tts, packFiles)

	n.Notify("s1", "done")

	if len(player.files) != 1 || player.files[0] != "/override/done.wav" {
		t.Fatalf("expected files override, got %v", player.files)
	}
}

func TestNoPackNoFilesStillSpeaks(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Triggers["done"] = true
	cfg.TTS.Enabled = true

	player := &mockPlayer{available: true}
	tts := &mockTTS{available: true}
	n := newTestNotifier(cfg, player, tts)

	n.Notify("s1", "done")

	if len(player.files) != 0 {
		t.Fatalf("expected no playback without files or pack")
	}
	if len(tts.texts) != 1 {
		t.Fatalf("expected TTS still works, got %d", len(tts.texts))
	}
}

func TestRandomSelection(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Triggers["waiting"] = true
	cfg.CooldownSeconds = 0

	packFiles := map[string][]string{
		"waiting": {"/pack/waiting-1.wav", "/pack/waiting-2.wav", "/pack/waiting-3.wav"},
	}

	player := &mockPlayer{available: true}
	tts := &mockTTS{available: false}

	callCount := 0
	n := newTestNotifierWithPack(cfg, player, tts, packFiles)
	n.randIntn = func(n int) int {
		idx := callCount % n
		callCount++
		return idx
	}
	n.cfg.CooldownSeconds = 0

	// Force cooldown to be 0 for multiple calls
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	n.now = func() time.Time {
		now = now.Add(time.Hour)
		return now
	}

	n.Notify("s1", "waiting")
	n.Notify("s2", "waiting")
	n.Notify("s3", "waiting")

	if len(player.files) != 3 {
		t.Fatalf("expected 3 playback calls, got %d", len(player.files))
	}

	// Verify different files were selected
	seen := make(map[string]bool)
	for _, f := range player.files {
		seen[f] = true
	}
	if len(seen) < 2 {
		t.Fatalf("expected variation in file selection, got %v", player.files)
	}
}

func TestSetPackValid(t *testing.T) {
	tmpDir := t.TempDir()
	packDir := filepath.Join(tmpDir, "testpack")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packDir, "done.wav"), []byte("fake"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	withPackBaseDir(t, tmpDir)

	cfg := DefaultConfig()
	cfg.Enabled = true
	player := &mockPlayer{available: true}
	tts := &mockTTS{available: false}
	n := newTestNotifier(cfg, player, tts)

	if err := n.SetPack("testpack"); err != nil {
		t.Fatalf("SetPack error: %v", err)
	}
	if n.cfg.Pack != "testpack" {
		t.Fatalf("expected cfg.Pack=testpack, got %q", n.cfg.Pack)
	}
	if n.packFiles == nil || len(n.packFiles["done"]) == 0 {
		t.Fatalf("expected packFiles populated, got %v", n.packFiles)
	}
}

func TestSetPackEmpty(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Pack = "old-pack"
	player := &mockPlayer{available: true}
	tts := &mockTTS{available: false}
	n := newTestNotifier(cfg, player, tts)
	n.packFiles = map[string][]string{"done": {"/old/done.wav"}}

	if err := n.SetPack(""); err != nil {
		t.Fatalf("SetPack error: %v", err)
	}
	if n.cfg.Pack != "" {
		t.Fatalf("expected cfg.Pack empty, got %q", n.cfg.Pack)
	}
	if n.packFiles != nil {
		t.Fatalf("expected packFiles nil, got %v", n.packFiles)
	}
}

func TestSetPackInvalidPreservesState(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Pack = "old-pack"
	player := &mockPlayer{available: true}
	tts := &mockTTS{available: false}
	n := newTestNotifier(cfg, player, tts)
	n.packFiles = map[string][]string{"done": {"/old/done.wav"}}

	err := n.SetPack("nonexistent-pack-xyz")
	if err == nil {
		t.Fatalf("expected error for invalid pack")
	}
	// State should be preserved
	if n.cfg.Pack != "old-pack" {
		t.Fatalf("expected cfg.Pack preserved, got %q", n.cfg.Pack)
	}
	if len(n.packFiles["done"]) != 1 {
		t.Fatalf("expected packFiles preserved")
	}
}

func TestSetPackNilSafety(t *testing.T) {
	var n *Notifier
	if err := n.SetPack("test"); err != nil {
		t.Fatalf("nil notifier SetPack should return nil, got %v", err)
	}
}

func TestActivePack(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Pack = "my-pack"
	player := &mockPlayer{available: true}
	tts := &mockTTS{available: false}
	n := newTestNotifier(cfg, player, tts)

	if got := n.ActivePack(); got != "my-pack" {
		t.Fatalf("expected ActivePack()=my-pack, got %q", got)
	}

	var nilN *Notifier
	if got := nilN.ActivePack(); got != "" {
		t.Fatalf("expected ActivePack() empty for nil notifier, got %q", got)
	}
}

func TestSetPackConcurrentWithNotify(t *testing.T) {
	tmpDir := t.TempDir()
	for _, pack := range []string{"pack-a", "pack-b"} {
		packDir := filepath.Join(tmpDir, pack)
		if err := os.MkdirAll(packDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(packDir, "done.wav"), []byte("fake"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	withPackBaseDir(t, tmpDir)

	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Triggers["done"] = true
	cfg.CooldownSeconds = 0

	player := &mockPlayer{available: true}
	tts := &mockTTS{available: false}
	n := newTestNotifier(cfg, player, tts)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	n.now = func() time.Time {
		now = now.Add(time.Hour)
		return now
	}

	// Hammer SetPack and Notify concurrently to detect races
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 50; i++ {
			pack := "pack-a"
			if i%2 == 0 {
				pack = "pack-b"
			}
			_ = n.SetPack(pack)
		}
	}()
	for i := 0; i < 50; i++ {
		n.Notify(fmt.Sprintf("s%d", i), "done")
		_ = n.ActivePack()
	}
	<-done
}

func TestVolumePassedToPlayer(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Triggers["done"] = true
	cfg.Files["done"] = "/tmp/done.wav"
	cfg.Volume.Global = 80
	cfg.Volume.Events["done"] = 0.7

	player := &mockPlayer{available: true}
	tts := &mockTTS{available: false}
	n := newTestNotifier(cfg, player, tts)

	n.Notify("s1", "done")

	if len(player.volumes) != 1 {
		t.Fatalf("expected 1 volume entry, got %d", len(player.volumes))
	}
	// 80 * 0.7 = 56
	if player.volumes[0] != 56 {
		t.Fatalf("expected volume 56, got %d", player.volumes[0])
	}
}
