# PBI-5: Attach/Detach Loop & Keyboard Navigation

[View in Backlog](../backlog.md)

## Overview

Implement the core UX for navigating sessions with keyboard, attaching to tmux sessions, detaching back to the TUI, and dismissing notifications.

## Problem Statement

Users need to interact with their Claude sessions through the TUI. The key interaction pattern is: see a session needs attention → press Enter to attach → interact with Claude → detach → return to dashboard. Additionally, users need keyboard navigation and the ability to dismiss notifications.

## User Stories

- As a user, I want to navigate sessions with arrow keys so I can select which one to attach to
- As a user, I want to press Enter to attach to a session so I can interact with Claude
- As a user, I want to detach from tmux and automatically return to the TUI so I can manage multiple sessions
- As a user, I want to dismiss a notification without attaching so I can acknowledge it without context switching
- As a user, I want to manually refresh the session list

## Technical Approach

1. Implement `Update()` method to handle:
   - `up`/`down`/`k`/`j` for cursor navigation
   - `enter` to attach to selected session
   - `d` to dismiss (reset status to `working`)
   - `r` to force refresh
   - `q` to quit

2. Implement `attachSession(name string)` command using `tea.ExecProcess`

3. Handle `attachDoneMsg` to refresh state after detach

### Attach/Detach Loop (from PRD)

When the user presses Enter on a session:
1. TUI calls `bubbletea.ExecProcess` to run `tmux attach-session -t <session_name>`
2. The TUI process suspends while tmux takes over the terminal
3. When the user detaches from tmux (`prefix + d`), the tmux attach process exits
4. Bubble Tea regains control, re-renders the TUI with fresh state

### Dismiss Behavior (from PRD)

Pressing `d` on a session resets its status to `working` (clears the notification). This overwrites the JSON file with `"status": "working"`.

## UX/UI Considerations

- Cursor wraps at top/bottom of list
- After attach/detach, cursor stays on the same session if it still exists
- Dismiss provides a way to acknowledge without context switching

## Acceptance Criteria

1. Arrow keys (and vim-style j/k) move cursor up/down
2. Enter attaches to the selected tmux session
3. After tmux detach, TUI automatically reappears with fresh data
4. `d` key dismisses selected session (writes status as `working`)
5. `r` key forces an immediate refresh
6. `q` key quits the application
7. Cursor position is preserved across refreshes when possible

## Dependencies

- PBI-1: Core types (`Model`)
- PBI-3: Session polling (to refresh after detach)
- PBI-4: TUI rendering (to display updated state)

## Open Questions

None

## Related Tasks

See [Tasks](./tasks.md)
