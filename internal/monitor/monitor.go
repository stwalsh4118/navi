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

	mu          sync.Mutex
	states      map[string]string
	agentStates map[string]map[string]string

	notifyFn func(sessionName, newStatus string)
}

// New creates a new attach monitor.
func New(notifier *audio.Notifier, statusDir string, pollInterval time.Duration) *AttachMonitor {
	m := &AttachMonitor{
		notifier:    notifier,
		statusDir:   statusDir,
		interval:    pollInterval,
		states:      make(map[string]string),
		agentStates: make(map[string]map[string]string),
	}
	m.notifyFn = m.notifyStatusChange
	return m
}

// Start launches the background polling loop.
func (m *AttachMonitor) Start(ctx context.Context, initialStates map[string]string, initialAgentStates map[string]map[string]string) {
	if m == nil {
		return
	}

	m.mu.Lock()
	m.states = copyStates(initialStates)
	m.agentStates = copyAgentStates(initialAgentStates)
	skipInitialPoll := len(m.states) == 0 && len(m.agentStates) == 0
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
				currentAgentStates := make(map[string]map[string]string)
				for _, s := range currentSessions {
					currentStates[s.TmuxSession] = s.Status
					if len(s.Agents) == 0 {
						continue
					}

					agentStates := make(map[string]string, len(s.Agents))
					for agentType, agent := range s.Agents {
						agentStates[agentType] = agent.Status
					}
					currentAgentStates[s.TmuxSession] = agentStates
				}

				m.mu.Lock()
				if skipInitialPoll {
					m.states = currentStates
					m.agentStates = currentAgentStates
					skipInitialPoll = false
					m.mu.Unlock()
					continue
				}

				for sessionName, newStatus := range currentStates {
					if oldStatus, ok := m.states[sessionName]; ok && oldStatus != newStatus {
						m.notifyFn(sessionName, newStatus)
					}
				}

				for sessionName, agentStates := range currentAgentStates {
					lastSessionAgentStates, ok := m.agentStates[sessionName]
					if !ok {
						continue
					}

					for agentType, newStatus := range agentStates {
						oldStatus, ok := lastSessionAgentStates[agentType]
						if !ok {
							continue
						}
						if oldStatus != newStatus {
							m.notifyFn(sessionName+":"+agentType, newStatus)
						}
					}
				}

				m.states = currentStates
				m.agentStates = currentAgentStates
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

// AgentStates returns a thread-safe deep copy of the monitor agent states.
func (m *AttachMonitor) AgentStates() map[string]map[string]string {
	if m == nil {
		return map[string]map[string]string{}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return copyAgentStates(m.agentStates)
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

func copyAgentStates(src map[string]map[string]string) map[string]map[string]string {
	dst := make(map[string]map[string]string, len(src))
	for sessionName, states := range src {
		dst[sessionName] = copyStates(states)
	}
	return dst
}
