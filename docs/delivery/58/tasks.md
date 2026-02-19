# Tasks for PBI 58: In-TUI Sound Pack Picker

This document lists all tasks associated with PBI 58.

**Parent PBI**: [PBI 58: In-TUI Sound Pack Picker](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--- | :----- | :---------- |
| 58-1 | [Audio SetPack and SavePackSelection](./58-1.md) | Done | Add Notifier.SetPack() for runtime hot-swap and SavePackSelection() for YAML persistence |
| 58-2 | [Dialog enum and model state](./58-2.md) | Done | Add DialogSoundPackPicker enum, picker state fields, and S keybind to open picker |
| 58-3 | [Picker update logic](./58-3.md) | Done | Handle keyboard navigation, selection, preview, and close in updateDialog routing |
| 58-4 | [Picker rendering](./58-4.md) | Done | Render scrollable pack list overlay with active marker, counts, and empty-state message |
| 58-5 | [E2E CoS Test](./58-5.md) | Done | End-to-end tests verifying all PBI-58 acceptance criteria |

## Dependency Graph

```
58-1 (Audio SetPack + SavePackSelection)
  │
  ├──► 58-2 (Dialog enum + model state)
  │      │
  │      ├──► 58-3 (Picker update logic)
  │      │
  │      └──► 58-4 (Picker rendering)
  │              │
  └──────────────┴──► 58-5 (E2E CoS Test)
```

## Implementation Order

1. **58-1** — Audio APIs first (no TUI dependencies, foundational for everything)
2. **58-2** — Dialog wiring (depends on 58-1 for message types and notifier access)
3. **58-3** — Update logic (depends on 58-2 for state fields and dialog routing)
4. **58-4** — Rendering (depends on 58-2 for state fields; can parallelize with 58-3)
5. **58-5** — E2E tests (depends on all prior tasks being complete)

## Complexity Ratings

| Task ID | Complexity | External Packages |
|---------|-----------|-------------------|
| 58-1 | Medium | None |
| 58-2 | Simple | None |
| 58-3 | Medium | None |
| 58-4 | Medium | None |
| 58-5 | Medium | None |

## External Package Research Required

None — all functionality uses existing internal packages (`internal/audio`, `internal/tui`) and the standard library.
