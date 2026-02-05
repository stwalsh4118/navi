# PBI-21: Auto-Start & Custom Hooks

[View in Backlog](../backlog.md)

## Overview

Add configuration-driven auto-start for sessions and custom hooks that trigger scripts when session status changes, enabling automation and workflow integration.

## Problem Statement

Users often start the same Claude sessions every day (per-project sessions) and want to run custom scripts when Claude events occur (notifications, logging, integrations). This requires manual setup and custom scripting outside navi.

## User Stories

- As a user, I want navi to auto-start my common sessions so I don't configure them daily
- As a user, I want to run custom scripts when session status changes for workflow automation
- As a user, I want startup configuration to include session-specific settings

## Technical Approach

### Auto-Start Configuration

```yaml
# ~/.config/navi/startup.yaml
auto_start:
  enabled: true
  delay_between_ms: 1000  # Stagger session creation

sessions:
  - name: main-app
    directory: ~/work/main-app
    branch: main  # Optional: checkout this branch first
    on_create: |
      echo "Starting main-app session"

  - name: api
    directory: ~/work/api
    command: "claude --model opus"  # Custom claude command
    tags: [backend]

  - name: docs
    directory: ~/work/docs
    condition: "test -f TODO.md"  # Only start if condition true
```

### Auto-Start Behavior

1. On `navi` startup, check for existing sessions
2. For each configured session not already running:
   - Check condition (if specified)
   - Create tmux session with configured parameters
   - Run `on_create` hook if specified
3. Show startup summary:
   ```
   Auto-started 2 sessions:
   • main-app
   • api
   Skipped (already running): docs
   ```

### Custom Status Hooks

```yaml
# ~/.config/navi/hooks.yaml
hooks:
  on_status_change:
    - name: log-all
      script: ~/.config/navi/scripts/log.sh
      statuses: [all]  # Trigger on any change

    - name: urgent-alert
      script: ~/.config/navi/scripts/alert.sh
      statuses: [permission, error]
      sessions: [production, deploy]  # Only these sessions

    - name: done-notify
      script: ~/.config/navi/scripts/done.sh
      statuses: [done]
      cooldown_seconds: 60
```

### Hook Script Environment

Scripts receive environment variables:
```bash
NAVI_SESSION="hyperion"
NAVI_STATUS="permission"
NAVI_PREVIOUS_STATUS="working"
NAVI_MESSAGE="Run: rm -rf ./dist?"
NAVI_CWD="/home/user/projects/hyperion"
NAVI_TIMESTAMP="1738627200"
```

### Implementation

1. **Startup Manager**
   - Parse startup.yaml on navi launch
   - Check existing tmux sessions
   - Create missing sessions sequentially
   - Run on_create hooks

2. **Hook Manager**
   - Track previous session states
   - On state change, find matching hooks
   - Execute scripts asynchronously
   - Respect cooldowns

3. **CLI Commands**
   ```bash
   navi startup         # Run auto-start manually
   navi startup --dry   # Show what would start
   navi hooks test      # Trigger test hook execution
   ```

### Safety Features

- Scripts run with timeout (default 30s)
- Failed scripts don't crash navi
- Hook execution logged
- Option to disable all hooks

## UX/UI Considerations

- Show auto-start progress on launch
- Indicate hook failures in footer/status
- Provide hook debug mode for troubleshooting
- Document environment variables clearly

## Acceptance Criteria

1. Auto-start configuration in startup.yaml
2. Sessions auto-created on navi launch
3. Conditions supported for conditional startup
4. `on_create` hooks run after session creation
5. Custom status hooks configurable in hooks.yaml
6. Hooks receive session data as environment variables
7. Hooks respect cooldown settings
8. Hook failures logged but don't crash navi
9. `navi startup` command for manual trigger
10. `navi startup --dry` shows what would happen

## Dependencies

- PBI-7: Session management (for session creation)
- PBI-9: Desktop notifications (similar notification patterns)

## Open Questions

- Should hooks be able to cancel/modify events?
- Should there be built-in hook templates?
- Should auto-start support dependencies between sessions?

## Related Tasks

See [Tasks](./tasks.md)
