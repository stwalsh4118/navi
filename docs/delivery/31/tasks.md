# Tasks for PBI 31: Vim-Style Exact-Match Search

This document lists all tasks associated with PBI 31.

**Parent PBI**: [PBI 31: Vim-Style Exact-Match Search](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--- | :----- | :---------- |
| 31-1 | [Replace fuzzy matching with exact substring matching](./31-1.md) | Proposed | Replace fuzzyMatch/fuzzyFilter with exact case-insensitive substring matching that returns match indices instead of filtering |
| 31-2 | [Implement search state and n/N match cycling in session list](./31-2.md) | Proposed | Add match index tracking, current match pointer, and n/N key bindings for cycling through session matches |
| 31-3 | [Render search highlights and match counter in session list view](./31-3.md) | Proposed | Highlight matching sessions, distinguish current match, and display match counter in the UI |
| 31-4 | [Apply vim-style search to task panel](./31-4.md) | Proposed | Port the exact-match search with n/N cycling and highlights to the task panel |
| 31-5 | [Remove legacy fuzzy search code and update tests](./31-5.md) | Proposed | Remove all fuzzy matching code, update existing tests to reflect exact-match behaviour |
| 31-6 | [E2E CoS Test](./31-6.md) | Proposed | End-to-end verification of all acceptance criteria |
