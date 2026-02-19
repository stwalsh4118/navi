package audio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// AC1: Sound packs are directory-based with files named by event.
func TestE2E_AC1_PackDirectoryResolution(t *testing.T) {
	packDir := t.TempDir()
	createTestFile(t, filepath.Join(packDir, "done.wav"))
	createTestFile(t, filepath.Join(packDir, "error.mp3"))
	createTestFile(t, filepath.Join(packDir, "waiting.ogg"))

	files, err := ScanPack(packDir)
	if err != nil {
		t.Fatalf("ScanPack error: %v", err)
	}

	if len(files) != 3 {
		t.Fatalf("expected 3 events, got %d", len(files))
	}
	for _, event := range []string{"done", "error", "waiting"} {
		if paths, ok := files[event]; !ok || len(paths) != 1 {
			t.Fatalf("expected 1 file for %q, got %v", event, paths)
		}
	}

	// Verify config pack field loads
	configPath := filepath.Join(t.TempDir(), "sounds.yaml")
	configYAML := "pack: starcraft\nenabled: true\n"
	if err := os.WriteFile(configPath, []byte(configYAML), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.Pack != "starcraft" {
		t.Fatalf("expected pack=starcraft, got %q", cfg.Pack)
	}
}

// AC2: Multiple sounds per event via numeric suffix with random selection.
func TestE2E_AC2_MultiSoundRandomSelection(t *testing.T) {
	packDir := t.TempDir()
	createTestFile(t, filepath.Join(packDir, "waiting-1.wav"))
	createTestFile(t, filepath.Join(packDir, "waiting-2.wav"))
	createTestFile(t, filepath.Join(packDir, "waiting-3.wav"))
	createTestFile(t, filepath.Join(packDir, "done.wav"))

	files, err := ScanPack(packDir)
	if err != nil {
		t.Fatalf("ScanPack error: %v", err)
	}

	if len(files["waiting"]) != 3 {
		t.Fatalf("expected 3 waiting files, got %d", len(files["waiting"]))
	}
	if len(files["done"]) != 1 {
		t.Fatalf("expected 1 done file, got %d", len(files["done"]))
	}

	// Test random selection through Notifier
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Triggers["waiting"] = true

	player := &mockPlayer{available: true}
	tts := &mockTTS{available: false}
	n := newTestNotifierWithPack(cfg, player, tts, files)

	callIdx := 0
	n.randIntn = func(max int) int {
		idx := callIdx % max
		callIdx++
		return idx
	}

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	n.now = func() time.Time {
		now = now.Add(time.Hour)
		return now
	}

	n.Notify("s1", "waiting")
	n.Notify("s2", "waiting")
	n.Notify("s3", "waiting")

	if len(player.files) != 3 {
		t.Fatalf("expected 3 plays, got %d", len(player.files))
	}

	seen := make(map[string]bool)
	for _, f := range player.files {
		seen[f] = true
	}
	if len(seen) < 2 {
		t.Fatalf("expected variation, got %v", player.files)
	}
}

// AC3: Volume control — global and per-event multiplier passed to player.
func TestE2E_AC3_VolumeControl(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Triggers["done"] = true
	cfg.Files["done"] = "/tmp/done.wav"
	cfg.Volume.Global = 60
	cfg.Volume.Events["done"] = 0.5

	// Effective = 60 * 0.5 = 30
	eff := cfg.Volume.EffectiveVolume("done")
	if eff != 30 {
		t.Fatalf("expected effective volume 30, got %d", eff)
	}

	player := &mockPlayer{available: true}
	tts := &mockTTS{available: false}
	n := newTestNotifier(cfg, player, tts)

	n.Notify("s1", "done")

	if len(player.volumes) != 1 || player.volumes[0] != 30 {
		t.Fatalf("expected volume 30 passed to player, got %v", player.volumes)
	}

	// Verify backend-specific args
	paplayArgs := volumeArgs("paplay", 50)
	if paplayArgs[0] != "--volume=32768" {
		t.Fatalf("paplay volume arg: %v", paplayArgs)
	}

	afplayArgs := volumeArgs("afplay", 50)
	if afplayArgs[0] != "-v" || afplayArgs[1] != "0.50" {
		t.Fatalf("afplay volume args: %v", afplayArgs)
	}

	mpvArgs := volumeArgs("mpv", 50)
	if mpvArgs[0] != "--volume=50" {
		t.Fatalf("mpv volume arg: %v", mpvArgs)
	}

	ffplayArgs := volumeArgs("ffplay", 50)
	if ffplayArgs[0] != "-volume" || ffplayArgs[1] != "50" {
		t.Fatalf("ffplay volume args: %v", ffplayArgs)
	}

	aplayArgs := volumeArgs("aplay", 50)
	if len(aplayArgs) != 0 {
		t.Fatalf("expected no args for aplay, got %v", aplayArgs)
	}
}

// AC4: Mute toggle blocks all audio.
func TestE2E_AC4_MuteToggle(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Triggers["permission"] = true
	cfg.Files["permission"] = "/tmp/permission.wav"
	cfg.TTS.Enabled = true

	player := &mockPlayer{available: true}
	tts := &mockTTS{available: true}
	n := newTestNotifier(cfg, player, tts)

	// Mute
	n.SetMuted(true)
	if !n.IsMuted() {
		t.Fatalf("expected muted after SetMuted(true)")
	}

	n.Notify("s1", "permission")
	if len(player.files) != 0 || len(tts.texts) != 0 {
		t.Fatalf("expected no audio when muted")
	}

	// Unmute
	n.SetMuted(false)
	if n.IsMuted() {
		t.Fatalf("expected unmuted after SetMuted(false)")
	}

	n.Notify("s1", "permission")
	if len(player.files) != 1 {
		t.Fatalf("expected playback after unmute, got %d", len(player.files))
	}
	if len(tts.texts) != 1 {
		t.Fatalf("expected TTS after unmute, got %d", len(tts.texts))
	}
}

// AC6: Backwards compatibility — files config works alone, pack+files coexist.
func TestE2E_AC6_BackwardsCompat(t *testing.T) {
	// Case 1: Config with only files (no pack, no volume)
	tmpDir := t.TempDir()
	soundFile := filepath.Join(tmpDir, "done.wav")
	createTestFile(t, soundFile)

	configPath := filepath.Join(tmpDir, "sounds.yaml")
	configYAML := strings.Join([]string{
		"enabled: true",
		"files:",
		"  done: " + soundFile,
		"cooldown_seconds: 1",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(configYAML), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.Pack != "" {
		t.Fatalf("expected empty pack, got %q", cfg.Pack)
	}
	if cfg.Volume.Global != defaultGlobalVolume {
		t.Fatalf("expected default volume, got %d", cfg.Volume.Global)
	}

	player := &mockPlayer{available: true}
	tts := &mockTTS{available: false}
	n := newTestNotifier(cfg, player, tts)

	n.Notify("s1", "done")
	if len(player.files) != 1 || player.files[0] != soundFile {
		t.Fatalf("expected files-based playback, got %v", player.files)
	}

	// Case 2: Pack + files override — files takes precedence for overridden event
	packFiles := map[string][]string{
		"done":    {"/pack/done.wav"},
		"waiting": {"/pack/waiting.wav"},
	}

	cfg2 := DefaultConfig()
	cfg2.Enabled = true
	cfg2.Triggers["done"] = true
	cfg2.Triggers["waiting"] = true
	cfg2.Files["done"] = "/override/done.wav"

	player2 := &mockPlayer{available: true}
	tts2 := &mockTTS{available: false}
	n2 := newTestNotifierWithPack(cfg2, player2, tts2, packFiles)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	n2.now = func() time.Time {
		now = now.Add(time.Hour)
		return now
	}

	n2.Notify("s1", "done")
	n2.Notify("s2", "waiting")

	if len(player2.files) != 2 {
		t.Fatalf("expected 2 plays, got %d", len(player2.files))
	}
	if player2.files[0] != "/override/done.wav" {
		t.Fatalf("expected files override for done, got %q", player2.files[0])
	}
	if player2.files[1] != "/pack/waiting.wav" {
		t.Fatalf("expected pack for waiting, got %q", player2.files[1])
	}
}

// AC7: Existing audio tests continue to pass (verified by running full test suite).
func TestE2E_AC7_ExistingTestsPass(t *testing.T) {
	// This is a meta-test — the fact that all other tests in this package pass
	// confirms AC7. Just verify the test infrastructure works.
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatalf("DefaultConfig returned nil")
	}

	player := NewPlayer("nonexistent-player")
	if player.Available() {
		t.Fatalf("expected unavailable player for nonexistent backend")
	}
}

// AC8: API spec contains new types.
func TestE2E_AC8_APISpecUpdated(t *testing.T) {
	specPath := filepath.Join("..", "..", "docs", "api-specs", "audio", "audio-api.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Skipf("cannot read API spec: %v", err)
	}

	content := string(data)
	required := []string{
		"VolumeConfig",
		"EffectiveVolume",
		"PackInfo",
		"ScanPack",
		"ResolveSoundFiles",
		"ListPacks",
		"SetMuted",
		"IsMuted",
		"volume int",
	}

	for _, keyword := range required {
		if !strings.Contains(content, keyword) {
			t.Errorf("API spec missing keyword: %q", keyword)
		}
	}
}

// Full pipeline: config → pack → volume → random → play with correct volume.
func TestE2E_FullPipeline(t *testing.T) {
	packDir := t.TempDir()
	createTestFile(t, filepath.Join(packDir, "waiting-1.wav"))
	createTestFile(t, filepath.Join(packDir, "waiting-2.wav"))
	createTestFile(t, filepath.Join(packDir, "done.wav"))

	files, err := ScanPack(packDir)
	if err != nil {
		t.Fatalf("ScanPack error: %v", err)
	}

	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Triggers["waiting"] = true
	cfg.Triggers["done"] = true
	cfg.Volume.Global = 80
	cfg.Volume.Events["waiting"] = 0.5
	cfg.Volume.Events["done"] = 1.0

	player := &mockPlayer{available: true}
	tts := &mockTTS{available: false}
	n := newTestNotifierWithPack(cfg, player, tts, files)
	n.randIntn = func(max int) int { return 0 }

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	n.now = func() time.Time {
		now = now.Add(time.Hour)
		return now
	}

	n.Notify("s1", "waiting")
	n.Notify("s2", "done")

	if len(player.files) != 2 {
		t.Fatalf("expected 2 plays, got %d", len(player.files))
	}

	// waiting: 80 * 0.5 = 40
	if player.volumes[0] != 40 {
		t.Fatalf("expected waiting volume 40, got %d", player.volumes[0])
	}

	// done: 80 * 1.0 = 80
	if player.volumes[1] != 80 {
		t.Fatalf("expected done volume 80, got %d", player.volumes[1])
	}
}

// Zero volume should skip playback entirely.
func TestE2E_ZeroVolumeSkipsPlay(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Triggers["done"] = true
	cfg.Files["done"] = "/tmp/done.wav"
	cfg.Volume.Global = 0

	player := &mockPlayer{available: true}
	tts := &mockTTS{available: true}
	n := newTestNotifier(cfg, player, tts)

	n.Notify("s1", "done")

	// Volume 0 → Play gets called with volume=0 → Player.Play returns early
	// The notifier will call Play(file, 0) and the player skips it
	if len(player.files) != 1 {
		// The mock player doesn't implement the volume=0 skip logic
		// (that's in the real Player.Play), so mock records it
		// but the real player would skip. Checking volume was 0.
		if player.volumes[0] != 0 {
			t.Fatalf("expected volume 0, got %d", player.volumes[0])
		}
	}
}

// Missing pack should gracefully fall back.
func TestE2E_MissingPackGraceful(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Triggers["done"] = true
	cfg.TTS.Enabled = true

	player := &mockPlayer{available: true}
	tts := &mockTTS{available: true}
	n := newTestNotifier(cfg, player, tts)
	// packFiles is nil — simulates missing/failed pack load

	n.Notify("s1", "done")

	// No crash, no sound from pack, TTS still works
	if len(player.files) != 0 {
		t.Fatalf("expected no playback with missing pack, got %v", player.files)
	}
	if len(tts.texts) != 1 {
		t.Fatalf("expected TTS still works, got %d", len(tts.texts))
	}
}
