# PBI-18: Session Logs & Export

[View in Backlog](../backlog.md)

## Overview

Capture complete session logs for replay and provide export functionality to save session transcripts as markdown or other formats for documentation and review.

## Problem Statement

Claude conversations contain valuable information - decisions made, code explanations, problem-solving approaches. This knowledge is lost when sessions end. Users need to save and replay session content for documentation, learning, and sharing.

## User Stories

- As a user, I want to export a session transcript so I can document what Claude did
- As a user, I want to replay past sessions so I can review the conversation
- As a user, I want session logs for compliance and auditing purposes

## Technical Approach

### Log Capture

1. **Continuous Logging**
   - Capture tmux pane content periodically
   - Store in: `~/.claude-sessions/logs/<session>/<timestamp>.log`
   - Rotate logs to prevent unbounded growth

2. **Log Format**
   ```
   === 2024-01-15 10:30:45 ===
   [CLAUDE] I'll analyze the codebase structure...

   === 2024-01-15 10:30:52 ===
   [CLAUDE] Reading src/main.go...

   === 2024-01-15 10:31:05 ===
   [USER] Focus on the authentication module

   === 2024-01-15 10:31:08 ===
   [CLAUDE] I'll examine the auth package...
   ```

3. **Log Collection Methods**
   - Option A: Periodic `tmux capture-pane` with diff
   - Option B: Hook into Claude Code's native logging
   - Option C: Use tmux's pipe-pane feature

### Export Formats

1. **Markdown** (default)
   ```markdown
   # Session: hyperion
   **Date**: 2024-01-15 10:30 - 12:45
   **Directory**: ~/projects/hyperion
   **Branch**: feature/auth

   ## Conversation

   ### Claude
   I'll analyze the codebase structure...

   ### User
   Focus on the authentication module

   ### Claude
   I'll examine the auth package...
   ```

2. **JSON** (for programmatic use)
   ```json
   {
     "session": "hyperion",
     "messages": [
       {"role": "claude", "content": "...", "timestamp": 1705312245},
       {"role": "user", "content": "...", "timestamp": 1705312265}
     ]
   }
   ```

3. **HTML** (for web viewing)
   - Styled conversation view
   - Syntax-highlighted code blocks
   - Collapsible tool outputs

### Replay Feature

1. Press `L` on a session (or in history view) to open log viewer
2. Scrollable view of conversation
3. Search within logs
4. Navigate by timestamp
5. Export from this view

### Implementation

1. **Log Viewer Component**
   - Scrollable text view (similar to `less`)
   - Syntax highlighting for code blocks
   - Search with `/` key
   - Go to timestamp with `g`

2. **Export Command**
   - `E` in log viewer exports current session
   - Prompt for format and destination
   - Or: `navi export <session> --format md --output file.md`

### Storage Management

- Log retention configurable (days or size limit)
- Compress old logs (gzip)
- `navi logs prune` command for cleanup

## UX/UI Considerations

- Log viewer should handle large files smoothly
- Export should show progress for large sessions
- Format selection should be intuitive
- Include session metadata in exports

## Acceptance Criteria

1. Session logs captured continuously during active sessions
2. Logs stored in structured format with timestamps
3. `L` opens log viewer for selected/historical session
4. Log viewer supports scrolling and search
5. `E` exports session to markdown format
6. Export supports JSON and HTML formats
7. CLI export command available
8. Log retention policy configurable
9. Old logs can be compressed/pruned
10. Export includes session metadata (date, directory, branch)

## Dependencies

- PBI-17: Session history (for historical log access)
- PBI-11: Preview pane (similar viewer patterns)

## Open Questions

- Should we capture tool inputs/outputs separately?
- Should export include file diffs created during session?
- Should logs be encrypted at rest (for sensitive work)?

## Related Tasks

See [Tasks](./tasks.md)
