# PBI-50: Proactive PM and Session List Integration

[View in Backlog](../backlog.md)

## Overview

Enhance the PM to detect patterns across time, generate proactive suggestions, and surface attention items in the session list footer even when the PM view is not active. Phase 3 — the PM gets smart and pervasive.

## Problem Statement

In Phase 2, the PM only speaks when you look at it (PM view). Developers spend most of their time in the session list. Important attention items (stalled projects, completed PBIs needing PRs) should be visible without toggling to PM view. The PM should also start recognizing patterns ("this is the third time X stalled on Y").

## User Stories

- As a developer, I want the PM to flag attention items in the session list so that I see what needs action without switching views.
- As a developer, I want the PM to notice recurring patterns so that its observations become more insightful over time.
- As a developer, I want proactive suggestions (e.g., "PBI is done, ready for a PR?") so that I don't forget follow-up actions.

## Technical Approach

- Extend PM system prompt with pattern detection instructions: identify recurring blockers, velocity changes, stale projects, completed PBIs without PRs.
- PM output already has `attention_items` — this PBI makes the session list consume them.
- Session list footer: when PM has attention items, show `PM: N items need attention (P to view)`.
- Stale detection: PM flags projects with no activity for a configurable threshold (default 24h for active sessions, 3 days for idle).
- Proactive suggestion types: PBI complete → suggest PR, project stale → suggest checking in, recurring blocker → surface pattern.

## UX/UI Considerations

- Session list footer is a single line, always visible at the bottom of the session list view.
- Footer text in warning color to attract attention without being intrusive.
- Pressing `P` from session list with attention items goes directly to PM view.
- Attention items in the footer are a count + prompt, not full text — full details in PM view.

## Acceptance Criteria

1. Session list footer shows `PM: N items need attention (P to view)` when attention items exist.
2. Footer disappears when there are no attention items.
3. PM system prompt includes instructions for pattern detection (recurring blockers, velocity trends).
4. PM generates proactive suggestions for completed PBIs (suggest PR creation).
5. PM flags stale projects that haven't had activity within threshold.
6. Pattern-based observations appear in PM briefing and/or attention items when relevant.

## Dependencies

- **Depends on**: PBI-49 (working PM briefing panel with attention items)
- **External**: None

## Open Questions

- Should the attention count in the footer include breadcrumbs, or only action-required items?

## Related Tasks

_Tasks will be created when this PBI moves to Agreed via `/plan-pbi 50`._
