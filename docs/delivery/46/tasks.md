# Tasks for PBI 46: PM Engine — Project Snapshots and Event Pipeline

This document lists all tasks associated with PBI 46.

**Parent PBI**: [PBI 46: PM Engine — Project Snapshots and Event Pipeline](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--- | :----- | :---------- |
| 46-1 | [PM Core Types and Constants](./46-1.md) | Proposed | Define all PM data structures: ProjectSnapshot, Event, EventType, TaskCounts, PMOutput |
| 46-2 | [Project Discovery and Snapshot Capture](./46-2.md) | Proposed | Discover projects from session CWDs, capture git/task/session state snapshots |
| 46-3 | [Snapshot Diffing and Event Generation](./46-3.md) | Proposed | Compare snapshots to detect changes and generate typed events |
| 46-4 | [JSONL Event Log and Pruning](./46-4.md) | Proposed | Append events to JSONL file with 24-hour rolling pruning |
| 46-5 | [PM Engine and TUI Integration](./46-5.md) | Proposed | Orchestrate snapshot-diff-emit pipeline and hook into TUI polling cycle |
| 46-6 | [E2E CoS Test](./46-6.md) | Proposed | Verify all PBI-46 acceptance criteria with integration tests |
