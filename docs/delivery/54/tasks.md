# Tasks for PBI 54: Session-Scoped Current PBI Resolution for PM View

This document lists all tasks associated with PBI 54.

**Parent PBI**: [PBI 54: Session-Scoped Current PBI Resolution for PM View](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--- | :----- | :---------- |
| 54-1 | [Extend Type Contracts for Current-PBI Resolution](./54-1.md) | Done | Add provider hint, session metadata, and snapshot provenance fields to existing types |
| 54-2 | [Branch-Pattern PBI Inference Utility](./54-2.md) | Done | Create utility to extract PBI ID from git branch names using configurable patterns |
| 54-3 | [Status-Priority Heuristic for Task Groups](./54-3.md) | Done | Implement intelligent group selection based on PBI status instead of array order |
| 54-4 | [Current-PBI Multi-Strategy Resolver](./54-4.md) | Done | Implement resolver with 5-level precedence chain and provenance tracking |
| 54-5 | [Update markdown-tasks Provider with Current-PBI Hint](./54-5.md) | Done | Modify provider script to emit current_pbi_id based on InProgress PBI detection |
| 54-6 | [Integrate Resolver into PM Snapshot Pipeline](./54-6.md) | Done | Replace getCurrentPBI with the multi-strategy resolver in CaptureSnapshot |
| 54-7 | [E2E CoS Test — Current-PBI Resolution](./54-7.md) | Done | End-to-end verification of all acceptance criteria across resolution strategies |

---

## Dependency Graph

```
54-1 (Types)
  ├──→ 54-2 (Branch Inference)
  ├──→ 54-3 (Status Heuristic)
  └──→ 54-5 (Provider Hint)
        │
54-2 ───┤
54-3 ───┼──→ 54-4 (Resolver) ──→ 54-6 (Pipeline Integration) ──→ 54-7 (E2E CoS Test)
54-5 ───┘
```

## Implementation Order

1. **54-1** — Types first; all other tasks depend on these contract changes
2. **54-2** — Branch inference utility (depends on 54-1 for types, no other deps)
3. **54-3** — Status heuristic (depends on 54-1 for types, no other deps)
4. **54-5** — Provider script update (depends on 54-1 for JSON field names)
5. **54-4** — Resolver (depends on 54-1, 54-2, 54-3 for all strategies)
6. **54-6** — Pipeline integration (depends on 54-4 for resolver function)
7. **54-7** — E2E CoS test (depends on all tasks being complete)

## Complexity Ratings

| Task ID | Complexity | External Packages |
|---------|------------|-------------------|
| 54-1 | Simple | None |
| 54-2 | Simple | None |
| 54-3 | Simple | None |
| 54-4 | Medium | None |
| 54-5 | Simple | None |
| 54-6 | Medium | None |
| 54-7 | Medium | None |

## External Package Research Required

None — all tasks use existing Go stdlib and project infrastructure.
