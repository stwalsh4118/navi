# PBI-44: Background Attach Monitor and tmux Status Bar

[View in Backlog](../backlog.md)

## Overview

Keep Navi's background monitoring alive while the user is attached to a tmux session. Before `tea.ExecProcess` yields terminal control, a background goroutine starts that independently polls session status files and triggers audio notifications on status transitions. Additionally, provide a `navi status` CLI command that outputs a condensed session summary for use in tmux's status bar configuration.

## Problem Statement

**Current state**: When a user attaches to a tmux session, `tea.ExecProcess` pauses the Bubble Tea event loop. All polling stops. Audio notifications stop. The user is blind to other session status changes until they detach.

**Desired state**: Audio notifications continue firing while the user is attached to a session. A `navi status` command provides a visual summary in the tmux status bar, so the user can glance at session status without detaching.

## User Stories

- As a user, I want audio notifications to continue playing while I'm attached to a tmux session so that I hear when other sessions need attention.
- As a user, I want a `navi status` command that outputs session status so I can display it in my tmux status bar.
- As a user, I want the background monitor to stop cleanly when I detach so there are no duplicate notifications or resource leaks.

## Technical Approach

### Background Attach Monitor

Create an `internal/monitor` package with an `AttachMonitor` struct that:
1. Runs a polling loop in a goroutine (reusing session file reading logic from `internal/tui/sessions.go`)
2. Tracks last-known session states (same pattern as `Model.lastSessionStates`)
3. Calls `audio.Notifier.Notify()` on status transitions
4. Accepts a `context.Context` for clean cancellation
5. Is started just before `tea.ExecProcess` and cancelled when the user detaches

**State handoff**: Before starting the background monitor, the TUI hands off its current `lastSessionStates` map. When the monitor stops (user detaches), it hands back its updated state map so the TUI doesn't re-fire the same transitions.

### CLI Command: `navi status`

Add CLI argument detection in `main.go`. When `os.Args[1] == "status"`:
- Read session status files from `~/.claude-sessions/`
- Include both local and remote session status files
- Count sessions by status
- Output a summary and exit

**Default output** (priority alerts only):
```
1 waiting, 2 permission
```
Empty string if no sessions need attention.

**`--verbose` output** (full counts):
```
3 working, 1 waiting, 2 permission, 1 idle
```

**`--format=tmux` output**: Same as default but plain text suitable for tmux `status-right`. No special tmux color codes — users style via their tmux config.

### tmux Integration

Users configure their tmux status bar:
```bash
set -g status-right '#(navi status)'
set -g status-interval 5
```

The `navi status` command is one-shot (run, print, exit), so tmux's `status-interval` handles periodic refresh.

## UX/UI Considerations

N/A — this is backend/CLI only. The only "UI" is the tmux status bar string, which is plain text.

## Acceptance Criteria

1. When the user attaches to a session via `tea.ExecProcess`, a background goroutine starts polling session status files and triggering audio notifications on status transitions
2. When the user detaches and returns to Navi, the background goroutine stops cleanly with no resource leaks
3. No duplicate notifications: the background monitor and TUI hand off session state so the same transition doesn't fire twice
4. `navi status` prints a one-line summary of sessions needing attention (waiting/permission counts) and exits with code 0
5. `navi status --verbose` prints full session counts by status
6. `navi status --format=tmux` outputs plain text suitable for tmux status bar
7. `navi status` exits cleanly with empty output when no sessions need attention (default mode) or when no sessions exist
8. The background monitor handles both local and remote session status files
9. All existing tests pass; new tests cover background monitor lifecycle, state handoff, and CLI output formatting

## Dependencies

- **Depends on**: PBI-42 (Audio Notifications, Done) — reuses `audio.Notifier`
- **Blocks**: None
- **External**: None — uses only existing packages

## Open Questions

- None (resolved during brainstorm: include remote sessions, keep tmux output plain)

## Related Tasks

_Tasks will be created when this PBI is approved via `/plan-pbi 44`._
