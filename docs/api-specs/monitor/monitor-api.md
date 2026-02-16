# Monitor API

Package: `internal/monitor`

## Attach Monitor

```go
type AttachMonitor struct{}

func New(notifier *audio.Notifier, statusDir string, pollInterval time.Duration) *AttachMonitor
func (m *AttachMonitor) Start(ctx context.Context, initialStates map[string]string, initialAgentStates map[string]map[string]string)
func (m *AttachMonitor) States() map[string]string
func (m *AttachMonitor) AgentStates() map[string]map[string]string
```

Behavior:
- Polls `session.ReadStatusFiles(statusDir)` on `pollInterval`
- Tracks session status transitions and external agent status transitions in internal state maps
- Calls `notifier.Notify(sessionName, newStatus)` on transitions when notifier is non-nil
- Calls `notifier.Notify(sessionName+":"+agentType, newStatus)` for external agent transitions
- Supports state handoff via `initialStates`/`initialAgentStates` input and `States()`/`AgentStates()` output
- Stops cleanly when `ctx.Done()` is closed
