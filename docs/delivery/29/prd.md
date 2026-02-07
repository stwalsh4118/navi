# PBI-29: In-App Content Viewer

## Overview

Add a reusable in-app content viewer to Navi that can display files, diffs, and other text content in a scrollable, styled pane without leaving the TUI. This replaces the need to shell out to external editors or browsers for read-only content viewing.

## Problem Statement

Currently, several features in Navi either lack a way to view associated content or rely on opening external applications:
- **Task view**: Pressing enter on a task does nothing for local markdown tasks (no URL to open)
- **Git diffs**: The diff view exists but is basic and could benefit from a richer viewer
- **General file viewing**: No way to view file contents (e.g. task detail markdown, config files) without leaving the TUI

A reusable content viewer component would unify these use cases and enable future features that need to display text content.

## User Stories

1. As a user, I want to press enter on a task in the task view and see the task's detail file rendered in-app so that I can read task details without leaving Navi.
2. As a user, I want to scroll through file content with familiar keybindings (j/k, up/down, page up/down) so that navigation feels natural.
3. As a user, I want git diffs displayed with syntax-aware coloring (additions in green, deletions in red) so that I can quickly understand changes.
4. As a user, I want to press Esc to close the viewer and return to the previous screen so that the workflow is non-disruptive.

## Technical Approach

- Build a reusable `ContentViewer` Bubble Tea component with:
  - Scrollable text display with line wrapping
  - Configurable title/header
  - Keyboard navigation (j/k, arrows, page up/down, g/G for top/bottom, Esc to close)
  - Optional syntax highlighting or diff coloring
- Integrate as a dialog/overlay mode (similar to existing `DialogGitDetail`)
- First integration: task view enter key opens the task's markdown file
- Second integration: replace the existing git diff view with the new viewer
- Future integrations: config files, session logs, etc.

## UX/UI Considerations

- Full-screen overlay with title bar showing the file path or content description
- Scroll position indicator (e.g. percentage or line numbers)
- Consistent with existing Navi styling (box borders, color scheme)
- Responsive to terminal size changes

## Acceptance Criteria

1. A reusable content viewer component exists that can display arbitrary text with scrolling
2. Pressing enter on a task in the task view opens the task's detail file in the viewer
3. The viewer supports j/k, up/down, page up/down, g/G, and Esc keybindings
4. Git diffs can be displayed with addition/deletion coloring
5. The viewer adapts to terminal resize events
6. Esc closes the viewer and returns to the previous screen

## Dependencies

- PBI-28 (Task View) for the task file viewing integration
- Existing git diff display for the diff viewer integration

## Open Questions

- Should the viewer support basic markdown rendering (bold, headers) or display raw text?
- Should there be a way to open the file in `$EDITOR` from within the viewer (e.g. `e` key)?

## Related Tasks

See [tasks.md](./tasks.md) when tasks are created.

[View in Backlog](../backlog.md#user-content-29)
