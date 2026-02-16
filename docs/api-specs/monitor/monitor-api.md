# Monitor API

Package: `internal/monitor`

## Attach Monitor

```go
type AttachMonitor struct{}

func New(notifier *audio.Notifier, statusDir string, pollInterval time.Duration) *AttachMonitor
func (m *AttachMonitor) Start(ctx context.Context, initialStates map[string]string)
func (m *AttachMonitor) States() map[string]string
```

Behavior:
- Polls `session.ReadStatusFiles(statusDir)` on `pollInterval`
- Tracks session status transitions in an internal state map
- Calls `notifier.Notify(sessionName, newStatus)` on transitions when notifier is non-nil
- Supports state handoff via `initialStates` input and `States()` output
- Stops cleanly when `ctx.Done()` is closed
