# Tasks for PBI 12: Session Metrics

This document lists all tasks associated with PBI 12.

**Parent PBI**: [PBI 12: Session Metrics](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--- | :----- | :---------- |
| 12-1 | [Define Metrics types and constants](./12-1.md) | Done | Add Metrics, TokenMetrics, TimeMetrics, and ToolMetrics structs |
| 12-2 | [Extend SessionInfo with Metrics field](./12-2.md) | Done | Add metrics field to SessionInfo and update JSON parsing |
| 12-3 | [Add session start time tracking](./12-3.md) | Done | Track session start time and persist across status updates |
| 12-4 | [Research Claude Code hook environment](./12-4.md) | Done | Investigate available environment variables for token/tool data |
| 12-5 | [Extend hook script for metrics capture](./12-5.md) | Done | Update notify.sh to capture and persist metrics data |
| 12-6 | [Implement status time accumulation](./12-6.md) | Done | Track time spent in each status (working, waiting, etc.) |
| 12-7 | [Add inline metrics badges](./12-7.md) | Done | Display compact token/time metrics next to session name |
| 12-8 | [Create metrics detail dialog](./12-8.md) | Done | Implement 'i' key for comprehensive metrics view |
| 12-9 | [Add aggregate metrics dashboard](./12-9.md) | Done | Display totals across all sessions in header or dedicated view |
| 12-10 | [E2E CoS Test](./12-10.md) | Done | End-to-end verification of all acceptance criteria |
