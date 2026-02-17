# Tasks for PBI 47: PM TUI View — Three-Zone Layout

This document lists all tasks associated with PBI 47.

**Parent PBI**: [PBI 47: PM TUI View — Three-Zone Layout](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--- | :----- | :---------- |
| 47-1 | [PM View State, Toggle, and Layout Skeleton](./47-1.md) | Proposed | Add PM view model state, P keybinding toggle, three-zone vertical layout with briefing placeholder |
| 47-2 | [Projects Zone Rendering and Navigation](./47-2.md) | Proposed | Render project rows from snapshots with columns, cursor navigation, Enter to jump to filtered sessions |
| 47-3 | [Events Zone Rendering](./47-3.md) | Proposed | Render reverse-chronological color-coded event log, scrollable when focused |
| 47-4 | [Focus System and Tab Navigation](./47-4.md) | Proposed | Tab to switch focus between Zone 2 and Zone 3, visual indicators, keyboard routing |
| 47-5 | [Project Expansion and Responsive Layout](./47-5.md) | Proposed | Space to expand project rows, graceful degradation on small terminals |
| 47-6 | [E2E CoS Test](./47-6.md) | Proposed | Verify all PBI-47 acceptance criteria with integration tests |

## Dependency Graph

```
47-1 (State, Toggle, Layout)
 ├── 47-2 (Projects Zone)
 │    └── 47-4 (Focus System)
 ├── 47-3 (Events Zone)
 │    └── 47-4 (Focus System)
 └── 47-4 (Focus System)
      └── 47-5 (Expansion + Responsive)
           └── 47-6 (E2E CoS Test)
```

## Implementation Order

1. **47-1** — Foundation: model state, toggle, layout skeleton, briefing placeholder
2. **47-2** — Projects zone: rendering, sorting, cursor navigation, session jump
3. **47-3** — Events zone: rendering, color coding, scrolling (parallel with 47-2 after 47-1)
4. **47-4** — Focus system: Tab switching, visual indicators, keyboard routing (requires 47-2 + 47-3)
5. **47-5** — Expansion and responsiveness: Space to expand, graceful degradation (requires 47-4)
6. **47-6** — E2E CoS test: verify all 9 acceptance criteria (requires all above)

## Complexity Ratings

| Task ID | Complexity | External Packages |
|---------|-----------|-------------------|
| 47-1 | Medium | None |
| 47-2 | Medium | None |
| 47-3 | Medium | None |
| 47-4 | Simple | None |
| 47-5 | Medium | None |
| 47-6 | Complex | None |

## External Package Research Required

None — all rendering uses existing lipgloss and Bubble Tea patterns already in the codebase.
