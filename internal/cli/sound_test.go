package cli

import (
	"testing"
)

func TestRunSoundNoArgs(t *testing.T) {
	code := RunSound(nil)
	if code != exitError {
		t.Fatalf("expected exit code %d, got %d", exitError, code)
	}
}

func TestRunSoundUnknownSubcommand(t *testing.T) {
	code := RunSound([]string{"unknown"})
	if code != exitError {
		t.Fatalf("expected exit code %d, got %d", exitError, code)
	}
}

func TestRunSoundTestMissingEvent(t *testing.T) {
	code := RunSound([]string{"test"})
	if code != exitError {
		t.Fatalf("expected exit code %d, got %d", exitError, code)
	}
}

func TestRunSoundTestUnknownEvent(t *testing.T) {
	code := RunSound([]string{"test", "invalid-event"})
	if code != exitError {
		t.Fatalf("expected exit code %d for unknown event, got %d", exitError, code)
	}
}

func TestIsValidEvent(t *testing.T) {
	for _, event := range validEvents {
		if !isValidEvent(event) {
			t.Fatalf("expected %q to be valid", event)
		}
	}

	if isValidEvent("bogus") {
		t.Fatalf("expected 'bogus' to be invalid")
	}
}

func TestRunSoundListNoPacksDir(t *testing.T) {
	// With default config (no packs installed), list should succeed
	code := runSoundList()
	if code != exitOK {
		t.Fatalf("expected exit code %d for list with no packs, got %d", exitOK, code)
	}
}
