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
	if cfg.Pack != "" {
		t.Fatalf("expected pack empty by default, got %q", cfg.Pack)
	}
	if cfg.Volume.Global != 100 {
		t.Fatalf("expected volume.global=100, got %d", cfg.Volume.Global)
	}
	if cfg.Volume.Events == nil {
		t.Fatalf("expected volume.events non-nil")
	}
}

func TestLoadConfigWithPackAndVolume(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "sounds.yaml")
	configYAML := strings.Join([]string{
		"enabled: true",
		"pack: starcraft",
		"volume:",
		"  global: 80",
		"  events:",
		"    error: 1.0",
		"    done: 0.7",
		"    waiting: 0.5",
	}, "\n")
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
	if cfg.Volume.Global != 80 {
		t.Fatalf("expected volume.global=80, got %d", cfg.Volume.Global)
	}
	if got := cfg.Volume.Events["error"]; got != 1.0 {
		t.Fatalf("expected error multiplier=1.0, got %f", got)
	}
	if got := cfg.Volume.Events["done"]; got != 0.7 {
		t.Fatalf("expected done multiplier=0.7, got %f", got)
	}
	if got := cfg.Volume.Events["waiting"]; got != 0.5 {
		t.Fatalf("expected waiting multiplier=0.5, got %f", got)
	}
}

func TestLoadConfigBackwardsCompatNoPack(t *testing.T) {
	tmpDir := t.TempDir()
	soundFile := filepath.Join(tmpDir, "done.wav")
	if err := os.WriteFile(soundFile, []byte(""), 0o644); err != nil {
		t.Fatalf("write sound file: %v", err)
	}

	configPath := filepath.Join(tmpDir, "sounds.yaml")
	configYAML := strings.Join([]string{
		"enabled: true",
		"files:",
		"  done: " + soundFile,
		"cooldown_seconds: 3",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(configYAML), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.Pack != "" {
		t.Fatalf("expected empty pack for backwards compat, got %q", cfg.Pack)
	}
	if cfg.Volume.Global != 100 {
		t.Fatalf("expected default volume.global=100 for backwards compat, got %d", cfg.Volume.Global)
	}
	if got := cfg.Files["done"]; got != soundFile {
		t.Fatalf("expected file path preserved, got %q", got)
	}
}

func TestEffectiveVolumeCalculation(t *testing.T) {
	tests := []struct {
		name     string
		global   int
		events   map[string]float64
		event    string
		expected int
	}{
		{"global 80 with 0.7 multiplier", 80, map[string]float64{"done": 0.7}, "done", 56},
		{"global 80 no multiplier", 80, map[string]float64{}, "done", 80},
		{"global 0", 0, map[string]float64{"done": 1.0}, "done", 0},
		{"global 100 full multiplier", 100, map[string]float64{"error": 1.0}, "error", 100},
		{"global 50 half multiplier", 50, map[string]float64{"waiting": 0.5}, "waiting", 25},
		{"global 100 zero multiplier", 100, map[string]float64{"idle": 0.0}, "idle", 0},
		{"rounding up", 75, map[string]float64{"done": 0.33}, "done", 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := VolumeConfig{Global: tt.global, Events: tt.events}
			got := v.EffectiveVolume(tt.event)
			if got != tt.expected {
				t.Fatalf("EffectiveVolume(%q) = %d, want %d", tt.event, got, tt.expected)
			}
		})
	}
}
