package audio

import (
	"errors"
	"runtime"
	"testing"
	"time"
)

func withTTSLookPathMock(t *testing.T, fn func(string) (string, error)) {
	t.Helper()
	old := ttsLookPath
	ttsLookPath = fn
	t.Cleanup(func() { ttsLookPath = old })
}

func withTTSRunCmdMock(t *testing.T, fn func(string, ...string) error) {
	t.Helper()
	old := ttsRunCmd
	ttsRunCmd = fn
	t.Cleanup(func() { ttsRunCmd = old })
}

func TestNewTTSOverrideUsesSpecifiedBinary(t *testing.T) {
	withTTSLookPathMock(t, func(name string) (string, error) {
		if name == "say" {
			return "/usr/bin/say", nil
		}
		return "", errors.New("not found")
	})

	tts := NewTTS("say")
	if !tts.Available() {
		t.Fatalf("expected tts available")
	}
	if got := tts.Backend(); got != "say" {
		t.Fatalf("expected backend say, got %q", got)
	}
}

func TestNewTTSAutoDetection(t *testing.T) {
	withTTSLookPathMock(t, func(name string) (string, error) {
		switch runtime.GOOS {
		case "darwin":
			if name == "say" {
				return "/usr/bin/say", nil
			}
		default:
			if name == "espeak" {
				return "/usr/bin/espeak", nil
			}
		}
		return "", errors.New("not found")
	})

	tts := NewTTS("auto")
	if !tts.Available() {
		t.Fatalf("expected available tts backend")
	}
	if runtime.GOOS == "darwin" {
		if got := tts.Backend(); got != "say" {
			t.Fatalf("expected say, got %q", got)
		}
	} else {
		if got := tts.Backend(); got != "espeak" {
			t.Fatalf("expected espeak, got %q", got)
		}
	}
}

func TestFormatAnnouncement(t *testing.T) {
	got := FormatAnnouncement("{session} — {status}", "navi", "permission")
	if got != "navi — permission" {
		t.Fatalf("unexpected formatted value: %q", got)
	}

	got = FormatAnnouncement("", "abc", "done")
	if got != "abc — done" {
		t.Fatalf("expected default template output, got %q", got)
	}

	got = FormatAnnouncement("plain text", "x", "y")
	if got != "plain text" {
		t.Fatalf("expected plain text unchanged, got %q", got)
	}
}

func TestSpeakReturnsImmediately(t *testing.T) {
	called := make(chan struct{}, 1)
	withTTSRunCmdMock(t, func(name string, args ...string) error {
		called <- struct{}{}
		time.Sleep(250 * time.Millisecond)
		return nil
	})

	tts := &TTS{backend: "say"}
	start := time.Now()
	if err := tts.Speak("hello"); err != nil {
		t.Fatalf("Speak error: %v", err)
	}
	if time.Since(start) > 100*time.Millisecond {
		t.Fatalf("Speak blocked too long")
	}

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatalf("expected async TTS runner to be invoked")
	}
}

func TestSpeakNoBackendIsNoop(t *testing.T) {
	tts := &TTS{backend: ""}
	if err := tts.Speak("hello"); err != nil {
		t.Fatalf("expected nil error for no backend, got %v", err)
	}
}
