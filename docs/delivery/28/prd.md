# PBI-28: Task View with Pluggable Providers

[View in Backlog](../backlog.md)

## Overview

Add a task view to Navi that displays project tasks alongside sessions, using a pluggable provider system so users can bring tasks from any source (GitHub Issues, Linear, Jira, local markdown, etc.).

## Problem Statement

Users managing Claude Code sessions are typically working against a task list or backlog, but that context lives in a completely separate system. There is no way to see what tasks exist for a project, which are in progress, or which need attention - all without leaving the TUI. Since every team uses a different task management system, a fixed integration approach would only serve a fraction of users.

## User Stories

- As a user, I want to see my project's tasks in Navi so I can understand what work exists without switching tools
- As a user, I want tasks to be automatically discovered from my active session's projects so I don't have to manually configure each one
- As a user, I want to use my existing task system (GitHub Issues, Linear, markdown files, etc.) as the data source so I don't have to change my workflow
- As a user, I want to navigate and filter tasks in the TUI so I can find relevant work quickly
- As a user, I want to open a task in its source system (browser) so I can see full details or update it

## Technical Approach

### Architecture: Script Provider Model

Tasks flow into Navi through **provider scripts** - commands that output a standard JSON format to stdout. Navi runs the configured provider, parses the output, and renders it. This decouples Navi from any specific task management system.

```
Provider Script (any language) → stdout (standard JSON) → Navi TUI (renders)
```

### Configuration: Hybrid Discovery

Two levels of configuration work together:

#### 1. Per-Project Config (`.navi.yaml` in repo root)

Each project defines its own task provider. This file can be committed and shared with the team.

```yaml
tasks:
  provider: "github-issues"          # built-in provider shorthand
  args:
    repo: "owner/repo"
```

```yaml
tasks:
  provider: "./scripts/fetch-tasks.sh"  # custom script
  args:
    project: "my-project"
```

#### 2. Global Config (`~/.navi/config.yaml`)

Global defaults and settings that apply across all projects.

```yaml
tasks:
  default_provider: "github-issues"   # fallback if no .navi.yaml found
  interval: 60s                       # refresh interval
  status_map:                         # normalize statuses for display
    open: "todo"
    in_progress: "active"
    closed: "done"
```

#### Discovery Flow

1. Collect unique working directories from active sessions (already known to Navi)
2. Walk up from each working directory to find `.navi.yaml` (similar to how git finds `.git`)
3. Run each project's configured provider
4. Aggregate results into the task view, grouped by project

Projects without a `.navi.yaml` and no matching global default simply show no tasks - no error, no noise.

### Standard Task JSON Format

This is the contract between any provider and Navi. Providers must output this format to stdout:

```json
{
  "groups": [
    {
      "id": "PBI-13",
      "title": "Search & Filter",
      "status": "in_progress",
      "url": "https://github.com/owner/repo/milestone/3",
      "tasks": [
        {
          "id": "13-1",
          "title": "Implement fuzzy search",
          "status": "done",
          "assignee": "ai-agent",
          "labels": ["feat", "tui"],
          "priority": 1,
          "url": "https://github.com/owner/repo/issues/42",
          "created": "2025-05-19T15:02:00Z",
          "updated": "2025-05-20T10:30:00Z"
        }
      ]
    }
  ]
}
```

**Required fields:** `tasks[].id`, `tasks[].title`, `tasks[].status`

**Optional fields:** Everything else. Groups are optional - a flat task list (no `groups` wrapper, just a top-level `tasks` array) is also valid.

**Status is a free-form string** since every system has different states. Navi normalizes them for display via the `status_map` config.

### Built-in Providers

Ship with a small set of ready-made provider scripts for common systems:

1. **`github-issues`** - Uses `gh` CLI to fetch issues and PRs. Groups by milestone or label.
2. **`markdown-tasks`** - Parses a local directory structure of markdown task files (like the one used in this project).

Additional providers can be contributed by the community or written by users in any language.

### Provider Script Contract

A provider script:
- Receives arguments via environment variables (from `args` in config): `NAVI_TASK_ARG_<KEY>=<value>`
- Outputs the standard task JSON to stdout
- Exits 0 on success, non-zero on failure
- Stderr is captured for error display in the TUI
- Must complete within a configurable timeout (default 30s)

### TUI Integration

#### Task View Toggle

Press `T` to toggle between the sessions view and the task view. The task view uses familiar navigation patterns:

```
+-Tasks---------------------------------------------+
|                                                    |
| > navi (3 sessions)                                |
|   done   13-1 Implement fuzzy search               |
|   done   13-2 Add status filter toggles            |
|   active 13-3 Sort mode cycling                    |
|                                                    |
| > api-server (2 sessions)                          |
|   active #142 Add rate limiting endpoint            |
|   todo   #138 Fix auth token refresh               |
|   todo   #145 Update OpenAPI spec                  |
|                                                    |
| > client-app (1 session)                           |
|   active NAV-23 Refactor state management          |
|                                                    |
+----------------------------------------------------+
```

#### Interactions

- Arrow keys to navigate tasks
- `Enter` to open task URL in browser (if available)
- `Space` to expand/collapse a group
- `/` to search/filter tasks (reuse existing search infrastructure)
- `r` to refresh task data on-demand
- `T` to switch back to sessions view
- Existing session keybindings are inactive while in task view

### Data Flow

```
Active Sessions
  -> unique working directories
  -> walk up to find .navi.yaml
  -> run provider script (cached by interval)
  -> parse standard JSON
  -> render in task view
```

Task data is refreshed on a configurable interval (default 60s) and on-demand via `r`. Results are cached per-project to avoid excessive provider invocations.

## UX/UI Considerations

- Task view is a distinct mode, not overlaid on sessions - keeps both views clean
- Project grouping matches the mental model of "I have sessions for this project, here are its tasks"
- Status normalization means consistent visual treatment regardless of source system
- Provider errors are shown inline (e.g., "github-issues: gh CLI not found") rather than crashing
- Empty state guidance: if no `.navi.yaml` found for any project, show a short help message explaining how to set one up

## Acceptance Criteria

1. `T` toggles between sessions view and task view
2. Per-project `.navi.yaml` config is discovered from active session working directories
3. Global `~/.navi/config.yaml` provides defaults and status mapping
4. Provider scripts are executed and their stdout parsed as standard task JSON
5. Tasks are displayed grouped by project with status indicators
6. `Enter` opens task URL in default browser
7. `Space` expands/collapses groups
8. `/` search works within the task view
9. `r` refreshes task data on-demand
10. Provider errors are displayed gracefully in the TUI
11. At least two built-in providers ship: `github-issues` and `markdown-tasks`
12. Custom provider scripts are supported via path in config
13. Task view is read-only - no write-back to source systems

## Dependencies

- PBI-3: Session polling (for working directory discovery)
- PBI-13: Search & Filter (reuse search/filter infrastructure)

## Design Decisions

- **Task-session linking**: Deferred to a follow-up PBI. This PBI keeps tasks and sessions as separate views.
- **Inline status updates**: No. Navi is read-only for tasks. Users open the task URL to update status in the source system.
- **Pinned projects**: No. Tasks only show for projects with active sessions. Keeps the view focused on what you're actively working on.
- **Collapsible groups**: Yes. Groups (projects, epics, milestones) can be expanded/collapsed with a keypress to manage noise.

## Open Questions

None - all resolved.

## Related Tasks

[View Tasks](./tasks.md)
