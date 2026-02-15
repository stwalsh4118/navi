package audio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigValidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	soundFile := filepath.Join(tmpDir, "permission.wav")
	if err := os.WriteFile(soundFile, []byte(""), 0o644); err != nil {
		t.Fatalf("write sound file: %v", err)
	}

	configPath := filepath.Join(tmpDir, "sounds.yaml")
	configYAML := strings.Join([]string{
		"enabled: true",
		"triggers:",
		"  permission: true",
		"files:",
		"  permission: " + soundFile,
		"tts:",
		"  enabled: true",
		"  template: \"{session}::{status}\"",
		"cooldown_seconds: 9",
		"player: paplay",
		"tts_engine: espeak-ng",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(configYAML), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	if !cfg.Enabled {
		t.Fatalf("expected enabled=true")
	}
	if !cfg.Triggers["permission"] {
		t.Fatalf("expected permission trigger=true")
	}
	if got := cfg.Files["permission"]; got != soundFile {
		t.Fatalf("expected file %q, got %q", soundFile, got)
	}
	if !cfg.TTS.Enabled {
		t.Fatalf("expected tts.enabled=true")
	}
	if got := cfg.TTS.Template; got != "{session}::{status}" {
		t.Fatalf("expected template override, got %q", got)
	}
	if got := cfg.CooldownSeconds; got != 9 {
		t.Fatalf("expected cooldown=9, got %d", got)
	}
	if got := cfg.Player; got != "paplay" {
		t.Fatalf("expected player=paplay, got %q", got)
	}
	if got := cfg.TTSEngine; got != "espeak-ng" {
		t.Fatalf("expected tts_engine=espeak-ng, got %q", got)
	}
}

func TestLoadConfigMissingFileReturnsDefault(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if cfg == nil {
		t.Fatalf("expected config")
	}
	if cfg.Enabled {
		t.Fatalf("expected enabled=false by default")
	}
}

func TestLoadConfigMalformedYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "sounds.yaml")
	if err := os.WriteFile(configPath, []byte("enabled: ["), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestLoadConfigExpandsTildePaths(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("home dir unavailable")
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "sounds.yaml")
	if err := os.WriteFile(configPath, []byte(strings.Join([]string{
		"files:",
		"  done: ~/sounds/done.wav",
	}, "\n")), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	expected := filepath.Join(home, "sounds", "done.wav")
	if got := cfg.Files["done"]; got != expected {
		t.Fatalf("expected expanded path %q, got %q", expected, got)
	}
}

func TestDefaultConfigValues(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Enabled {
		t.Fatalf("expected enabled=false by default")
	}
	if cfg.CooldownSeconds != 5 {
		t.Fatalf("expected cooldown=5, got %d", cfg.CooldownSeconds)
	}
	if cfg.Player != "auto" {
		t.Fatalf("expected player=auto, got %q", cfg.Player)
	}
	if cfg.TTSEngine != "auto" {
		t.Fatalf("expected tts_engine=auto, got %q", cfg.TTSEngine)
	}
	if cfg.TTS.Template != "{session} — {status}" {
		t.Fatalf("unexpected default template: %q", cfg.TTS.Template)
	}
}
