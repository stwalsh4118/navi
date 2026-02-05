# PBI-20: Team Dashboard

[View in Backlog](../backlog.md)

## Overview

Create a shared team dashboard where team members can see each other's Claude sessions, enabling collaboration and awareness of what Claude is doing across the team.

## Problem Statement

Teams using Claude lack visibility into each other's work. Questions like "Is anyone working on the auth module?" or "Did Claude finish that deployment?" require asking around. A shared dashboard provides team-wide Claude visibility.

## User Stories

- As a team lead, I want to see all team members' Claude sessions so I can monitor project progress
- As a developer, I want to know if a teammate's Claude is working on related code to avoid conflicts
- As a team, we want shared awareness of Claude activity for coordination

## Technical Approach

### Architecture Options

#### Option A: Shared File System
- All team members mount a shared directory (NFS, SSHFS)
- Each user writes to their subdirectory
- Simple but requires shared storage

#### Option B: Central Server
- Lightweight HTTP server aggregates sessions
- Users run agent that reports to server
- More complex but works without shared storage

#### Option C: Peer-to-Peer
- Users broadcast session status via mDNS/UDP
- No central infrastructure
- Works on local networks only

### Recommended: Central Server (Option B)

```
┌─────────┐    ┌─────────┐    ┌─────────┐
│  User A │    │  User B │    │  User C │
│  navi   │    │  navi   │    │  navi   │
└────┬────┘    └────┬────┘    └────┬────┘
     │              │              │
     └──────────────┼──────────────┘
                    │
              ┌─────▼─────┐
              │   navi    │
              │  server   │
              └─────┬─────┘
                    │
              ┌─────▼─────┐
              │   Team    │
              │ Dashboard │
              └───────────┘
```

### Server Component

1. **navi-server** binary
   - Simple HTTP/WebSocket server
   - Receives session status updates
   - Broadcasts to connected dashboards
   - In-memory storage (no persistence needed)

2. **API Endpoints**
   ```
   POST /api/sessions      - Report session status
   GET  /api/sessions      - Get all team sessions
   WS   /api/ws            - Real-time updates
   ```

3. **Session Payload**
   ```json
   {
     "user": "sean",
     "session": "hyperion",
     "status": "working",
     "message": "Implementing auth...",
     "cwd": "~/projects/hyperion",
     "timestamp": 1738627200
   }
   ```

### Client Integration

1. Configure team server in navi config:
   ```yaml
   team:
     enabled: true
     server: https://navi.team.internal
     user: sean
   ```

2. navi reports local session status to server
3. navi fetches team sessions and displays them

### Dashboard View

Press `T` to open team view:
```
╭─ Team Sessions ────────────────────────────────────────────╮
│                                                            │
│  Sean                                                      │
│  ⚙️  hyperion     working    ~/projects/hyperion           │
│  ✅  api          done       ~/projects/api                │
│                                                            │
│  Alex                                                      │
│  ⏳  frontend     waiting    ~/projects/frontend           │
│      "Should I use React Query or SWR?"                    │
│                                                            │
│  Jordan                                                    │
│  ❓  deploy       permission ~/deploy/production           │
│      "Run: kubectl apply -f deploy.yaml?"                  │
│                                                            │
╰────────────────────────────────────────────────────────────╯
```

### Privacy Controls

- Users can mark sessions as private (not shared)
- CWD paths sanitized (show project name only)
- Messages optionally hidden
- User must explicitly enable team features

## UX/UI Considerations

- Clear visual separation between own and team sessions
- Indicate which sessions are "mine" vs others
- Don't allow attaching to others' sessions directly
- Respect privacy settings
- Handle server disconnection gracefully

## Acceptance Criteria

1. navi-server binary can run as team server
2. navi reports session status to configured server
3. `T` opens team dashboard view
4. Team sessions displayed with user attribution
5. Own sessions distinguishable from team sessions
6. Privacy controls allow hiding specific sessions
7. Server handles multiple concurrent users
8. Real-time updates via WebSocket
9. Graceful degradation when server unavailable
10. Authentication/authorization for team access

## Dependencies

- PBI-3: Session polling (base session data)
- PBI-10: Webhooks (similar outbound notification pattern)

## Open Questions

- Should we support message/chat between team members?
- Should server persist session history?
- Should there be role-based access (who sees what)?

## Related Tasks

See [Tasks](./tasks.md)
