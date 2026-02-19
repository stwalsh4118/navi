package audio

import (
	"fmt"
	"math/rand/v2"
	"os"
	"sync"
	"time"
)

const ttsDelayAfterSound = 150 * time.Millisecond

type soundPlayer interface {
	Play(filePath string, volume int) error
	Available() bool
	Backend() string
}

type speechEngine interface {
	Speak(text string) error
	Available() bool
	Backend() string
}

// Notifier orchestrates audio playback and TTS with per-session cooldown tracking.
type Notifier struct {
	cfg       *Config
	player    soundPlayer
	tts       speechEngine
	cooldowns map[string]time.Time
	packFiles map[string][]string

	mu       sync.Mutex
	muted    bool
	now      func() time.Time
	runAsync func(func())
	ttsDelay time.Duration
	randIntn func(n int) int
}

// NewNotifier creates a notifier from configuration and auto-detected backends.
func NewNotifier(cfg *Config) *Notifier {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	notifier := &Notifier{
		cfg:       cfg,
		player:    NewPlayer(cfg.Player),
		tts:       NewTTS(cfg.TTSEngine),
		cooldowns: make(map[string]time.Time),
		now:       time.Now,
		runAsync: func(fn func()) {
			go fn()
		},
		ttsDelay: ttsDelayAfterSound,
		randIntn: rand.IntN,
	}

	if cfg.Pack != "" {
		files, err := ResolveSoundFiles(cfg.Pack)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load sound pack %q: %v\n", cfg.Pack, err)
		} else {
			notifier.packFiles = files
		}
	}

	if cfg.Enabled {
		if !notifier.player.Available() {
			fmt.Fprintln(os.Stderr, "Warning: no audio player backend found; sound playback disabled")
		}
		if cfg.TTS.Enabled && !notifier.tts.Available() {
			fmt.Fprintln(os.Stderr, "Warning: no TTS backend found; speech announcements disabled")
		}
	}

	return notifier
}

// SetMuted sets the mute state (thread-safe).
func (n *Notifier) SetMuted(muted bool) {
	if n == nil {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.muted = muted
}

// IsMuted returns the current mute state (thread-safe).
func (n *Notifier) IsMuted() bool {
	if n == nil {
		return false
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.muted
}

// Enabled reports whether notifications can currently produce audio output.
func (n *Notifier) Enabled() bool {
	if n == nil || n.cfg == nil || !n.cfg.Enabled {
		return false
	}
	if n.player.Available() {
		return true
	}
	if n.cfg.TTS.Enabled && n.tts.Available() {
		return true
	}
	return false
}

// Notify triggers status-change notifications according to triggers and cooldown settings.
func (n *Notifier) Notify(sessionName, newStatus string) {
	if !n.Enabled() || sessionName == "" || newStatus == "" {
		return
	}

	if n.IsMuted() {
		return
	}

	triggerEnabled, ok := n.cfg.Triggers[newStatus]
	if !ok || !triggerEnabled {
		return
	}

	if !n.tryAcquireCooldown(sessionName) {
		return
	}

	volume := n.cfg.Volume.EffectiveVolume(newStatus)
	filePath := n.resolveSound(newStatus)

	soundPlayed := false
	if filePath != "" && n.player.Available() {
		if err := n.player.Play(filePath, volume); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to play notification sound: %v\n", err)
		} else {
			soundPlayed = true
		}
	}

	if !n.cfg.TTS.Enabled || !n.tts.Available() {
		return
	}

	announcement := FormatAnnouncement(n.cfg.TTS.Template, sessionName, newStatus)
	if soundPlayed {
		n.runAsync(func() {
			if n.ttsDelay > 0 {
				time.Sleep(n.ttsDelay)
			}
			if err := n.tts.Speak(announcement); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to announce notification: %v\n", err)
			}
		})
		return
	}

	if err := n.tts.Speak(announcement); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to announce notification: %v\n", err)
	}
}

// resolveSound returns the file path for a status, following the resolution order:
// 1. cfg.Files[status] (explicit override) — single file, no randomization
// 2. packFiles[status] — random selection if multiple files
// 3. empty string (no sound)
func (n *Notifier) resolveSound(status string) string {
	if filePath, ok := n.cfg.Files[status]; ok && filePath != "" {
		return filePath
	}

	if files, ok := n.packFiles[status]; ok && len(files) > 0 {
		if len(files) == 1 {
			return files[0]
		}
		return files[n.randIntn(len(files))]
	}

	return ""
}

func (n *Notifier) tryAcquireCooldown(sessionName string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	cooldown := time.Duration(n.cfg.CooldownSeconds) * time.Second
	now := n.now()

	if last, ok := n.cooldowns[sessionName]; ok {
		if now.Sub(last) < cooldown {
			return false
		}
	}

	n.cooldowns[sessionName] = now
	return true
}
