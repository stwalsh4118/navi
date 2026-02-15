# Audio API

Package: `internal/audio`

## Configuration

```go
type Config struct {
    Enabled         bool
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

## Audio Player

```go
type Player struct{}

func NewPlayer(override string) *Player
func (p *Player) Play(filePath string) error
func (p *Player) Available() bool
func (p *Player) Backend() string
```

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
```

Behavior:
- Checks `cfg.Enabled` and per-status `cfg.Triggers`
- Enforces per-session cooldown (`CooldownSeconds`)
- Plays status-mapped sound first (if configured)
- Speaks TTS announcement after sound when enabled
- Non-blocking execution and graceful no-op when backends unavailable

## TUI Integration

Model fields in `internal/tui/model.go`:
- `audioNotifier *audio.Notifier`
- `lastSessionStates map[string]string`

Integration points:
- Local poll updates (`sessionsMsg`) call status-change detection
- Remote poll updates (`remoteSessionsMsg`) call status-change detection
- On status transition, TUI calls `Notify(session, status)`
- First poll initializes state without emitting notifications
