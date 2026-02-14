# PBI-42: Audio Notifications with Sound Packs and TTS Session Announcements

[View in Backlog](../backlog.md)

## Overview

Add configurable audio notifications to navi that play user-provided sound files and announce session names via text-to-speech when session statuses change. This gives users audio awareness of their Claude sessions without needing to watch the TUI.

## Problem Statement

Users often run navi in a terminal while working in other applications or monitors. Visual status changes in the TUI go unnoticed unless the user is actively watching. Audio feedback — themed sound effects plus spoken session names — provides immediate, hands-free awareness of which session changed and what it needs, inspired by the satisfying audio cues of games like Warcraft and StarCraft.

## User Stories

- As a user, I want to hear a sound when a session's status changes so that I know something needs attention without looking at the screen
- As a user, I want to assign different sound files to different statuses so that I can distinguish between permission requests, completions, and errors by ear
- As a user, I want the session name spoken aloud when a status changes so that I know which session needs attention
- As a user, I want to configure which status transitions trigger sounds so that I'm not overwhelmed by noise
- As a user, I want a cooldown period so that rapid status changes don't spam me with sound

## Technical Approach

### State-Change Detection

The TUI already polls sessions every 500ms (`session.PollInterval`). The audio system needs to:

1. Track previous session statuses in the TUI model (map of session name → last known status)
2. On each poll result (`sessionsMsg` / `remoteSessionsMsg`), compare new statuses against previous
3. For each detected transition, check config and fire audio events

This detection lives in `internal/tui/model.go` within the existing `Update()` message loop.

### Audio Playback

A new `internal/audio/` package handles sound playback:

- **Player abstraction**: Auto-detect available system binary
  - Linux: `paplay` (PulseAudio), `aplay` (ALSA), `ffplay`, `mpv`
  - macOS: `afplay`
- **Non-blocking**: Launch playback in a goroutine via `exec.Command`, fire-and-forget
- **Supported formats**: `.wav`, `.mp3`, `.ogg` (depends on system player capabilities)
- **No Go audio library dependency**: Uses system binaries only, keeping the binary small

### Text-to-Speech

Same package handles TTS:

- **TTS backends**:
  - macOS: `say` (built-in)
  - Linux: `espeak-ng`, `espeak`, or `spd-say`
- **Announcement format**: Configurable template, default: `"{session} — {status}"`
- **Non-blocking**: Same fire-and-forget goroutine pattern
- **Sequencing**: TTS plays after the sound effect (small delay)

### Configuration

```yaml
# ~/.config/navi/sounds.yaml
enabled: true

# Which status transitions trigger audio
triggers:
  waiting: true
  permission: true
  working: false
  idle: false
  stopped: false
  done: true
  error: true

# Map statuses to sound files (absolute paths or relative to config dir)
files:
  waiting: ~/sounds/waiting.wav
  permission: ~/sounds/permission.mp3
  done: ~/sounds/done.ogg
  error: ~/sounds/error.wav

# Text-to-speech settings
tts:
  enabled: true
  template: "{session} — {status}"

# Playback settings
cooldown_seconds: 5
player: auto       # auto-detect, or specify: paplay, afplay, aplay, mpv
tts_engine: auto   # auto-detect, or specify: say, espeak-ng, espeak, spd-say
```

### Integration Points

- **Model**: `internal/tui/model.go` — add previous-state tracking and audio trigger logic in `Update()`
- **New package**: `internal/audio/` — player detection, sound playback, TTS, cooldown management
- **Config loading**: New config loader for `~/.config/navi/sounds.yaml`

## UX/UI Considerations

- N/A — this is an audio-only feature with no visual TUI changes
- Audio playback must never block or slow the TUI event loop
- If no audio player or TTS engine is found, log a warning at startup and silently skip audio events
- Cooldown is per-session to prevent the same session from spamming sounds

## Acceptance Criteria

1. Status transitions trigger playback of user-provided sound files (`.wav`, `.mp3`, `.ogg`)
2. Each status (`waiting`, `permission`, `working`, `idle`, `stopped`, `done`, `error`) can be individually mapped to a sound file in config
3. Text-to-speech announces the session name and status when a sound plays
4. User configures triggers, sound file paths, TTS settings, and cooldown via `~/.config/navi/sounds.yaml`
5. Auto-detects available system audio player (`paplay`, `afplay`, `aplay`) and TTS engine (`espeak-ng`, `say`)
6. Audio playback is non-blocking — never stalls the TUI
7. Per-session cooldown prevents sound spam from rapid status changes (configurable interval, default 5s)
8. Graceful degradation — warns if no audio player/TTS found, does not crash or error

## Dependencies

- **Depends on**: None (session polling from PBI-3 already exists and is stable)
- **Blocks**: None
- **External**: System audio players (`paplay`, `afplay`, `aplay`) and TTS engines (`espeak-ng`, `say`) — not Go library dependencies

## Open Questions

- Should team agent status changes (individual agents within a team session) also trigger sounds, or only the main session status?
- Should there be a global mute toggle / keybind in the TUI to quickly silence sounds?

## Related Tasks

[View Tasks](./tasks.md)
