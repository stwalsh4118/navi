# PBI-19: Remote Sessions

[View in Backlog](../backlog.md)

## Overview

Aggregate Claude sessions from remote machines via SSH, allowing users to monitor and manage Claude instances running on multiple servers from a single navi dashboard.

## Problem Statement

Developers often run Claude on multiple machines - local workstation, cloud servers, CI runners. Currently, each machine's navi instance is isolated. Users need a unified view across all their machines.

## User Stories

- As a user, I want to see Claude sessions from my remote servers so I can monitor all my work
- As a user, I want to attach to remote sessions through navi so I don't need separate SSH terminals
- As a user, I want session sync across machines so bookmarks and tags are consistent

## Technical Approach

### Remote Configuration

```yaml
# ~/.config/navi/remotes.yaml
remotes:
  - name: dev-server
    host: dev.example.com
    user: sean
    key: ~/.ssh/id_rsa
    sessions_dir: ~/.claude-sessions  # default

  - name: staging
    host: staging.example.com
    user: deploy
    key: ~/.ssh/deploy_key
    jump_host: bastion.example.com
```

### Remote Session Polling

1. **SSH Connection Pool**
   - Maintain persistent SSH connections to remotes
   - Use multiplexing (ControlMaster) for efficiency
   - Handle reconnection on failure

2. **Remote Polling**
   - Periodically run on each remote:
     ```bash
     cat ~/.claude-sessions/*.json
     ```
   - Parse and merge with local sessions
   - Tag sessions with remote name

3. **Session Display**
   ```
     ⚙️  hyperion                                   2m ago
         ~/projects/hyperion
         "Implementing feature..."

     ⚙️  api [dev-server]                          5m ago
         ~/work/api
         "Running tests..."

     ✅  deploy [staging]                          10m ago
         ~/app
         "Deployment complete"
   ```

### Remote Attach

When attaching to a remote session:
```bash
ssh -t user@host "tmux attach-session -t <session>"
```

This opens an SSH connection with tmux attached directly.

### Implementation

1. **SSH Manager**
   - Handles connection lifecycle
   - Connection pooling and reuse
   - Error handling and reconnection

2. **Remote Poller**
   - Parallel polling of all remotes
   - Merges remote sessions with local
   - Handles network latency/timeouts

3. **Extended Model**
   ```go
   type SessionInfo struct {
       // ... existing fields
       Remote string  // empty for local, host name for remote
   }

   type Model struct {
       // ... existing fields
       remotes []RemoteConfig
       sshPool *SSHPool
   }
   ```

### Security Considerations

- SSH key authentication only (no passwords stored)
- Support for jump hosts/bastions
- Connection timeout and retry limits
- No sensitive data cached locally from remotes

## UX/UI Considerations

- Remote sessions clearly labeled
- Connection status indicator per remote
- Handle offline remotes gracefully
- Show latency/freshness of remote data
- Filter by remote or show all

## Acceptance Criteria

1. Remote machines configurable via YAML file
2. Sessions from remotes appear in main list
3. Remote sessions labeled with host name
4. Attach to remote session opens SSH with tmux
5. Connection pooling reuses SSH connections
6. Offline remotes handled gracefully (shown as unavailable)
7. Per-remote status indicator (connected/disconnected)
8. Jump host support for bastion access
9. SSH key authentication (no password prompts)
10. Filter sessions by local/remote

## Dependencies

- PBI-3: Session polling (for local session handling)
- PBI-5: Attach mechanism (extend for SSH attach)

## Open Questions

- Should remote status files be cached locally?
- Should we support remote session creation?
- Should there be a "remotes health" view?

## Related Tasks

See [Tasks](./tasks.md)
