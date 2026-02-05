# PBI-4: TUI Rendering & Styling

[View in Backlog](../backlog.md)

## Overview

Build the Bubble Tea TUI with Lip Gloss styling that displays all Claude sessions with status icons, session names, working directories, and messages.

## Problem Statement

Users need a visual dashboard to see all running Claude Code sessions at a glance. The TUI must clearly indicate which sessions need attention through colored status icons and display relevant context like working directory and Claude's message.

## User Stories

- As a user, I want to see all my Claude sessions in a dashboard so I can monitor multiple projects
- As a user, I want clear status icons so I can instantly see which sessions need attention
- As a user, I want to see the working directory and Claude's message for context

## Technical Approach

1. Implement Lip Gloss styles for:
   - Status icons with appropriate colors
   - Session name (bold)
   - Age display (right-aligned)
   - Working directory (dimmed)
   - Message (dimmed/italic, truncated)
   - Selected row highlight
   - Header and footer boxes

2. Implement the `View()` method that renders:
   - Header with title and session count
   - Session list with cursor indicator
   - Footer with keybindings

### TUI Layout (from PRD)

```
╭─────────────────────────────────────────────────────────────╮
│  Claude Sessions                               3 active │
╰─────────────────────────────────────────────────────────────╯

  ⏳  hyperion                                      12s ago
      ~/projects/hyperion
      "Should I proceed with the refactor?"

  ✅  api                                            3m ago
      ~/projects/parallel-api
      Done

▸ ⚙️  dotfiles                                      working
      ~/dotfiles

  ❓  scratch                                       45s ago
      ~/tmp/scratch
      "Run: rm -rf ./dist?"

╭─────────────────────────────────────────────────────────────╮
│  ↑/↓ navigate  ⏎ attach  d dismiss  q quit  r refresh  │
╰─────────────────────────────────────────────────────────────╯
```

### Status Icons (from PRD)

| Status       | Icon | Color   | Meaning                          |
|------------- |------|---------|----------------------------------|
| `waiting`    | ⏳   | Yellow  | Claude asked a question          |
| `done`       | ✅   | Green   | Task completed                   |
| `permission` | ❓   | Magenta | Needs tool use approval          |
| `working`    | ⚙️   | Cyan    | Actively processing              |
| `error`      | ❌   | Red     | Something went wrong             |
| `unknown`    | ○    | Dim     | No status yet / stale            |

### Row Content

Each row shows:
- Status icon (colored)
- Session name in bold
- Age of last status update, right-aligned
- Second line: Working directory (shortened with `~`), dimmed
- Third line (if present): The message from Claude, truncated to terminal width, dimmed/italic

## UX/UI Considerations

- Currently selected row has `▸` marker and is highlighted
- Messages are truncated to terminal width to prevent wrapping
- Working directories use `~` shorthand for home directory
- Color choices follow common conventions (yellow=attention, green=success, red=error)

## Acceptance Criteria

1. Lip Gloss styles are defined for all UI elements
2. Header displays title and active session count
3. Each session row displays: status icon, name, age, cwd, message
4. Status icons use correct colors per PRD specification
5. Selected row is highlighted with `▸` marker
6. Footer displays keybinding help
7. View handles terminal resize (width/height from Model)
8. Messages are truncated appropriately

## Dependencies

- PBI-1: Core types (`Model`, `SessionInfo`)
- PBI-3: Session polling provides data to render

## Open Questions

None

## Related Tasks

See [Tasks](./tasks.md)
