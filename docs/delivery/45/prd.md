# PBI-45: Agent-Aware Audio Notifications for External Agents

[View in Backlog](../backlog.md)

## Overview

Extend navi's audio notification system to fire on external agent (OpenCode, etc.) status changes, not just the main Claude Code session status. Currently, when an external agent transitions to `permission`, `idle`, `error`, etc., no audio fires — only the root session status triggers notifications.

## Problem Statement

PBI-42 added audio notifications that trigger on Claude Code session status changes. PBI-40/41 added external agent status tracking and TUI display for agents like OpenCode. However, the audio notification system only monitors the root-level `session.Status` field — it ignores status changes in `session.Agents` (the external agent map). This means a user won't hear an audio alert when OpenCode needs permission or finishes work, even though the visual indicator updates in the TUI.

## User Stories

- As a user, I want audio notifications when an external agent's status changes so that I have the same audio awareness for OpenCode as I do for Claude Code sessions.

## Technical Approach

### State Tracking Extension

Add a parallel state map `lastAgentStates map[string]map[string]string` (keyed by `[sessionName][agentType]`) alongside the existing `lastSessionStates` in the TUI model. On each poll cycle, compare current `session.Info.Agents` against this map to detect transitions.

### Notification Trigger

When an external agent status change is detected, call `audioNotifier.Notify()` with a composite key like `"sessionName:agentType"` (e.g., `"my-session:opencode"`). This gives each agent independent cooldown tracking. The existing `Notify()` method already accepts any string key — no changes needed to the audio package.

### Background Attach Monitor

Extend `internal/monitor/monitor.go` with the same agent-aware detection so that external agent status changes fire audio while the user is attached to a tmux session.

### Files to Modify

- `internal/tui/model.go` — Add `lastAgentStates`, extend `detectStatusChanges()`, add `notifyAgentStatusChange()`
- `internal/monitor/monitor.go` — Add agent state tracking to background polling loop

### Files Unchanged

- `internal/audio/` — No changes needed; `Notify()` already works with composite keys
- `hooks/notify.sh` — Already preserves external agent data in status files
- `internal/session/` — `ExternalAgent` model already parsed from JSON

## UX/UI Considerations

N/A — no visual changes. Audio behavior uses the same sounds.yaml config that already exists. External agent status changes map to the same trigger names (working, idle, permission, error, done) as Claude Code sessions.

## Acceptance Criteria

1. When an external agent's status changes (e.g., OpenCode idle → permission), an audio notification fires using the same sound/TTS config as Claude Code sessions.
2. Each external agent has independent cooldown tracking — an OpenCode notification doesn't suppress a simultaneous Claude Code notification on the same session.
3. The background attach monitor (`internal/monitor/`) also fires audio for external agent status changes while the user is attached to a tmux session.
4. Existing main-session audio notification behavior is unchanged.
5. All existing tests pass; new tests cover external agent status change detection.

## Dependencies

- **Depends on**: PBI-40 (external agent data model), PBI-41 (external agent TUI display), PBI-42 (audio notification system)
- **Blocks**: None
- **External**: None

## Open Questions

None.

## Related Tasks

_Tasks will be created when this PBI is planned via `/plan-pbi 45`._
