# PBI-54: Session-Scoped Current PBI Resolution for PM View

[View in Backlog](../backlog.md)

## Overview

Improve PM "current PBI" detection so it reflects what each active session is actually working on, even when multiple sessions are active in the same project. Replace the current first-group heuristic with a deterministic resolver that uses session-scoped state, provider hints, and configurable fallback inference.

## Problem Statement

The PM snapshot currently infers current PBI from the first task group returned by the provider, which is often incorrect when providers return all PBIs in backlog order. This produces misleading PM Zone 2 rows (for example always showing PBI-1) and makes project state less trustworthy during real concurrent workflows.

## User Stories

- As a developer, I want PM current-PBI selection to reflect active session work so that PM rows show the PBI I am actually executing.
- As a developer, I want concurrent sessions in the same project to avoid clobbering each other so that PM state remains accurate under parallel work.
- As a provider author, I want to emit an explicit current-PBI hint so that PM can consume high-confidence intent directly.

## Technical Approach

- Add session-scoped current-work metadata support (per session file, not global singleton).
- Extend provider contract to support explicit current-PBI hints (`current_pbi_id` and/or per-group `is_current`).
- Implement PM current-PBI resolver with precedence:
  1. provider explicit current hint
  2. freshest session-scoped current-work signal for the same project
  3. branch-pattern inference (configurable patterns)
  4. status-priority heuristic from provider task groups
  5. legacy first-group fallback
- Include provenance fields in PM snapshot/resolution output (for example `current_pbi_source`) for observability and debugging.
- Keep resolver deterministic and fully unit-tested; PM agent may consume this result later but is not the source of truth for current-PBI selection.

## UX/UI Considerations

- PM Zone 2 should display correct current PBI for active project rows with no UI affordance changes required for baseline support.
- Optional follow-up: show a subtle indicator when multiple concurrent PBIs are active in one project.

## Acceptance Criteria

1. PM no longer relies solely on first provider group for current-PBI detection.
2. Provider output can mark a current PBI explicitly; resolver prefers this when present.
3. Session-scoped current-work metadata can represent multiple concurrent sessions without collisions.
4. Resolver supports configurable branch-pattern inference and falls back cleanly when patterns do not match.
5. Resolver uses stable status-priority heuristics when explicit signals are absent.
6. PM snapshot includes current-PBI provenance source for diagnostics.
7. Unit/integration tests cover provider-hint, session-scoped, branch-inferred, heuristic, and fallback paths.

## Dependencies

- **Depends on**: PBI-46 (PM snapshot/event pipeline), PBI-47 (PM TUI view)
- **Blocks**: None
- **External**: None

## Open Questions

- Exact schema and storage location for session-scoped current-work metadata (reuse existing session state files vs dedicated navi folder).
- Conflict behavior when two active sessions in one project declare different PBIs (single primary row value vs multi-value indicator).

## Related Tasks

[View Tasks](./tasks.md)
