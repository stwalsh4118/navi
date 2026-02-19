# Tasks for PBI 57: Session RAM Usage Monitoring

This document lists all tasks associated with PBI 57.

**Parent PBI**: [PBI 57: Session RAM Usage Monitoring](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--- | :----- | :---------- |
| 57-1 | [ResourceMetrics type and FormatBytes helper](./57-1.md) | Proposed | Add ResourceMetrics struct to metrics package with RSSBytes field and human-readable byte formatting function |
| 57-2 | [/proc process tree RSS calculator](./57-2.md) | Proposed | New internal/resource package that walks /proc to sum RSS for all processes in a tmux session's process tree |
| 57-3 | [Resource polling integration in TUI](./57-3.md) | Proposed | Add 30-second independent resource tick, poll resource metrics for local sessions, merge onto session data |
| 57-4 | [RAM badge rendering in sidebar](./57-4.md) | Proposed | Extend renderMetricsBadges to display RAM badge alongside existing time/tools/tokens badges |
| 57-5 | [E2E CoS Test](./57-5.md) | Proposed | End-to-end verification of all 6 acceptance criteria for session RAM monitoring |

## Dependency Graph

```
57-1  ResourceMetrics type
  │
  ├──► 57-2  /proc tree walker (depends on ResourceMetrics type)
  │      │
  │      └──► 57-3  Resource polling in TUI (depends on walker)
  │             │
  │             └──► 57-4  RAM badge rendering (depends on polling data)
  │                    │
  │                    └──► 57-5  E2E CoS Test (depends on all above)
  │
  └──► 57-4  RAM badge rendering (also depends on FormatBytes)
```

## Implementation Order

1. **57-1** — Types/constants must exist before any code references them
2. **57-2** — Core /proc logic needed before TUI can poll it
3. **57-3** — Polling must store data before rendering can display it
4. **57-4** — Rendering depends on both the format helper (57-1) and polled data (57-3)
5. **57-5** — E2E test verifies the complete integrated feature

## Complexity Ratings

| Task ID | Complexity | External Packages |
|---------|------------|-------------------|
| 57-1    | Simple     | None              |
| 57-2    | Medium     | None              |
| 57-3    | Medium     | None              |
| 57-4    | Simple     | None              |
| 57-5    | Medium     | None              |

## External Package Research Required

None — all functionality uses Go stdlib and the Linux /proc filesystem.
