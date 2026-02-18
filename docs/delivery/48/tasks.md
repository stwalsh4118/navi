# Tasks for PBI 48: PM Agent — Claude CLI Invoker and Memory System

This document lists all tasks associated with PBI 48.

**Parent PBI**: [PBI 48: PM Agent — Claude CLI Invoker and Memory System](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--- | :----- | :---------- |
| 48-1 | [PM Briefing Types and Embedded Templates](./48-1.md) | Done | Define PMBriefing, AttentionItem, Breadcrumb, InboxPayload types; embed system-prompt.md and output-schema.json templates; add TriggerType constants |
| 48-2 | [PM Storage Initialization and Session Management](./48-2.md) | Done | Create EnsureStorageLayout(); seed memory file templates; manage storage directory layout |
| 48-3 | [Inbox Construction](./48-3.md) | Done | Build JSON inbox payload combining accumulated events, project snapshots, trigger type, and timestamp for Claude invocation |
| 48-4 | [Claude CLI Invoker Core](./48-4.md) | Done | Implement exec.Command wrapper for `claude -p` with fresh sessions, streaming, model selection, and CLI flag assembly |
| 48-5 | [Output Parsing and Caching](./48-5.md) | Done | Parse Claude's structured_output JSON into PMBriefing struct; cache to last-output.json; load cached output with staleness tracking |
| 48-6 | [Error Recovery and Resilience](./48-6.md) | Done | Handle non-zero exit, unparseable JSON, and rate limiting with fallback and backoff strategies |
| 48-7 | [Trigger System and TUI Integration](./48-7.md) | Done | Wire trigger evaluation (task_completed, commit, on_demand) into TUI message loop; async goroutine invocation; streaming status; cached briefing on startup |
| 48-8 | [E2E CoS Test](./48-8.md) | Done | End-to-end verification of acceptance criteria for PM agent initialization, invocation, output, triggers, error recovery, and async behavior |

## Dependency Graph

```
48-1 (Types & Templates)
 ├──▶ 48-2 (Storage Init)
 ├──▶ 48-3 (Inbox Construction)
 └──▶ 48-5 (Output Parsing)
        │
48-2 ───┤
48-3 ───┼──▶ 48-4 (CLI Invoker Core)
        │
48-4 ───┼──▶ 48-6 (Error Recovery)
48-5 ───┘
        │
48-6 ───┼──▶ 48-7 (Trigger System & TUI)
        │
48-7 ───┴──▶ 48-8 (E2E CoS Test)
```

## Implementation Order

1. **48-1** — Types and templates first; everything else imports these.
2. **48-2** — Storage layer; needed before invoker can read/write session state.
3. **48-3** — Inbox construction; needed to feed the invoker.
4. **48-5** — Output parsing; needed by invoker to return structured results.
5. **48-4** — CLI invoker core; depends on storage, inbox, and output parsing.
6. **48-6** — Error recovery; wraps the invoker with resilience logic.
7. **48-7** — TUI integration; wires everything into the running application.
8. **48-8** — E2E test; verifies the full pipeline end-to-end.

## Complexity Ratings

| Task ID | Complexity | External Packages |
|---------|-----------|-------------------|
| 48-1 | Simple | None |
| 48-2 | Medium | None |
| 48-3 | Simple | None |
| 48-4 | Complex | Claude CLI (flags, JSON output format) |
| 48-5 | Medium | None |
| 48-6 | Medium | None |
| 48-7 | Complex | None |
| 48-8 | Complex | None |

## External Package Research Required

| Task ID | Package | Guide Document |
|---------|---------|---------------|
| 48-4 | Claude CLI (`claude -p`, `--resume`, `--json-schema`, `--output-format json`) | `48-4-claude-cli-guide.md` |
