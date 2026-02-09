# Tasks for PBI 33: Agent Team Awareness and Hook Robustness

This document lists all tasks associated with PBI 33.

**Parent PBI**: [PBI 33: Agent Team Awareness and Hook Robustness](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--- | :----- | :---------- |
| 33-1 | [Fix stale permission status with PostToolUse hook](./33-1.md) | Proposed | Add PostToolUse hook to clear permission/question icon after tool approval |
| 33-2 | [Extend session data model with agent team types](./33-2.md) | Proposed | Add AgentInfo, TeamInfo types to session.go and update sorting to consider teammate statuses |
| 33-3 | [Refactor hook scripts for teammate-aware status writing](./33-3.md) | Proposed | Update notify.sh to parse stdin JSON, detect teammates, write team data, and register new hook events |
| 33-4 | [Render inline agent team status in TUI](./33-4.md) | Proposed | Display team agents with status icons below session rows in the TUI |
| 33-5 | [E2E CoS Test](./33-5.md) | Proposed | End-to-end verification of all acceptance criteria |
