# PBI-24: Themes & Keybindings

[View in Backlog](../backlog.md)

## Overview

Add customizable color themes and configurable keybindings so users can personalize the TUI to their preferences and workflows.

## Problem Statement

Users have different preferences for colors (dark/light themes, colorblind accessibility) and keybindings (vim users expect hjkl, others prefer arrows). A one-size-fits-all approach doesn't accommodate diverse workflows.

## User Stories

- As a user, I want to choose a color theme that matches my terminal aesthetic
- As a user, I want to customize keybindings to match my muscle memory
- As a user, I want a compact mode for when I have limited screen space
- As a user, I want a help overlay showing all available keybindings

## Technical Approach

### Theme Configuration

```yaml
# ~/.config/navi/theme.yaml
theme: dark  # or: light, solarized, nord, custom

# Custom theme definition
custom:
  colors:
    background: "#1e1e1e"
    foreground: "#d4d4d4"
    selected_bg: "#264f78"
    header_bg: "#252526"
    border: "#454545"

  status_icons:
    waiting:
      icon: "⏳"
      color: "#dcdcaa"
    done:
      icon: "✓"
      color: "#6a9955"
    permission:
      icon: "?"
      color: "#c586c0"
    working:
      icon: "⚙"
      color: "#4ec9b0"
    error:
      icon: "✗"
      color: "#f44747"
    offline:
      icon: "○"
      color: "#808080"
```

### Built-in Themes

1. **Dark** (default) - Dark background, vibrant status colors
2. **Light** - Light background, muted colors
3. **Solarized Dark** - Solarized color palette (dark variant)
4. **Solarized Light** - Solarized color palette (light variant)
5. **Nord** - Nord color scheme
6. **High Contrast** - Accessibility-focused, maximum contrast

### Keybinding Configuration

```yaml
# ~/.config/navi/keybindings.yaml
keybindings:
  navigation:
    up: ["k", "up"]
    down: ["j", "down"]
    top: ["g", "home"]
    bottom: ["G", "end"]

  actions:
    attach: ["enter", "l"]
    dismiss: ["d"]
    kill: ["x"]
    quit: ["q", "ctrl+c"]
    refresh: ["r"]

  views:
    search: ["/"]
    help: ["?"]
    preview: ["p", "tab"]
    split: ["\\"]

  # Disable default binding
  # kill: []  # uncomment to disable kill key
```

### Compact Mode

```yaml
# In theme.yaml
display:
  compact: false  # or true
  show_cwd: true
  show_message: true
  show_age: true
  row_padding: 1  # lines between sessions
```

Compact layout (1 line per session):
```
⏳ hyperion   ~/projects/hyperion              2m "Should I proceed?"
✓  api        ~/work/api                       15m Done
⚙  scratch    ~/tmp/scratch                    working
```

### Help Overlay

Press `?` to show:
```
╭─ Keybindings ──────────────────────────────────────────────╮
│                                                            │
│  Navigation                                                │
│  ↑/k        Move up                                        │
│  ↓/j        Move down                                      │
│  g/Home     Go to top                                      │
│  G/End      Go to bottom                                   │
│                                                            │
│  Actions                                                   │
│  Enter/l    Attach to session                              │
│  d          Dismiss notification                           │
│  x          Kill session                                   │
│  r          Refresh                                        │
│  q          Quit                                           │
│                                                            │
│  Views                                                     │
│  /          Search                                         │
│  ?          Show this help                                 │
│  p/Tab      Toggle preview                                 │
│  \          Toggle split view                              │
│                                                            │
│  Press any key to close                                    │
╰────────────────────────────────────────────────────────────╯
```

### Implementation

1. **Theme Engine**
   ```go
   type Theme struct {
       Colors      ColorPalette
       StatusIcons map[string]StatusStyle
   }

   func LoadTheme(name string) (*Theme, error)
   ```

2. **Keybinding Engine**
   ```go
   type Keybindings struct {
       Actions map[string][]key.Binding
   }

   func (k *Keybindings) Matches(action string, msg tea.KeyMsg) bool
   ```

3. **Help Component**
   - Generate from current keybindings config
   - Overlay component with dismiss on any key

## UX/UI Considerations

- Theme changes should apply immediately
- Invalid colors should fall back gracefully
- Keybinding conflicts should be detected and warned
- Help overlay should be context-aware (show relevant bindings)

## Acceptance Criteria

1. Themes configurable via theme.yaml
2. At least 4 built-in themes available
3. Custom colors definable per-element
4. Keybindings configurable via keybindings.yaml
5. Multiple keys can bind to same action
6. Keys can be unbound by setting to empty list
7. Compact mode reduces vertical space per session
8. `?` shows help overlay with current bindings
9. Theme and binding errors handled gracefully
10. Live theme switching (if possible) or on restart

## Dependencies

- PBI-4: TUI rendering (base styling system)

## Open Questions

- Should themes support 24-bit colors only or also 256/16?
- Should there be theme import/export?
- Should keybindings support key chords (e.g., "g g")?

## Related Tasks

See [Tasks](./tasks.md)
