# PBI-9: Desktop Notifications

[View in Backlog](../backlog.md)

## Overview

Add native desktop notifications when sessions change to attention-requiring states (permission, waiting, done, error) so users don't need to constantly monitor the TUI.

## Problem Statement

Users often have navi running in a terminal while working in other applications. When Claude needs attention (permission request, question, completion), users may not notice for extended periods. Desktop notifications provide immediate awareness without requiring constant visual monitoring.

## User Stories

- As a user, I want desktop notifications when Claude asks a question so I can respond promptly
- As a user, I want notifications when Claude needs permission so security-sensitive operations don't stall
- As a user, I want to configure which statuses trigger notifications so I'm not overwhelmed
- As a user, I want notification cooldown so rapid status changes don't spam me

## Technical Approach

### Notification Backends

Support multiple backends based on platform:

1. **Linux**: `notify-send` (libnotify)
2. **macOS**: `osascript` with display notification
3. **Generic**: Write to a notification file for external tools

### Implementation

1. Track previous session states in Model
2. On poll, compare new states to previous
3. For each state transition to a notifiable status:
   - Check cooldown (don't notify same session within N seconds)
   - Check user preferences (which statuses to notify)
   - Send notification via appropriate backend

### Notification Content

```
Title: "Claude - {session_name}"
Body: "{message}" or status description
Icon: navi icon or status-specific icon
Actions: "Open" → attach to session
```

### Configuration

```yaml
# ~/.config/navi/config.yaml
notifications:
  enabled: true
  backend: auto  # auto, notify-send, osascript, file
  cooldown_seconds: 30
  statuses:
    permission: true
    waiting: true
    done: false
    error: true
  sound: false  # or path to sound file
```

### Sound Alerts (Optional)

- Play system sound or custom audio file
- Use `paplay` (Linux) or `afplay` (macOS)
- Respect system Do Not Disturb settings

## UX/UI Considerations

- Notifications should be non-blocking
- Include session context in notification body
- Clicking notification should focus terminal (if possible)
- Provide visual indicator in TUI when notifications are enabled
- Show "notifications disabled" warning if backend unavailable

## Acceptance Criteria

1. Notifications sent when session status changes to configured states
2. Linux support via notify-send
3. macOS support via osascript
4. Notification cooldown prevents spam (configurable interval)
5. Configuration file controls notification behavior
6. Sound alerts optionally play on notification
7. TUI shows notification status in footer
8. Graceful fallback if notification backend unavailable

## Dependencies

- PBI-3: Session polling (for state change detection)

## Open Questions

- Should notifications group if multiple sessions change at once?
- Should we support notification actions (click to attach)?
- Should notifications include a preview of Claude's message?

## Related Tasks

See [Tasks](./tasks.md)
