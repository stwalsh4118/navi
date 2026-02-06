package session

import (
	"sort"
	"time"

	"github.com/stwalsh4118/navi/internal/git"
	"github.com/stwalsh4118/navi/internal/metrics"
)

// Status constants
const (
	StatusWaiting    = "waiting"
	StatusPermission = "permission"
	StatusWorking    = "working"
)

// Polling constants
const (
	PollInterval      = 500 * time.Millisecond
	DefaultStatusDir  = "~/.claude-sessions"
)

// StatusDir is the directory where session status files are stored.
// Tests can override this to use a temporary directory.
var StatusDir = DefaultStatusDir

// Info represents the status data for a single Claude Code session.
type Info struct {
	TmuxSession string           `json:"tmux_session"`
	Status      string           `json:"status"`
	Message     string           `json:"message"`
	CWD         string           `json:"cwd"`
	Timestamp   int64            `json:"timestamp"`
	Git         *git.Info        `json:"git,omitempty"`
	Remote      string           `json:"remote,omitempty"`
	Metrics     *metrics.Metrics `json:"metrics,omitempty"`
}

// FilterMode represents the session filter state.
type FilterMode int

const (
	FilterAll    FilterMode = iota // Show all sessions
	FilterLocal                    // Show only local sessions
	FilterRemote                   // Show only remote sessions
)

// SortSessions sorts sessions with priority statuses (waiting, permission) first,
// then by timestamp descending (most recent first).
func SortSessions(sessions []Info) {
	sort.Slice(sessions, func(i, j int) bool {
		iPriority := sessions[i].Status == StatusWaiting || sessions[i].Status == StatusPermission
		jPriority := sessions[j].Status == StatusWaiting || sessions[j].Status == StatusPermission

		if iPriority != jPriority {
			return iPriority
		}

		return sessions[i].Timestamp > sessions[j].Timestamp
	})
}

// AggregateMetrics calculates combined metrics across all sessions.
// Returns nil if no sessions have metrics data.
func AggregateMetrics(sessions []Info) *metrics.Metrics {
	if len(sessions) == 0 {
		return nil
	}

	aggregate := &metrics.Metrics{
		Tokens: &metrics.TokenMetrics{},
		Time:   &metrics.TimeMetrics{},
		Tools:  &metrics.ToolMetrics{Counts: make(map[string]int)},
	}

	hasData := false

	for _, s := range sessions {
		if s.Metrics == nil {
			continue
		}

		if s.Metrics.Tokens != nil {
			hasData = true
			aggregate.Tokens.Input += s.Metrics.Tokens.Input
			aggregate.Tokens.Output += s.Metrics.Tokens.Output
			aggregate.Tokens.Total += s.Metrics.Tokens.Total
		}

		if s.Metrics.Time != nil {
			hasData = true
			aggregate.Time.TotalSeconds += s.Metrics.Time.TotalSeconds
			aggregate.Time.WorkingSeconds += s.Metrics.Time.WorkingSeconds
			aggregate.Time.WaitingSeconds += s.Metrics.Time.WaitingSeconds
		}

		if s.Metrics.Tools != nil && s.Metrics.Tools.Counts != nil {
			hasData = true
			for tool, count := range s.Metrics.Tools.Counts {
				aggregate.Tools.Counts[tool] += count
			}
		}
	}

	if !hasData {
		return nil
	}

	return aggregate
}
