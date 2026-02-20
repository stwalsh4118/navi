# Resource API

Process tree RSS monitoring via Linux `/proc` filesystem.

**Package**: `internal/resource`

## Functions

```go
// SessionRSS returns total RSS in bytes for all processes in a tmux session's
// process tree. Gets pane PIDs via `tmux list-panes -s`, then walks /proc
// recursively. Returns 0 on error or empty session.
func SessionRSS(sessionName string) int64
```

## Internal Functions (unexported)

| Function | Description |
|----------|-------------|
| `getPanePIDs(sessionName string) []int` | Runs `tmux list-panes -s -t <session> -F '#{pane_pid}'` |
| `processTreeRSS(pid int) int64` | Recursive tree walk summing RSS |
| `readRSSBytes(pid int) int64` | Reads `/proc/<pid>/statm` field 2, converts pages to bytes |
| `getChildPIDs(pid int) []int` | Globs `/proc/<pid>/task/*/children`, deduplicates |

## Related Types (metrics package)

```go
// ResourceMetrics tracks resource usage for a session.
type ResourceMetrics struct {
    RSSBytes int64 `json:"rss_bytes"`
}

// FormatBytes returns human-readable byte size (e.g., "256M", "1.2G").
func FormatBytes(bytes int64) string
```

## TUI Integration

- **Poll interval**: `resourcePollInterval = 2 * time.Second`
- **Message types**: `resourceTickMsg`, `resourcePollMsg map[string]int64`
- **Cache**: `Model.resourceCache map[string]int64` persists RSS across session refreshes
- **Badge**: `🧠 <formatted>` rendered by `renderMetricsBadges()` in view.go
