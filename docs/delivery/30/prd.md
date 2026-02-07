# PBI-30: Provider-Supplied File Paths for Content Viewer

## Overview

Add a `file` field to the task provider JSON output so that providers can supply the local file path for each task's detail document. The content viewer should use this provider-supplied path instead of hardcoding path derivation logic in the TUI.

## Problem Statement

Currently, `openTaskDetail` in `model.go` hardcodes the path derivation for task detail files, assuming the `markdown-tasks` provider's directory structure (`docs/delivery/<pbi-num>/<taskID>.md`) and group ID format (`PBI-<num>`). This works for the built-in markdown provider but would fail for any custom provider that uses different group ID formats or file structures. The path logic should be owned by the provider, not the TUI.

This was identified during the PBI-29 PR review (PR #9, Greptile comment on `openTaskDetail`).

## User Stories

1. As a provider author, I want to specify the file path for each task so that the content viewer can open my task detail files regardless of my directory structure.
2. As a user, I want Enter on any task with a local file to open it in the content viewer so that the feature works consistently across all providers.

## Technical Approach

- Add a `file` field (or `detail_path`) to the `Task` struct in `internal/task/types.go`
- Update `markdown-tasks.sh` to output the file path for each task
- Update `openTaskDetail` in `model.go` to use `item.file` from the provider output instead of deriving the path
- Remove the hardcoded `PBI-` prefix stripping and `docs/delivery/` path construction
- The `file` field should be relative to the project directory; the TUI joins it with `taskFocusedProject`

## UX/UI Considerations

- No visible UX change — Enter on a task still opens the detail file
- Error handling remains the same (show error in content viewer if file not found)

## Acceptance Criteria

1. The `Task` struct has a `file` field for local file paths
2. `markdown-tasks.sh` outputs the file path for each task
3. `openTaskDetail` uses the provider-supplied file path instead of hardcoded path logic
4. Tasks without a `file` or `url` show an appropriate message (e.g. "No detail file available")
5. Existing tests pass; new tests cover the file path flow

## Dependencies

- PBI-29 (In-App Content Viewer) — provides the content viewer and `openTaskDetail` method
- PBI-28 (Task View) — provides the task provider system

## Open Questions

- Should the `file` field be absolute or relative to the project directory?
- Should `github-issues.sh` also be updated (it already has URLs, so `file` would be unused)?

## Related Tasks

[View Tasks](./tasks.md)

[View in Backlog](../backlog.md#user-content-30)
