# PBI-3: Session Polling & State Management

[View in Backlog](../backlog.md)

## Overview

Implement the polling logic that reads status files from `~/.claude-sessions/`, parses JSON, validates against live tmux sessions, and manages stale cleanup.

## Problem Statement

The TUI needs to continuously monitor the status directory for updates. It must parse JSON files, cross-reference with live tmux sessions to remove stale entries, and provide sorted session data to the view layer.

## User Stories

- As a user, I want the TUI to automatically refresh session status so that I see real-time updates
- As a user, I want stale sessions (where tmux session was killed) to be cleaned up automatically so the display stays accurate

## Technical Approach

1. Implement `pollSessions()` function that:
   - Reads all `*.json` files from `~/.claude-sessions/`
   - Parses each file into `SessionInfo` struct
   - Gets live tmux sessions via `tmux list-sessions -F '#{session_name}'`
   - Removes status files for sessions that no longer exist
   - Sorts sessions: `waiting` and `permission` first, then by timestamp descending
   - Returns `sessionsMsg` with the session list

2. Implement tick command that triggers polling every 500ms

3. Handle file read errors gracefully

### Polling Logic (from PRD)

On each tick:
1. Read all `*.json` files from the status directory
2. Parse into `[]SessionInfo` structs
3. Cross-reference with live tmux sessions via `tmux list-sessions -F '#{session_name}'`
4. Remove any status files whose session no longer exists (stale cleanup)
5. Sort: `waiting` and `permission` first (needs attention), then by timestamp descending
6. Send as a message to the Bubble Tea model to trigger re-render

## UX/UI Considerations

- Polling interval of 500ms provides responsive updates without excessive CPU usage
- Sorting prioritizes sessions that need attention

## Acceptance Criteria

1. `pollSessions()` reads all JSON files from status directory
2. JSON is correctly parsed into `SessionInfo` structs
3. Live tmux sessions are queried via `tmux list-sessions`
4. Status files for dead sessions are deleted (stale cleanup)
5. Sessions are sorted with `waiting`/`permission` first, then by timestamp
6. Tick command fires every 500ms
7. File read/parse errors are handled gracefully without crashing

## Dependencies

- PBI-1: Core types (`SessionInfo`, message types)
- PBI-2: Hook system must write JSON files for polling to read

## Open Questions

None

## Related Tasks

See [Tasks](./tasks.md)
