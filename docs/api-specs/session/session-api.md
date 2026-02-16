# Session API

Package: `internal/session`

## Types

```go
type Info struct {
    TmuxSession string
    Status      string
    Message     string
    CWD         string
    Timestamp   int64
    Git         *git.Info
    Remote      string
    Metrics     *metrics.Metrics
    Team        *TeamInfo
    Agents      map[string]ExternalAgent
}

type ExternalAgent struct {
    Status    string
    Timestamp int64
}
```

## Constants

```go
const (
    StatusWaiting    = "waiting"
    StatusPermission = "permission"
    StatusWorking    = "working"
    StatusIdle       = "idle"
    StatusStopped    = "stopped"

    PollInterval     = 500 * time.Millisecond
    DefaultStatusDir = "~/.claude-sessions"
)
```

## Status File IO

```go
func ReadStatusFiles(dir string) ([]Info, error)
```

Behavior:
- Reads all `*.json` files in `dir` and unmarshals into `[]Info`
- Returns empty slice and nil error when directory does not exist
- Skips unreadable files and malformed JSON entries

## Session Utilities

```go
func SortSessions(sessions []Info)
func AggregateMetrics(sessions []Info) *metrics.Metrics
func HasPriorityTeammate(s Info) bool
```
