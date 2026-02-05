# PBI-17: Session History & Bookmarks

[View in Backlog](../backlog.md)

## Overview

Track session activity history over time and allow users to bookmark important sessions for easy access, providing a record of what Claude accomplished.

## Problem Statement

Once a session ends or is closed, there's no record of what was done. Users can't review past sessions, learn from previous work, or quickly return to important sessions. History and bookmarks solve these problems.

## User Stories

- As a user, I want to see a history of completed sessions so I can review past work
- As a user, I want to bookmark important sessions so I can find them quickly
- As a user, I want to see what a session accomplished over its lifetime

## Technical Approach

### Session History

1. **History Storage**
   - Location: `~/.claude-sessions/history/`
   - Format: `<session>-<timestamp>.json`
   - Store when session ends or status changes to "done"/"offline"

2. **History Entry**
   ```json
   {
     "session_name": "hyperion",
     "started": 1738620000,
     "ended": 1738627200,
     "duration_seconds": 7200,
     "cwd": "/home/user/projects/hyperion",
     "summary": {
       "status_changes": 12,
       "tools_used": ["Read", "Edit", "Bash"],
       "files_modified": ["main.go", "handler.go"],
       "final_message": "Task completed successfully"
     },
     "git": {
       "branch": "feature/auth",
       "commits_made": 2
     }
   }
   ```

3. **History Collection**
   - Hook on `SessionEnd` captures final state
   - Optionally track file changes via git diff
   - Store tool usage counts accumulated during session

### Bookmarks

1. **Bookmark Storage**
   - Location: `~/.claude-sessions/bookmarks.json`
   - Simple list of session names with metadata

2. **Bookmark Entry**
   ```json
   {
     "session": "hyperion",
     "note": "OAuth implementation",
     "created": 1738627200,
     "tags": ["important", "auth"]
   }
   ```

3. **Bookmark Operations**
   - Press `b` to toggle bookmark on current session
   - Press `B` to view bookmarks list
   - Bookmarked sessions show ★ indicator

### History View

Press `H` to open history browser:
```
╭─ Session History ──────────────────────────────────────────╮
│                                                            │
│  Today                                                     │
│  ────────────────────────────────────────────────────────  │
│  hyperion          2h 15m    feature/auth    3 commits    │
│  api               45m       fix/user-bug    1 commit     │
│                                                            │
│  Yesterday                                                 │
│  ────────────────────────────────────────────────────────  │
│  dotfiles          1h 30m    main            2 commits    │
│                                                            │
│  [Enter] View details  [/] Search  [q] Close              │
╰────────────────────────────────────────────────────────────╯
```

### History Detail View

Select a history entry to see:
- Full timeline of status changes
- Tools used with counts
- Files modified
- Git commits made
- Final message

## UX/UI Considerations

- History view should be searchable
- Bookmarks should be prominently visible in main list
- History retention policy (delete after N days, or keep last N entries)
- Export option for history data

## Acceptance Criteria

1. Session history captured when sessions end
2. History includes duration, tools used, files modified
3. History stored in structured files
4. `H` opens history browser with grouped entries
5. History entries can be viewed in detail
6. `b` toggles bookmark on current session
7. Bookmarked sessions show ★ indicator
8. `B` opens bookmarks list
9. Bookmarks persist across restarts
10. History can be searched by name, date, or content

## Dependencies

- PBI-2: Hook system (for end-of-session capture)
- PBI-12: Session metrics (for detailed activity data)
- PBI-8: Session organization (for bookmark tags)

## Open Questions

- How long should history be retained?
- Should history include full session transcripts?
- Should bookmarks support folders/categories?

## Related Tasks

See [Tasks](./tasks.md)
