# PBI-31: Vim-Style Exact-Match Search

## Overview

Replace the current fuzzy subsequence search with a vim-style exact-match search that highlights matches in place and allows cycling through them with `n`/`N`, rather than filtering the list down to only matching items.

[View in Backlog](../backlog.md#user-content-31)

## Problem Statement

The current search implementation uses fuzzy subsequence matching and filters the session/task list to only show matches. This behaviour differs from the vim search model that many terminal users expect, where:

- Search is exact (literal substring match)
- All items remain visible with matches highlighted
- The user cycles through matches with `n` (next) and `N` (previous)
- The cursor jumps to the next/previous match rather than the list being filtered

Users familiar with vim find the current fuzzy-filter approach unintuitive for quickly locating a specific session or task by name.

## User Stories

- As a user, I want search to match exact substrings so that I get predictable, precise results
- As a user, I want to press `n` to jump to the next match and `N` to jump to the previous match so that I can cycle through all occurrences
- As a user, I want all sessions/tasks to remain visible during search so that I maintain context of the full list
- As a user, I want the current match highlighted distinctly so that I can see which result I'm on

## Technical Approach

### Current State

- Search activated with `/`, uses fuzzy subsequence matching (`filter.go:fuzzyMatch`)
- Results are filtered — non-matching items are hidden
- Cursor moves within the filtered subset
- Applies to both session list and task panel search

### Proposed Changes

- Replace fuzzy matching with exact case-insensitive substring matching
- Keep all items visible during search; highlight matching items
- Track a list of match indices and a "current match" pointer
- `n` moves to next match, `N` moves to previous match (with wrap-around)
- `Enter` confirms selection (attaches to session / opens task), `Esc` exits search mode
- Show match count indicator (e.g., "2/5 matches") in the search bar or status line
- Apply to both session list search and task panel search

## UX/UI Considerations

- Search bar appears at the bottom or top as current (`/ <query>`)
- Match count shown: e.g., `[2/5]` next to the search input
- Matched text within items should be visually highlighted (bold, colour, or inverse)
- Current match (where cursor is) should have a distinct highlight vs other matches
- When no matches found, show "No matches" indicator
- Wrap-around behaviour: cycling past last match goes to first, and vice versa

## Acceptance Criteria

1. Pressing `/` enters search mode; typing produces exact case-insensitive substring matches
2. All items remain visible during search (no filtering)
3. Matching items are visually highlighted in the list
4. `n` jumps cursor to next match, `N` jumps to previous match
5. Match cycling wraps around (last -> first, first -> last)
6. A match counter (e.g., "2/5") is displayed showing current position and total matches
7. `Esc` exits search mode and clears highlights
8. `Enter` on a match performs the default action (attach/open) and search persists (highlights and match state remain)
9. Only `Esc` clears the search state
10. Fuzzy search is fully removed — no legacy fuzzy matching code remains
11. Works in both session list and task panel
12. Existing status filters (1-5 keys), sort, and offline toggle continue to work independently of search

## Dependencies

- None — this is a self-contained change to existing search behaviour

## Open Questions

- ~~Should the old fuzzy search be retained as an alternative mode (e.g., toggled), or fully replaced?~~ **Resolved**: Fully replace fuzzy search.
- ~~Should search persist after `Enter` (like vim) or clear?~~ **Resolved**: Search persists after `Enter`, like vim. Highlights and match state remain until the user explicitly clears with `Esc`.

## Related Tasks

[View Tasks](./tasks.md)
