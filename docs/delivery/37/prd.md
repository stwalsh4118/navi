# PBI-37: Task View Enhancements — Sorting, Filtering, Progress, and Navigation

## Overview

The task view currently displays groups and tasks in source order with no ability to sort, filter, or summarize. With 30+ PBIs in the backlog (most Done), the useful information is buried under noise. This PBI adds sorting, filtering, progress indicators, summary stats, and navigation improvements to make the task view significantly more useful at scale.

[View in Backlog](../backlog.md#user-content-37)

## Problem Statement

The task view works well for small backlogs but becomes unwieldy as the number of groups grows:

- **No sorting** — Groups display in source order (by ID). Active work is buried below dozens of completed PBIs. Users must scroll through everything to find what's in progress.
- **No filtering** — All groups are always visible. There's no way to hide completed work and focus on what's active or upcoming.
- **No progress indicators** — Group headers show task count but not completion progress. You have to expand each group to see how many tasks are done vs remaining.
- **No summary stats** — The header shows total task count and group count, but no status breakdown. There's no at-a-glance view of overall project health.
- **Limited navigation** — Moving between groups requires stepping through every task. No way to jump between group headers directly.
- **No bulk expand/collapse** — With many groups, expanding/collapsing one at a time is tedious.
- **No manual refresh** — Must wait for the 60s auto-refresh cycle.

## User Stories

- As a user, I want to sort task groups by status so that active work appears at the top and completed work sinks to the bottom.
- As a user, I want to filter out completed groups so that I only see work that needs attention.
- As a user, I want to see progress indicators per group so that I know completion status at a glance.
- As a user, I want summary stats in the header so that I can see overall project health without scrolling.
- As a user, I want to jump between groups quickly so that I can navigate a large backlog efficiently.
- As a user, I want to expand/collapse all groups at once so that I can quickly scan or drill into the full list.
- As a user, I want a manual refresh trigger so that I can see updated data immediately after making changes.

## Technical Approach

### 1. Sorting

#### Group sorting

Add a `taskSortMode` field to the model with the following modes:

| Mode | Behavior | Use case |
|------|----------|----------|
| `source` | Original provider order (current behavior, default) | Matches backlog priority order |
| `status` | Groups ordered by status priority: active → review → todo → blocked → done | See what needs attention first |
| `name` | Alphabetical by group title | Find a specific group quickly |
| `progress` | Groups with lowest completion percentage first | Focus on what needs the most work |

**Status priority order** (for status sort): `active` > `review` > `blocked` > `todo` > `done`. Within the same status, preserve source order as a stable secondary sort.

#### Task sorting within groups

Tasks within each group are sorted by the same status priority when group sort is `status`. In other modes, tasks remain in source order.

#### Keybinding

- `s` — Cycle through sort modes: source → status → name → progress → source
- Show current sort mode in the header (e.g., `sort:status`)

### 2. Filtering

#### Status filtering

Add a `taskFilterMode` field to the model with the following modes:

| Mode | Behavior | Use case |
|------|----------|----------|
| `all` | Show everything (current behavior, default) | Full backlog view |
| `active` | Show only groups with status active, review, or blocked | Focus on in-flight work |
| `incomplete` | Show groups that are not done (active, review, todo, blocked) | Hide completed work |

#### Keybinding

- `f` — Cycle through filter modes: all → active → incomplete → all
- Show current filter mode in the header (e.g., `filter:active`)
- Show count of hidden groups when filter is active (e.g., `22 hidden`)

#### Interaction with search

When both search and filter are active, search operates within the filtered set. Clearing the filter (cycling back to `all`) does not clear the search, and vice versa.

### 3. Progress indicators

#### Per-group progress

Replace the simple task count `(5)` in group headers with a progress display:

```
▶ 3. Enhanced Session Creation  [3/5] ██░░  todo
```

- `[done/total]` — Numeric progress
- Mini progress bar — 4 characters wide, filled proportionally (using `█` and `░`)
- Only shown for groups with tasks (groups with 0 tasks show `(0)` as before)

#### Header summary

Extend the task panel header to include a status breakdown:

```
─ Tasks  navi  (47 tasks, 34 groups)  12 done · 3 active · 1 review · 18 todo
```

The summary counts reflect the **unfiltered** totals so the user always has context of the full backlog, even when filtering. The filtered count is shown separately if a filter is active.

### 4. Navigation improvements

#### Jump between groups

- `J` (shift-j) — Jump to next group header (skip tasks within current group)
- `K` (shift-k) — Jump to previous group header

This is distinct from `j`/`k` which move one item at a time. Group jumping allows rapid navigation through a long list.

#### Expand/collapse all

- `e` — Toggle expand/collapse all groups
  - If any groups are collapsed → expand all
  - If all groups are expanded → collapse all

#### Accordion mode

- `a` — Toggle accordion mode on/off
  - When on: expanding a group automatically collapses all others
  - When off: standard independent expand/collapse (default)
  - Show `accordion` indicator in header when active

### 5. Manual refresh

- `r` — Trigger immediate task data refresh (invalidate cache, re-run provider)
- Show a brief "Refreshing..." indicator while the provider runs
- Does not affect the auto-refresh timer

### 6. State persistence

Sort mode, filter mode, and accordion mode should persist for the session (stored in model state). They do not need to persist across restarts (that can be a future enhancement if needed).

## UX/UI Considerations

### Header layout

The header needs to accommodate new information without becoming cluttered. Proposed layout:

```
─ Tasks  navi  (47 tasks, 34 groups)  sort:status  filter:active (22 hidden)
  12 done · 3 active · 1 review · 18 todo
```

- First line: panel title, project name, counts, active sort/filter modes
- Second line: status summary (only shown when panel is tall enough)

### Visual indicators

- Sort mode: shown in header as `sort:<mode>`, dimmed style
- Filter mode: shown in header as `filter:<mode>`, dimmed style, with hidden count
- Accordion mode: shown as `accordion` indicator in header when active
- Progress bar: uses `█` (filled) and `░` (empty), 4 chars wide, dimmed style
- Refresh: brief flash message "Refreshing..." in header area

### Keybinding summary

| Key | Action | Context |
|-----|--------|---------|
| `s` | Cycle sort mode | Task panel focused |
| `f` | Cycle filter mode | Task panel focused |
| `e` | Expand/collapse all | Task panel focused |
| `a` | Toggle accordion mode | Task panel focused |
| `r` | Manual refresh | Task panel focused |
| `J` | Jump to next group | Task panel focused |
| `K` | Jump to previous group | Task panel focused |

Existing keybindings unchanged: `j/k` (move), `space/enter` (toggle group), `/` (search), `n/N` (search nav), `g/G` (top/bottom), `[/]` (resize), `tab` (return focus), `T` (toggle panel).

## Acceptance Criteria

1. **Sorting**: Groups can be sorted by source order, status priority, name, or progress. `s` cycles through modes. Current mode shown in header.
2. **Task sorting**: When group sort is `status`, tasks within groups are also sorted by status priority.
3. **Filtering**: Groups can be filtered to show all, active-only, or incomplete-only. `f` cycles through modes. Current mode and hidden count shown in header.
4. **Progress indicators**: Group headers show `[done/total]` with a mini progress bar for groups with tasks.
5. **Summary stats**: Header shows status breakdown across all groups (done/active/review/todo counts).
6. **Group jumping**: `J`/`K` jump between group headers, skipping tasks.
7. **Expand/collapse all**: `e` toggles all groups expanded or collapsed.
8. **Accordion mode**: `a` toggles accordion mode where expanding one group collapses others.
9. **Manual refresh**: `r` triggers immediate provider re-execution and cache invalidation.
10. **No keybinding conflicts**: New keybindings don't conflict with existing task panel or global keybindings.
11. **Search interaction**: Search works correctly with active filters (searches within filtered set).
12. **Cursor stability**: Cursor remains on a valid item after sort/filter changes. If the current item is filtered out, cursor moves to nearest visible item.
13. **All existing tests pass**; new tests cover sorting, filtering, progress calculation, and navigation.

## Dependencies

- None. This PBI builds on the existing task view infrastructure.

## Open Questions

- None.

## Related Tasks

[View Tasks](./tasks.md)
