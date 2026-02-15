package audio

import (
	"fmt"
	"os"
	"sync"
	"time"
)

const ttsDelayAfterSound = 150 * time.Millisecond

type soundPlayer interface {
	Play(filePath string) error
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

	mu       sync.Mutex
	now      func() time.Time
	runAsync func(func())
	ttsDelay time.Duration
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

	triggerEnabled, ok := n.cfg.Triggers[newStatus]
	if !ok || !triggerEnabled {
		return
	}

	if !n.tryAcquireCooldown(sessionName) {
		return
	}

	soundPlayed := false
	if filePath, ok := n.cfg.Files[newStatus]; ok && filePath != "" && n.player.Available() {
		if err := n.player.Play(filePath); err != nil {
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
