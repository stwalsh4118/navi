# PBI-12: Session Metrics

[View in Backlog](../backlog.md)

## Overview

Display session metrics including token usage, time tracking, and tool activity so users can understand resource consumption and Claude's behavior patterns.

## Problem Statement

Users have limited visibility into what Claude is doing and how much it's consuming. Token usage affects costs, time tracking helps estimate work, and tool activity reveals what operations Claude is performing. This information should be visible without deep investigation.

## User Stories

- As a user, I want to see token usage per session so I can track API costs
- As a user, I want to see how long a session has been active and working so I can estimate completion
- As a user, I want to see which tools Claude is using so I can understand what operations are happening

## Technical Approach

### Data Collection

Metrics require extending the hook system to capture more data:

1. **Token Usage**
   - Hook into Claude Code's output or logs to extract token counts
   - Track input/output tokens separately
   - Accumulate per session

2. **Time Tracking**
   - Track total session lifetime
   - Track time in each status (working, waiting, etc.)
   - Calculate "active" vs "idle" time

3. **Tool Activity**
   - Capture tool use events from hooks
   - Track tool call counts by type
   - Track recent tools in a rolling window

### Extended Session JSON

```json
{
  "tmux_session": "hyperion",
  "status": "working",
  "message": "Implementing feature...",
  "cwd": "/home/user/projects/hyperion",
  "timestamp": 1738627200,
  "metrics": {
    "tokens": {
      "input": 45000,
      "output": 12000,
      "total": 57000
    },
    "time": {
      "started": 1738620000,
      "total_seconds": 7200,
      "working_seconds": 3600,
      "waiting_seconds": 1800
    },
    "tools": {
      "recent": ["Read", "Edit", "Bash"],
      "counts": {
        "Read": 45,
        "Edit": 12,
        "Bash": 8,
        "Write": 3
      }
    }
  }
}
```

### Display Options

1. **Inline Badges**
   - Show token count and time next to session name
   - `hyperion  12k tokens  1h 23m`

2. **Detail View**
   - Press `i` for detailed info overlay
   - Show all metrics in a modal

3. **Dashboard View**
   - Aggregate metrics across all sessions
   - Total tokens, total time, tool breakdown

### Implementation

1. Extend notify.sh hook to capture metrics from Claude Code environment
2. Add metrics parsing to session loader
3. Create metrics display components
4. Add aggregate dashboard view

## UX/UI Considerations

- Keep inline display minimal (don't clutter the list)
- Use abbreviations (12k, 1h 23m) for compact display
- Color-code token usage if exceeding thresholds
- Detail view should be comprehensive but organized
- Consider graphs for time tracking (ASCII sparklines)

## Acceptance Criteria

1. Token usage (input/output) is captured and displayed per session
2. Time tracking shows session duration and active time
3. Tool activity shows recent and cumulative tool usage
4. Metrics update in real-time with session status
5. Detail view (`i` key) shows comprehensive metrics
6. Aggregate dashboard shows totals across sessions
7. Metrics persist across TUI restarts (stored in session files)

## Dependencies

- PBI-2: Hook system (for extended data capture)
- PBI-11: Preview pane (similar UI patterns for detail view)

## Open Questions

- ~~Can we access token counts from Claude Code's output?~~ **RESOLVED: No** - Research confirmed tokens are only available via OpenTelemetry telemetry, not hooks.
- Should metrics be stored separately from status files?
- Should we support exporting metrics to CSV/JSON?

## Implementation Notes

### Token Tracking Limitation
Research in task 12-4 confirmed that **token counts are NOT available via Claude Code hooks**. The only way to access token usage data is through OpenTelemetry telemetry, which requires setting up external infrastructure (OTLP endpoint, Honeycomb, etc.). This is outside the scope of the hook-based architecture.

The implementation tracks:
- **Time metrics**: Session duration, working time, waiting time
- **Tool metrics**: Tool call counts and recent tools list

Token tracking could be added in the future if OpenTelemetry integration is implemented.

## Related Tasks

[View Tasks](./tasks.md)
