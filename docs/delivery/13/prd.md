# PBI-13: Search & Filter

[View in Backlog](../backlog.md)

## Overview

Add fuzzy search, status filtering, and sort options so users can quickly find sessions when managing many concurrent Claude instances.

## Problem Statement

With many sessions running, scrolling through a flat list becomes inefficient. Users need to quickly locate specific sessions by name, filter by status, and sort by different criteria.

## User Stories

- As a user, I want to type to search sessions by name so I can quickly find what I'm looking for
- As a user, I want to filter by status so I can focus on sessions needing attention
- As a user, I want to sort sessions by different criteria so I can organize my view
- As a user, I want to hide offline sessions so my list stays focused on active work

## Technical Approach

### Fuzzy Search

1. Press `/` to enter search mode
2. Fuzzy match against:
   - Session name
   - Working directory path
   - Message content
   - Tags (if PBI-8 implemented)
3. Use a fuzzy matching algorithm (e.g., Smith-Waterman or simple subsequence)
4. Show match count and highlight matches
5. Press `Esc` to clear search

### Status Filters

Quick toggle filters:
- `1` - Show only "waiting" sessions
- `2` - Show only "permission" sessions
- `3` - Show only "working" sessions
- `4` - Show only "done" sessions
- `5` - Show only "error" sessions
- `0` - Show all (clear filter)
- `o` - Toggle offline sessions visibility

### Sort Options

Press `s` to cycle through sort modes:
1. **Priority** (default) - Attention-needed first, then by time
2. **Name** - Alphabetical by session name
3. **Age** - Most recent activity first
4. **Status** - Grouped by status type
5. **Directory** - Grouped by working directory

### Implementation

1. Add filter state to Model:
   ```go
   type Model struct {
       // ... existing fields
       searchQuery   string
       searchMode    bool
       statusFilter  string  // empty = show all
       showOffline   bool
       sortMode      SortMode
   }
   ```

2. Filter sessions in `View()` before rendering:
   ```go
   func (m Model) filteredSessions() []SessionInfo {
       result := m.sessions
       if m.statusFilter != "" {
           result = filterByStatus(result, m.statusFilter)
       }
       if !m.showOffline {
           result = filterOutOffline(result)
       }
       if m.searchQuery != "" {
           result = fuzzyFilter(result, m.searchQuery)
       }
       return sortSessions(result, m.sortMode)
   }
   ```

3. Update footer to show current filter/sort state

### UI Elements

Search bar (when active):
```
╭─────────────────────────────────────────────────────────────╮
│  🔍 hyper_                                                  │
╰─────────────────────────────────────────────────────────────╯
```

Filter indicator in footer:
```
╭─────────────────────────────────────────────────────────────╮
│  Filter: permission | Sort: priority | 3/8 shown           │
╰─────────────────────────────────────────────────────────────╯
```

## UX/UI Considerations

- Search should be incremental (filter as you type)
- Clear visual feedback when filters are active
- Show filtered count vs total count
- Search should not block other keyboard input
- `Esc` should clear search first, then exit if pressed again

## Acceptance Criteria

1. `/` opens search mode with fuzzy matching
2. Search filters session list in real-time
3. Number keys (0-5) toggle status filters
4. `o` toggles offline session visibility
5. `s` cycles through sort modes
6. Current filter/sort state shown in footer
7. Filtered count displayed (e.g., "3/8 shown")
8. `Esc` clears search and filters
9. Cursor position preserved when possible during filtering

## Dependencies

- PBI-3: Session polling (base session data)
- PBI-8: Session organization (for tag-based filtering, optional)

## Open Questions

- Should search persist across polls?
- Should we support regex search?
- Should filter combinations be saveable as "views"?

## Related Tasks

See [Tasks](./tasks.md)
