# PBI-49: PM Briefing Panel — Live PM Output in TUI

[View in Backlog](../backlog.md)

## Overview

Replace the Phase 1 briefing placeholder with live PM output: narrative briefing text, attention items, breadcrumbs, and staleness indicator. Connect the TUI to the PM agent's cached output and show async loading state during invocations.

## Problem Statement

PBI-47 created the PM view layout with a placeholder briefing zone. PBI-48 wired up the PM agent that produces structured output. This PBI bridges them — rendering the PM's voice in the TUI so the developer actually sees the briefings, attention items, and breadcrumbs.

## User Stories

- As a developer, I want the PM's briefing rendered in the TUI so that I can read its analysis of what happened across my projects.
- As a developer, I want attention items visually highlighted so that I immediately see what needs action.
- As a developer, I want breadcrumbs showing where I left off so that I can resume work without context-switching overhead.
- As a developer, I want to know how fresh the briefing is so that I can trigger a refresh if needed.

## Technical Approach

- Modify `pmview.go` Zone 1 to read from `PMOutput` cached by the invoker (PBI-48).
- Briefing text: rendered as wrapped prose in the top section of Zone 1.
- Attention items: rendered below briefing as `⚠` lines in yellow/warning color.
- Breadcrumbs: rendered below attention items when the PM includes them (context-switching hints).
- Staleness: header shows "updated Xm ago" computed from `PMOutput.GeneratedAt`.
- Async loading: when PM invocation is in progress, show "updating..." indicator alongside last output.
- On-demand refresh: keybinding in PM view triggers `on_demand` invocation.
- Project status in Zone 2 can optionally reflect PM's project status assessment (needs_input, pbi_complete, active, stale, idle) alongside raw data.

## UX/UI Considerations

- Briefing text wraps naturally — no bullet points, short prose sentences per the PM's personality.
- Attention items use `⚠` prefix and yellow/warning styling to stand out.
- Breadcrumbs are contextual — only shown when the PM determines the developer is returning after absence.
- "Updated Xm ago" in the header gives immediate freshness signal.
- "Updating..." indicator is subtle — doesn't obscure the last briefing.

## Acceptance Criteria

1. Zone 1 renders the PM's `briefing` text as wrapped prose (replaces "No PM briefing yet" placeholder).
2. Attention items render as `⚠`-prefixed lines in warning color below the briefing.
3. Breadcrumbs render below attention items when present in PM output.
4. Header shows "updated Xm ago" based on when the PM last produced output.
5. An "updating..." indicator appears during active PM invocations without hiding the last briefing.
6. A keybinding in PM view triggers on-demand PM refresh.
7. If no PM output exists yet (first run, before first invocation completes), the placeholder is shown.

## Dependencies

- **Depends on**: PBI-47 (PM view layout), PBI-48 (PM agent producing output)
- **External**: None

## Open Questions

- None.

## Related Tasks

_Tasks will be created when this PBI moves to Agreed via `/plan-pbi 49`._
