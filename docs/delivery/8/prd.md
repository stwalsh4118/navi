# PBI-8: Session Organization

[View in Backlog](../backlog.md)

## Overview

Add the ability to organize sessions with groups, tags, and templates so users can manage large numbers of sessions across different projects effectively.

## Problem Statement

As users work on multiple projects with multiple Claude sessions, the flat list becomes unwieldy. Users need ways to categorize, filter, and quickly set up sessions with predefined configurations.

## User Stories

- As a user, I want to assign tags to sessions so I can filter by project or category
- As a user, I want to group sessions visually so related sessions appear together
- As a user, I want session templates so I can quickly create sessions with predefined settings

## Technical Approach

### Tags

1. Add `tags` field to session JSON: `"tags": ["frontend", "urgent"]`
2. Press `t` to open tag editor for selected session
3. Tags stored in separate metadata file: `~/.claude-sessions/meta/<session>.json`
4. Display tags as colored badges next to session name

### Groups

1. Groups defined in config: `~/.config/navi/groups.yaml`
2. Groups can be based on:
   - Explicit assignment
   - Directory patterns (e.g., `~/work/*` → "Work")
   - Tag matching
3. Press `g` to cycle through group views or show all
4. Collapsible group headers in the list

### Templates

1. Templates defined in: `~/.config/navi/templates.yaml`
2. Template includes:
   - Name pattern (e.g., `{project}-{date}`)
   - Working directory
   - Initial prompt (optional)
   - Default tags
3. Press `N` (shift+n) to create from template
4. Show template picker if multiple defined

### Config File Example

```yaml
# ~/.config/navi/groups.yaml
groups:
  - name: Work
    patterns:
      - ~/work/*
      - ~/company/*
    color: blue
  - name: Personal
    patterns:
      - ~/projects/*
    color: green
  - name: Urgent
    tags:
      - urgent
      - blocking
    color: red

# ~/.config/navi/templates.yaml
templates:
  - name: New Feature
    directory: ~/work/main-app
    tags: [feature]
    session_name: "feature-{input}"
  - name: Bug Fix
    directory: ~/work/main-app
    tags: [bugfix, urgent]
    session_name: "fix-{input}"
```

## UX/UI Considerations

- Tags should be visually distinct but not cluttering
- Group headers should be collapsible with session count
- Template picker should be searchable if many templates
- Consider keyboard shortcuts for favorite groups (1-9)

## Acceptance Criteria

1. Users can add and remove tags from sessions
2. Tags persist across TUI restarts
3. Tags are displayed as badges in the session list
4. Groups can be defined via config file
5. Sessions are automatically grouped by pattern or tag
6. Group view can be toggled with keyboard
7. Templates can be defined in config
8. Users can create new sessions from templates
9. Filter view respects both tags and groups

## Dependencies

- PBI-7: Session management (for create session integration)
- PBI-13: Search & filter (for tag-based filtering)

## Open Questions

- Should tags support auto-complete from existing tags?
- Should groups support manual ordering?
- Should templates support environment variables?

## Related Tasks

See [Tasks](./tasks.md)
