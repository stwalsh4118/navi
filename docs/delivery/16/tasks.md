# Tasks for PBI 16: Git Integration

This document lists all tasks associated with PBI 16.

**Parent PBI**: [PBI 16: Git Integration](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--- | :----- | :---------- |
| 16-1 | [Define GitInfo types and constants](./16-1.md) | Done | Define data structures for git information and display constants |
| 16-2 | [Implement git status collection functions](./16-2.md) | Done | Create functions to gather git branch, dirty status, ahead/behind from working directory |
| 16-3 | [Extend SessionInfo with GitInfo](./16-3.md) | Done | Add GitInfo field to SessionInfo and update JSON handling |
| 16-4 | [Implement git info caching and polling](./16-4.md) | Done | Add separate git polling with caching to avoid slowing main status updates |
| 16-5 | [Display git info in session rows](./16-5.md) | Done | Render branch name, dirty indicator, and ahead/behind counts in session list |
| 16-6 | [Implement GitHub remote detection](./16-6.md) | Done | Parse remote URL and detect GitHub repositories |
| 16-7 | [Implement PR detection from branch names](./16-7.md) | Done | Extract issue/PR numbers from branch naming patterns |
| 16-8 | [Add G keybinding for git detail view](./16-8.md) | Done | Implement modal git detail view with full branch info |
| 16-9 | [Implement diff preview in git detail view](./16-9.md) | Done | Show file changes and diff stats in git detail view |
| 16-10 | [Add PR link action in git detail view](./16-10.md) | Done | Allow opening GitHub PR/issue links in browser |
| 16-11 | [E2E CoS Test](./16-11.md) | Done | End-to-end verification of all acceptance criteria |
