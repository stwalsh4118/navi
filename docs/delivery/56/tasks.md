# Tasks for PBI 56: Enhanced Audio — Sound Packs, Volume Control, and UX Polish

This document lists all tasks associated with PBI 56.

**Parent PBI**: [PBI 56: Enhanced Audio — Sound Packs, Volume Control, and UX Polish](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--- | :----- | :---------- |
| 56-1 | [Config Schema — Pack and Volume Fields](./56-1.md) | Done | Add Pack, Volume struct, and updated YAML parsing to audio Config |
| 56-2 | [Sound Pack Scanner and File Resolution](./56-2.md) | Done | New pack.go with directory scanning, extension detection, multi-file discovery, and pack listing |
| 56-3 | [Volume Control — Per-Backend CLI Flags](./56-3.md) | Done | Add volume parameter to Player.Play() with backend-specific flag mapping and volume calculation |
| 56-4 | [Notifier Integration — Pack Resolution, Volume, Random Selection, and Mute](./56-4.md) | Done | Wire pack scanner, volume, random sound selection, and mute state into the Notifier |
| 56-5 | [Mute Toggle — TUI Keybind and Visual Indicator](./56-5.md) | Done | Add m keybind to toggle mute with a visual indicator in the TUI footer |
| 56-6 | [CLI Subcommands — navi sound test/list](./56-6.md) | Done | Add navi sound test, test-all, and list subcommands for previewing and managing sound packs |
| 56-7 | [E2E CoS Test — Full Audio Enhancement Verification](./56-7.md) | Done | End-to-end tests verifying all 8 acceptance criteria for PBI-56 |

## Dependency Graph

```
56-1 (Config Schema)
 ├──► 56-2 (Pack Scanner)
 │     └──► 56-4 (Notifier Integration) ──► 56-5 (Mute Toggle TUI)
 └──► 56-3 (Volume Control)                    │
       └──► 56-4 ◄────────────────────────────┘
                                                │
56-2 ──► 56-6 (CLI Subcommands) ◄── 56-3       │
                                                │
56-4 ──► 56-5                                   │
56-4 ──► 56-6                                   │
                                                ▼
                                          56-7 (E2E CoS Test)
                                          depends on ALL above
```

## Implementation Order

1. **56-1** Config Schema — no dependencies, foundation for all other tasks
2. **56-2** Pack Scanner — depends on 56-1 (uses Config.Pack field)
3. **56-3** Volume Control — depends on 56-1 (uses VolumeConfig); parallel-safe with 56-2
4. **56-4** Notifier Integration — depends on 56-1, 56-2, 56-3 (wires everything together)
5. **56-5** Mute Toggle TUI — depends on 56-4 (uses SetMuted/IsMuted)
6. **56-6** CLI Subcommands — depends on 56-2, 56-3, 56-4 (uses pack scanner, volume, notifier)
7. **56-7** E2E CoS Test — depends on ALL above (final verification gate)

## Complexity Ratings

| Task ID | Complexity | External Packages |
|---------|------------|-------------------|
| 56-1 | Simple | None |
| 56-2 | Medium | None |
| 56-3 | Medium | None |
| 56-4 | Complex | None |
| 56-5 | Simple | None |
| 56-6 | Medium | None |
| 56-7 | Complex | None |

## External Package Research Required

None — all features use Go stdlib and existing dependencies (YAML, file I/O, exec).
