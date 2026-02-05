# PBI-7: Session Management Actions

[View in Backlog](../backlog.md)

## Overview

Add the ability to create new Claude Code sessions, kill existing sessions, and rename sessions directly from the TUI without needing to use tmux commands separately.

## Problem Statement

Currently, users must exit the TUI or open another terminal to create new Claude sessions, kill sessions, or rename them. This breaks the workflow and requires knowledge of tmux commands. Users should be able to manage the full session lifecycle from within navi.

## User Stories

- As a user, I want to press a key to create a new Claude session so I can start new work without leaving the dashboard
- As a user, I want to kill a session from the TUI so I can clean up finished or stuck sessions
- As a user, I want to rename sessions so I can give them meaningful names that reflect the work being done

## Technical Approach

### New Session Creation

1. Press `n` to open a new session dialog
2. Prompt for:
   - Session name (default: auto-generated or directory-based)
   - Working directory (default: current directory or home)
3. Execute: `tmux new-session -d -s <name> -c <dir> "claude"`
4. Session appears in list after next poll

### Kill Session

1. Press `x` on selected session
2. Show confirmation dialog: "Kill session '<name>'? (y/n)"
3. On confirm: `tmux kill-session -t <name>`
4. Delete corresponding status JSON file
5. Session removed from list after next poll

### Rename Session

1. Press `R` on selected session
2. Show text input for new name
3. Execute: `tmux rename-session -t <old> <new>`
4. Rename status JSON file to match
5. Update display after next poll

### Implementation Details

- Use Bubble Tea's text input component for name entry
- Add confirmation dialog component for destructive actions
- Handle edge cases:
  - Session name conflicts
  - Invalid characters in names
  - Killing attached sessions (warn user)

## UX/UI Considerations

- New session dialog should be minimal and fast
- Kill confirmation must be explicit to prevent accidents
- Rename should preserve cursor position in list
- Show loading indicator while tmux commands execute
- Display error messages inline if commands fail

## Acceptance Criteria

1. Pressing `n` opens new session dialog with name and directory inputs
2. New sessions are created in tmux and appear in the list
3. Pressing `x` prompts for confirmation before killing a session
4. Killed sessions are removed from tmux and status files are cleaned up
5. Pressing `R` allows renaming with validation for valid tmux session names
6. All operations show appropriate error messages on failure
7. Footer keybindings are updated to show new actions

## Dependencies

- PBI-5: Attach/detach mechanism (for understanding session handling)

## Open Questions

- Should new sessions support custom commands beyond `claude`?
- Should we support session duplication (clone)?

## Related Tasks

See [Tasks](./tasks.md)
