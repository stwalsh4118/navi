# PBI-39: Remote Session Lifecycle Parity

[View in Backlog](../backlog.md)

## Overview

Bring remote session lifecycle behavior to full parity with local sessions. While PBI-27 added interactive actions (kill, rename, dismiss, git, preview) for remote sessions, several lifecycle gaps remain: stale session cleanup doesn't cover remotes, remote sessions cannot be created from the TUI, and status updates rely on polling rather than the hook-based real-time updates that local sessions enjoy.

## Problem Statement

Local sessions have a complete lifecycle managed by navi: creation via the TUI, real-time status updates via hooks, and automatic cleanup of stale sessions when their tmux session dies. Remote sessions are missing all three of these lifecycle stages:

1. **No stale cleanup**: `cleanStaleSessions()` only checks local tmux sessions. When a remote tmux session is killed (externally or via navi), its `.json` status file persists on the remote machine indefinitely. The next poll cycle reads it back, so the dead session reappears in the TUI. The only thing preventing duplicates is the merge logic, but the stale file is never removed.

2. **No remote session creation**: The `n` key opens a new-session dialog that only creates local tmux sessions. There is no way to create a session on a remote machine from the TUI, forcing users to SSH in manually.

3. **Polling-only status updates**: Local sessions get real-time status updates via Claude Code hooks (`notify.sh`, `tool-tracker.sh`) that write directly to `~/.claude-sessions/`. Remote session status is only discovered by periodically SSHing in and `cat`-ing the JSON files, introducing latency proportional to the poll interval plus SSH round-trip time.

## User Stories

- As a user, I want dead remote sessions to be automatically cleaned up so that my dashboard doesn't show stale sessions that no longer exist
- As a user, I want to create new Claude sessions on remote machines from the TUI so that I don't have to SSH in manually to start work
- As a user, I want remote session status updates to arrive promptly so that the dashboard reflects current state without significant delay

## Technical Approach

### Stale Remote Session Cleanup

During remote polling, after fetching the list of `.json` files, also run `tmux list-sessions -F "#{session_name}"` on the remote to get live tmux sessions. Compare the two lists and remove any `.json` files on the remote that don't have a corresponding live tmux session. This mirrors the local `cleanStaleSessions()` logic.

The cleanup command can be bundled into the existing remote poll SSH call to avoid an extra round-trip:

```bash
# Combined poll + cleanup in a single SSH call
LIVE=$(tmux list-sessions -F '#{session_name}' 2>/dev/null)
for f in "$HOME/.claude-sessions"/*.json; do
  [ -f "$f" ] || continue
  name=$(basename "$f" .json)
  echo "$LIVE" | grep -qx "$name" || rm -f "$f"
done
cat "$HOME/.claude-sessions"/*.json 2>/dev/null || true
```

### Remote Session Creation

Add a dialog flow for creating sessions on remote machines:

1. When `n` is pressed, detect if remotes are configured
2. Offer a choice between local and each configured remote
3. For remote creation, SSH in and execute `tmux new-session -d -s <name> -c <dir>` on the remote
4. Optionally send keys to start Claude in the new session (same as local)
5. The remote's hooks will create the status file, which navi will pick up on the next poll

### Improved Remote Status Freshness

Reduce the effective latency of remote status updates. Options include:

- **Adaptive polling**: Poll more frequently when sessions are in attention-requiring states (e.g., `permission_requested`)
- **On-demand refresh**: Allow manual refresh of remote sessions (e.g., a keybinding to force-poll a specific remote)
- **Event-driven polling**: After performing an action on a remote session (kill, rename, dismiss), immediately re-poll that remote rather than waiting for the next tick

## UX/UI Considerations

- Stale cleanup should be invisible to the user — dead sessions simply stop appearing
- Remote session creation dialog should clearly indicate which machine the session will be created on
- The creation flow should handle SSH errors gracefully (remote unreachable, tmux not installed, etc.)
- Status freshness improvements should be transparent — the dashboard should just feel more responsive

## Acceptance Criteria

1. When a remote tmux session is killed, its status file is automatically cleaned up on the remote machine within one poll cycle, and it disappears from the TUI
2. Pressing `n` when remotes are configured offers a choice of local or remote machine for session creation
3. Creating a remote session via the TUI starts a tmux session on the chosen remote machine
4. After performing a remote action (kill, rename, dismiss), the remote is immediately re-polled to reflect the change
5. Stale cleanup does not remove status files for sessions that are still alive on the remote
6. All remote operations reuse the existing SSHPool connection pool
7. SSH errors during cleanup or creation are handled gracefully with user-visible feedback

## Dependencies

- PBI-19: Remote Sessions (foundation — SSH pool, remote config, session polling)
- PBI-27: Enhanced Remote Session Management (remote actions — kill, rename, dismiss, git, preview)

## Open Questions

- Should the remote creation dialog allow specifying the working directory on the remote machine, or default to the home directory? (Likely needs tab-completion or recent-directories support on the remote, which adds complexity)
- Should there be a configurable option to disable stale cleanup for specific remotes? (Edge case: shared machines where multiple users have sessions)
- Is adaptive polling worth the added complexity, or is immediate re-poll after actions sufficient for perceived responsiveness?

## Related Tasks

[View Tasks](./tasks.md)
