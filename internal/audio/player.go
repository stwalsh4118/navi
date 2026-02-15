package audio

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

var (
	playerLookPath = exec.LookPath
	playerRunCmd   = func(name string, args ...string) error {
		cmd := exec.Command(name, args...)
		return cmd.Run()
	}
)

var errAudioFileRequired = errors.New("audio file path is required")

// Player wraps a system audio backend for sound playback.
type Player struct {
	backend string
}

// NewPlayer creates a player with either explicit backend override or auto-detection.
func NewPlayer(override string) *Player {
	backend := detectPlayerBackend(override)
	return &Player{backend: backend}
}

// Available reports whether a usable backend was detected.
func (p *Player) Available() bool {
	return p != nil && p.backend != ""
}

// Backend returns the selected backend name.
func (p *Player) Backend() string {
	if p == nil {
		return ""
	}
	return p.backend
}

// Play validates the audio file then starts non-blocking playback.
func (p *Player) Play(filePath string) error {
	if filePath == "" {
		return errAudioFileRequired
	}
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("audio file not found: %w", err)
	}
	if !p.Available() {
		return nil
	}

	backend := p.backend
	args := playerArgs(backend, filePath)

	go func() {
		if err := playerRunCmd(backend, args...); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to play sound via %s: %v\n", backend, err)
		}
	}()

	return nil
}

func detectPlayerBackend(override string) string {
	if override != "" && override != defaultBackendAuto {
		if _, err := playerLookPath(override); err == nil {
			return override
		}
		return ""
	}

	for _, candidate := range autoPlayerCandidates() {
		if _, err := playerLookPath(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

func autoPlayerCandidates() []string {
	if runtime.GOOS == "darwin" {
		return []string{"afplay"}
	}
	return []string{"paplay", "aplay", "ffplay", "mpv"}
}

func playerArgs(backend, filePath string) []string {
	switch backend {
	case "ffplay":
		return []string{"-nodisp", "-autoexit", "-loglevel", "quiet", filePath}
	case "mpv":
		return []string{"--no-video", filePath}
	default:
		return []string{filePath}
	}
}
