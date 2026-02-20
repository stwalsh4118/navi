# PBI-57: Session RAM Usage Monitoring

[View in Backlog](../backlog.md)

## Overview

Add per-session RAM usage monitoring to the Navi TUI sidebar. Each session displays a compact RAM badge showing the total resident memory of all processes in its tmux session process tree, polled every 2 seconds.

## Problem Statement

Claude Code and OpenCode have memory leaks that cause steadily increasing RAM usage over time. Currently there is no way to see how much memory a session is consuming without manually inspecting processes. Users need an at-a-glance indicator in the session list to spot runaway memory usage before it impacts system stability.

## User Stories

- As a user, I want to see the RAM usage of each tmux session in the sidebar so that I can spot memory leaks at a glance without leaving the TUI

## Technical Approach

### Process Tree Walking via /proc

For each tmux session, get the root shell PIDs using `tmux list-panes -t <session> -F '#{pane_pid}'`, then recursively walk the process tree via `/proc/<pid>/children` (or `/proc/<pid>/stat`) to find all descendant processes. Sum the RSS values from `/proc/<pid>/statm` or `/proc/<pid>/status` for the full tree.

This approach avoids spawning subprocesses (no `ps` calls) and is efficient for a 2-second polling interval.

### Data Model

Add a resource metrics type (e.g., `ResourceMetrics` with `RSSBytes int64`) to the session model. This sits alongside the existing token/time/tool metrics.

### Polling

Use a separate tick interval (2 seconds) from the main 500ms session poll. The resource poll runs independently and updates session data in place. Only local sessions are measured — remote sessions will not display a RAM badge.

### Rendering

Add a RAM badge to the existing metrics badge line in the sidebar session row, following the same compact format as tokens/tools (e.g., "256M", "1.2G").

## UX/UI Considerations

- RAM badge renders inline with existing metric badges (time, tools, tokens)
- Uses human-readable format: bytes < 1024 → "512K", megabytes → "256M", gigabytes → "1.2G"
- Remote sessions show no RAM badge (data not available without remote process inspection)
- Badge should not add visual clutter — same dimmed style as existing metric badges

## Acceptance Criteria

1. Each local session in the sidebar displays a RAM usage badge showing total RSS of all processes in the tmux session's process tree
2. RAM is calculated by walking the process tree from tmux pane PIDs via `/proc` filesystem (no subprocess spawning)
3. RAM usage is polled every 2 seconds, independently of the main 500ms session poll
4. RAM is formatted human-readably (e.g., "256M", "1.2G")
5. Remote sessions do not display a RAM badge
6. RAM badge renders alongside existing metric badges (time, tools, tokens) with consistent styling

## Dependencies

- **Depends on**: None
- **Blocks**: None
- **External**: Linux `/proc` filesystem (platform-specific; no Windows support needed per current backlog)

## Open Questions

None — resolved during brainstorm:
- Approach: `/proc` filesystem walking (no subprocess spawning) ✓
- Display location: Session list sidebar ✓
- Poll interval: 2 seconds ✓
- Alerts: Informational only (no thresholds) ✓

## Related Tasks

[View Tasks](./tasks.md)
