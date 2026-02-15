package audio

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

var (
	ttsLookPath = exec.LookPath
	ttsRunCmd   = func(name string, args ...string) error {
		cmd := exec.Command(name, args...)
		return cmd.Run()
	}
)

// TTS wraps a system text-to-speech backend.
type TTS struct {
	backend string
}

// NewTTS creates a TTS engine with explicit override or auto-detection.
func NewTTS(override string) *TTS {
	backend := detectTTSBackend(override)
	return &TTS{backend: backend}
}

// Available reports whether a usable TTS backend was detected.
func (t *TTS) Available() bool {
	return t != nil && t.backend != ""
}

// Backend returns the selected TTS backend name.
func (t *TTS) Backend() string {
	if t == nil {
		return ""
	}
	return t.backend
}

// Speak starts non-blocking speech synthesis.
func (t *TTS) Speak(text string) error {
	if !t.Available() || strings.TrimSpace(text) == "" {
		return nil
	}

	backend := t.backend
	args := ttsArgs(backend, text)

	go func() {
		if err := ttsRunCmd(backend, args...); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to speak via %s: %v\n", backend, err)
		}
	}()

	return nil
}

// FormatAnnouncement renders the announcement text from template placeholders.
func FormatAnnouncement(template, session, status string) string {
	value := template
	if strings.TrimSpace(value) == "" {
		value = defaultTTSTemplate
	}

	value = strings.ReplaceAll(value, "{session}", session)
	value = strings.ReplaceAll(value, "{status}", status)
	return value
}

func detectTTSBackend(override string) string {
	if override != "" && override != defaultBackendAuto {
		if _, err := ttsLookPath(override); err == nil {
			return override
		}
		return ""
	}

	for _, candidate := range autoTTSCandidates() {
		if _, err := ttsLookPath(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

func autoTTSCandidates() []string {
	if runtime.GOOS == "darwin" {
		return []string{"say"}
	}
	return []string{"espeak-ng", "espeak", "spd-say"}
}

func ttsArgs(backend, text string) []string {
	switch backend {
	case "spd-say":
		return []string{"--wait", text}
	default:
		return []string{text}
	}
}
