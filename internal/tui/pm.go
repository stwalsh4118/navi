package tui

import (
	"maps"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/debug"
	"github.com/stwalsh4118/navi/internal/pm"
	"github.com/stwalsh4118/navi/internal/session"
	"github.com/stwalsh4118/navi/internal/task"
)

const pmPollInterval = 60 * time.Second

// pmTriggerEvents are event types that trigger a PM invocation.
var pmTriggerEvents = map[pm.EventType]pm.TriggerType{
	pm.EventTaskCompleted: pm.TriggerTaskCompleted,
	pm.EventCommit:        pm.TriggerCommit,
}

func pmTickCmd() tea.Cmd {
	return tea.Tick(pmPollInterval, func(t time.Time) tea.Msg {
		return pmTickMsg(t)
	})
}

func pmRunCmd(engine *pm.Engine, sessions []session.Info, taskResults map[string]*task.ProviderResult) tea.Cmd {
	sessionCopy := append([]session.Info(nil), sessions...)
	resultsCopy := maps.Clone(taskResults)

	return func() tea.Msg {
		output, err := engine.Run(sessionCopy, resultsCopy)
		return pmOutputMsg{output: output, err: err}
	}
}

// pmInvokeCmd runs InvokeWithRecoveryStream in a goroutine, sending streaming
// status updates through a channel consumed by pmStreamReadCmd.
func pmInvokeCmd(invoker *pm.Invoker, trigger pm.TriggerType, snapshots []pm.ProjectSnapshot, events []pm.Event) tea.Cmd {
	snapshotsCopy := append([]pm.ProjectSnapshot(nil), snapshots...)
	eventsCopy := append([]pm.Event(nil), events...)

	debug.Log("tui: pm invoke cmd queued, trigger=%s, snapshots=%d, events=%d", trigger, len(snapshotsCopy), len(eventsCopy))

	streamCh := make(chan pm.StreamEvent, 16)

	// Launch the invoke goroutine — it writes to streamCh and closes it when done.
	invokeResultCh := make(chan pmInvokeMsg, 1)
	go func() {
		debug.Log("tui: pm invoke goroutine started")
		inbox, err := pm.BuildInbox(trigger, snapshotsCopy, eventsCopy)
		if err != nil {
			debug.Log("tui: pm inbox build failed: %v", err)
			close(streamCh)
			invokeResultCh <- pmInvokeMsg{err: err}
			return
		}
		briefing, isStale, err := invoker.InvokeWithRecoveryStream(inbox, streamCh)
		close(streamCh)
		if err != nil {
			debug.Log("tui: pm invoke returned error: %v", err)
		} else {
			debug.Log("tui: pm invoke succeeded, stale=%t", isStale)
		}
		invokeResultCh <- pmInvokeMsg{briefing: briefing, isStale: isStale, err: err}
	}()

	// Return a command that reads the first stream event (or the final result).
	return pmStreamReadCmd(streamCh, invokeResultCh)
}

// pmStreamReadCmd returns a tea.Cmd that reads the next stream event or the
// final invoke result, whichever comes first.
func pmStreamReadCmd(streamCh <-chan pm.StreamEvent, resultCh <-chan pmInvokeMsg) tea.Cmd {
	return func() tea.Msg {
		select {
		case event, ok := <-streamCh:
			if !ok {
				// Stream closed — read final result.
				return <-resultCh
			}
			return pmStreamMsg{
				status:   event.Status,
				streamCh: streamCh,
				resultCh: resultCh,
			}
		case result := <-resultCh:
			return result
		}
	}
}

// pmCheckTrigger scans engine events for trigger types. Returns the trigger type if
// a PM invocation should be fired, or empty string if not.
func pmCheckTrigger(events []pm.Event) pm.TriggerType {
	for _, event := range events {
		if trigger, ok := pmTriggerEvents[event.Type]; ok {
			return trigger
		}
	}
	return ""
}
