# Follow-Ups

Ideas, improvements, and deferred work captured during planning and implementation.
Review periodically — good candidates become new PBIs via `/new-pbi`.

## Open

| # | Type | Summary | Source | Date | Notes |
|---|------|---------|--------|------|-------|
| 1 | enhancement | Add RAM usage threshold alerts (color change at configurable limits) | PBI-57 | 2026-02-20 | PRD noted "informational only" — thresholds deferred as future enhancement |
| 2 | tech-debt | Pre-existing test failure: TestPMRefreshFlow_OpenRunOutputCycle | PBI-57 | 2026-02-20 | Fails on main branch, unrelated to PBI-57 changes — needs investigation |
| 3 | enhancement | Support macOS resource monitoring (no /proc — use sysctl or ps fallback) | PBI-57 | 2026-02-20 | Current implementation is Linux-only; PRD noted no Windows needed but macOS unaddressed |
