# PBI-58: In-TUI Sound Pack Picker

[View in Backlog](../backlog.md)

## Overview

Add an in-TUI overlay for browsing and switching sound packs. Users press `S` to open a picker showing all installed packs with event/file counts, preview sounds, and select a pack — which is immediately applied and persisted to `sounds.yaml`.

## Problem Statement

PBI-56 introduced sound packs with `ListPacks()`, `ScanPack()`, and the `navi sound list/test` CLI. However, switching packs currently requires manually editing `sounds.yaml` and restarting navi. Users should be able to browse, preview, and switch packs without leaving the TUI.

## User Stories

- As a user, I want to press `S` to see all available sound packs so that I can browse what's installed.
- As a user, I want to select a pack from the picker so that it takes effect immediately without restarting.
- As a user, I want my pack selection persisted to `sounds.yaml` so that it survives restarts.
- As a user, I want to preview a pack's sounds before committing to it so that I can hear what it sounds like.

## Technical Approach

### TUI Layer (`internal/tui/`)
- Add `S` keybind to open a pack picker overlay (similar to existing filter/dialog patterns)
- Render a scrollable list of packs from `audio.ListPacks()` with the active pack marked
- Arrow keys to navigate, `Enter` to select, `Esc` to cancel
- Optional preview key (e.g., `Space` or `p`) to play a sample sound from the highlighted pack

### Audio Layer (`internal/audio/`)
- Add `Notifier.SetPack(packName string) error` to hot-swap pack files at runtime (thread-safe)
- Add `SavePackSelection(configPath, packName string) error` to update the `pack:` field in `sounds.yaml` without clobbering other settings
- Reuse existing `ListPacks()`, `ResolveSoundFiles()`, and `Player.Play()` APIs

### Config Persistence
- Read existing YAML, update only the `pack:` field, write back
- Preserve all other user settings (volume, triggers, files, TTS, etc.)
- Create the config file with sensible defaults if it doesn't exist yet

## UX/UI Considerations

- Overlay should show: pack name, event count, file count, active marker
- Match existing TUI styling (borders, colors, selection highlight)
- Show "No sound packs installed" with a hint about the soundpacks directory if empty
- Active pack should be clearly marked (e.g., checkmark or highlight)

## Acceptance Criteria

1. Pressing `S` in session view opens a scrollable pack picker overlay listing all installed packs
2. Each pack shows name, event count, and file count (from `ListPacks()`)
3. The currently active pack is visually marked
4. Selecting a pack with `Enter` hot-swaps it on the running Notifier (immediate effect)
5. The selection is persisted to `sounds.yaml` (survives restart)
6. Pressing `Esc` closes the picker without changing the pack
7. When no packs are installed, the picker shows a helpful message
8. Existing keybinds and audio functionality are unaffected

## Dependencies

- **Depends on**: PBI-56 (sound pack infrastructure must be complete)
- **Blocks**: None
- **External**: None

## Open Questions

- Should there be a preview key to play a sample sound from the highlighted pack? (Proposed: `Space` or `p` plays a random "done" sound from the pack)
- Should the picker also allow adjusting volume, or keep that config-file-only for now?

## Related Tasks

[View Tasks](./tasks.md)
