# PBI-47: PM TUI View — Three-Zone Layout

[View in Backlog](../backlog.md)

## Overview

Add a new PM view to Navi's TUI, toggled via `P`, with three zones: briefing (top), projects (middle), and events (bottom). In Phase 1, the briefing zone shows a placeholder; projects and events are rendered from snapshot and event data produced by PBI-46.

## Problem Statement

Navi's TUI currently shows a flat session list. There's no project-level aggregate view — no way to see "which projects changed, which need attention, what happened recently" without checking each session individually. The PM view provides this birds-eye project view.

## User Stories

- As a developer, I want to toggle a PM view that shows all my projects in one place so that I can orient myself quickly.
- As a developer, I want to see a chronological event log so that I know what happened across all projects recently.
- As a developer, I want to navigate between projects and jump to their sessions so that I can drill into details when needed.

## Technical Approach

- `internal/tui/pmview.go`: New Bubble Tea component rendering the three-zone layout.
- Modify `model.go` to add PM toggle state (`P` keybinding), mutually exclusive with session list.
- Zone 1 (briefing, ~30%): Static, shows "No PM briefing yet" in dim text for Phase 1. Will be populated by PBI-49.
- Zone 2 (projects, ~30%): One row per project from latest snapshots. Columns: status icon, name, PBI ID + title, progress (done/total), session status, last activity. Sorted by attention priority. Cursor navigation, Enter to jump to session.
- Zone 3 (events, ~40%): Reverse chronological from `events.jsonl`. Color-coded by type (green=task, yellow=status, cyan=commit). Scrollable with j/k.
- Tab to move focus between Zone 2 (default) and Zone 3. Zone 1 is never focused.
- Responsive layout — zone heights proportional to terminal size. Minimum 80 columns.

## UX/UI Considerations

- Layout matches the PRD mockup: briefing top, projects middle, events bottom.
- Status icons reuse Navi's existing icon/color scheme.
- PBI title truncates first on narrow terminals. Briefing collapses to one line on very short terminals.
- `P` toggle is mutually exclusive with session list — same Bubble Tea program, different view.
- Space to expand a project row: task breakdown, branch details, recent commits.

## Acceptance Criteria

1. `P` (capital) toggles PM view on/off, mutually exclusive with the session list.
2. Three-zone layout renders with proportional heights (briefing ~30%, projects ~30%, events ~40%).
3. Zone 1 shows "No PM briefing yet" placeholder in dim text.
4. Zone 2 lists one row per project with columns: status icon, name, PBI ID + title, progress, session status, last activity.
5. Zone 2 supports cursor navigation; Enter jumps to session list filtered to that project.
6. Zone 3 renders events in reverse chronological order, color-coded by event type.
7. Zone 3 is scrollable when focused (j/k, arrow keys).
8. Tab moves focus between Zone 2 and Zone 3.
9. Layout degrades gracefully on terminals shorter than normal (briefing collapses, titles truncate).

## Dependencies

- **Depends on**: PBI-46 (snapshot and event data to render)
- **External**: None

## Open Questions

- None.

## Related Tasks

_Tasks will be created when this PBI moves to Agreed via `/plan-pbi 47`._
