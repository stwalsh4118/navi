# Navi Orchestrator Mode

## Stage: PRD

## Overview
An LLM-powered project manager built into Navi. It continuously watches all active Claude Code sessions, detects meaningful changes (task completions, commits), and uses a persistent Claude session with a file-based memory system to produce natural language briefings about what's happening across all your projects. Not a dashboard — a PM that reads the room, remembers context across invocations, and tells you what matters.

## Problem Statement
When running multiple Claude Code sessions across multiple projects, there's no unified view of what's happening. You have to cursor through each session individually to piece together the state of everything. Navi already monitors sessions, but it shows you session-level status — not project-level understanding. There's no one telling you "Apollo finished while you were away, Navi has been stuck for an hour, and you haven't touched Mnemos in three days."

## Target Users
Developers running multiple concurrent Claude Code sessions across multiple projects via tmux, using Navi as their session monitor.

## Core Value Proposition
A PM that knows things and keeps you on track. You open the PM view and in 10 seconds you know: what changed, what needs you, and where you left off. The LLM layer turns raw events into human-readable context. The memory system means the PM gets smarter about your projects over time — it notices patterns, remembers history, and doesn't repeat itself.

## User Stories
- As a developer, I want to see the state of all my projects in one place so I don't have to check each session individually.
- As a developer, I want to know what changed since I last looked so I can quickly orient myself after being away.
- As a developer, I want the PM to flag things that need my attention so nothing slips through the cracks.
- As a developer, I want breadcrumbs that tell me where I left off so I can resume work without context-switching overhead.
- As a developer, I want the PM to remember my projects over time so its briefings get more useful, not more generic.

## Features & Scope

### Phase 1: Aggregate View + Event Stream
- PM view toggled via `P` keybinding, mutually exclusive with session list
- Three-zone TUI layout: briefing (top), projects (middle), events (bottom)
- Project list derived from active sessions (session CWD → project discovery)
- Diff detection: cache project state snapshots, compare on refresh cycle
- Event log: rolling 24-hour JSONL file of structured change events
- Commit detection via HEAD SHA comparison on 30-second git polling cycle
- No LLM — structured data only. Briefing zone shows "No PM briefing yet"

### Phase 2: PM Agent + Memory System
- Persistent `claude -p --resume` session as the PM agent
- Memory system: short-term (working memory), long-term (institutional knowledge), per-project memories
- Invocation triggers: task completions, commits, on-demand
- PM produces structured JSON output (enforced via `--json-schema`)
- Briefing panel shows PM narrative, attention items, breadcrumbs
- PM manages its own memory files via `Read/Edit/Write` tools

### Phase 3: Proactive PM
- Pattern detection across time ("this is the third time X stalled on Y")
- Proactive suggestions ("PBI-8 is done, ready for a PR?")
- PM attention items surfaced in session list footer even outside PM view
- Conversational interaction — input mechanism in TUI to ask the PM questions

### Phase 4: All-Project Awareness
- Config-based project registration in `~/.navi/config.yaml`
- Track projects without active sessions — read git state and task status directly
- Full portfolio-level awareness and memory

### Future: Active Orchestration
- Start/stop sessions from the PM view
- Cross-project dependency tracking
- Delegate tasks to sessions

---

## System Design

### Architecture Overview
The PM is a feature within Navi, not a separate service. It adds three new internal packages alongside Navi's existing architecture:

```
internal/
├── pm/
│   ├── engine.go       — Core PM loop: diff detection, event accumulation, invocation scheduling
│   ├── snapshot.go     — Project state snapshotting and diff computation
│   ├── events.go       — Event types, event log (JSONL read/write)
│   ├── invoker.go      — Claude CLI invocation (exec.Command, --resume, --json-schema)
│   └── types.go        — Inbox, output, event structs
├── tui/
│   ├── pmview.go       — PM view rendering (three zones)
│   └── (existing files) — model.go gains PM toggle state
└── (existing packages)
```

### Data Flow

