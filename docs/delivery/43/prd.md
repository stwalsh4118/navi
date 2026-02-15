# PBI-43: Embedded Terminal Mode

[View in Backlog](../backlog.md)

## Overview

Replace the current `tea.ExecProcess` attach model with an embedded terminal that renders the tmux session inside the Navi TUI. The user sees a Navi top bar and bottom status bar with the tmux session occupying the middle of the screen. The Bubble Tea event loop stays running, so session polling, git polling, audio notifications, and all other background activity continues uninterrupted while the user works inside a session.

## Problem Statement

**Current state**: When a user presses Enter to attach to a tmux session, Navi calls `tea.ExecProcess` which yields full terminal control to `tmux attach-session`. The Bubble Tea event loop pauses — no polling, no notifications, no audio alerts. The user is blind to other session status changes until they detach.

**Desired state**: The tmux session renders inside the Navi UI with minimal chrome (top bar, bottom status bar). The event loop stays alive. Session status polling continues, audio notifications fire, and the user can glance at the status bar to see if other sessions need attention — all without detaching.

## User Stories

- As a user, I want tmux sessions embedded inside the Navi UI so that polling, notifications, and audio alerts continue working while I interact with a session.
- As a user, I want a keybinding (Ctrl+\) to toggle focus between the embedded terminal and Navi chrome so that I can access Navi features without leaving the session.
- As a user, I want the option to do a full raw tmux attach (the old behavior) as a fallback when I need maximum terminal fidelity.

## Technical Approach

### Library Choice

Use `taigrr/bubbleterm` as the primary terminal emulation component. It provides:
- Ready-made Bubble Tea model integration (Init/Update/View)
- PTY management (spawns commands, reads output, forwards input)
- Focus management (Focus/Blur/Focused)
- Dynamic resizing
- Process exit monitoring

`bubbleterm` internally uses `charmbracelet/x/vt` for terminal emulation and handles ANSI parsing, cursor positioning, colors, and alternate screen buffer.

If `bubbleterm` proves insufficient, the fallback path is direct integration with `charmbracelet/x/vt` + `charmbracelet/x/xpty` + `creack/pty`.

### Architecture

```
┌─────────────────────────────────┐
│  Top bar: session name, status  │  ← Bubble Tea renders (lipgloss)
├─────────────────────────────────┤
│                                 │
│   bubbleterm.Model              │  ← PTY running `tmux attach -t <name>`
│   (terminal emulation)          │
│                                 │
├─────────────────────────────────┤
│  Bottom bar: alerts, hints      │  ← Bubble Tea renders (lipgloss)
└─────────────────────────────────┘
```

### Key Components

1. **`internal/terminal/` package**: Wrapper around `bubbleterm` that:
   - Creates a PTY running `tmux attach-session -t <name>` (local) or the SSH command (remote)
   - Manages focus state and the escape keybinding (Ctrl+\)
   - Handles resize propagation from Navi's `tea.WindowSizeMsg` to the PTY
   - Forwards scroll wheel events to the PTY

2. **New view mode in `Model`**: An `embeddedMode` flag (or a new `DialogMode` value) that switches the `View()` output from the session list to the embedded terminal layout.

3. **Input routing**: When in embedded mode and the terminal is focused, all key events except the escape key (Ctrl+\) are forwarded to the PTY. When Navi chrome is focused, normal keybindings apply (e.g., navigate the status bar, view notifications).

4. **Dual attach modes**: Enter = embedded mode (default), Shift+Enter (or configurable) = full raw `tea.ExecProcess` attach (fallback).

### Scroll Support

Mouse scroll wheel events are translated to terminal scroll sequences and forwarded to the PTY. This covers the user's primary mouse interaction (scrolling through output). Full mouse positioning (click at cell X,Y) is out of scope for this PBI.

### Resize Handling

When `tea.WindowSizeMsg` arrives:
1. Navi recalculates the embedded terminal area (total height minus top bar minus bottom bar)
2. Calls `bubbleterm`'s resize method to update the PTY dimensions
3. The PTY sends `SIGWINCH` to tmux, which redraws at the new size

## UX/UI Considerations

### Minimal Chrome (v1)

- **Top bar** (1 line): Session name, status icon, working directory
- **Bottom bar** (1 line): Notification alerts (e.g., "2 sessions waiting"), keybinding hint ("Ctrl+\ toggle focus | Shift+Enter raw attach")
- Terminal area gets all remaining rows

### Focus Indicator

When Navi chrome is focused (after pressing Ctrl+\), the top/bottom bars highlight to indicate focus is on Navi. The terminal area dims slightly or shows a subtle border change. Pressing Ctrl+\ again returns focus to the terminal.

### Returning to Session List

- Press Ctrl+\ to focus Navi chrome, then press Esc or q to exit embedded mode and return to the session list
- If the tmux session exits (user types `exit` or the process ends), automatically return to session list

## Acceptance Criteria

1. Pressing Enter on a session opens it in embedded terminal view with Navi top bar (session name, status icon, CWD) and bottom status bar (notification alerts, keybinding hints)
2. The Bubble Tea event loop continues running — session polling, git polling, and audio notifications remain active while in embedded mode
3. Keyboard input is forwarded to the embedded tmux session; Ctrl+\ toggles focus between the terminal and Navi chrome
4. Scroll wheel events inside the embedded terminal are forwarded to the PTY
5. Terminal resize events propagate correctly through Navi -> PTY -> tmux (no rendering glitches on resize)
6. Shift+Enter (or configurable keybinding) triggers full raw `tea.ExecProcess` attach as a fallback
7. Returning from embedded mode (via Ctrl+\ then Esc, or session exit) restores the normal session list view
8. Remote sessions (via SSH) work in embedded mode using the SSH command instead of local tmux attach
9. All existing tests continue to pass; new tests cover embedded terminal creation, focus toggling, and mode transitions

## Dependencies

- **Depends on**: None (PBI-5 attach/detach is Done and provides the baseline)
- **Blocks**: None directly, but enables future PBI-15 (Split View)
- **External**: `taigrr/bubbleterm` (0BSD license), which depends on `charmbracelet/x/vt`, `charmbracelet/x/xpty`

## Open Questions

- Performance: Does `bubbleterm`'s full-screen redraw per frame cause noticeable latency with tmux output? May need damage tracking optimization in a follow-up.
- `bubbleterm` known issue: backspace rendering quirks. The author recommends running tmux inside the emulator (which is exactly our use case), so this may be a non-issue.
- Should the escape keybinding (Ctrl+\) be configurable via settings, or is hardcoded sufficient for v1?

## Related Tasks

_Tasks will be created when this PBI is approved via `/plan-pbi 43`._
