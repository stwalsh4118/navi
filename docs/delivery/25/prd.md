# PBI-25: Mouse Support

[View in Backlog](../backlog.md)

## Overview

Add mouse support to the TUI for clicking to select sessions, scrolling the list, and interacting with UI elements, making navi more accessible to users who prefer mouse-driven interfaces.

## Problem Statement

While keyboard navigation is efficient for power users, many users instinctively reach for their mouse. Lack of mouse support makes the TUI feel unresponsive to these users and limits accessibility.

## User Stories

- As a user, I want to click on a session to select it
- As a user, I want to scroll the session list with my mouse wheel
- As a user, I want to double-click to attach to a session
- As a user, I want to click buttons in dialogs

## Technical Approach

### Bubble Tea Mouse Support

Bubble Tea provides built-in mouse support:

```go
func main() {
    p := tea.NewProgram(
        initialModel(),
        tea.WithAltScreen(),
        tea.WithMouseCellMotion(),  // Enable mouse tracking
    )
}
```

### Mouse Events to Handle

| Event | Action |
|-------|--------|
| Left click on session | Select that session |
| Double-click on session | Attach to session |
| Right-click on session | Context menu (if implemented) |
| Scroll wheel | Scroll session list |
| Click on header | Collapse/expand (if applicable) |
| Click on footer action | Execute that action |

### Implementation

1. **Click Detection**
   ```go
   func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
       switch msg := msg.(type) {
       case tea.MouseMsg:
           if msg.Type == tea.MouseLeft {
               if session := m.sessionAtY(msg.Y); session != nil {
                   m.cursor = m.indexOfSession(session)
               }
           }
           if msg.Type == tea.MouseLeft && msg.Action == tea.MouseActionRelease {
               // Check for double-click timing
               if m.isDoubleClick(msg) {
                   return m, attachSession(m.selectedSession())
               }
           }
       }
   }
   ```

2. **Scroll Handling**
   ```go
   case tea.MouseMsg:
       if msg.Type == tea.MouseWheelUp {
           m.cursor = max(0, m.cursor-1)
       }
       if msg.Type == tea.MouseWheelDown {
           m.cursor = min(len(m.sessions)-1, m.cursor+1)
       }
   ```

3. **Hit Testing**
   - Track Y positions of rendered sessions
   - Map click Y coordinate to session index
   - Handle variable row heights (with messages)

### Clickable Footer

Make footer actions clickable:
```
╭─────────────────────────────────────────────────────────────╮
│  [↑↓] navigate  [⏎] attach  [d] dismiss  [q] quit         │
╰─────────────────────────────────────────────────────────────╯
```

Click on `[⏎] attach` executes attach action.

### Context Menu (Optional)

Right-click on session shows menu:
```
┌──────────────────┐
│ Attach           │
│ Kill             │
│ Rename           │
│ ─────────────────│
│ Add Tag          │
│ Bookmark         │
│ ─────────────────│
│ View Details     │
└──────────────────┘
```

### Configuration

```yaml
# ~/.config/navi/config.yaml
mouse:
  enabled: true
  double_click_ms: 400
  scroll_speed: 3  # sessions per wheel tick
  context_menu: true
```

### Terminal Compatibility

Mouse support varies by terminal:
- Full support: iTerm2, Kitty, Alacritty, Windows Terminal
- Partial support: gnome-terminal, Terminal.app
- Limited/None: some older terminals

Detect capabilities and degrade gracefully.

## UX/UI Considerations

- Mouse should complement, not replace, keyboard navigation
- Visual feedback on hover (if terminal supports)
- Double-click timing should be configurable
- Context menu should be keyboard-dismissable
- Don't break tmux mouse mode

## Acceptance Criteria

1. Single click selects a session
2. Double-click attaches to a session
3. Mouse wheel scrolls the session list
4. Scroll speed is configurable
5. Footer actions are clickable
6. Mouse can be disabled via config
7. Works correctly within tmux (mouse passthrough)
8. Graceful degradation on unsupported terminals
9. Double-click timing configurable
10. Right-click context menu (optional)

## Dependencies

- PBI-4: TUI rendering (for hit testing coordinates)

## Open Questions

- Should we support drag-and-drop for reordering?
- Should preview pane support mouse scrolling independently?
- Should there be mouse-based resize for split views?

## Related Tasks

See [Tasks](./tasks.md)
