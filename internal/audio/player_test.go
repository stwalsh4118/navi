package audio

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func withPlayerLookPathMock(t *testing.T, fn func(string) (string, error)) {
	t.Helper()
	old := playerLookPath
	playerLookPath = fn
	t.Cleanup(func() { playerLookPath = old })
}

func withPlayerRunCmdMock(t *testing.T, fn func(string, ...string) error) {
	t.Helper()
	old := playerRunCmd
	playerRunCmd = fn
	t.Cleanup(func() { playerRunCmd = old })
}

func TestNewPlayerOverrideUsesSpecifiedBinary(t *testing.T) {
	withPlayerLookPathMock(t, func(name string) (string, error) {
		if name == "paplay" {
			return "/usr/bin/paplay", nil
		}
		return "", errors.New("not found")
	})

	player := NewPlayer("paplay")
	if !player.Available() {
		t.Fatalf("expected player available")
	}
	if got := player.Backend(); got != "paplay" {
		t.Fatalf("expected backend paplay, got %q", got)
	}
}

func TestNewPlayerAutoDetectionOrder(t *testing.T) {
	withPlayerLookPathMock(t, func(name string) (string, error) {
		switch runtime.GOOS {
		case "darwin":
			if name == "afplay" {
				return "/usr/bin/afplay", nil
			}
		default:
			if name == "ffplay" {
				return "/usr/bin/ffplay", nil
			}
		}
		return "", errors.New("not found")
	})

	player := NewPlayer("auto")
	if !player.Available() {
		t.Fatalf("expected detected backend")
	}

	if runtime.GOOS == "darwin" {
		if got := player.Backend(); got != "afplay" {
			t.Fatalf("expected afplay backend, got %q", got)
		}
	} else {
		if got := player.Backend(); got != "ffplay" {
			t.Fatalf("expected ffplay backend, got %q", got)
		}
	}
}

func TestAvailableFalseWhenNoneFound(t *testing.T) {
	withPlayerLookPathMock(t, func(name string) (string, error) {
		return "", errors.New("not found")
	})

	player := NewPlayer("auto")
	if player.Available() {
		t.Fatalf("expected unavailable player")
	}
	if got := player.Backend(); got != "" {
		t.Fatalf("expected empty backend, got %q", got)
	}
}

func TestPlayMissingFileReturnsError(t *testing.T) {
	player := &Player{backend: "paplay"}
	err := player.Play(filepath.Join(t.TempDir(), "missing.wav"))
	if err == nil {
		t.Fatalf("expected error for missing file")
	}
}

func TestPlayReturnsImmediately(t *testing.T) {
	called := make(chan struct{}, 1)
	withPlayerRunCmdMock(t, func(name string, args ...string) error {
		called <- struct{}{}
		time.Sleep(250 * time.Millisecond)
		return nil
	})

	tmpDir := t.TempDir()
	sound := filepath.Join(tmpDir, "tone.wav")
	if err := os.WriteFile(sound, []byte(""), 0o644); err != nil {
		t.Fatalf("write sound file: %v", err)
	}

	player := &Player{backend: "paplay"}
	start := time.Now()
	err := player.Play(sound)
	if err != nil {
		t.Fatalf("Play error: %v", err)
	}
	if time.Since(start) > 100*time.Millisecond {
		t.Fatalf("Play blocked too long")
	}

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatalf("expected async command runner to be invoked")
	}
}
