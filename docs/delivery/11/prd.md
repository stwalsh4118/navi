# PBI-11: Session Preview Pane

[View in Backlog](../backlog.md)

## Overview

Add a preview pane that shows recent output from a tmux session without fully attaching, allowing users to see what's happening in a session at a glance.

## Problem Statement

Currently, to see what Claude is doing in a session, users must fully attach to it. This is disruptive when monitoring multiple sessions. A preview pane would let users peek into a session's recent activity without context switching.

## User Stories

- As a user, I want to see recent output from a session so I can check its status without attaching
- As a user, I want to expand the preview for more detail
- As a user, I want the preview to update in real-time so I can watch Claude work

## Technical Approach

### Capturing Session Output

Use tmux's `capture-pane` command:
```bash
tmux capture-pane -t <session> -p -S -50
```
- `-p`: Print to stdout
- `-S -50`: Capture last 50 lines

### Layout Options

1. **Side Panel** (default)
   - Split screen: sessions list on left, preview on right
   - Preview updates when cursor moves
   - Resizable split (drag or keyboard)

2. **Bottom Panel**
   - Sessions list on top, preview on bottom
   - Good for wide terminals

3. **Inline Expand**
   - Press `space` to expand selected session in-place
   - Shows last N lines below the session entry
   - Press `space` again to collapse

### Implementation

1. Add preview mode toggle: `p` or `Tab`
2. Create `capturePane()` function that runs tmux capture-pane
3. Add preview component with its own styling
4. Poll preview content on timer (every 1-2s) or on cursor change
5. Handle ANSI colors in captured output (strip or render)

### Preview Styling

```
╭─ hyperion ──────────────────────────────────────────╮
│ Searching for files matching "*.go"...             │
│ Found 47 files                                      │
│                                                     │
│ I'll analyze these files to understand the         │
│ project structure. Let me start with main.go...    │
│                                                     │
│ Reading main.go...                                  │
│ ████████████████░░░░░░░░░░░░░░░░░░ 45%             │
╰─────────────────────────────────────────────────────╯
```

### Performance Considerations

- Don't capture while attached (unnecessary)
- Debounce capture when rapidly navigating list
- Cache captured content briefly
- Limit capture to reasonable line count

## UX/UI Considerations

- Preview should feel responsive (<100ms to update)
- ANSI colors should be preserved if possible
- Preview panel should be dismissible
- Keyboard-only navigation must work
- Consider scroll within preview pane

## Acceptance Criteria

1. Press `p` or `Tab` toggles preview panel visibility
2. Preview shows last N lines of selected session's tmux output
3. Preview updates when cursor moves to different session
4. Preview updates periodically while visible
5. ANSI escape codes are handled (stripped or rendered)
6. Preview panel can be resized
7. Layout adapts to terminal size
8. Preview is disabled when terminal too narrow

## Dependencies

- PBI-4: TUI rendering (base styling and layout)

## Open Questions

- Should preview support scrolling within the pane?
- Should we render ANSI colors or strip them?
- Should preview show input line (what Claude is about to do)?

## Related Tasks

[View Tasks](./tasks.md)
