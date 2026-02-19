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
	err := player.Play(filepath.Join(t.TempDir(), "missing.wav"), fullVolume)
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
	err := player.Play(sound, fullVolume)
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

func TestPlayZeroVolumeSkipsPlayback(t *testing.T) {
	called := false
	withPlayerRunCmdMock(t, func(name string, args ...string) error {
		called = true
		return nil
	})

	tmpDir := t.TempDir()
	sound := filepath.Join(tmpDir, "tone.wav")
	if err := os.WriteFile(sound, []byte(""), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	player := &Player{backend: "paplay"}
	err := player.Play(sound, 0)
	if err != nil {
		t.Fatalf("Play error: %v", err)
	}
	if called {
		t.Fatalf("expected no playback for volume 0")
	}
}

func TestPlayFullVolumeNoVolumeFlag(t *testing.T) {
	done := make(chan []string, 1)
	withPlayerRunCmdMock(t, func(name string, args ...string) error {
		done <- args
		return nil
	})

	tmpDir := t.TempDir()
	sound := filepath.Join(tmpDir, "tone.wav")
	if err := os.WriteFile(sound, []byte(""), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	player := &Player{backend: "paplay"}
	if err := player.Play(sound, fullVolume); err != nil {
		t.Fatalf("Play error: %v", err)
	}

	select {
	case capturedArgs := <-done:
		for _, arg := range capturedArgs {
			if arg == "--volume=65536" || arg == "--volume=" {
				t.Fatalf("expected no volume flag at full volume, got args: %v", capturedArgs)
			}
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for playback")
	}
}

func TestVolumeArgsPaplay(t *testing.T) {
	args := volumeArgs("paplay", 50)
	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d: %v", len(args), args)
	}
	if args[0] != "--volume=32768" {
		t.Fatalf("expected --volume=32768, got %q", args[0])
	}
}

func TestVolumeArgsAfplay(t *testing.T) {
	args := volumeArgs("afplay", 50)
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d: %v", len(args), args)
	}
	if args[0] != "-v" || args[1] != "0.50" {
		t.Fatalf("expected [-v 0.50], got %v", args)
	}
}

func TestVolumeArgsMpv(t *testing.T) {
	args := volumeArgs("mpv", 50)
	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d: %v", len(args), args)
	}
	if args[0] != "--volume=50" {
		t.Fatalf("expected --volume=50, got %q", args[0])
	}
}

func TestVolumeArgsFfplay(t *testing.T) {
	args := volumeArgs("ffplay", 50)
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d: %v", len(args), args)
	}
	if args[0] != "-volume" || args[1] != "50" {
		t.Fatalf("expected [-volume 50], got %v", args)
	}
}

func TestVolumeArgsAplay(t *testing.T) {
	args := volumeArgs("aplay", 50)
	if len(args) != 0 {
		t.Fatalf("expected no args for aplay, got %v", args)
	}
}

func TestVolumeArgsBoundaryValues(t *testing.T) {
	args1 := volumeArgs("paplay", 1)
	if args1[0] != "--volume=655" {
		t.Fatalf("volume=1 paplay: expected --volume=655, got %q", args1[0])
	}

	args99 := volumeArgs("paplay", 99)
	if args99[0] != "--volume=64881" {
		t.Fatalf("volume=99 paplay: expected --volume=64881, got %q", args99[0])
	}
}
