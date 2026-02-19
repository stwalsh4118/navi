package audio

import (
	"fmt"
	"math"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/stwalsh4118/navi/internal/pathutil"
)

const (
	// DefaultConfigPath is the default path for audio notification configuration.
	DefaultConfigPath = "~/.config/navi/sounds.yaml"

	defaultCooldownSeconds = 5
	defaultBackendAuto     = "auto"
	defaultTTSTemplate     = "{session} — {status}"
	defaultGlobalVolume    = 100
	minVolume              = 0
	maxVolume              = 100
	minEventMultiplier     = 0.0
	maxEventMultiplier     = 1.0
)

// VolumeConfig defines global and per-event volume settings.
type VolumeConfig struct {
	Global int                `yaml:"global"`
	Events map[string]float64 `yaml:"events"`
}

// EffectiveVolume returns the effective volume for an event, clamped to 0-100.
// It multiplies Global by the per-event multiplier (defaulting to 1.0 if unset).
func (v VolumeConfig) EffectiveVolume(event string) int {
	multiplier := 1.0
	if m, ok := v.Events[event]; ok {
		multiplier = m
	}
	result := math.Round(float64(v.Global) * multiplier)
	if result < float64(minVolume) {
		return minVolume
	}
	if result > float64(maxVolume) {
		return maxVolume
	}
	return int(result)
}

// Config defines audio notification settings loaded from sounds.yaml.
type Config struct {
	Enabled         bool              `yaml:"enabled"`
	Pack            string            `yaml:"pack"`
	Volume          VolumeConfig      `yaml:"volume"`
	Triggers        map[string]bool   `yaml:"triggers"`
	Files           map[string]string `yaml:"files"`
	TTS             TTSConfig         `yaml:"tts"`
	CooldownSeconds int               `yaml:"cooldown_seconds"`
	Player          string            `yaml:"player"`
	TTSEngine       string            `yaml:"tts_engine"`
}

// TTSConfig configures text-to-speech announcements.
type TTSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Template string `yaml:"template"`
}

// DefaultConfig returns default audio configuration.
func DefaultConfig() *Config {
	return &Config{
		Enabled: false,
		Volume: VolumeConfig{
			Global: defaultGlobalVolume,
			Events: make(map[string]float64),
		},
		Triggers: map[string]bool{
			"waiting":    true,
			"permission": true,
			"working":    false,
			"idle":       false,
			"stopped":    false,
			"done":       true,
			"error":      true,
		},
		Files: make(map[string]string),
		TTS: TTSConfig{
			Enabled:  true,
			Template: defaultTTSTemplate,
		},
		CooldownSeconds: defaultCooldownSeconds,
		Player:          defaultBackendAuto,
		TTSEngine:       defaultBackendAuto,
	}
}

// LoadConfig reads and parses the audio configuration from YAML.
// Missing files return defaults and no error.
func LoadConfig(path string) (*Config, error) {
	configPath := path
	if configPath == "" {
		configPath = DefaultConfigPath
	}
	configPath = pathutil.ExpandPath(configPath)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse audio config: %w", err)
	}

	normalizeConfig(cfg)
	ValidateConfig(cfg)

	return cfg, nil
}

// ValidateConfig validates sound file configuration and logs warnings for missing files.
func ValidateConfig(cfg *Config) {
	if cfg == nil {
		return
	}

	if cfg.Volume.Global < minVolume || cfg.Volume.Global > maxVolume {
		fmt.Fprintf(os.Stderr, "Warning: volume.global %d outside valid range 0-100\n", cfg.Volume.Global)
	}
	for event, multiplier := range cfg.Volume.Events {
		if multiplier < minEventMultiplier || multiplier > maxEventMultiplier {
			fmt.Fprintf(os.Stderr, "Warning: volume.events.%s multiplier %.2f outside valid range 0.0-1.0\n", event, multiplier)
		}
	}

	for status, filePath := range cfg.Files {
		if filePath == "" {
			continue
		}
		if _, err := os.Stat(filePath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: audio file for status %q not found: %s\n", status, filePath)
		}
	}
}

func normalizeConfig(cfg *Config) {
	if cfg.Triggers == nil {
		cfg.Triggers = make(map[string]bool)
	}
	if cfg.Files == nil {
		cfg.Files = make(map[string]string)
	}
	if cfg.Volume.Events == nil {
		cfg.Volume.Events = make(map[string]float64)
	}
	if cfg.CooldownSeconds <= 0 {
		cfg.CooldownSeconds = defaultCooldownSeconds
	}
	if cfg.Player == "" {
		cfg.Player = defaultBackendAuto
	}
	if cfg.TTSEngine == "" {
		cfg.TTSEngine = defaultBackendAuto
	}
	if cfg.TTS.Template == "" {
		cfg.TTS.Template = defaultTTSTemplate
	}

	for status, filePath := range cfg.Files {
		cfg.Files[status] = pathutil.ExpandPath(filePath)
	}
}
