# PBI-40: OpenCode Status Hook Integration

[View in Backlog](../backlog.md#user-content-40)

## Overview

Add OpenCode as a second supported AI coding agent in navi by creating an OpenCode plugin that pushes status updates into the existing session status file. This mirrors the Claude Code hooks pattern (PBI-2) — OpenCode's plugin system actively writes status rather than navi passively detecting processes.

## Problem Statement

Navi currently tracks only Claude Code sessions via its hook system. The user's workflow has evolved to run multiple AI agents in the same tmux session (e.g., Claude Code in one pane, OpenCode in another). OpenCode sessions are completely invisible to navi.

### Current Limitations

- **Claude Code only**: Only Claude Code is recognized as an agent; OpenCode is invisible
- **Single-agent data model**: `session.Info` has no concept of multiple agent types per session
- **No extensibility**: No mechanism exists for other agents to push status into navi's pipeline

### Why Plugin-Based (Not Process Detection)

OpenCode provides a plugin/hooks system that can actively push status updates — the same pattern Claude Code hooks use. This is more reliable and richer than passive process tree inspection because:

- **Active push vs passive poll**: Status updates arrive immediately on events, no scanning delay
- **Richer data**: Plugin receives structured event data (session status, permissions, etc.)
- **No PID walking**: Avoids fragile process tree inspection across platforms
- **Extensible**: New agents with plugin/hook systems can follow the same pattern

## User Stories

- As a user, I want OpenCode to report its status to navi so that I can see OpenCode session state alongside Claude Code.
- As a user, I want the integration to work automatically after installation so that I don't have to configure anything manually.
- As a user, I want the existing Claude Code tracking to be unaffected so that nothing breaks for my current workflow.

## Technical Approach

### OpenCode Plugin

Create a navi plugin for OpenCode that hooks into lifecycle events and writes status to the session's JSON file. The plugin is written in JavaScript/TypeScript and deployed to `~/.config/opencode/plugins/`.

#### Event-to-Status Mapping

| OpenCode Hook | Navi Status |
|---|---|
| `session.created` | `working` |
| `tool.execute.after` | `working` |
| `permission.asked` | `permission` |
| `permission.replied` | `working` |
| `session.idle` | `idle` |
| `session.error` | `error` |

#### Plugin Behavior

1. Determine the tmux session name (via `tmux display-message -p '#{session_name}'`)
2. Read the existing status file (`~/.claude-sessions/{SESSION_NAME}.json`)
3. Update the `.agents.opencode` section with current status and timestamp
4. Write the file back atomically

```javascript
export const NaviPlugin = async ({ $ }) => {
  return {
    "permission.asked": async () => {
      await updateStatus($, "permission")
    },
    "tool.execute.after": async () => {
      await updateStatus($, "working")
    },
    "session.idle": async () => {
      await updateStatus($, "idle")
    },
    // ... etc
  }
}
```

The plugin writes JSON directly — no shell script intermediary.

### Single Status File (Shared)

Both Claude Code and OpenCode write to the same `{SESSION_NAME}.json` file. The structure extends to include an `agents` map:

```json
{
  "tmux_session": "api",
  "status": "working",
  "message": "Running tests",
  "cwd": "/home/user/api",
  "timestamp": 1707506400,
  "agents": {
    "opencode": {
      "status": "idle",
      "timestamp": 1707506380
    }
  }
}
```

- **Root-level fields** (`status`, `message`, etc.) remain Claude Code's domain — written by `notify.sh` as today
- **`agents` map** holds non-Claude-Code agent statuses, keyed by agent type
- Each writer does read-modify-write, preserving the other's fields
- Race risk is negligible (events fire seconds apart, write window is microseconds, next event self-corrects)

### Data Model Extension

Extend `session.Info` with an `Agents` field:

```go
type ExternalAgent struct {
    Status    string `json:"status"`
    Timestamp int64  `json:"timestamp"`
}

type Info struct {
    // ... existing fields ...
    Agents map[string]ExternalAgent `json:"agents,omitempty"`
}
```

### notify.sh Compatibility

Update `notify.sh` to preserve the `agents` field during its read-modify-write cycle. This is a one-line addition to the existing field preservation logic.

### Install Script

Extend `install.sh` to:

1. Copy the OpenCode plugin to `~/.config/opencode/plugins/`
2. Create the plugins directory if it doesn't exist
3. Display setup confirmation

### Scope Boundaries

This PBI covers infrastructure only — no TUI changes. Just status for now — no metrics, tool tracking, or team support for OpenCode. Those can be added in future PBIs. PBI-41 handles the TUI display.

## Acceptance Criteria

1. An OpenCode plugin exists that writes status updates to the session status file on lifecycle events
2. The `agents` map in the status JSON correctly reflects OpenCode's current status
3. `session.Info` struct includes `Agents map[string]ExternalAgent` and is populated from JSON
4. `notify.sh` preserves the `agents` field (does not clobber OpenCode data)
5. Install script deploys the OpenCode plugin to `~/.config/opencode/plugins/`
6. Existing Claude Code hook behavior is completely unaffected
7. All existing tests continue to pass
8. New unit tests cover agent field parsing and multi-agent status file reading

## Dependencies

- PBI-2 (Claude Code Hooks) — Done. Established the hooks → status file → poller pattern.
- PBI-33 (Agent Team Awareness) — Done. Established multi-agent data model patterns.

## Open Questions

1. ~~Should we use process tree detection or plugin-based status pushing?~~ Plugin-based — active push is more reliable and richer than passive detection.
2. ~~Single status file or separate files per agent?~~ Single file with `agents` map — avoids merge logic in the poller.
3. Should the plugin cache the tmux session name or look it up on every event? Recommend caching at plugin init.

## Related Tasks

[View Tasks](./tasks.md)
