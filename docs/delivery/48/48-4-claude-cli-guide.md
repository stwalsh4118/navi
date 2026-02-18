# Task 48-4 Claude CLI Guide

Date: 2026-02-18

## Sources

- https://docs.anthropic.com/en/cli-reference
- https://docs.anthropic.com/en/docs/claude-code/overview

## Relevant CLI Behaviors

1. `claude -p` (or `--print`) runs non-interactive print mode and exits.
2. `--resume` / `-r` resumes an existing session by ID or name.
3. `--output-format json` returns structured envelope output suitable for machine parsing.
4. `--allowedTools` accepts a comma-separated list (for example `Read,Edit,Write`).
5. `--system-prompt-file` is the file-based system prompt flag in print mode.
6. `--json-schema` accepts a JSON schema string, not a file path.

## Invocation Pattern Used by Navi PM

The PM invoker should:

- run in print mode (`-p`)
- pass input inbox JSON via stdin
- request JSON envelope output (`--output-format json`)
- constrain tools to memory-safe operations (`--allowedTools Read,Edit,Write`)
- set a system prompt from file (`--system-prompt-file <path>`)
- load schema file content and pass it as the `--json-schema` argument
- include `--resume <session-id>` only when a stored session exists

## Example Commands

First run (no resume):

```bash
claude -p \
  --output-format json \
  --allowedTools Read,Edit,Write \
  --system-prompt-file ~/.config/navi/pm/system-prompt.md \
  --json-schema "$(cat ~/.config/navi/pm/output-schema.json)"
```

Resume run:

```bash
claude -p \
  --resume 550e8400-e29b-41d4-a716-446655440000 \
  --output-format json \
  --allowedTools Read,Edit,Write \
  --system-prompt-file ~/.config/navi/pm/system-prompt.md \
  --json-schema "$(cat ~/.config/navi/pm/output-schema.json)"
```

## Notes for Implementation

- The invoker should read `output-schema.json` from disk and pass file contents to `--json-schema`.
- The response envelope in JSON mode includes metadata plus result payload, so parser logic must extract nested structured content.
