# PBI-6: Installation Script

[View in Backlog](../backlog.md)

## Overview

Create an installation script that sets up the hooks, configures Claude Code settings, and optionally installs the TUI binary.

## Problem Statement

Users need an easy way to deploy claude-sessions. This involves copying hook scripts, creating directories, merging hook configuration into Claude Code's settings.json, and building/installing the binary.

## User Stories

- As a user, I want a single command to install claude-sessions so I don't have to manually configure everything
- As a user, I want the installer to preserve my existing Claude Code settings while adding the hooks

## Technical Approach

1. Create `install.sh` that:
   - Creates `~/.claude-sessions/` directory
   - Copies `notify.sh` to `~/.claude-sessions/hooks/` and makes it executable
   - Reads `~/.claude/settings.json`
   - Detects and handles conflicts with existing hooks
   - Merges hook configuration (with user consent if conflicts)
   - Writes updated settings.json back
   - Builds the Go binary with `go build`
   - Optionally copies binary to `~/.local/bin/`

### Installation Steps (from PRD)

1. Copies `notify.sh` to `~/.claude-sessions/hooks/` and makes it executable
2. Creates `~/.claude-sessions/` if it doesn't exist
3. Reads `~/.claude/settings.json`, detects conflicts, prompts user if needed, merges the hook config, writes it back
4. Builds the Go binary and optionally copies it to `~/.local/bin/`

## UX/UI Considerations

- Script should be idempotent (safe to run multiple times)
- Clear output showing what was done
- Handle missing settings.json gracefully (create new file)

### Conflict Detection and Resolution

When merging hook configuration, the installer must detect conflicts with existing hooks:

1. **No Conflicts**: If the user has no existing hooks for `Notification`, `Stop`, `PreToolUse`, or `SubagentStop`, merge automatically
2. **Conflicts Detected**: If existing hooks use any of the same hook types:
   - Display which hook types have existing configurations
   - Show the existing configuration for those hooks
   - Prompt user with options:
     - **Force merge/override**: Replace conflicting hooks with navi hooks (backup original first)
     - **Manual setup**: Copy navi's hook config to a file and display instructions for manual merging
     - **Abort**: Exit without making changes

Example conflict prompt:
```
⚠ Existing hooks detected for: Notification, Stop

Your current Notification hooks:
  [{"matcher": "...", "hooks": [...]}]

Options:
  1) Override - Replace with navi hooks (backup saved to settings.json.bak)
  2) Manual - Save navi config to ~/.claude-sessions/hooks/config.json for manual merging
  3) Abort - Exit without changes

Choose [1/2/3]:
```

## Acceptance Criteria

1. Script creates `~/.claude-sessions/` and `~/.claude-sessions/hooks/` directories
2. Script copies `notify.sh` and makes it executable
3. Script reads existing `settings.json` without corrupting it
4. Script detects conflicts with existing hook configurations
5. Script prompts user when conflicts are detected with options to override, manual merge, or abort
6. Script backs up settings.json before overriding
7. Script saves navi config for manual merging when requested
8. Script builds the Go binary successfully
9. Script optionally installs binary to `~/.local/bin/`
10. Script is idempotent (can run multiple times safely)
11. Script provides clear feedback on actions taken

## Dependencies

- PBI-1: Go project must be buildable
- PBI-2: Hook script must exist to copy

## Open Questions

- Should we support other install locations besides `~/.local/bin/`?
- Should we add an uninstall option?

## Related Tasks

See [Tasks](./tasks.md)
