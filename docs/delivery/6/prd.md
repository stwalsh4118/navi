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
   - Reads `~/.config/claude/settings.json`
   - Merges hook configuration (preserving existing hooks)
   - Writes updated settings.json back
   - Builds the Go binary with `go build`
   - Optionally copies binary to `~/.local/bin/`

### Installation Steps (from PRD)

1. Copies `notify.sh` to `~/.claude-sessions/hooks/` and makes it executable
2. Creates `~/.claude-sessions/` if it doesn't exist
3. Reads `~/.config/claude/settings.json`, merges the hook config (preserving existing hooks), writes it back
4. Builds the Go binary and optionally copies it to `~/.local/bin/`

## UX/UI Considerations

- Script should be idempotent (safe to run multiple times)
- Clear output showing what was done
- Handle missing settings.json gracefully (create new file)

## Acceptance Criteria

1. Script creates `~/.claude-sessions/` and `~/.claude-sessions/hooks/` directories
2. Script copies `notify.sh` and makes it executable
3. Script reads existing `settings.json` without corrupting it
4. Script merges hook configuration, preserving existing hooks
5. Script builds the Go binary successfully
6. Script optionally installs binary to `~/.local/bin/`
7. Script is idempotent (can run multiple times safely)
8. Script provides clear feedback on actions taken

## Dependencies

- PBI-1: Go project must be buildable
- PBI-2: Hook script must exist to copy

## Open Questions

- Should we support other install locations besides `~/.local/bin/`?
- Should we add an uninstall option?

## Related Tasks

See [Tasks](./tasks.md)
