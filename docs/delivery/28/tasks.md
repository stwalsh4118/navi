# Tasks for PBI 28: Task View with Pluggable Providers

This document lists all tasks associated with PBI 28.

**Parent PBI**: [PBI 28: Task View with Pluggable Providers](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--- | :----- | :---------- |
| 28-1 | [Define task data types, provider contract, and JSON parsing](./28-1.md) | Done | Define Go structs for tasks, groups, provider results, and config; implement standard JSON format parsing |
| 28-2 | [Implement config discovery and loading](./28-2.md) | Done | Walk up from session CWDs to find .navi.yaml, load global ~/.navi/config.yaml, merge defaults |
| 28-3 | [Implement provider execution engine](./28-3.md) | Done | Execute provider scripts with timeout, capture stdout/stderr, parse output, cache results |
| 28-4 | [Integrate task view into TUI model and input handling](./28-4.md) | Done | Add task state to Model, wire up T toggle, task-mode keybindings, async commands, and config discovery |
| 28-5 | [Implement task view rendering and styling](./28-5.md) | Done | Render grouped task list with collapsible groups, status indicators, search/filter, empty state, and error display |
| 28-6 | [Create built-in github-issues provider](./28-6.md) | Done | Shell script using gh CLI to fetch issues grouped by milestone in standard JSON format |
| 28-7 | [Create built-in markdown-tasks provider](./28-7.md) | Done | Shell script parsing local docs/delivery/ markdown task structure into standard JSON format |
| 28-8 | [E2E CoS Test](./28-8.md) | Done | End-to-end verification of all acceptance criteria with mock providers |
