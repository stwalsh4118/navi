# Tasks for PBI 11: Session Preview Pane

This document lists all tasks associated with PBI 11.

**Parent PBI**: [PBI 11: Session Preview Pane](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--- | :----- | :---------- |
| 11-1 | [Add tmux capture-pane function](./11-1.md) | Done | Implement core function to capture output from a tmux session pane |
| 11-2 | [Define preview pane types and constants](./11-2.md) | Done | Add types for preview mode, layout options, and configuration constants |
| 11-3 | [Add preview state to Model](./11-3.md) | Done | Extend the Model struct with preview visibility, content, and settings |
| 11-4 | [Implement preview toggle keybindings](./11-4.md) | Done | Handle `p` or `Tab` key to toggle preview pane visibility |
| 11-5 | [Create preview pane renderer](./11-5.md) | Done | Implement the visual rendering of the preview pane component |
| 11-6 | [Add preview content polling](./11-6.md) | Done | Implement periodic refresh with debouncing for cursor navigation |
| 11-7 | [Handle ANSI escape codes](./11-7.md) | Done | Strip or render ANSI escape sequences in captured output |
| 11-8 | [Add preview pane resizing](./11-8.md) | Done | Allow keyboard-based resizing of the preview panel |
| 11-9 | [Add terminal size adaptation](./11-9.md) | Done | Disable preview when terminal is too narrow, adapt layout |
| 11-10 | [E2E CoS Test](./11-10.md) | Done | End-to-end verification of all acceptance criteria |
