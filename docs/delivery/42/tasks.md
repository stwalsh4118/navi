# Tasks for PBI 42: Audio Notifications with Sound Packs and TTS Session Announcements

This document lists all tasks associated with PBI 42.

**Parent PBI**: [PBI 42: Audio Notifications with Sound Packs and TTS Session Announcements](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--- | :----- | :---------- |
| 42-1 | [Define audio config types and YAML loader](./42-1.md) | Proposed | Config structs, YAML loading, validation, and defaults for sounds.yaml |
| 42-2 | [Implement audio player detection and playback](./42-2.md) | Proposed | Detect system audio players and play sound files non-blocking |
| 42-3 | [Implement TTS engine detection and speech](./42-3.md) | Proposed | Detect system TTS engines and speak session announcements non-blocking |
| 42-4 | [Implement notification manager with cooldown](./42-4.md) | Proposed | Orchestrate player + TTS with per-session cooldown tracking |
| 42-5 | [Integrate audio notifications into TUI model](./42-5.md) | Proposed | Add state-change detection to Model and trigger audio on status transitions |
| 42-6 | [E2E CoS Test](./42-6.md) | Proposed | End-to-end verification of the full audio notification pipeline |