```
Navi's existing polling (500ms session status, 30s git/tasks)
    │
    ▼
PM Engine (new)
    ├── Snapshot current state per project
    ├── Diff against cached snapshots → structured events
    ├── Append events to ~/.navi/pm/events.jsonl
    ├── Check trigger conditions (task completion? commit? on-demand?)
    │
    ▼ (on trigger)
PM Invoker
    ├── Construct inbox JSON (events + current state)
    ├── exec.Command: claude -p --resume $SESSION_ID --json-schema ... --allowedTools ...
    ├── Parse structured JSON output
    ├── Cache output for TUI rendering
    │
    ▼
PM View (TUI)
    ├── Zone 1: briefing + attention_items + breadcrumbs (from PM output)
    ├── Zone 2: project list (from latest snapshots, sorted by priority)
    └── Zone 3: event log (from events.jsonl)
```

### Key Technical Decisions

**Diff detection via snapshot comparison.** On each refresh cycle (piggybacks on existing 30s task/git polling), snapshot each project's state: task statuses, HEAD SHA, branch name, session status. Compare against the cached snapshot. Emit typed events for any changes. Cache the new snapshot.

**HEAD SHA for commit detection.** Compare `git rev-parse HEAD` against the cached SHA per project. If different, run `git log --oneline <old>..<new>` to get the new commits. One command per project per 30s cycle — negligible cost. Store last-seen SHA in the project snapshot cache.

**Event log as rolling 24-hour JSONL.** Events are appended to `~/.navi/pm/events.jsonl`. On each write cycle, drop events older than 24 hours. File persists across Navi restarts. The event log serves two purposes: feeding the PM on invocation (events since last invocation), and rendering the event zone in the TUI.

**`claude -p --resume` for PM persistence.** The PM session runs indefinitely. Claude Code handles context compression automatically as the session grows. If the session errors or gets corrupted, Navi deletes the session ID and creates a fresh one — memory files carry the real continuity. No proactive reset needed.

**Multiple sessions per project: group by project root.** If multiple sessions share the same project root directory (common with teams), deduplicate to one project row. Use the most recently active session's status. In expandable project detail, list all sessions. Team agents are already nested under their parent session in Navi's data model.

### PM Session Initialization (First Run)
1. Check for `~/.navi/pm/session_id` — if missing, this is a first run
2. Create directory structure: `~/.navi/pm/memory/projects/`
3. Seed `short-term.md` with empty template: `# Short-Term Memory\n\nNo prior context. First invocation.`
4. Seed `long-term.md` with empty template: `# Long-Term Memory\n\nNo observations yet.`
5. Invoke `claude -p` WITHOUT `--resume` (creates new session), trigger type `initialization`, with full current state and no events
6. Capture session ID from response JSON, write to `~/.navi/pm/session_id`
7. All subsequent invocations use `--resume`

