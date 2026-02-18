# Tasks for PBI 55: PM View Refresh UX and Provider Performance

This document lists all tasks associated with PBI 55.

**Parent PBI**: [PBI 55: PM View Refresh UX and Provider Performance](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--- | :----- | :---------- |
| 55-1 | [PM Loading Indicator](./55-1.md) | Done | Display a visible loading indicator in the PM briefing zone while PM refresh is in flight |
| 55-2 | [Immediate PM Refresh on View Entry](./55-2.md) | Review | Trigger an immediate PM engine run when the PM view is opened via P key |
| 55-3 | [Manual PM Refresh Key](./55-3.md) | Review | Add r key handler in PM view to manually trigger a PM refresh with cache invalidation |
| 55-4 | [Concurrent Provider Execution](./55-4.md) | Review | Execute task providers concurrently with bounded concurrency to reduce multi-project refresh latency |
| 55-5 | [E2E CoS Test](./55-5.md) | Review | Verify all PBI-55 acceptance criteria through automated tests |

## Dependency Graph

```
55-1 (PM Loading Indicator)
  ↓
55-2 (Immediate Refresh on Entry) ──→ 55-5 (E2E CoS Test)
  ↓
55-3 (Manual PM Refresh Key) ────────→ 55-5
                                         ↑
55-4 (Concurrent Providers) ─────────→ 55-5
```

## Implementation Order

1. **55-1** — PM Loading Indicator: Foundation for UX feedback; must exist before triggering immediate refreshes so users see loading state.
2. **55-2** — Immediate PM Refresh on View Entry: Core feature; depends on 55-1 for visual feedback.
3. **55-3** — Manual PM Refresh Key: Extends PM view interaction; depends on 55-1 for visual feedback, follows same pattern as 55-2.
4. **55-4** — Concurrent Provider Execution: Independent performance optimization in the task package; can be done in parallel with 55-2/55-3 but ordered here for sequential flow.
5. **55-5** — E2E CoS Test: Final verification; depends on all other tasks.

## Complexity Ratings

| Task ID | Complexity | External Packages |
|---------|-----------|-------------------|
| 55-1 | Simple | None |
| 55-2 | Simple | None |
| 55-3 | Simple | None |
| 55-4 | Medium | None |
| 55-5 | Medium | None |

## External Package Research Required

None
