# PBI-33: Agent Team Awareness and Hook Robustness

[View in Backlog](../backlog.md#user-content-33)

## Overview

Improve the hook system and TUI to correctly handle Claude Code agent teams and fix stale status issues. When a session spawns an agent team, the TUI should show inline information about the team's agents and their statuses, rather than treating each teammate's hook events as if they were the main session. Additionally, fix the issue where answering or dismissing a permission/question dialog leaves the status icon stuck.

## Problem Statement

The current hook and status system was designed for single-session Claude Code usage. Two categories of bugs have emerged:

### 1. Stale Permission/Question Status
When Claude Code fires a `PermissionRequest` hook, navi sets the `❓` icon. However, **no hook fires when the user answers or dismisses the dialog**, so the icon stays stuck until the next `UserPromptSubmit` event. This means:
- Approving a permission clears the icon only indirectly (when `PostToolUse` fires after the tool runs)
- Denying or canceling leaves the `❓` stuck indefinitely
- The same applies to the `waiting` status set by question dialogs

### 2. Agent Team Interference
When Claude Code agent teams are used, each teammate runs in its own tmux pane with the same hooks config. This causes:
- **False "done" status**: A teammate finishing fires `Stop`, writing `done` ✅ to its session file, even though the main session is still working
- **False "offline" status**: A teammate's `SessionEnd` writes `offline` ⏹️
- **Permission noise**: Teammate permission requests show as if the main session needs attention
- **No team visibility**: Users have no way to see that an agent team is active, how many agents are running, or what each agent is doing

### Root Cause
The hooks write status based on tmux session name alone, with no awareness of whether the hook caller is the main session or a teammate sub-process. Claude Code now provides teammate-identifying fields (`teammate_name`, `team_name`) in hook input JSON, but navi doesn't read or use them.

## User Stories

- As a user, I want the permission/question icon to clear when I answer or dismiss the dialog so that the session status is always accurate.
- As a user, I want to see when a session is running an agent team so that I understand why it's active for a long time.
- As a user, I want to see each agent in the team and its current status (working, idle, done) so that I can monitor team progress inline.
- As a user, I want teammate hook events to not corrupt the main session's status so that the dashboard remains reliable.

## Technical Approach

### Hook Changes

1. **Read stdin JSON in hooks**: Update `notify.sh` to parse the JSON input from stdin to extract `teammate_name`, `team_name`, `hook_event_name`, and `session_id`.

2. **Add new hook events**: Register hooks for:
   - `PostToolUse` → `notify.sh working` (clears permission/question status after tool approval)
   - `SubagentStart` → track agent spawn with `agent_type` and `agent_id`
   - `SubagentStop` → track agent completion
   - `TeammateIdle` → track teammate going idle (includes `teammate_name`, `team_name`)
   - `TaskCompleted` → track task completion (includes `teammate_name`, `team_name`, `task_id`)

3. **Teammate-aware status writing**: When a hook detects `teammate_name` in the input JSON, write teammate status to a nested structure within the session JSON rather than overwriting the main session status. For example:
   ```json
   {
     "tmux_session": "api",
     "status": "working",
     "team": {
       "name": "my-project",
       "agents": [
         {"name": "researcher", "status": "working", "timestamp": 1707506400},
         {"name": "implementer", "status": "idle", "timestamp": 1707506350}
       ]
     }
   }
   ```

4. **Fix stale permission status**: Add a `PostToolUse` hook that sets status back to `working`, which naturally clears the `❓` after a permission is approved and the tool runs.

### Session Data Model Changes

5. **Extend `session.Info`**: Add team-related fields:
   ```go
   type AgentInfo struct {
       Name      string `json:"name"`
       Status    string `json:"status"`
       Timestamp int64  `json:"timestamp"`
   }

   type TeamInfo struct {
       Name   string      `json:"name"`
       Agents []AgentInfo `json:"agents"`
   }
   ```
   Add `Team *TeamInfo` field to `session.Info`.

### TUI Changes

6. **Render team info inline**: When a session has team data, display it below the existing session lines:
   - Summary line showing team name and agent count (e.g., `Team: my-project (3 agents)`)
   - Each agent rendered with its own status icon and name (e.g., `  ⚙️ researcher  ⏳ implementer  ✅ tester`)
   - Compact format that fits within the session's existing space allocation

7. **Session sorting awareness**: If any teammate has `permission` or `waiting` status, the main session must sort to the top as if it had that status itself.

### Hook Script Architecture

8. **Generalize notify.sh**: Refactor to handle the richer hook input. The script needs to:
   - Always read stdin JSON (currently only `tool-tracker.sh` reads stdin)
   - Detect main session vs. teammate based on presence of `teammate_name`
   - Route status updates to the correct location in the session JSON
   - Handle new hook event types appropriately

## UX/UI Considerations

- Team info should be visually subordinate to the main session info - indented, dimmed, or compact
- Agent status icons should use the same icon set as sessions for consistency
- When no team is active, no extra lines should appear (zero visual overhead)
- The team agent list is always shown when a team is active (no collapse toggle)
- Consider showing a small agent count badge next to the session name when a team is active (e.g., `api ⚙️ [3 agents]`)

## Acceptance Criteria

1. Answering or dismissing a permission request clears the `❓` icon from the session status
2. Teammate hook events do not overwrite the main session's status
3. When an agent team is active, the session shows the team name and number of active agents
4. Each teammate's name and current status is visible inline under the session
5. Teammate statuses update in real-time as hooks fire (within the normal polling interval)
6. When all teammates finish or the team is shut down, the team info is cleaned up
7. The existing session display is unchanged when no agent team is running
8. All existing tests continue to pass
9. New tests cover teammate status parsing, team info rendering, and the permission-clearing fix

## Dependencies

- PBI-2 (Hooks) - Done. Provides the existing hook infrastructure.
- PBI-4 (Styled TUI) - Done. Provides the session rendering that will be extended.
- PBI-12 (Session Metrics) - Done. Provides the metrics system that teammates should not corrupt.

## Open Questions

All resolved:

1. ~~Should teammate metrics (tokens, time, tools) be aggregated into the main session's metrics, shown separately per agent, or both?~~ **Resolved: Ignore agent metrics for now. Only track agent status, not their token/time/tool metrics.**
2. ~~When a teammate needs permission, should the main session's sort priority be elevated (surfacing it as needing attention)?~~ **Resolved: Yes - if any teammate has permission/waiting status, the main session sorts to the top.**
3. ~~Should there be a keybinding to expand/collapse the team agent list for a session?~~ **Resolved: No - always show the agent list when a team is active. No toggle needed.**
4. ~~How should the `Notification` hook's `permission_prompt` matcher interact with the existing `PermissionRequest` hook?~~ **Resolved: Keep `PermissionRequest` only. Fix the stale icon purely with `PostToolUse` clearing it back to working. Don't add the `Notification` hook.**

## Related Tasks

See [tasks.md](./tasks.md) for the task breakdown.
