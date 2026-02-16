# PBI-52: All-Project Awareness

[View in Backlog](../backlog.md)

## Overview

Enable the PM to track projects that don't have active sessions. Config-based project registration in `~/.config/navi/config.yaml` lets the PM monitor git state and task status for any project directory, providing full portfolio-level awareness and memory.

## Problem Statement

The PM currently only knows about projects with active Claude Code sessions. But developers have projects they're not actively running sessions for — they still want to know "you haven't touched Mnemos in three days" or "Apollo's main branch has 5 new commits since your feature branch." Portfolio-level awareness requires tracking projects independent of session state.

## User Stories

- As a developer, I want to register projects in config so that the PM tracks them even without active sessions.
- As a developer, I want portfolio-level briefings that cover all my projects so that nothing falls through the cracks.
- As a developer, I want the PM to notice when I haven't touched a project in a while so that I'm reminded of stale work.

## Technical Approach

- Extend `~/.config/navi/config.yaml` with a `projects` section for registering project directories.
- For registered projects without active sessions: scan git state (`git rev-parse HEAD`, branch, dirty state) and task status (read task files from `docs/delivery/`) directly.
- Merge registered-project snapshots with session-derived project snapshots in the PM engine.
- PM view Zone 2 shows all projects — session-backed and registered — with appropriate status icons (e.g., `⏹` for stopped/no session).
- Per-project memory files work the same for registered projects as session-backed ones.
- PM briefings can reference registered projects in their narrative and attention items.

## UX/UI Considerations

- Registered projects without sessions show a distinct status icon (stopped/offline indicator).
- Project rows for registered projects don't have session-specific columns (session status shows "no session").
- Expandable detail for registered projects shows git state and task progress.

## Acceptance Criteria

1. `~/.config/navi/config.yaml` supports a `projects` section listing project directories and names.
2. Registered projects without active sessions have their git state and task status scanned on the 30s polling cycle.
3. Registered projects appear in PM view Zone 2 alongside session-backed projects.
4. Registered projects without sessions show an appropriate status indicator (stopped/offline).
5. PM briefings can reference registered projects — they're included in the inbox alongside session-backed projects.
6. Per-project memory files are created and maintained for registered projects.
7. Stale detection works for registered projects (e.g., "you haven't touched X in 3 days").

## Dependencies

- **Depends on**: PBI-48 (PM agent and memory system)
- **External**: None

## Open Questions

- Should auto-discovered session projects be automatically added to config for persistence, or remain ephemeral?
- What's the config format? Simple list of paths, or richer objects with name/path/tags?

## Related Tasks

_Tasks will be created when this PBI moves to Agreed via `/plan-pbi 52`._
