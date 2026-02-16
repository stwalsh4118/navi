package monitor

import (
	"context"
	"sync"
	"time"

	"github.com/stwalsh4118/navi/internal/audio"
	"github.com/stwalsh4118/navi/internal/session"
)

// AttachMonitor polls session status files in the background while users are attached.
type AttachMonitor struct {
	notifier  *audio.Notifier
	statusDir string
	interval  time.Duration

	mu     sync.Mutex
	states map[string]string

	notifyFn func(sessionName, newStatus string)
}

// New creates a new attach monitor.
func New(notifier *audio.Notifier, statusDir string, pollInterval time.Duration) *AttachMonitor {
	m := &AttachMonitor{
		notifier:  notifier,
		statusDir: statusDir,
		interval:  pollInterval,
		states:    make(map[string]string),
	}
	m.notifyFn = m.notifyStatusChange
	return m
}

// Start launches the background polling loop.
func (m *AttachMonitor) Start(ctx context.Context, initialStates map[string]string) {
	if m == nil {
		return
	}

	m.mu.Lock()
	m.states = copyStates(initialStates)
	skipInitialPoll := len(m.states) == 0
	m.mu.Unlock()

	go func() {
		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				currentSessions, err := session.ReadStatusFiles(m.statusDir)
				if err != nil {
					continue
				}

				currentStates := make(map[string]string, len(currentSessions))
				for _, s := range currentSessions {
					currentStates[s.TmuxSession] = s.Status
				}

				m.mu.Lock()
				if skipInitialPoll {
					m.states = currentStates
					skipInitialPoll = false
					m.mu.Unlock()
					continue
				}

				for sessionName, newStatus := range currentStates {
					if oldStatus, ok := m.states[sessionName]; ok && oldStatus != newStatus {
						m.notifyFn(sessionName, newStatus)
					}
				}
				m.states = currentStates
				m.mu.Unlock()
			}
		}
	}()
}

// States returns a thread-safe copy of the monitor states.
func (m *AttachMonitor) States() map[string]string {
	if m == nil {
		return map[string]string{}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return copyStates(m.states)
}

func (m *AttachMonitor) notifyStatusChange(sessionName, newStatus string) {
	if m.notifier != nil {
		m.notifier.Notify(sessionName, newStatus)
	}
}

func copyStates(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
