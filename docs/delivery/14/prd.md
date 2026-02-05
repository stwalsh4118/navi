# PBI-14: Multi-Session Operations

[View in Backlog](../backlog.md)

## Overview

Add the ability to select multiple sessions and perform bulk actions like killing, tagging, approving permissions, or broadcasting input across all selected sessions.

## Problem Statement

When managing many Claude sessions, performing the same action on multiple sessions one-by-one is tedious. Users need to select multiple sessions and act on them as a group.

## User Stories

- As a user, I want to select multiple sessions so I can act on them together
- As a user, I want to kill multiple idle sessions at once to clean up
- As a user, I want to approve permissions across multiple sessions quickly
- As a user, I want to add the same tag to multiple sessions for organization

## Technical Approach

### Selection Mechanism

1. Press `Space` to toggle selection on current session
2. Selected sessions marked with `✓` or highlight
3. Press `a` to select all visible sessions
4. Press `A` to deselect all
5. Selection persists across cursor movement

### Bulk Actions

When sessions are selected, action keys apply to all:

| Key | Action | Confirmation |
|-----|--------|--------------|
| `x` | Kill all selected | Yes |
| `d` | Dismiss notifications on all | No |
| `t` | Add tag to all selected | Prompt for tag |
| `Space` (in permission) | Approve all pending permissions | Yes |

### Implementation

1. Add selection state to Model:
   ```go
   type Model struct {
       // ... existing fields
       selected map[string]bool  // session name -> selected
   }
   ```

2. Modify action handlers to check for selections:
   ```go
   func (m *Model) handleKill() tea.Cmd {
       targets := m.getSelectedOrCurrent()
       if len(targets) > 1 {
           return confirmBulkKill(targets)
       }
       return confirmKill(targets[0])
   }
   ```

3. Add bulk action confirmation dialog:
   ```
   ╭─────────────────────────────────────────────────╮
   │  Kill 5 selected sessions?                     │
   │                                                 │
   │  • hyperion                                     │
   │  • api                                          │
   │  • dotfiles                                     │
   │  • scratch                                      │
   │  • feature-auth                                 │
   │                                                 │
   │  [Y]es  [N]o                                   │
   ╰─────────────────────────────────────────────────╯
   ```

### Broadcast Input

For advanced use case, allow sending input to multiple sessions:
1. Press `B` to enter broadcast mode
2. Type input
3. Press `Enter` to send to all selected sessions
4. Uses `tmux send-keys -t <session> "<input>" Enter`

**Warning**: This is powerful and potentially dangerous. Show clear warnings.

## UX/UI Considerations

- Selection state must be visually clear
- Show selection count in footer: "3 selected"
- Confirmations required for destructive bulk actions
- Broadcast mode needs prominent warning
- Actions should show progress for long operations

## Acceptance Criteria

1. `Space` toggles session selection
2. Multiple sessions can be selected simultaneously
3. `a` selects all visible, `A` deselects all
4. Selection count shown in footer
5. `x` kills all selected sessions (with confirmation)
6. `d` dismisses notifications on all selected
7. `t` adds tag to all selected
8. Bulk permission approval available for permission-status sessions
9. Broadcast input sends to all selected (with warning)
10. Selection persists across list updates

## Dependencies

- PBI-7: Session management (for kill action)
- PBI-8: Session organization (for bulk tagging)
- PBI-22: Permission rules (for bulk approval)

## Open Questions

- Should selection persist across filters?
- Should we add range selection (Shift+Up/Down)?
- Should broadcast support command history?

## Related Tasks

See [Tasks](./tasks.md)
