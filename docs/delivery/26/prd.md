# PBI-26: Token Metrics from Session Transcripts

[View in Backlog](../backlog.md)

## Overview

Parse Claude Code's session transcript files (`~/.claude/projects/*/*.jsonl`) to extract and display per-session token usage. This completes the token tracking feature that was blocked in PBI-12 due to the mistaken belief that OpenTelemetry was required.

## Problem Statement

PBI-12 implemented time and tool tracking but could not implement token tracking because research indicated tokens were only available via OpenTelemetry. However, further investigation revealed that Claude Code stores complete session transcripts with per-message token data in `~/.claude/projects/*/`. This data can be parsed directly.

## User Stories

- As a user, I want to see token usage per session so I can track API costs
- As a user, I want to see input vs output token breakdown so I can understand usage patterns
- As a user, I want token counts in both the inline badges and detail view

## Technical Approach

### Data Source

Claude Code stores session transcripts at:
```
~/.claude/projects/<project-path>/<session-id>.jsonl
```

Where `<project-path>` is the working directory with `/` replaced by `-` (e.g., `-home-sean-workspace-navi`).

Each assistant message contains token usage:
```json
{
  "type": "assistant",
  "message": {
    "usage": {
      "input_tokens": 1,
      "output_tokens": 4,
      "cache_read_input_tokens": 17601,
      "cache_creation_input_tokens": 13870
    }
  }
}
```

### Implementation

1. **Path Conversion**: Convert session CWD to project folder path
   - `/home/sean/workspace/navi` → `-home-sean-workspace-navi`

2. **Session Matching**: Find the most recently modified `.jsonl` file in the project folder
   - Match by modification time (most recent = current session)
   - Or parse the file to match session start time

3. **Token Parsing**: Sum tokens from all assistant messages
   - `input_tokens` + `cache_read_input_tokens` + `cache_creation_input_tokens` = total input
   - `output_tokens` = total output
   - Sum both for total

4. **Integration**: Add token data to existing Metrics struct
   - Update during session poll cycle
   - Cache results to avoid re-parsing unchanged files

### Display

Update existing metrics display:
- **Inline badge**: `⏱ 1h 23m  🔧 45  📊 57k tokens`
- **Detail view**: Add token section with input/output/cache breakdown
- **Header aggregate**: Include total tokens across sessions

## Acceptance Criteria

1. Token counts are extracted from session transcript files
2. Input and output tokens are tracked separately
3. Cache tokens (read + creation) are included in totals
4. Tokens display in inline badges
5. Tokens display in detail view with breakdown
6. Aggregate tokens shown in header
7. Token parsing doesn't significantly impact poll performance

## Dependencies

- PBI-12: Session Metrics (provides Metrics struct and UI components)

## Open Questions

- Should we track cost estimates based on token counts?
- Should cache tokens be shown separately or combined with input?

## Related Tasks

[View Tasks](./tasks.md)
