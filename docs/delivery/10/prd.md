# PBI-10: Webhook Integrations

[View in Backlog](../backlog.md)

## Overview

Add webhook integrations for Slack, Discord, and custom HTTP endpoints so users can receive Claude session alerts in their team communication tools or trigger external automations.

## Problem Statement

Desktop notifications are local to one machine. Teams working with Claude across multiple sessions and machines need centralized alerting. Developers may also want to integrate session events with other tools (monitoring, logging, custom dashboards).

## User Stories

- As a user, I want Slack notifications when Claude needs attention so my team can respond even if I'm away
- As a user, I want Discord webhooks for my personal server to track Claude activity
- As a user, I want custom webhooks to integrate with my monitoring stack

## Technical Approach

### Webhook Types

1. **Slack Incoming Webhook**
   - Uses Slack's webhook URL format
   - Formats message as Slack blocks with status, session, message
   - Supports channel mention (@here, @channel)

2. **Discord Webhook**
   - Uses Discord's webhook URL format
   - Formats as Discord embed with color based on status
   - Supports role mentions

3. **Generic HTTP Webhook**
   - POST to any URL
   - JSON payload with session data
   - Configurable headers (for auth tokens)

### Payload Format (Generic)

```json
{
  "event": "status_change",
  "session": {
    "name": "hyperion",
    "status": "permission",
    "previous_status": "working",
    "message": "Run: rm -rf ./dist?",
    "cwd": "/home/user/projects/hyperion",
    "timestamp": 1738627200
  },
  "host": "workstation-1",
  "navi_version": "1.0.0"
}
```

### Configuration

```yaml
# ~/.config/navi/config.yaml
webhooks:
  - name: team-slack
    type: slack
    url: https://hooks.slack.com/services/XXX/YYY/ZZZ
    statuses: [permission, error]
    mention: "@here"

  - name: personal-discord
    type: discord
    url: https://discord.com/api/webhooks/XXX/YYY
    statuses: [permission, waiting, done]

  - name: monitoring
    type: http
    url: https://my-monitoring.com/api/events
    method: POST
    headers:
      Authorization: "Bearer ${MONITORING_TOKEN}"
    statuses: [error]
```

### Implementation

1. On status change (similar to desktop notifications):
   - Check if status is in webhook's configured statuses
   - Apply cooldown per webhook per session
   - Queue webhook call (don't block main loop)
2. Use goroutine pool for async webhook delivery
3. Retry failed webhooks with exponential backoff
4. Log webhook failures but don't crash

## UX/UI Considerations

- Show webhook delivery status in TUI (optional debug mode)
- Provide `navi webhook test` command to verify configuration
- Support environment variable substitution in config
- Warn if webhook URLs look malformed

## Acceptance Criteria

1. Slack webhooks send formatted messages with session context
2. Discord webhooks send embeds with status-appropriate colors
3. Generic HTTP webhooks POST JSON to configured URL
4. Webhooks respect per-webhook status filters
5. Webhooks apply cooldown to prevent spam
6. Failed webhooks retry with backoff
7. Configuration supports multiple webhooks
8. Environment variables can be used in config (for secrets)
9. Test command verifies webhook connectivity

## Dependencies

- PBI-9: Desktop notifications (shared status change detection logic)

## Open Questions

- Should we support webhook templates (custom payload format)?
- Should we add Microsoft Teams support?
- Should there be a "webhook log" view in the TUI?

## Related Tasks

See [Tasks](./tasks.md)
