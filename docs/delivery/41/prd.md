# PBI-41: Multi-Agent TUI Display

[View in Backlog](../backlog.md#user-content-41)

## Overview

Update the TUI to display per-agent status indicators for sessions with multiple detected agents, so users can see both Claude Code and Codex state in a unified view. This builds on PBI-40's pane detection infrastructure.

## Problem Statement

Once PBI-40 provides detected agent data per session, there is no way to see it. The TUI currently shows a single status icon and message per session, driven entirely by Claude Code hooks. Users running dual-agent workflows (Claude Code + Codex) have no visibility into the Codex pane's state without manually switching to it.

### Current Limitations

- **Single status display**: Each session row shows one status icon derived from Claude Code hooks
- **No agent breakdown**: No way to see which agents are in a session or their individual states
- **No Codex visibility**: Codex running/stopped state is completely invisible in the TUI

## User Stories

- As a user, I want to see which agents are active in each session so that I know what's running at a glance.
- As a user, I want to see Codex's running/stopped status alongside Claude Code's status so that I can monitor my dual-agent workflow.
- As a user, I want agent indicators to be compact and unobtrusive so that sessions with a single agent don't look cluttered.

## Technical Approach

### Session List Row - Agent Indicators

Add compact agent type/status indicators to each session row. When a session has detected agents beyond the primary Claude Code instance, show them inline:

```
  api        ⚙️ working    [CX ●]     main ✓  2m ago
  frontend   ✅ done        [CX ○]     feat/nav ~  5m ago
  backend    ❓ permission               main ✓  1m ago
```

- `[CX ●]` = Codex running
- `[CX ○]` = Codex stopped
- No indicator shown if only Claude Code is detected (no visual overhead for single-agent sessions)

The exact rendering (icons, positioning, colors) will be refined during implementation to fit the existing TUI style.

### Detail / Preview Area

When a session is selected, the detail area (preview pane, metrics, git info) should include an "Agents" section showing:

- Agent type and status for each detected agent
- Pane ID for reference
- For Claude Code: the existing hook-driven status and message
- For Codex: running/stopped

### Status Aggregation

Session-level sorting and attention indicators should consider all agents:

- If Codex is running and Claude Code is idle/done, the session is still "active"
- The primary sort/attention logic remains driven by Claude Code hooks (permission, waiting take priority)
- A session with any running agent should not sort as fully "done"

### Color and Icon Design

- Each agent type gets a short label: `CC` (Claude Code), `CX` (Codex)
- Status uses filled/hollow circle or similar compact indicator
- Colors follow existing TUI palette (green for active, dim for stopped)
- Agent indicators are visually subordinate to the main session status

## UX/UI Considerations

- **Zero overhead for single-agent sessions**: If only Claude Code is detected (the common legacy case), the session row looks identical to today
- **Compact by default**: Agent indicators should not push other columns (git, timestamp) off screen
- **Consistent iconography**: Reuse existing status icon conventions where possible
- **Accessible**: Don't rely on color alone; use text labels (`CX`) alongside icons

## Acceptance Criteria

1. Sessions with a detected Codex agent show a compact `[CX ●/○]` indicator in the session list
2. Sessions with only Claude Code show no additional indicators (unchanged from today)
3. The detail/preview area shows per-agent status when a multi-agent session is selected
4. Session sorting accounts for detected agents (sessions with any running agent are not fully "done")
5. Agent indicators use consistent styling that fits the existing TUI design
6. All existing tests continue to pass
7. New tests cover agent indicator rendering and status aggregation logic

## Dependencies

- PBI-40 (Tmux Pane Agent Detection) - Provides the `[]DetectedAgent` data on `session.Info`
- PBI-4 (Styled TUI) - Done. Provides the rendering infrastructure being extended.
- PBI-33 (Agent Team Awareness) - Done. Established inline agent display patterns.

## Open Questions

1. Should the agent indicators be configurable (show/hide per agent type)? Recommend no for now, keep it simple.
2. Should there be a keybinding to cycle focus between agent panes? Potentially useful but recommend deferring to a future PBI.

## Related Tasks

See [tasks.md](./tasks.md) for the task breakdown.
