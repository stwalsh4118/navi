# PBI-41: Multi-Agent TUI Display

[View in Backlog](../backlog.md#user-content-41)

## Overview

Update the TUI to display per-agent status indicators for sessions with multiple agents, so users can see both Claude Code and OpenCode state in a unified view. This builds on PBI-40's OpenCode plugin integration.

## Problem Statement

Once PBI-40 provides OpenCode status data in the session status file, there is no way to see it. The TUI currently shows a single status icon and message per session, driven entirely by Claude Code hooks. Users running dual-agent workflows (Claude Code + OpenCode) have no visibility into OpenCode's state without manually switching to that pane.

### Current Limitations

- **Single status display**: Each session row shows one status icon derived from Claude Code hooks
- **No agent breakdown**: No way to see which agents are in a session or their individual states
- **No OpenCode visibility**: OpenCode running/idle/permission state is completely invisible in the TUI

## User Stories

- As a user, I want to see which agents are active in each session so that I know what's running at a glance.
- As a user, I want to see OpenCode's status alongside Claude Code's status so that I can monitor my dual-agent workflow.
- As a user, I want agent indicators to be compact and unobtrusive so that sessions with a single agent don't look cluttered.

## Technical Approach

### Session List Row — Agent Indicators

Add compact agent type/status indicators to each session row. When a session has agents in the `agents` map, show them inline:

```
  api        ⚙️ working    [OC ●]     main ✓  2m ago
  frontend   ✅ done        [OC ○]     feat/nav ~  5m ago
  backend    ❓ permission               main ✓  1m ago
```

- `[OC ●]` = OpenCode running/working
- `[OC ○]` = OpenCode idle/stopped
- No indicator shown if the `agents` map is empty (no visual overhead for single-agent sessions)

The exact rendering (icons, positioning, colors) will be refined during implementation to fit the existing TUI style.

### Detail / Preview Area

When a session is selected, the detail area should include an "Agents" section showing:

- Agent type and status for each entry in the `agents` map
- For Claude Code: the existing hook-driven status and message (root-level fields)
- For OpenCode: status from `agents.opencode`

### Status Aggregation

Session-level sorting and attention indicators should consider all agents:

- If OpenCode has `permission` status, the session needs attention (similar to Claude Code permission)
- The primary sort/attention logic remains driven by Claude Code hooks (permission, waiting take priority)
- A session with any agent in `working` or `permission` status should not sort as fully "done"

### Color and Icon Design

- Each agent type gets a short label: `CC` (Claude Code), `OC` (OpenCode)
- Status uses filled/hollow circle or similar compact indicator
- Colors follow existing TUI palette (green for active, dim for stopped/idle)
- Agent indicators are visually subordinate to the main session status

## UX/UI Considerations

- **Zero overhead for single-agent sessions**: If the `agents` map is empty, the session row looks identical to today
- **Compact by default**: Agent indicators should not push other columns (git, timestamp) off screen
- **Consistent iconography**: Reuse existing status icon conventions where possible
- **Accessible**: Don't rely on color alone; use text labels (`OC`) alongside icons

## Acceptance Criteria

1. Sessions with an OpenCode entry in `agents` show a compact `[OC ●/○]` indicator in the session list
2. Sessions with no `agents` entries show no additional indicators (unchanged from today)
3. The detail/preview area shows per-agent status when a multi-agent session is selected
4. Session sorting accounts for agent statuses (sessions with any active/permission agent are not fully "done")
5. Agent indicators use consistent styling that fits the existing TUI design
6. All existing tests continue to pass
7. New tests cover agent indicator rendering and status aggregation logic

## Dependencies

- PBI-40 (OpenCode Status Hook Integration) — Provides the `agents` map on `session.Info`
- PBI-4 (Styled TUI) — Done. Provides the rendering infrastructure being extended.
- PBI-33 (Agent Team Awareness) — Done. Established inline agent display patterns.

## Open Questions

1. Should the agent indicators be configurable (show/hide per agent type)? Recommend no for now, keep it simple.
2. Should there be a keybinding to focus/switch to a specific agent's pane? Potentially useful but recommend deferring to a future PBI.

## Related Tasks

_Tasks will be created when this PBI is planned via `/plan-pbi 41`._
