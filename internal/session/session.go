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
	StatusIdle       = "idle"
	StatusStopped    = "stopped"
)

// Polling constants
const (
	PollInterval     = 500 * time.Millisecond
	DefaultStatusDir = "~/.claude-sessions"
)

// StatusDir is the directory where session status files are stored.
// Tests can override this to use a temporary directory.
var StatusDir = DefaultStatusDir

// AgentInfo represents the status of a single agent in a team.
type AgentInfo struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
}

// TeamInfo represents an active agent team within a session.
type TeamInfo struct {
	Name   string      `json:"name"`
	Agents []AgentInfo `json:"agents"`
}

// ExternalAgent represents a non-Claude-Code agent status entry.
type ExternalAgent struct {
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
}

// Info represents the status data for a single Claude Code session.
type Info struct {
	TmuxSession string                   `json:"tmux_session"`
	Status      string                   `json:"status"`
	Message     string                   `json:"message"`
	CWD         string                   `json:"cwd"`
	Timestamp   int64                    `json:"timestamp"`
	Git         *git.Info                `json:"git,omitempty"`
	Remote      string                   `json:"remote,omitempty"`
	Metrics     *metrics.Metrics         `json:"metrics,omitempty"`
	Team        *TeamInfo                `json:"team,omitempty"`
	Agents      map[string]ExternalAgent `json:"agents,omitempty"`
}

// FilterMode represents the session filter state.
type FilterMode int

const (
	FilterAll    FilterMode = iota // Show all sessions
	FilterLocal                    // Show only local sessions
	FilterRemote                   // Show only remote sessions
)

// HasPriorityTeammate returns true if any agent in the session's team
// has a priority status (waiting or permission).
func HasPriorityTeammate(s Info) bool {
	if s.Team == nil || len(s.Team.Agents) == 0 {
		return false
	}
	for _, agent := range s.Team.Agents {
		if agent.Status == StatusWaiting || agent.Status == StatusPermission {
			return true
		}
	}
	return false
}

// SortSessions sorts sessions with priority statuses (waiting, permission) first,
// then by timestamp descending (most recent first).
func SortSessions(sessions []Info) {
	sort.Slice(sessions, func(i, j int) bool {
		iPriority := sessions[i].Status == StatusWaiting || sessions[i].Status == StatusPermission || HasPriorityTeammate(sessions[i])
		jPriority := sessions[j].Status == StatusWaiting || sessions[j].Status == StatusPermission || HasPriorityTeammate(sessions[j])

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
