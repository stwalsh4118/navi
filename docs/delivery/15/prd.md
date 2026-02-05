# PBI-15: Split View

[View in Backlog](../backlog.md)

## Overview

Add a split view mode that shows two session previews side-by-side so users can monitor and compare related sessions simultaneously.

## Problem Statement

When working on related tasks across multiple Claude sessions (e.g., frontend and backend), users need to monitor both without constantly switching. A split view allows parallel monitoring of two sessions.

## User Stories

- As a user, I want to view two sessions side-by-side so I can monitor related work
- As a user, I want to compare session outputs to coordinate changes
- As a user, I want to switch focus between split panes without losing the view

## Technical Approach

### Layout

```
╭─────────────────────────────────────────────────────────────────────────╮
│  Claude Sessions                                           2 active   │
╰─────────────────────────────────────────────────────────────────────────╯

╭─ hyperion ─────────────────────╮ ╭─ api ───────────────────────────────╮
│ Working on frontend component  │ │ Implementing REST endpoint...      │
│                                 │ │                                     │
│ Reading Button.tsx...           │ │ Writing handlers/user.go...         │
│ I'll update the onClick handler │ │ Added CreateUser endpoint           │
│ to include the new validation.  │ │                                     │
│                                 │ │ Now I'll add the validation:        │
│ ```tsx                          │ │ ```go                               │
│ onClick={() => {                │ │ func (h *Handler) CreateUser(       │
│   if (validate(input)) {        │ │   w http.ResponseWriter,            │
│ ```                             │ │ ```                                 │
╰─────────────────────────────────╯ ╰─────────────────────────────────────╯

╭─────────────────────────────────────────────────────────────────────────╮
│  ←/→ switch focus  ⏎ attach  Tab toggle split  q quit                 │
╰─────────────────────────────────────────────────────────────────────────╯
```

### Implementation

1. **Split Mode Toggle**
   - Press `Tab` to toggle split view (if PBI-11 preview is implemented)
   - Press `\` for dedicated split toggle

2. **Pane Management**
   - Left/right panes each have their own "cursor"
   - `←`/`→` or `h`/`l` switch focus between panes
   - `↑`/`↓` or `j`/`k` navigate within focused pane
   - Each pane shows a session preview (captured output)

3. **Session Assignment**
   - Focused pane shows currently selected session
   - Navigate to assign different session to focused pane
   - Press `Enter` to attach to focused pane's session

4. **State Model**
   ```go
   type Model struct {
       // ... existing fields
       splitMode     bool
       leftSession   string  // session name for left pane
       rightSession  string  // session name for right pane
       focusedPane   Pane    // Left or Right
   }
   ```

5. **Rendering**
   - Calculate pane widths based on terminal width
   - Minimum width threshold (disable split if too narrow)
   - Use lipgloss.JoinHorizontal for side-by-side layout

### Advanced Features

- **Synchronized Scroll**: Option to scroll both panes together
- **Quick Swap**: Press `X` to swap left and right sessions
- **Pin Session**: Lock a session to one pane while browsing in other

## UX/UI Considerations

- Focused pane should have visible border highlight
- Minimum terminal width required (suggest 120+ columns)
- Graceful fallback to single view on narrow terminals
- Session names in pane headers for clarity

## Acceptance Criteria

1. `Tab` or `\` toggles split view mode
2. Two session previews displayed side-by-side
3. `←`/`→` switches focus between panes
4. `↑`/`↓` navigates sessions within focused pane
5. Each pane shows captured session output
6. Press `Enter` attaches to focused pane's session
7. Pane headers show session name and status
8. Minimum width enforced (fallback to single view)
9. Split view state persists during session updates

## Dependencies

- PBI-11: Session preview pane (core preview functionality)
- PBI-4: TUI rendering (layout foundation)

## Open Questions

- Should split support more than two panes?
- Should pane sizing be adjustable?
- Should split configuration persist across restarts?

## Related Tasks

See [Tasks](./tasks.md)
