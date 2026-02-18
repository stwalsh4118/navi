# PBI-48: PM Agent — Claude CLI Invoker and Memory System

[View in Backlog](../backlog.md)

## Overview

Wire up the PM agent: a persistent Claude session via `claude -p --resume` that processes project events, maintains a file-based memory system, and produces structured JSON briefings. This is the Phase 2 backend — the PM comes alive.

## Problem Statement

The event pipeline (PBI-46) captures what happened, but raw events aren't useful to a developer who wants a quick summary. An LLM layer is needed to synthesize events into human-readable briefings, track context across invocations via memory, and surface actionable attention items.

## User Stories

- As a developer, I want an AI PM that processes project events and produces concise briefings so that I can understand project state in 10 seconds.
- As a developer, I want the PM to remember context across invocations so that its briefings get smarter over time and don't repeat themselves.

## Technical Approach

- `internal/pm/invoker.go`: Claude CLI invocation wrapper using `exec.Command`. Handles `claude -p --resume`, `--json-schema`, `--allowedTools Read,Edit,Write`, `--output-format json`.
- Session initialization flow: detect first run (no `~/.config/navi/pm/session_id`), create directory structure, seed memory files, invoke without `--resume`, capture and store session ID.
- Inbox construction: build JSON payload from accumulated events + current project snapshots.
- Output parsing: parse `structured_output` from Claude's JSON response into `PMOutput` struct.
- Trigger system: invoke PM on `task_completed`, `commit`, or `on_demand`. Other events accumulate and are included in the next triggered invocation.
- Error recovery: non-zero exit → keep last output, show staleness. Unparseable JSON → same fallback. `--resume` failure → delete session_id, re-initialize. Rate limited → backoff and retry.
- PM invocation is async (goroutine) — TUI never blocks.
- Storage layout under `~/.config/navi/pm/`: `session_id`, `system-prompt.md`, `output-schema.json`, `last-output.json`, `events.jsonl`, `snapshots/`, `memory/`.
- System prompt and output schema files are created during initialization from embedded templates.
- Memory files: `memory/short-term.md` (~2000 tokens), `memory/long-term.md` (~3000 tokens), `memory/projects/<name>.md` (~1000 tokens each). PM manages these via Read/Edit/Write tools.

## UX/UI Considerations

N/A — backend PBI. Output is consumed by PBI-49 (briefing panel rendering).

## Acceptance Criteria

1. First-run initialization creates `~/.config/navi/pm/` directory structure, seeds `short-term.md` and `long-term.md` with empty templates, writes `system-prompt.md` and `output-schema.json`.
2. First invocation runs `claude -p` without `--resume`, captures session ID, writes to `~/.config/navi/pm/session_id`.
3. Subsequent invocations use `--resume` with the stored session ID.
4. Inbox JSON includes: timestamp, trigger type, events since last invocation, and current project state snapshot.
5. Output is parsed into `PMOutput` struct (briefing, projects, attention_items, breadcrumbs) and cached to `last-output.json`.
6. PM is invoked on `task_completed` and `commit` triggers; other events accumulate for next invocation.
7. On-demand trigger works when user requests refresh.
8. Non-zero exit or parse failure falls back to last successful output with staleness indicator.
9. Corrupted `--resume` session triggers re-initialization; memory files persist.
10. PM invocation runs in a goroutine — TUI responsiveness is unaffected.

## Dependencies

- **Depends on**: PBI-46 (event pipeline and snapshot data for inbox construction)
- **External**: Claude Code CLI (`claude -p`, `--resume`, `--json-schema`, `--allowedTools`)

## Open Questions

- None — the PRD specifies the invocation flow, flags, error recovery, and memory system in detail.

## Related Tasks

[View Tasks](./tasks.md)
