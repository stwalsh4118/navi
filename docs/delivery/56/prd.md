# PBI-56: Enhanced Audio — Sound Packs, Volume Control, and UX Polish

[View in Backlog](../backlog.md)

## Overview

Enhance navi's audio notification system with directory-based swappable sound packs, volume management via system player CLI flags, multiple sounds per event with random selection, and UX polish features including a mute toggle keybind, sound preview/test, and pack listing.

## Problem Statement

The current audio system (PBI-42) supports one sound file per event with no volume control, no variety, and no quick way to mute or test sounds. Users who want themed audio experiences must manually edit config paths, and hearing the same sound repeatedly for every event becomes monotonous. There is no way to adjust volume without changing system-wide settings, no way to quickly mute notifications, and no way to preview sounds without triggering a real status change.

## User Stories

- As a user, I want to switch between themed sound packs so that I can personalize my audio experience (e.g., sci-fi, retro, minimal)
- As a user, I want multiple sounds per event that trigger randomly so that notifications feel varied and less repetitive
- As a user, I want global and per-event volume control so that I can balance notification loudness without changing system volume
- As a user, I want a mute toggle keybind so that I can instantly silence and unmute notifications without editing config
- As a user, I want to preview/test sounds so that I can hear what a sound pack sounds like without waiting for real status changes
- As a user, I want to list available sound packs so that I can see what's installed and which is active

## Technical Approach

### Sound Pack Directory Structure

Sound packs live in `~/.config/navi/soundpacks/<pack-name>/`. Each pack is a directory containing audio files named by event:

```
~/.config/navi/soundpacks/
  starcraft/
    waiting.wav
    permission.wav
    done.wav
    error.wav
  retro/
    waiting.mp3
    permission.mp3
    done.mp3
    error.mp3
```

The active pack is selected via `pack:` in `sounds.yaml`. When a pack is set, the system resolves sound files from the pack directory by event name, auto-detecting the file extension (`.wav`, `.mp3`, `.ogg`).

### Multiple Sounds Per Event

A pack can include multiple files for the same event using a numeric suffix:

```
starcraft/
  waiting-1.wav
  waiting-2.wav
  waiting-3.wav
  done.wav          # single sound, no randomization
```

When multiple files exist for an event, one is selected randomly on each trigger. The system scans for files matching `<event>.*` and `<event>-<N>.*` patterns at pack load time and caches the list.

### Volume Control

Volume is controlled via CLI flags passed to the system audio player:

| Player | Flag | Range |
|--------|------|-------|
| `paplay` | `--volume=<0-65536>` | 0 = silent, 65536 = 100% (PulseAudio native) |
| `afplay` | `-v <0.0-1.0>` | Float volume multiplier |
| `mpv` | `--volume=<0-100>` | Percentage |
| `ffplay` | `-volume <0-100>` | Percentage |
| `aplay` | N/A | No volume flag — use global volume only via system mixer |

Config schema:

```yaml
volume:
  global: 80          # 0-100, applied to all sounds
  events:             # optional per-event multiplier (0.0-1.0)
    error: 1.0        # errors at full configured volume
    done: 0.7         # done at 70% of global
    waiting: 0.5      # waiting at 50% of global
```

The player translates the effective volume (global * event multiplier) into the backend-specific flag format.

### Mute Toggle

A keybind in the TUI (e.g., `m`) toggles mute state on/off. When muted:
- No sounds play
- No TTS announcements
- A visual indicator shows muted state (e.g., mute icon in the status bar or footer)
- Mute state is session-only (resets on restart)

### Sound Preview/Test

A CLI subcommand allows testing sounds:

```bash
navi sound test <event>          # play the sound for an event using active pack
navi sound test-all              # play all event sounds sequentially
navi sound list                  # list available packs and active pack
```

This reuses the existing audio player infrastructure.

### Backwards Compatibility

The existing `files:` config key continues to work. Resolution order:
1. If `pack:` is set, resolve from pack directory
2. If `files:` has explicit paths, use those (overrides pack for that event)
3. Fall back to no sound for unconfigured events

This means users can set a pack and override individual sounds via `files:`.

### Updated Config Schema

```yaml
# ~/.config/navi/sounds.yaml
enabled: true
pack: starcraft                  # NEW: active sound pack name

volume:                          # NEW: volume settings
  global: 80                     # 0-100
  events:                        # optional per-event multiplier
    error: 1.0
    done: 0.7

triggers:
  waiting: true
  permission: true
  working: false
  idle: false
  stopped: false
  done: true
  error: true

files:                           # existing per-file overrides (still supported)
  permission: ~/custom/alert.wav

tts:
  enabled: true
  template: "{session} — {status}"

cooldown_seconds: 5
player: auto
tts_engine: auto
```

### Components Affected

- `internal/audio/config.go` — new fields: `Pack`, `Volume`, multi-sound file lists
- `internal/audio/player.go` — volume flag support per backend
- `internal/audio/notifier.go` — random sound selection, volume passing, mute state
- `internal/tui/model.go` — mute toggle keybind handling, mute indicator rendering
- New: `internal/audio/pack.go` — pack directory scanning, file resolution, listing
- New: `cmd/` changes for `navi sound` subcommands

## UX/UI Considerations

- Mute toggle keybind (`m`) with visual indicator (e.g., muted icon or text in footer/status area)
- Mute state is session-only — does not persist to config on restart
- Sound preview commands provide immediate audible feedback
- Pack listing shows available packs with the active one highlighted

## Acceptance Criteria

1. Sound packs are directory-based (`~/.config/navi/soundpacks/<pack-name>/`) with files named by event. Active pack selected via `pack:` in `sounds.yaml`
2. Multiple sounds per event supported via numeric suffix (e.g., `waiting-1.wav`, `waiting-2.wav`). One selected randomly on each trigger
3. Global volume (0-100) and optional per-event volume multiplier configured in YAML, passed as CLI flags to the system audio player
4. Mute toggle keybind in the TUI instantly silences/unmutes all audio with a visual indicator
5. `navi sound test <event>` plays the configured sound for testing; `navi sound list` shows available packs
6. Backwards-compatible — existing `files:` config still works; `pack:` and `files:` can coexist with `files:` overriding pack sounds
7. All existing audio tests continue to pass; new features have test coverage
8. Audio API spec updated to reflect new config schema and public interfaces

## Dependencies

- **Depends on**: PBI-42 (Done) — foundation audio system
- **Blocks**: None
- **External**: System audio players (paplay, afplay, mpv, ffplay, aplay) — existing dependency, no new external deps

## Open Questions

None — all questions resolved during brainstorm.

## Related Tasks

_Tasks will be created when this PBI is approved via `/plan-pbi 56`._
