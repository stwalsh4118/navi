# Audio API

Package: `internal/audio`

## Configuration

```go
type VolumeConfig struct {
    Global int                // 0-100, default 100
    Events map[string]float64 // per-event multiplier 0.0-1.0
}

func (v VolumeConfig) EffectiveVolume(event string) int // returns Global * multiplier, clamped 0-100

type Config struct {
    Enabled         bool
    Pack            string            // active sound pack name (empty = no pack)
    Volume          VolumeConfig
    Triggers        map[string]bool
    Files           map[string]string
    TTS             TTSConfig
    CooldownSeconds int
    Player          string
    TTSEngine       string
}

type TTSConfig struct {
    Enabled  bool
    Template string
}
```

Default config path:
- `DefaultConfigPath = "~/.config/navi/sounds.yaml"`

Public config functions:

```go
func DefaultConfig() *Config
func LoadConfig(path string) (*Config, error)
func ValidateConfig(cfg *Config)
```

## Sound Packs

```go
type PackInfo struct {
    Name       string
    EventCount int
    FileCount  int
}

func ScanPack(packDir string) (map[string][]string, error)
func ResolveSoundFiles(packName string) (map[string][]string, error)
func ListPacks() ([]PackInfo, error)
```

Pack directory: `~/.config/navi/soundpacks/<pack-name>/`
Supported extensions: `.wav`, `.mp3`, `.ogg`, `.flac`
File naming: `<event>.ext` (single) or `<event>-<N>.ext` (multi-sound variants)

## Audio Player

```go
type Player struct{}

func NewPlayer(override string) *Player
func (p *Player) Play(filePath string, volume int) error // volume 0-100; 0 skips playback
func (p *Player) Available() bool
func (p *Player) Backend() string
```

Volume flags per backend:
- `paplay`: `--volume=<0-65536>`
- `afplay`: `-v <0.0-1.0>`
- `mpv`: `--volume=<0-100>`
- `ffplay`: `-volume <0-100>`
- `aplay`: no volume flag

Detection behavior:
- macOS: `afplay`
- Linux/other: `paplay`, `aplay`, `ffplay`, `mpv`

## Text-to-Speech

```go
type TTS struct{}

func NewTTS(override string) *TTS
func (t *TTS) Speak(text string) error
func (t *TTS) Available() bool
func (t *TTS) Backend() string
func FormatAnnouncement(template, session, status string) string
```

Detection behavior:
- macOS: `say`
- Linux/other: `espeak-ng`, `espeak`, `spd-say`

## Notification Manager

```go
type Notifier struct{}

func NewNotifier(cfg *Config) *Notifier
func (n *Notifier) Notify(sessionName, newStatus string)
func (n *Notifier) Enabled() bool
func (n *Notifier) SetMuted(muted bool)
func (n *Notifier) IsMuted() bool
```

Behavior:
- Checks `cfg.Enabled` and per-status `cfg.Triggers`
- Mute check: if muted, skips all sound and TTS
- Enforces per-session cooldown (`CooldownSeconds`)
- Sound resolution order: `cfg.Files[status]` (override) → pack files (random if multiple) → no sound
- Calculates effective volume via `cfg.Volume.EffectiveVolume(status)`
- Plays sound with volume, then speaks TTS after delay when enabled
- Non-blocking execution and graceful no-op when backends unavailable
- SetMuted/IsMuted are thread-safe (session-only, not persisted)

## TUI Integration

Model fields in `internal/tui/model.go`:
- `audioNotifier *audio.Notifier`
- `lastSessionStates map[string]string`

Integration points:
- Local poll updates (`sessionsMsg`) call status-change detection
- Remote poll updates (`remoteSessionsMsg`) call status-change detection
- On status transition, TUI calls `Notify(session, status)`
- First poll initializes state without emitting notifications
