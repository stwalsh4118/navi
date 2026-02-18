# PM API Specification

Package `internal/pm` — Claude CLI invoker, briefing types, event pipeline, and memory system.

## Types

### PMBriefing

```go
type PMBriefing struct {
    Summary        string            `json:"summary"`
    Projects       []ProjectBriefing `json:"projects"`
    AttentionItems []AttentionItem   `json:"attention_items"`
    Breadcrumbs    []Breadcrumb      `json:"breadcrumbs"`
}

type ProjectBriefing struct {
    Name           string `json:"name"`
    Status         string `json:"status"`
    CurrentWork    string `json:"current_work"`
    RecentActivity string `json:"recent_activity"`
}

type AttentionItem struct {
    Priority    string `json:"priority"`    // "high", "medium", "low"
    Title       string `json:"title"`
    Description string `json:"description"`
    ProjectName string `json:"project_name"`
}

type Breadcrumb struct {
    Timestamp string `json:"timestamp"`
    Summary   string `json:"summary"`
}
```

### InboxPayload

```go
type InboxPayload struct {
    Timestamp   time.Time         `json:"timestamp"`
    TriggerType TriggerType       `json:"trigger_type"`
    Events      []Event           `json:"events"`
    Snapshots   []ProjectSnapshot `json:"snapshots"`
}
```

### InvokeResult

```go
type InvokeResult struct {
    Output *PMBriefing
    Raw    []byte
    Usage  *InvokeUsage
}

type InvokeUsage struct {
    InputTokens  int     `json:"input_tokens"`
    OutputTokens int     `json:"output_tokens"`
    CostUSD      float64 `json:"cost_usd"`
    DurationMS   int     `json:"duration_ms"`
    NumTurns     int     `json:"num_turns"`
}
```

### Trigger Types

```go
const (
    TriggerTaskCompleted TriggerType = "task_completed"
    TriggerCommit        TriggerType = "commit"
    TriggerOnDemand      TriggerType = "on_demand"
)
```

## Invoker

```go
func NewInvoker() (*Invoker, error)
func (i *Invoker) Invoke(inbox *InboxPayload) (*InvokeResult, error)
func (i *Invoker) InvokeStream(inbox *InboxPayload, stream chan<- StreamEvent) (*InvokeResult, error)
```

- Each invocation is a fresh Claude CLI session (no `--resume`).
- Memory files under `~/.config/navi/pm/memory/` provide continuity between runs.
- Uses `--model sonnet` for cost efficiency.
- Streaming mode sends `StreamEvent` status updates through the channel.

## Recovery

```go
func (i *Invoker) InvokeWithRecovery(inbox *InboxPayload) (*PMBriefing, bool, error)
func (i *Invoker) InvokeWithRecoveryStream(inbox *InboxPayload, stream chan<- StreamEvent) (*PMBriefing, bool, error)
```

- Rate-limit retries with exponential backoff (1s, 2s, 4s).
- Falls back to cached output on failure (`isStale=true`).
- Returns error only if both invocation and cache fail.

## Inbox Construction

```go
func BuildInbox(trigger TriggerType, snapshots []ProjectSnapshot, events []Event) (*InboxPayload, error)
func InboxToJSON(inbox *InboxPayload) ([]byte, error)
```

## Output Parsing and Caching

```go
func ParseOutput(raw []byte) (*PMBriefing, error)
func CacheOutput(briefing *PMBriefing) error
func LoadCachedOutput() (*CachedOutput, error)
```

- Parses `structured_output` or `result` envelope from Claude CLI JSON.
- Falls back to direct JSON parsing and markdown-embedded JSON extraction.
- Cache stored at `~/.config/navi/pm/last-output.json`.

## Storage

```go
func EnsureStorageLayout() error
```

Directory structure under `~/.config/navi/pm/`:
- `memory/short-term.md` — seeded on first run, managed by Claude
- `memory/long-term.md` — seeded on first run, managed by Claude
- `memory/projects/` — per-project memory files managed by Claude
- `snapshots/` — project state snapshots
- `system-prompt.md` — embedded template, overwritten on each startup
- `output-schema.json` — embedded template, overwritten on each startup
- `last-output.json` — cached briefing output

## TUI Integration

PM trigger events in `internal/tui/pm.go`:
- `EventTaskCompleted` and `EventCommit` trigger PM invocation.
- On-demand trigger via `i` key in PM view.
- 60-second poll interval for snapshot/event collection.
- Cached briefing loaded on startup for immediate display.
- Streaming status shown during invocation.
