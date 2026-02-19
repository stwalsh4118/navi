# CLI API

Package: `internal/cli`

## Status Command

```go
func RunStatus(args []string) int
```

Flags:
- `--verbose`: include all non-zero status counts
- `--format=plain|tmux`: plain text output mode (tmux currently same formatting)

Behavior:
- Reads sessions from `session.ReadStatusFiles(pathutil.ExpandPath(session.StatusDir))`
- Default output includes only priority statuses: `waiting`, `permission`
- Verbose output includes non-zero counts in this order: `working`, `waiting`, `permission`, `idle`, `stopped`
- Prints empty output when nothing matches the selected mode
- Returns exit code `0` on success and `1` on flag/IO errors

## Sound Command

```go
func RunSound(args []string) int
```

Subcommands:
- `navi sound test <event>` — play the configured sound for an event
- `navi sound test-all` — play all enabled trigger event sounds sequentially
- `navi sound list` — list available sound packs and show active pack

Behavior:
- Loads audio config from `audio.LoadConfig("")`
- Sound resolution: `cfg.Files[event]` (override) → pack files → none
- Uses `audio.NewPlayer(cfg.Player)` for playback
- `test-all` plays with ~1.5s delay between events
- `list` shows pack name, event count, file count, active marker
- Returns exit code `0` on success, `1` on error
