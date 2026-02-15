package audio

import (
	"testing"
	"time"
)

type mockPlayer struct {
	available bool
	files     []string
}

func (m *mockPlayer) Play(filePath string) error {
	m.files = append(m.files, filePath)
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
	}
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
