# PBI-27: Enhanced Remote Session Management

[View in Backlog](../backlog.md)

## Overview

Extend remote session capabilities to achieve feature parity with local sessions. Currently, remote sessions are view-only and support attaching, but lack git integration, preview, kill, rename, and dismiss functionality. This PBI adds all of these via on-demand SSH command execution reusing the existing SSH connection pool.

## Problem Statement

Remote sessions (introduced in PBI-19) currently only fetch session status JSON files. All interactive features (git info, preview pane, kill, rename, dismiss) are explicitly disabled for remote sessions with early returns in the TUI. Users managing Claude across multiple machines cannot use the full feature set without SSHing in manually, which defeats the purpose of the unified dashboard.

## User Stories

- As a user, I want to see git branch and status for remote sessions so I have project context without leaving navi
- As a user, I want to preview remote session output so I can see what's happening without attaching
- As a user, I want to kill remote sessions from navi so I can clean up without separate SSH terminals
- As a user, I want to rename remote sessions from navi so I can organize them like local sessions
- As a user, I want to dismiss remote session notifications so I can clear them from my view

## Technical Approach

### On-Demand SSH Command Execution

Rather than polling for git/preview data on every tick (expensive over SSH), fetch data on-demand when the user explicitly requests it:

- **Git info** (`G` key): SSH into remote, run git commands in the session's `cwd`, return structured git data
- **Preview** (`p` key): SSH into remote, run `tmux capture-pane` for the session, return output lines

### One-Off SSH Actions

Actions that modify state are executed as single SSH commands when triggered:

- **Kill** (`x` key): `ssh remote "tmux kill-session -t <session>"`
- **Rename** (`R` key): `ssh remote "tmux rename-session -t <old> <new>"`
- **Dismiss** (`d` key): `ssh remote "rm <sessions_dir>/<file>.json"`

### Reuse Existing Infrastructure

All SSH operations go through the existing `SSHPool` from PBI-19, which provides:
- Connection pooling and reuse
- Keepalive and auto-reconnect
- Jump host support
- 30-second command timeout

### Remote Git Info Script

Bundle git commands into a single SSH call to minimize round trips:

```bash
cd <cwd> && echo "BRANCH:$(git rev-parse --abbrev-ref HEAD 2>/dev/null)" && \
echo "DIRTY:$(git status --porcelain 2>/dev/null | head -1)" && \
echo "REMOTE:$(git remote get-url origin 2>/dev/null)" && \
echo "AHEAD:$(git rev-list --count @{u}..HEAD 2>/dev/null)" && \
echo "BEHIND:$(git rev-list --count HEAD..@{u} 2>/dev/null)"
```

Parse the structured output back in Go and populate the same `GitInfo` struct used for local sessions. Once the remote URL and branch are known, use the local `gh` CLI to look up PR info (same as the local git flow) - no need for `gh` on the remote machine.

### Remote Preview

```bash
tmux capture-pane -t <session> -p -S -<lines>
```

Returns the last N lines of the tmux pane, same as local preview but executed via SSH.

### Caching

On-demand results should be cached with a reasonable TTL (e.g., 10-30 seconds) to avoid re-fetching on rapid key presses. Cache is keyed by (remote, session, data type).

## UX/UI Considerations

- Remote git/preview should feel the same as local - same keybindings, same display
- Brief loading indicator for on-demand fetches (SSH has latency)
- Error handling: if SSH command fails, show inline error rather than crashing
- Kill/rename/dismiss should show confirmation feedback same as local
- No visual distinction in the data itself - only the `[remote-name]` label differentiates

## Acceptance Criteria

1. Pressing `G` on a remote session fetches and displays git info via SSH
2. Pressing `p` on a remote session fetches and displays tmux preview via SSH
3. Pressing `x` on a remote session kills the remote tmux session via SSH
4. Pressing `R` on a remote session renames the remote tmux session via SSH
5. Pressing `d` on a remote session dismisses it by removing the status file via SSH
6. All SSH operations reuse the existing SSHPool connection pool
7. On-demand fetches (git, preview) are cached with a TTL to avoid redundant SSH calls
8. SSH errors are handled gracefully with user-visible feedback (not crashes)
9. Remote actions provide the same confirmation/feedback as their local equivalents

## Dependencies

- PBI-19: Remote Sessions (foundation - SSH pool, remote config, session polling)

## Resolved Questions

- **Cache TTL**: Hardcoded default to start. Can be made configurable later if needed.
- **GitHub PR info**: Fetch git metadata (remote URL, branch) from the remote via SSH, then use the local `gh` CLI to look up PR info - same as the local flow. No need for `gh` installed on remote machines.
- **Remote token metrics**: Out of scope. Remote token metric fetching will be a separate PBI if needed after PBI-26 lands.

## Related Tasks

[View Tasks](./tasks.md)