### PM Error Recovery
- If `claude -p` exits non-zero: log the error, keep last successful output, show "Last updated: Xm ago" in TUI
- If `claude -p` returns unparseable JSON (shouldn't happen with `--json-schema`, but): same fallback
- If `--resume` fails (session expired/corrupted): delete `session_id`, re-run initialization flow. Memory files persist — the PM picks up where it left off.
- If rate-limited: back off, retry after delay. Navi's TUI remains responsive — only the LLM layer is affected.

---

## LLM Integration

### Invocation Flow

```
1. Navi detects trigger event (task completion, commit, or on-demand)
2. Construct inbox JSON: events since last invocation + current state snapshot
3. Invoke:
     claude -p --resume $PM_SESSION_ID \
       --system-prompt-file ~/.navi/pm/system-prompt.md \
       --json-schema "$(cat ~/.navi/pm/output-schema.json)" \
       --allowedTools "Read,Edit,Write" \
       --output-format json \
       "$INBOX_JSON"
4. PM reads memory files (short-term.md, long-term.md, active project memories)
5. PM processes events against memory context
6. PM produces structured JSON output (schema-enforced)
7. PM updates memory files (rewrite short-term, optionally append long-term/project)
8. Navi parses structured_output from response JSON
9. Cache output, render in TUI PM view
```

### Invocation Triggers

| Trigger | When | Priority |
|---------|------|----------|
| `task_completed` | A task status changes to Done | Primary |
| `commit` | New commit detected (HEAD SHA changed) | Primary |
| `on_demand` | User presses refresh key in PM view | User-initiated |
| `initialization` | First run, no prior session | One-time |

Everything else (session status changes, metric updates, branch movements) is captured in the event log and included as context on the next triggered invocation, but doesn't trigger its own invocation.

Session needing input (waiting/permission) is surfaced immediately in the TUI — no LLM needed.

### Inbox Format (Navi → PM)

```json
{
  "timestamp": "2026-02-16T15:12:00Z",
  "trigger": "task_completed",
  "events_since_last_invocation": [
    {
      "timestamp": "2026-02-16T15:10:00Z",
      "type": "task_completed",
      "project": "apollo",
      "details": {
        "pbi": "PBI-8",
        "task_id": "8-5",
        "task_title": "Connection resolver"
      }
    },
    {
      "timestamp": "2026-02-16T14:30:00Z",
      "type": "session_status_change",
      "project": "navi",
      "details": { "from": "working", "to": "waiting" }
    },
    {
      "timestamp": "2026-02-16T14:23:00Z",
      "type": "commit",
      "project": "apollo",
      "details": {
        "sha": "abc1234",
        "message": "implement connection resolver",
        "branch": "feat/research-agent"
      }
    }
  ],
  "current_state": {
    "projects": [
      {
        "name": "apollo",
        "session_name": "apollo",
        "session_status": "idle",
        "current_pbi": { "id": "PBI-8", "title": "Research Agent" },
        "tasks": { "done": 5, "in_progress": 0, "todo": 0, "total": 5 },
        "git": {
          "branch": "feat/research-agent",
          "commits_ahead": 12,
          "dirty": false,
          "has_pr": false
        },
        "metrics": {
          "tokens_total": 57000,
          "working_seconds": 3600,
          "waiting_seconds": 120
        }
      }
    ]
  }
}
```

**Event types:** `task_completed`, `task_started`, `commit`, `session_status_change`, `pbi_completed`, `branch_created`, `pr_created`

### Output Schema (PM → Navi)

Enforced via `--json-schema`:

```json
{
  "type": "object",
  "required": ["briefing", "projects"],
  "properties": {
    "briefing": {
      "type": "string",
      "description": "2-4 sentence freeform narrative. The PM's voice."
    },
    "projects": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["name", "status", "summary"],
        "properties": {
          "name": { "type": "string" },
          "status": {
            "type": "string",
            "enum": ["needs_input", "pbi_complete", "active", "stale", "idle"]
          },
          "summary": { "type": "string" }
        }
      }
    },
    "attention_items": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["project", "message"],
        "properties": {
          "project": { "type": "string" },
          "message": { "type": "string" }
        }
      }
    },
    "breadcrumbs": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["project", "message"],
        "properties": {
          "project": { "type": "string" },
          "message": { "type": "string" }
        }
      }
    }
  }
}
```

### Memory System

```
~/.navi/pm/memory/
├── short-term.md          — Working memory (rewritten every cycle, ≤2000 tokens)
├── long-term.md           — Institutional knowledge (append-only, ≤3000 tokens)
└── projects/
    ├── apollo.md          — Per-project mental model (≤1000 tokens each)
    ├── navi.md
    └── mnemos.md
```

**Short-term memory** — Rewritten every invocation. Contains: current state of each project (1-2 lines), what the PM told the developer last time, pending observations not yet surfaced. The PM compresses aggressively — this file never grows. ~2000 token target.

**Long-term memory** — Append-only, pruned periodically by the PM itself. Contains: project arcs and velocity patterns, recurring blockers, workflow preferences, cross-project observations. Updated only when the PM notices something worth remembering weeks from now. Most invocations don't touch it. ~3000 token target.

**Per-project memory** — One file per project. Contains: project purpose, PBI history, common blockers, velocity patterns. Updated on PBI completion or significant events. Only loaded for projects with active sessions. ~1000 tokens each.

Total memory budget: ~6000-8000 tokens across all files. Combined with system prompt (~1500 tokens) and inbox data (~2000-4000 tokens), fits well within context limits.

The PM manages its own memory via `Read/Edit/Write` tools. Navi doesn't enforce size limits externally — it trusts the system prompt's instructions. If memory files grow too large, the PM is instructed to prune.

### PM System Prompt

```markdown
You are a project manager embedded in a terminal session monitor called Navi.
You track active software projects and keep the developer informed about what's
happening across all of them.

## Your Responsibilities
1. Process incoming events (task completions, commits) and update your
   understanding of each project
2. Produce concise briefings that tell the developer what matters right now
3. Maintain your memory files to preserve context across invocations
4. Flag things that need attention — don't narrate things that are going fine

## Personality
- Direct and concise. This renders in a TUI — every line counts.
- Opinionated when you see patterns. "This is the third time Apollo stalled
  on integration tasks" is useful.
- Don't repeat yourself. Check your last briefing in short-term memory before
  writing a new one.
- Prioritize action items over status reports. "PBI-8 is done, needs a PR"
  beats "PBI-8 has 5/5 tasks complete."
- Use plain language. No corporate PM speak. No bullet points in the briefing
  — write in short sentences.

## Memory Management

### Short-Term Memory (~/.navi/pm/memory/short-term.md)
Read this FIRST every invocation. It's your working context.
After processing, REWRITE it with:
- Current state of each active project (1-2 lines each)
- Summary of what you told the developer last time (so you don't repeat it)
- Pending observations you haven't surfaced yet
KEEP IT UNDER 2000 TOKENS. Compress aggressively. Drop old events. Summarize
rather than list. This file should never grow — you rewrite it every cycle.

### Long-Term Memory (~/.navi/pm/memory/long-term.md)
Your institutional knowledge. Read every invocation.
APPEND when you notice:
- Recurring patterns (project velocity, common blockers, workflow habits)
- Important milestones (PBI completions, major architectural changes)
- Cross-project observations (shared tech needs, dependencies, synergies)
DO NOT append every invocation. Only when something is genuinely worth
remembering weeks from now. Most invocations should not touch this file.
Periodically review and prune entries that are no longer relevant.
KEEP UNDER 3000 TOKENS.

### Per-Project Memory (~/.navi/pm/memory/projects/<name>.md)
Your mental model of one project. Read for active projects only.
Update when:
- A PBI completes or a new one starts
- You notice a pattern specific to this project (recurring blockers, velocity)
- Something architecturally significant happens
KEEP EACH FILE UNDER 1000 TOKENS.

## Output
Respond with valid JSON matching the required schema.
- The `briefing` field is your voice — write naturally, be concise, 2-4
  sentences max. This is what the developer reads first.
- Structured fields must accurately reflect current state from the input data.
- Attention items are for things that NEED ACTION, not status updates.
- Breadcrumbs are for context-switching: what was the developer last working on?

## Rules
- Never fabricate information. Only report what's in the input data or your
  memory. If you don't have data, omit the field.
- If nothing meaningful changed, say so in one sentence. Don't pad.
- After producing output, update your memory files via the tool.
```

---

## TUI Layout

### Toggling
`P` (capital) toggles PM view. Mutually exclusive with the session list — you're either in sessions mode or PM mode. Same Bubble Tea program, different view.

### Layout: Three Zones

```
┌─ PM ────────────────────────────────── updated 2m ago ─┐
│                                                         │
│  Apollo finished PBI-8 while you were away — all 5      │
│  tasks done, branch 12 commits ahead, no PR. Navi has   │
│  been waiting on permission for 47 min.                  │
│                                                         │
│  ⚠ navi — Permission request pending for 47 min         │
│  ⚠ apollo — All tasks done, create PR?                  │
│                                                         │
├─ Projects ──────────────────────────────────────────────┤
│  ❓ navi       PBI-29 Orchestrator    2/7  waiting   3m │
│  ✅ apollo     PBI-8  Research Agent  5/5  idle      1h │
│  ⏹  mnemos    PBI-3  Classification  0/4  stopped   3d │
├─ Events ────────────────────────────────────────────────┤
│  15:12  apollo   4 commits on feat/research-agent       │
│  15:10  apollo   Task 8-5 (Connection resolver) done    │
│  14:30  navi     working → waiting (permission)         │
│  14:23  apollo   Task 8-3 (Schema migrations) done      │
└─────────────────────────────────────────────────────────┘
```

**Zone 1: Briefing + Attention (top, ~30%)**
- PM's `briefing` text rendered as wrapped prose
- `attention_items` rendered below as `⚠` lines in yellow
- `breadcrumbs` shown when returning after absence
- Header shows "updated Xm ago" for staleness
- Static — never focused, always fully visible
- Before LLM is integrated (phase 1): "No PM briefing yet" in dim

**Zone 2: Projects (middle, ~30%)**
- One row per project, sorted by attention priority (needs_input → pbi_complete → active → stale → idle)
- Columns: status icon | name | PBI ID + title | progress (done/total) | session status | last activity
- Existing Navi status icons and color scheme
- Cursor navigation, Enter to jump to session list filtered to that project
- Space to expand: task breakdown, branch details, recent commits

**Zone 3: Event Log (bottom, ~40%)**
- Reverse chronological, newest at top
- Scrollable when focused (j/k, arrow keys)
- Each line: `[time] [project] [description]`
- Color-coded: task completions green, status changes yellow, commits cyan
- Dim styling for older events

**Zone focus:** Tab to move between Zone 2 (default) and Zone 3. Zone 1 is static.

**Responsive:** Zone heights proportional to terminal. Minimum 80 columns. Briefing collapses to one line + attention on very short terminals. PBI title truncates first.

**Session list integration:** When PM has attention items, session list footer shows: `PM: 2 items need attention (P to view)`

---

## Data Model

### Project Snapshot (cached per project)

```go
type ProjectSnapshot struct {
    Name          string          `json:"name"`
    ProjectDir    string          `json:"project_dir"`
    SessionNames  []string        `json:"session_names"`
    HeadSHA       string          `json:"head_sha"`
    Branch        string          `json:"branch"`
    CommitsAhead  int             `json:"commits_ahead"`
    Dirty         bool            `json:"dirty"`
    HasPR         bool            `json:"has_pr"`
    CurrentPBI    *PBISnapshot    `json:"current_pbi"`
    SessionStatus string          `json:"session_status"`
    LastActivity  time.Time       `json:"last_activity"`
    CapturedAt    time.Time       `json:"captured_at"`
}

type PBISnapshot struct {
    ID    string `json:"id"`
    Title string `json:"title"`
    Tasks TaskCounts `json:"tasks"`
}

type TaskCounts struct {
    Done       int `json:"done"`
    InProgress int `json:"in_progress"`
    Todo       int `json:"todo"`
    Total      int `json:"total"`
}
```

### Event

```go
type Event struct {
    Timestamp time.Time       `json:"timestamp"`
    Type      string          `json:"type"`
    Project   string          `json:"project"`
    Details   json.RawMessage `json:"details"`
}
```

### PM Output (parsed from LLM response)

```go
type PMOutput struct {
    Briefing       string           `json:"briefing"`
    Projects       []PMProject      `json:"projects"`
    AttentionItems []AttentionItem   `json:"attention_items,omitempty"`
    Breadcrumbs    []Breadcrumb     `json:"breadcrumbs,omitempty"`
    GeneratedAt    time.Time        `json:"-"` // set by Navi, not the PM
}

type PMProject struct {
    Name    string `json:"name"`
    Status  string `json:"status"`
    Summary string `json:"summary"`
}

type AttentionItem struct {
    Project string `json:"project"`
    Message string `json:"message"`
}

type Breadcrumb struct {
    Project string `json:"project"`
    Message string `json:"message"`
}
```

### Storage Layout

```
~/.navi/pm/
├── session_id              — PM Claude session ID for --resume
├── system-prompt.md        — PM system prompt
├── output-schema.json      — JSON schema for --json-schema
├── last-output.json        — Most recent PM output (for TUI rendering)
├── events.jsonl            — Rolling 24-hour event log
├── snapshots/
│   ├── apollo.json         — Latest project snapshot
│   ├── navi.json
│   └── mnemos.json
└── memory/
    ├── short-term.md       — PM working memory
    ├── long-term.md        — PM institutional knowledge
    └── projects/
        ├── apollo.md       — PM's mental model per project
        ├── navi.md
        └── mnemos.md
```

---

## Tech Stack

This is a feature within Navi. Inherits Navi's stack:

| Layer | Choice | Rationale |
|-------|--------|-----------|
| Language | Go | Navi is Go |
| TUI | Bubble Tea + Lip Gloss | Navi's existing framework |
| LLM | Claude Code CLI (`claude -p`) | Zero marginal cost on Max plan, `--resume` for persistence, `--json-schema` for reliable output |
| Storage | JSON files + JSONL | Follows Navi's file-based pattern (`~/.claude-sessions/`, `~/.navi/`) |
| PM Memory | Markdown files | Human-readable, editable by the PM via Write tool, inspectable by the developer |

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| LLM latency blocks TUI | PM view feels slow | PM invocation is async (goroutine). TUI shows last output + "updating..." indicator. Structured data always available. |
| PM produces poor/repetitive briefings | Noise, user ignores PM | System prompt is specific about personality and anti-repetition. Short-term memory includes last briefing. Iterate on prompt. |
| Memory files grow unbounded | Context window fills, PM quality degrades | Size targets in system prompt (2000/3000/1000 tokens). PM self-prunes. If that fails, Navi can truncate externally as a safety valve. |
| `--resume` session gets corrupted | PM invocation fails | Delete session_id, re-initialize. Memory files carry continuity. Seamless recovery. |
| Max plan rate limits | PM invocations fail during bursts | Back off and retry. Don't invoke on every commit — batch events over 1-5 minutes. |
| Git commands slow across many projects | 30s polling adds latency | `git rev-parse HEAD` is ~1ms. `git log` only runs when SHA changed. Cost is negligible even with 10+ projects. |
| Claude Code CLI interface changes | Invocation breaks | Pin to known-working CLI flags. Wrap invocation in a single function for easy updates. |

---

## Milestones

| Phase | Scope | Depends On |
|-------|-------|------------|
| **Phase 1** | PM engine: snapshot diffing, event log, aggregate project view in TUI. No LLM. | Navi's existing session/task/git infrastructure |
| **Phase 2** | PM agent: `claude -p --resume`, memory system, inbox/output contracts, briefing panel. The PM comes alive. | Phase 1 (event pipeline) |
| **Phase 3** | Proactive PM: pattern detection, proactive suggestions, attention in session list footer, conversational interaction. | Phase 2 (working PM agent) |
| **Phase 4** | All-project awareness: config-based registration, sessionless project tracking. | Phase 2 (memory system) |

Phase 1 is the prerequisite — it builds the data pipeline. Phase 2 is the product — it's when the PM actually works. Phase 3 and 4 are enhancements.

---

## References
- Navi source: `~/workspace/navi/`
- Navi PRD: `~/workspace/thoth/ideas/prds/2026-02-07-navi.md`
- Apollo PRD (session orchestration patterns): `~/workspace/thoth/ideas/prds/2026-02-14-apollo.md`
- Claude Code CLI docs: `claude -p --help`, `--resume`, `--json-schema`, `--allowedTools`
