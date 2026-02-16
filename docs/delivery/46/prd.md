# PBI-46: PM Engine — Project Snapshots and Event Pipeline

[View in Backlog](../backlog.md)

## Overview

Build the core PM data pipeline: discover projects from active sessions, capture state snapshots (git, tasks, session status), detect changes via snapshot diffing, and emit structured events to a rolling JSONL log. This is the Phase 1 backend — no LLM, purely structured data.

## Problem Statement

Navi monitors individual sessions but has no concept of "projects" — grouping sessions by their working directory, tracking project-level state (which PBI is active, how many tasks are done, what branch/commits exist), or detecting meaningful changes over time. The PM engine provides this foundational data layer.

## User Stories

- As a developer, I want Navi to automatically discover projects from my active sessions so that I don't have to manually configure them.
- As a developer, I want project state changes (task completions, new commits, status transitions) captured as structured events so that the PM can reason about what happened.

## Technical Approach

- New `internal/pm/` package alongside existing Navi packages.
- `types.go`: Core data types — `ProjectSnapshot`, `PBISnapshot`, `TaskCounts`, `Event`, `PMOutput`.
- `snapshot.go`: Project discovery (session CWD → project root), snapshot capture (git rev-parse, task status parsing, session status aggregation), snapshot caching, diff computation.
- `events.go`: Event type definitions, JSONL append/read, 24-hour rolling pruning.
- `engine.go`: Core PM loop — hooks into Navi's existing 30s git/task refresh cycle, runs snapshot → diff → event pipeline.
- Multiple sessions sharing the same CWD are grouped into one project. Most recently active session determines project status.

## UX/UI Considerations

N/A — backend/infrastructure PBI. Events and snapshots are consumed by PBI-47 (TUI) and PBI-48 (PM agent).

## Acceptance Criteria

1. Projects are automatically discovered from active session working directories, with deduplication when multiple sessions share the same project root.
2. Project snapshots capture: HEAD SHA, branch name, commits ahead, dirty state, current PBI (ID + title + task counts), session status, last activity time.
3. Snapshot diffing detects and emits typed events: `task_completed`, `task_started`, `commit`, `session_status_change`, `pbi_completed`, `branch_created`, `pr_created`.
4. Events are appended to `~/.config/navi/pm/events.jsonl` with timestamps and project names.
5. Events older than 24 hours are pruned on each write cycle.
6. The engine runs on Navi's existing 30s polling cycle without adding perceptible latency.
7. HEAD SHA comparison uses `git rev-parse HEAD`; `git log --oneline <old>..<new>` only runs when SHA changes.

## Dependencies

- **Depends on**: None (uses existing session, git, and task infrastructure in Navi)
- **External**: None

## Open Questions

- None — the PRD is specific about the snapshot/diff/event design.

## Related Tasks

_Tasks will be created when this PBI moves to Agreed via `/plan-pbi 46`._
