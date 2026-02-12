# Tasks for PBI 38: Enhanced GitHub PR Integration

This document lists all tasks associated with PBI 38.

**Parent PBI**: [PBI 38: Enhanced GitHub PR Integration](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--- | :----- | :---------- |
| 38-1 | [Define PR detail data model and types](./38-1.md) | Proposed | Define PRDetail, Reviewer, Check, CheckSummary structs and related constants |
| 38-2 | [Implement PR detail fetching and JSON parsing](./38-2.md) | Proposed | Fetch extended PR data via gh pr view --json, parse response, support local and remote sessions |
| 38-3 | [Enhance git detail view with PR metadata](./38-3.md) | Proposed | Extend the git detail dialog to display full PR info: title, state, draft, mergeable, labels, review status, checks, changed files |
| 38-4 | [Enhance session list PR indicator](./38-4.md) | Proposed | Update inline PR indicator to show aggregate check status, comment count, and draft state |
| 38-5 | [Implement PR comment fetching](./38-5.md) | Proposed | Fetch PR review comments and issue comments via gh api, parse into displayable structures |
| 38-6 | [Add PR comment viewer](./38-6.md) | Proposed | Scrollable comment viewer panel accessible from git detail view via 'c' keybinding |
| 38-7 | [Add auto-refresh for pending checks](./38-7.md) | Proposed | Ticker-based auto-refresh of PR data when checks are pending and git detail view is open |
| 38-8 | [E2E CoS Test](./38-8.md) | Proposed | End-to-end verification of all acceptance criteria |
