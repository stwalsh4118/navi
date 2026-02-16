# Tasks for PBI 40: OpenCode Status Hook Integration

This document lists all tasks associated with PBI 40.

**Parent PBI**: [PBI 40: OpenCode Status Hook Integration](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--- | :----- | :---------- |
| 40-1 | [ExternalAgent data model and JSON parsing](./40-1.md) | Review | Add ExternalAgent struct and Agents map field to session.Info with unit tests |
| 40-2 | [Preserve agents field in notify.sh](./40-2.md) | Review | Update notify.sh to read and preserve the agents field during write cycles |
| 40-3 | [OpenCode navi plugin](./40-3.md) | Review | Create the OpenCode plugin that maps lifecycle events to navi status updates |
| 40-4 | [Install script OpenCode plugin deployment](./40-4.md) | Review | Extend install.sh to deploy the OpenCode plugin to ~/.config/opencode/plugins/ |
| 40-5 | [E2E CoS verification](./40-5.md) | Review | Verify all acceptance criteria end-to-end |
