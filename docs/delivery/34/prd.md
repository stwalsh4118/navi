# PBI-34: Enhanced Session Creation

[View in Backlog](../backlog.md#user-content-34)

## Overview

Upgrade the session creation dialog with more flexible options: the ability to create a tmux session without auto-launching Claude, directory path tab-completion, cloning an existing session's settings, and an option to immediately attach to the newly created session.

## Problem Statement

The current session creation flow (press `n`) always creates a tmux session **and** immediately launches `claude` inside it via `tmux send-keys`. This is limiting in several ways:

1. **No shell-only sessions**: Sometimes you want a tmux workspace ready — maybe to set up the environment, install dependencies, or run other commands before starting Claude. There's no way to create a "blank" tmux session from Navi.

2. **Manual directory entry is tedious**: The directory field requires typing a full path with no assistance. For deeply nested project directories, this is slow and error-prone. Users must remember exact paths or tab out to a terminal to check.

3. **No way to clone sessions**: When you want a second session in the same directory with the same flags (e.g., another agent working on the same project), you have to manually re-enter all the same values. There's no shortcut to duplicate an existing session's configuration.

4. **No immediate attach**: After creating a session, you return to the session list and must manually select and press Enter to attach. For the common case where you want to jump right into the new session, this adds an unnecessary extra step.

## User Stories

- As a user, I want to create a tmux session without auto-launching Claude so that I can prepare the workspace first.
- As a user, I want tab-completion on the directory field so that I can quickly navigate to the right path without typing it fully.
- As a user, I want to clone an existing session's settings into a new session so that I don't have to re-enter the same directory and flags.
- As a user, I want the option to immediately attach to a newly created session so that I can start working without an extra selection step.

## Technical Approach

### 1. Tmux-Only Mode

- Add a new boolean toggle to the session creation dialog: `[ ] Shell only (no Claude)`.
- When enabled, `createSessionCmd` skips the `tmux send-keys` step that launches `claude`.
- The initial status file should use a new or existing status value (e.g., `idle`) to differentiate shell-only sessions from active Claude sessions.
- The session appears in Navi's list with an appropriate status icon.
- The user can later manually start Claude from within the tmux session, and hooks will take over status tracking normally.

### 2. Directory Tab-Completion

- When the directory input field is focused and the user presses Tab (while not cycling focus), trigger filesystem path completion.
- Expand `~` to the home directory before completion.
- If the current value is a valid directory prefix, list matching entries and complete to the longest common prefix. If there's exactly one match, complete fully.
- If multiple matches exist with no further common prefix, show a brief completion list below the input (or cycle through matches on repeated Tab presses).
- Tab should only trigger completion when focus is on the directory field; it continues to cycle focus when on other fields.
- Need to rework the focus cycling mechanism — currently Tab always cycles. Options:
  - Use a different key for focus cycling (e.g., Shift+Tab or Ctrl+Tab) when on the directory field.
  - Use Tab for completion when the directory field has partial input, and for focus cycling when it's empty or the path is already complete.
  - Simplest approach: Tab completes when directory field is focused; use Shift+Tab (or a dedicated key) to cycle between fields. This is consistent with how most terminal forms work.

### 3. Clone Session

- Add a new keybinding (e.g., `c` or `C`) on the session list to open the "New Session" dialog pre-populated with the selected session's directory and flags.
- The session name field should be empty (or auto-suffixed like `session-name-2`) so the user provides a unique name.
- `skipPermissions` and any other flags should carry over from the source session.
- The source session's CWD is read from its status file or tmux.
- This reuses the existing dialog — it's just pre-populated rather than blank.

### 4. Post-Create Attach

- Add a checkbox to the session creation dialog: `[x] Attach after create`.
- Default: on (matches the common workflow of jumping right in).
- When enabled, after successful session creation, instead of just closing the dialog and refreshing, immediately trigger `attachSessionCmd` for the new session.
- The TUI suspends (as it does for normal attach) and the user lands directly in the new tmux session.

### Dialog Layout (Updated)

```
╭─ New Session ──────────────────────────────────╮
│                                                │
│ Name:      [                                ]  │
│ Directory: [~/workspace/nav               ▸]  │
│ [ ] Skip permissions                           │
│ [ ] Shell only (no Claude)                     │
│ [x] Attach after create                        │
│                                                │
│ Tab: complete dir  Shift+Tab: switch field     │
│ Space: toggle  Enter: create  Esc: cancel      │
╰────────────────────────────────────────────────╯
```

## UX/UI Considerations

- The new toggles (shell only, attach after create) should appear as checkboxes below the existing skip-permissions toggle, maintaining visual consistency.
- Directory completion should feel snappy — filesystem reads are fast for listing a single directory's contents.
- When showing completion candidates, limit the display to avoid overwhelming the dialog. Show at most 5-10 candidates, with an indicator if more exist.
- Clone session should be discoverable — include `c clone` in the help bar when a session is selected.
- The focus cycling key change (Tab → Shift+Tab for field switching) should be clearly communicated in the dialog footer hints.

## Acceptance Criteria

1. A "Shell only" toggle exists in the session creation dialog; when enabled, the tmux session is created but `claude` is not launched.
2. Shell-only sessions appear in the session list with an appropriate status (not `working`).
3. Tab-completion works on the directory field: pressing Tab completes the path based on filesystem contents.
4. Tab-completion handles `~` expansion, partial paths, and multiple matches gracefully.
5. Pressing `c` (or chosen key) on a session in the list opens the new session dialog pre-populated with that session's directory and flags.
6. The cloned session's name field is empty or auto-suffixed, not duplicated.
7. An "Attach after create" toggle exists; when enabled, the TUI attaches to the new session immediately after creation.
8. All existing session creation functionality continues to work (name validation, directory validation, skip-permissions).
9. The help bar / footer hints accurately reflect the new keybindings.
10. All existing tests continue to pass.
11. New tests cover shell-only creation, clone pre-population, attach-after-create flow, and directory completion logic.

## Dependencies

- PBI-7 (Session Management Actions) - Done. Provides the existing session creation dialog and command infrastructure.
- PBI-32 (Scrollable Panels) - Done. Dialog rendering may need to account for the taller dialog with more options.

## Open Questions

All resolved:

1. ~~What key should trigger clone session from the session list?~~ **Resolved: `c` — intuitive mnemonic for "clone". Verify no conflicts with existing bindings during implementation.**
2. ~~Should the directory completion show a dropdown-style list of candidates, or cycle through them one at a time on repeated Tab presses?~~ **Resolved: Bash-style — complete to the longest common prefix first, then show a candidate list only when ambiguous.**
3. ~~For shell-only sessions, should the status be `idle` (existing status) or a new dedicated status like `shell`?~~ **Resolved: Reuse `idle` — simpler, no new status value needed.**
4. ~~Should "Attach after create" default to on or off?~~ **Resolved: Default on — matches the common workflow of wanting to jump right into the new session.**

## Related Tasks

See [tasks.md](./tasks.md) for the task breakdown.
