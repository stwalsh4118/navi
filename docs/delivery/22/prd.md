# PBI-22: Permission Rules

[View in Backlog](../backlog.md)

## Overview

Add configurable permission rules that auto-approve certain Claude tool calls based on tool type, session, or directory patterns, reducing interruptions for low-risk operations.

## Problem Statement

Claude's permission system is thorough but can be interruptive for frequently-used, low-risk operations (e.g., Read, Glob in safe directories). Users who trust Claude in certain contexts want to pre-approve specific operations while maintaining control over higher-risk ones.

## User Stories

- As a user, I want to auto-approve Read operations so file reading doesn't require my input
- As a user, I want to auto-approve operations in my personal scratch directory
- As a user, I want dangerous operations (rm, git push) to always require approval
- As a user, I want per-session permission profiles for different trust levels

## Technical Approach

### Permission Rules Configuration

```yaml
# ~/.config/navi/permissions.yaml
rules:
  # Always approve these tools
  - action: approve
    tools: [Read, Glob, Grep]

  # Approve Bash for specific commands in specific directories
  - action: approve
    tools: [Bash]
    directories:
      - ~/scratch/*
      - ~/tmp/*
    commands:
      - "ls *"
      - "cat *"
      - "head *"

  # Approve Edit only in certain projects
  - action: approve
    tools: [Edit, Write]
    directories:
      - ~/work/sandbox/*

  # Always require approval for dangerous operations
  - action: require
    tools: [Bash]
    commands:
      - "rm *"
      - "git push *"
      - "sudo *"
    priority: 100  # High priority, checked first

  # Session-specific rules
  - action: approve
    tools: [Bash]
    sessions: [scratch, playground]
    # All Bash approved in these sessions

  # Deny and notify for suspicious patterns
  - action: deny
    tools: [Bash]
    commands:
      - "curl * | sh"
      - "wget * | sh"
    notify: true
```

### Rule Evaluation

1. Rules evaluated in priority order (higher first)
2. First matching rule determines action
3. Default action: require approval (existing behavior)

### Actions

| Action | Behavior |
|--------|----------|
| `approve` | Auto-approve, no user prompt |
| `require` | Always require user approval |
| `deny` | Block the operation entirely |

### Integration with Claude Code

This requires integration with Claude Code's permission system:

1. **Option A: Hook-based**
   - On `PermissionRequest` hook, evaluate rules
   - If `approve`, send approval keystroke to tmux
   - If `deny`, send denial keystroke

2. **Option B: Configuration-based**
   - Generate Claude Code's permission config from navi rules
   - Sync on navi startup

### Implementation

1. **Rule Engine**
   ```go
   type PermissionRule struct {
       Action      string   // approve, require, deny
       Tools       []string // tool names
       Directories []string // glob patterns
       Commands    []string // command patterns
       Sessions    []string // session names
       Priority    int
   }

   func (r *RuleEngine) Evaluate(request PermissionRequest) Action
   ```

2. **Hook Handler**
   - Intercept permission requests
   - Match against rules
   - Execute appropriate action

3. **Audit Log**
   - Log all auto-approved operations
   - Log denied operations
   - Provide audit trail

### Safety Features

- Deny rules always take precedence over approve
- Audit log for all auto-approved operations
- Command pattern matching is conservative (explicit matches only)
- Easy to disable all auto-approve rules
- Show auto-approve count in session display

## UX/UI Considerations

- Visual indicator when auto-approve is active
- Show auto-approve count per session
- Easy toggle to temporarily disable auto-approve
- Clear warning when configuring approve rules
- Audit log accessible from TUI

## Acceptance Criteria

1. Permission rules configurable in YAML
2. Rules support tool, directory, command, and session matching
3. `approve` action auto-approves matching operations
4. `require` action forces approval regardless of other rules
5. `deny` action blocks operation entirely
6. Rules evaluated in priority order
7. Audit log records all auto-approved operations
8. Visual indicator shows when auto-approve is active
9. Global toggle to disable all auto-approve rules
10. Dangerous operations (rm, sudo, etc.) require explicit opt-in

## Dependencies

- PBI-2: Hook system (for permission request handling)
- PBI-3: Session polling (for session context)

## Open Questions

- Should rules support time-based conditions (e.g., only during work hours)?
- Should there be per-file rules (not just per-directory)?
- Should denied operations notify the user automatically?

## Related Tasks

See [Tasks](./tasks.md)
