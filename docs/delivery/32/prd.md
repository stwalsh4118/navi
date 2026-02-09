# PBI-32: Scrollable Panels

[View in Backlog](../backlog.md#user-content-32)

## Overview

Add scrollable content to all TUI panels that currently truncate content when it exceeds the available viewport height. The content viewer (PBI-29) already implements a proven scrolling pattern with offset-based scrolling, vim-style keybindings, and scroll indicators. This PBI extends that pattern to the task panel, preview pane, session list, and dialog overlays.

## Problem Statement

In the current TUI, when panels don't have enough vertical space to display all their content (especially in vertical layout or with smaller terminal windows), the content is silently truncated. Users have no indication that more content exists below the visible area, and no way to scroll to see it. The expand/shrink hotkeys (`[`/`]`) help somewhat but don't solve the fundamental problem - content that doesn't fit is simply invisible.

**Specific issues:**
- **Task panel**: Items beyond `maxLines` are cut off with no scroll indicator. The cursor can wrap around but the viewport doesn't follow.
- **Preview pane**: Only shows the last N lines of content; there's no way to scroll up through earlier output.
- **Session list**: Uses cursor-based navigation with wrapping but no viewport scrolling - when the list exceeds the visible area, off-screen sessions are inaccessible.
- **Dialog overlays**: Fixed-size panels with no scrolling for long content (e.g., git status with many files).

## User Stories

- As a user, I want to scroll through the task panel when there are more tasks than fit on screen so that I can see all my tasks without expanding the panel.
- As a user, I want to scroll through the preview pane so that I can see earlier output, not just the most recent lines.
- As a user, I want the session list to scroll properly when I have many sessions so that I can navigate to any session.
- As a user, I want scroll indicators on panels so that I know when there's more content above or below the visible area.

## Technical Approach

### Pattern to Follow

The content viewer (`internal/tui/contentviewer.go`) establishes the scrolling pattern to replicate:

1. **Scroll state**: Integer offset tracking the first visible line/item
2. **Viewport calculation**: Derive visible area from panel height minus chrome (borders, headers, footers)
3. **Visible item slicing**: Render only items within `[scrollOffset, scrollOffset+viewportHeight)`
4. **Scroll clamping**: Ensure scroll offset never exceeds `max(0, totalItems - viewportHeight)`
5. **Auto-follow cursor**: When cursor moves beyond the viewport, adjust scroll to keep it visible
6. **Scroll indicators**: Show position information (e.g., line count, percentage, or arrow indicators)
7. **Keybindings**: vim-style navigation (j/k line-by-line, PgUp/PgDn for pages, g/G for top/bottom) when the panel is focused

### Panel-Specific Considerations

- **Task panel**: Already has cursor navigation and focus state. Needs a `taskScrollOffset` field and viewport-aware rendering in `renderTaskPanelList()`. Cursor movement should auto-scroll the viewport.
- **Preview pane**: Needs a `previewScrollOffset` field. Currently shows last N lines; should instead show lines from scroll offset. When new content arrives, auto-scroll to bottom (unless user has scrolled up).
- **Session list**: Already has cursor navigation. Needs a `sessionScrollOffset` field so that cursor movement scrolls the viewport when the cursor reaches the edge.
- **Dialogs**: Assess which dialogs can have overflowing content and add scrolling where needed. May be lower priority if dialog content rarely overflows.

### Shared Scrolling Utility

Consider extracting common scrolling logic into a reusable helper to avoid duplicating the clamping, viewport calculation, and auto-follow logic across panels. This could be a simple struct or set of functions rather than a full component.

## UX/UI Considerations

- Scroll indicators should be subtle but clear - showing that more content exists above/below
- When a panel has focus, its scroll keybindings should take priority
- Auto-scrolling behavior (e.g., preview pane following new output) should be intuitive
- The expand/shrink hotkeys (`[`/`]`) should continue to work alongside scrolling
- Scrolling should feel responsive with no visual lag

## Acceptance Criteria

1. The task panel scrolls its content when items exceed the viewport height, with the viewport following cursor movement
2. The preview pane supports scrolling through its full content (not just the last N lines)
3. The session list scrolls properly when sessions exceed the visible area
4. All scrollable panels show visual indicators when content extends beyond the viewport
5. Vim-style keybindings (j/k, PgUp/PgDn, g/G) work in focused panels for scrolling
6. The existing expand/shrink (`[`/`]`) panel resize functionality continues to work
7. All existing tests continue to pass
8. New tests cover scroll offset clamping, viewport calculation, and cursor-follow behavior

## Dependencies

- PBI-29 (Content Viewer) - Done. Provides the scrolling pattern to follow.
- PBI-28 (Task View) - Done. Task panel implementation that needs scrolling added.
- PBI-11 (Preview Pane) - Done. Preview pane implementation that needs scrolling added.

## Open Questions

1. Should dialogs also get scrolling, or is that a separate PBI? (Recommend: include basic dialog scrolling if straightforward, otherwise defer)
2. Should the preview pane auto-scroll to bottom when new content arrives, or always stay at the user's scroll position? (Recommend: auto-scroll to bottom unless user has manually scrolled up, similar to terminal behavior)
3. Should scroll position persist when switching between panels, or reset? (Recommend: persist)

## Related Tasks

See [tasks.md](./tasks.md) for the task breakdown.
