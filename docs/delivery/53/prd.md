# PBI-53: Composite Session Status — Unified Multi-Agent Display

[View in Backlog](../backlog.md)

## Overview

Make external agents (OpenCode, etc.) first-class citizens in the session status display. The main session icon and message should reflect the aggregate state of all agents — if any agent is working, the session shows working. The current design treats external agents as subordinate indicators; this PBI promotes them to equal participants in the session's displayed state.

## Problem Statement

PBI-41 added per-agent indicators (`[OC ●]`) as secondary, visually subordinate elements alongside the main session status. The main status icon is driven solely by Claude Code's hook-reported status. This means a session where OpenCode is actively working but Claude Code is idle shows `⏸ idle [OC ●]` — at a glance, the session looks idle when work is actually happening. The developer has to scan the secondary indicators to understand the real state, which defeats the purpose of status-at-a-glance.

The core issue: both agents are doing real work simultaneously, but only one agent's status drives the primary visual.

## User Stories

- As a user, I want the session's main status icon to reflect whether any agent is working so that I can see real activity at a glance without scanning secondary indicators.
- As a user, I want to know which agent is driving the displayed status so that I can tell whether it's Claude Code or OpenCode that needs attention.

## Technical Approach

### Composite Status Function

Add a `CompositeStatus(s Info) (status string, source string)` function to `internal/session/session.go` that returns the highest-priority status across all agents and which agent it came from.

Priority order (highest to lowest):
1. `permission` — any agent needs human input
2. `waiting` — any agent is waiting
3. `working` — any agent is actively working
4. `error` — any agent has errored
5. `idle` — all agents idle
6. `stopped` — all agents stopped
7. `done` — all agents done

Inputs: the root-level `Status` field (Claude Code) and all entries in the `Agents` map. When Claude Code is the source, `source` returns `""` (empty — it's the default, no annotation needed). When an external agent is the source, `source` returns the agent type (e.g., `"opencode"`).

### Session Row Rendering

Modify `renderSession()` in `internal/tui/view.go`:

- The main status icon uses `StatusIcon(compositeStatus)` instead of `StatusIcon(s.Status)`
- When the composite status source is an external agent, append the source to the status message: e.g., `"working (opencode)"` instead of just `"working"`
- Per-agent indicators (`[CC ⚙️] [OC ⏸]`) remain visible for the individual breakdown. When composite status is active, both agents' individual indicators show so the user sees the per-agent picture alongside the aggregate

### Per-Agent Indicators Update

With the main icon now composite, the per-agent indicators shift from "here's the secondary agent" to "here's the per-agent breakdown." When composite status differs from Claude Code's raw status, show indicators for all agents including Claude Code (using `CC` label) so the user can see the full picture. When only Claude Code is present (no external agents), no indicators are shown — identical to today.

### Sorting Alignment

The existing `sessionSortTier()` already considers external agents for tier 0 (permission) and tier 1 (active). Verify this aligns with the new composite status logic and refactor if needed to share the priority logic with `CompositeStatus()`.

## UX/UI Considerations

- **Single-agent sessions unchanged**: Sessions with no external agents look identical to today — no visual regression.
- **Message format**: When the composite status comes from an external agent, the message shows `"<status> (agenttype)"` — e.g., `"working (opencode)"`. This tells the user which agent is driving the status without cluttering the display.
- **Per-agent breakdown**: The `[CC ⏸] [OC ⚙️]` indicators provide the detailed picture. These only appear when external agents exist.
- **Color consistency**: The main icon color changes to match the composite status — e.g., cyan for working even if Claude Code is idle.

## Acceptance Criteria

1. The main session status icon reflects the highest-priority status across Claude Code and all external agents (priority: permission > waiting > working > error > idle > stopped > done).
2. When an external agent is the source of the composite status, the status message includes the agent source: e.g., `"working (opencode)"`.
3. When Claude Code is the source, the status message displays as it does today (no annotation).
4. Per-agent breakdown indicators (`[CC ⏸] [OC ⚙️]`) show individual agent states when external agents are present.
5. Sessions with no external agents render identically to today — zero visual change.
6. Session sorting uses the composite status logic consistently.
7. All existing tests pass; new tests cover composite status computation, message formatting, and rendering with various agent state combinations.

## Dependencies

- **Depends on**: PBI-40 (Done — external agent data model), PBI-41 (Done — per-agent TUI indicators)
- **Blocks**: None
- **External**: None

## Open Questions

None — message format decided: `"<status> (agenttype)"`.

## Related Tasks

[View Tasks](./tasks.md)
