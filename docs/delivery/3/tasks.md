# Tasks for PBI 3: Session Polling & State Management

This document lists all tasks associated with PBI 3.

**Parent PBI**: [PBI 3: Session Polling & State Management](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--- | :----- | :---------- |
| 3-1 | [Implement JSON file reading and parsing](./3-1.md) | Done | Read and parse session JSON files from status directory |
| 3-2 | [Implement tmux session listing](./3-2.md) | Done | Query live tmux sessions for cross-referencing |
| 3-3 | [Implement stale session cleanup](./3-3.md) | Done | Delete status files for sessions that no longer exist |
| 3-4 | [Implement session sorting](./3-4.md) | Done | Sort sessions by priority (waiting/permission first) then timestamp |
| 3-5 | [Implement tick-based polling](./3-5.md) | Done | Set up 500ms polling interval with Bubble Tea commands |
