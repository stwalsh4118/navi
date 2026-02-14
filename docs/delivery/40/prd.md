# PBI-40: Tmux Pane Agent Detection

[View in Backlog](../backlog.md#user-content-40)

## Overview

Extend navi's session monitoring to be pane-aware, automatically detecting which AI coding agents (Claude Code, Codex, etc.) are running in each tmux session's panes. This provides the infrastructure for multi-agent workflow tracking where a single tmux session may contain multiple agent panes.

## Problem Statement

Navi currently models each tmux session as a single entity tracked via Claude Code hooks. The user's workflow has evolved to run multiple AI agents in separate panes within the same tmux session (e.g., Claude Code for planning in one pane, Codex for implementation in another). Navi has no awareness of individual tmux panes or non-Claude-Code agents, so there is no visibility into what agents are active in a session beyond what the hooks report.

### Current Limitations

- **No pane awareness**: Navi only queries `tmux list-sessions`, never `tmux list-panes`
- **Single-agent assumption**: The data model assumes one agent (Claude Code) per session
- **No process inspection**: No mechanism exists to detect what program is running in a given pane
- **Claude Code only**: Only Claude Code is recognized as an agent; other tools like Codex are invisible

## User Stories

- As a user, I want navi to detect which AI agents are running in each tmux session so that I can see all active agents at a glance.
- As a user, I want agent detection to work automatically without manual configuration so that I don't have to label panes myself.
- As a user, I want the detection system to be extensible so that new agents can be added easily in the future.

## Technical Approach

### Pane Discovery

Add a pane polling step to the existing session polling cycle:

```bash
tmux list-panes -t <session> -F "#{pane_id} #{pane_pid} #{pane_current_command}"
```

This gives the PID of the shell process in each pane, which is the starting point for process tree inspection.

### Process Tree Walking

From each pane's shell PID, walk the child process tree to find known agent binaries:

```bash
pgrep -P <pane_pid> -a
```

Or recursively via `/proc/<pid>/children` on Linux. Match process names against a known agent registry:

- `claude` → Claude Code
- `codex` → Codex

### Agent Registry

An extensible mapping of process names to agent types:

```go
type AgentType struct {
    ID          string   // "claude-code", "codex"
    ProcessNames []string // ["claude"], ["codex"]
    DisplayName string   // "Claude Code", "Codex"
}
```

New agents are added by extending this registry. No code changes needed beyond adding an entry.

### Data Model Extension

Extend `session.Info` to carry per-pane agent detection results:

```go
type DetectedAgent struct {
    Type   string `json:"type"`    // "claude-code", "codex"
    PaneID string `json:"pane_id"`
    Status string `json:"status"`  // "running", "stopped"
    PID    int    `json:"pid"`
}
```

Add an `Agents []DetectedAgent` field to `session.Info`. Claude Code's hook-driven status remains the primary session status; detected agents are supplementary metadata.

### Integration with Existing Polling

The pane scan runs as part of the existing 500ms poll cycle (or at a slower cadence if performance requires it). For each session returned by `tmux list-sessions`:

1. Run `tmux list-panes` to get pane PIDs
2. For each pane, walk the process tree looking for known agents
3. Attach detection results to the session's `Info` struct
4. Claude Code detection cross-references with the existing hook-driven status (don't duplicate)

### Remote Session Considerations

For remote sessions polled via SSH, pane detection would require running the same pane/process inspection commands over SSH. This can be deferred to a follow-up PBI if needed, focusing on local sessions first.

## UX/UI Considerations

This PBI is infrastructure only - no TUI changes. PBI-41 handles display.

## Acceptance Criteria

1. Navi detects `claude` and `codex` processes running in tmux panes
2. Detection results are available on the `session.Info` struct as `[]DetectedAgent`
3. Detection does not interfere with or duplicate existing hook-driven Claude Code status
4. The agent registry is extensible (adding a new agent type requires only a config/data change)
5. Pane scanning does not degrade polling performance (under 50ms overhead per cycle)
6. All existing tests continue to pass
7. New unit tests cover process detection, agent classification, and pane parsing

## Dependencies

- PBI-3 (Session Polling) - Done. Provides the polling infrastructure being extended.
- PBI-33 (Agent Team Awareness) - Done. Established the `AgentInfo` pattern and team-aware data model.

## Open Questions

1. ~~Should pane detection run every poll cycle (500ms) or at a slower cadence?~~ To be determined during implementation based on performance testing.
2. Should remote sessions also get pane detection, or defer to a follow-up? Recommend deferring.

## Related Tasks

See [tasks.md](./tasks.md) for the task breakdown.
