# PBI-2: Claude Code Hook System

[View in Backlog](../backlog.md)

## Overview

Create the notification hooks that Claude Code fires to write status updates to the shared status directory. This enables the TUI to monitor session states in real-time.

## Problem Statement

The TUI needs a way to know the current status of each Claude Code session. Claude Code supports hooks that fire on specific events. We need to implement hook scripts that write JSON status files that the TUI can read.

## User Stories

- As a user, I want Claude Code to automatically report its status so that the TUI can show me which sessions need attention
- As a user, I want status updates to include the session name, status, message, and working directory so that I have context at a glance

## Technical Approach

1. Create the hooks directory structure at `~/.claude-sessions/hooks/`
2. Implement `notify.sh` script that:
   - Accepts a status argument (`working`, `waiting`, `done`, `permission`, `error`)
   - Reads `CLAUDE_NOTIFICATION` environment variable for the message
   - Gets tmux session name via `tmux display-message`
   - Gets current working directory via tmux
   - Writes JSON to `~/.claude-sessions/<session>.json`

### Hook Script (from PRD)

```bash
#!/bin/bash
STATUS="$1"
MESSAGE="${CLAUDE_NOTIFICATION:-}"
DIR="$HOME/.claude-sessions"
mkdir -p "$DIR"

SESSION=$(tmux display-message -p '#{session_name}' 2>/dev/null || echo "unknown")
CWD=$(tmux display-message -p '#{pane_current_path}' 2>/dev/null || echo "")

cat > "$DIR/$SESSION.json" <<EOF
{
  "tmux_session": "$SESSION",
  "status": "$STATUS",
  "message": "$MESSAGE",
  "cwd": "$CWD",
  "timestamp": $(date +%s)
}
EOF
```

### Claude Code Hook Configuration

```json
{
  "hooks": {
    "Notification": [
      {
        "matcher": "",
        "hooks": [{ "type": "command", "command": "~/.claude-sessions/hooks/notify.sh waiting" }]
      }
    ],
    "Stop": [
      {
        "hooks": [{ "type": "command", "command": "~/.claude-sessions/hooks/notify.sh done" }]
      }
    ],
    "PreToolUse": [
      {
        "hooks": [{ "type": "command", "command": "~/.claude-sessions/hooks/notify.sh permission" }]
      }
    ],
    "SubagentStop": [
      {
        "hooks": [{ "type": "command", "command": "~/.claude-sessions/hooks/notify.sh working" }]
      }
    ]
  }
}
```

## UX/UI Considerations

N/A - This PBI is backend/hook infrastructure.

## Acceptance Criteria

1. `notify.sh` script is created and executable
2. Script correctly identifies tmux session name
3. Script writes valid JSON to the status directory
4. JSON includes all required fields: `tmux_session`, `status`, `message`, `cwd`, `timestamp`
5. Script handles cases where tmux is not available gracefully
6. Hook configuration JSON is documented and ready for installation

## Dependencies

- PBI-1: Core types must be defined to ensure JSON structure matches `SessionInfo`

## Open Questions

None

## Related Tasks

See [Tasks](./tasks.md)
