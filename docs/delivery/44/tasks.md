# Tasks for PBI 44: Background Attach Monitor and tmux Status Bar

This document lists all tasks associated with PBI 44.

**Parent PBI**: [PBI 44: Background Attach Monitor and tmux Status Bar](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--- | :----- | :---------- |
| 44-1 | [Extract session reading into shared package-level function](./44-1.md) | Done | Move readSessions from unexported tui func to an exported session package function so both the TUI and the background monitor can use it |
| 44-2 | [Create internal/monitor package with AttachMonitor](./44-2.md) | Done | Background goroutine that polls session files and triggers audio notifications, with context-based cancellation and state handoff |
| 44-3 | [Integrate AttachMonitor into TUI attach flow](./44-3.md) | Done | Start the background monitor before tea.ExecProcess, stop it on detach, hand off lastSessionStates in both directions |
| 44-4 | [Add navi status CLI command](./44-4.md) | Done | One-shot CLI command that reads session files and prints a status summary with --verbose and --format=tmux flags |
| 44-5 | [E2E CoS verification tests](./44-5.md) | Done | Comprehensive tests covering background monitor lifecycle, state handoff, CLI output formatting, and no-duplicate-notification guarantee |
