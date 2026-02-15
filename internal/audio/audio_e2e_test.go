package audio

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	e2eLogEnvVar   = "NAVI_AUDIO_E2E_LOG"
	waitTimeout    = 3 * time.Second
	pollInterval   = 20 * time.Millisecond
	nonBlockingMax = 150 * time.Millisecond
)

func TestAudioPipelineE2E(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audio.log")
	t.Setenv(e2eLogEnvVar, logPath)

	playerBin, ttsBin := backendBinariesForOS()
	createMockExecutable(t, tmpDir, playerBin, false)
	createMockExecutable(t, tmpDir, ttsBin, false)

	// Ensure auto-detection finds our mock binaries first.
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	sounds := map[string]string{
		"permission": filepath.Join(tmpDir, "permission.wav"),
		"done":       filepath.Join(tmpDir, "done.wav"),
		"error":      filepath.Join(tmpDir, "error.wav"),
	}
	for _, path := range sounds {
		if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
			t.Fatalf("write sound file: %v", err)
		}
	}

	configPath := filepath.Join(tmpDir, "sounds.yaml")
	configData := strings.Join([]string{
		"enabled: true",
		"triggers:",
		"  permission: true",
		"  done: true",
		"  error: true",
		"  working: false",
		"files:",
		"  permission: " + sounds["permission"],
		"  done: " + sounds["done"],
		"  error: " + sounds["error"],
		"tts:",
		"  enabled: true",
		"  template: \"{session}::{status}\"",
		"cooldown_seconds: 1",
		"player: auto",
		"tts_engine: auto",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(configData), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if !cfg.Enabled || !cfg.TTS.Enabled || cfg.CooldownSeconds != 1 {
		t.Fatalf("config values were not loaded correctly")
	}

	notifier := NewNotifier(cfg)
	if !notifier.Enabled() {
		t.Fatalf("expected notifier enabled with detected backends")
	}

	// CoS 1/2/3/5: mapped sound playback + per-status files + TTS + auto-detection.
	notifier.Notify("s-perm", "permission")
	notifier.Notify("s-done", "done")
	notifier.Notify("s-error", "error")

	waitForLogContains(t, logPath, []string{
		"player:" + sounds["permission"],
		"player:" + sounds["done"],
		"player:" + sounds["error"],
		"tts:s-perm::permission",
		"tts:s-done::done",
		"tts:s-error::error",
	})
}

func TestAudioPipelineNonBlockingAndCooldownE2E(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audio.log")
	t.Setenv(e2eLogEnvVar, logPath)

	playerBin, ttsBin := backendBinariesForOS()
	createMockExecutable(t, tmpDir, playerBin, true)
	createMockExecutable(t, tmpDir, ttsBin, true)
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	sound := filepath.Join(tmpDir, "permission.wav")
	if err := os.WriteFile(sound, []byte(""), 0o644); err != nil {
		t.Fatalf("write sound file: %v", err)
	}

	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.CooldownSeconds = 1
	cfg.Triggers["permission"] = true
	cfg.Files["permission"] = sound
	cfg.TTS.Enabled = true
	cfg.TTS.Template = "{session}:{status}"

	notifier := NewNotifier(cfg)
	if !notifier.Enabled() {
		t.Fatalf("expected notifier enabled")
	}

	// CoS 6: Notify should return quickly despite slow backend commands.
	start := time.Now()
	for i := 0; i < 10; i++ {
		notifier.Notify(fmt.Sprintf("s-%d", i), "permission")
	}
	if elapsed := time.Since(start); elapsed > nonBlockingMax {
		t.Fatalf("Notify appears blocking; elapsed=%v", elapsed)
	}

	// CoS 7: cooldown is per-session.
	notifier.Notify("same-session", "permission")
	notifier.Notify("same-session", "permission")
	notifier.Notify("other-session", "permission")

	waitForLogContains(t, logPath, []string{
		"player:" + sound,
		"tts:same-session:permission",
		"tts:other-session:permission",
	})

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	entries := strings.Split(strings.TrimSpace(string(logData)), "\n")
	countSameSession := 0
	for _, entry := range entries {
		if strings.Contains(entry, "same-session:permission") {
			countSameSession++
		}
	}
	if countSameSession != 1 {
		t.Fatalf("expected one same-session entry within cooldown, got %d", countSameSession)
	}
}

func TestAudioPipelineGracefulDegradationE2E(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Triggers["permission"] = true
	cfg.Files["permission"] = filepath.Join(t.TempDir(), "missing.wav")
	cfg.Player = "definitely-missing-player"
	cfg.TTS.Enabled = true
	cfg.TTSEngine = "definitely-missing-tts"

	notifier := NewNotifier(cfg)
	if notifier.Enabled() {
		t.Fatalf("expected disabled notifier when no backends are available")
	}

	// CoS 8: should be a no-op, not panic.
	notifier.Notify("session", "permission")
}

func backendBinariesForOS() (string, string) {
	if runtime.GOOS == "darwin" {
		return "afplay", "say"
	}
	return "paplay", "espeak-ng"
}

func createMockExecutable(t *testing.T, dir, name string, slow bool) {
	t.Helper()

	lines := []string{"#!/bin/sh"}
	if slow {
		lines = append(lines, "sleep 1")
	}

	prefix := "player"
	if name == "say" || name == "espeak-ng" {
		prefix = "tts"
	}
	lines = append(lines, fmt.Sprintf("printf '%s:%%s\\n' \"$*\" >> \"$%s\"", prefix, e2eLogEnvVar))

	script := strings.Join(lines, "\n") + "\n"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write mock executable %s: %v", name, err)
	}
}

func waitForLogContains(t *testing.T, logPath string, expected []string) {
	t.Helper()

	deadline := time.Now().Add(waitTimeout)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(logPath)
		if err != nil && !os.IsNotExist(err) {
			t.Fatalf("read log: %v", err)
		}

		content := string(data)
		allFound := true
		for _, needle := range expected {
			if !strings.Contains(content, needle) {
				allFound = false
				break
			}
		}
		if allFound {
			return
		}

		time.Sleep(pollInterval)
	}

	data, _ := os.ReadFile(logPath)
	t.Fatalf("timed out waiting for log content. expected=%v actual=%q", expected, string(data))
}
