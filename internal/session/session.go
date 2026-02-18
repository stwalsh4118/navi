package session

import (
	"sort"
	"time"

	"github.com/stwalsh4118/navi/internal/git"
	"github.com/stwalsh4118/navi/internal/metrics"
)

const (
	sortTierPriority = 0
	sortTierActive   = 1
	sortTierDefault  = 2
)

// Status constants
const (
	StatusWaiting    = "waiting"
	StatusPermission = "permission"
	StatusWorking    = "working"
	StatusError      = "error"
	StatusIdle       = "idle"
	StatusStopped    = "stopped"
	StatusDone       = "done"
)

// StatusPriority defines status precedence from highest to lowest.
var StatusPriority = []string{
	StatusPermission,
	StatusWaiting,
	StatusWorking,
	StatusError,
	StatusIdle,
	StatusStopped,
	StatusDone,
}

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
	TmuxSession     string                   `json:"tmux_session"`
	Status          string                   `json:"status"`
	Message         string                   `json:"message"`
	CWD             string                   `json:"cwd"`
	CurrentPBI      string                   `json:"current_pbi,omitempty"`
	CurrentPBITitle string                   `json:"current_pbi_title,omitempty"`
	Timestamp       int64                    `json:"timestamp"`
	Git             *git.Info                `json:"git,omitempty"`
	Remote          string                   `json:"remote,omitempty"`
	Metrics         *metrics.Metrics         `json:"metrics,omitempty"`
	Team            *TeamInfo                `json:"team,omitempty"`
	Agents          map[string]ExternalAgent `json:"agents,omitempty"`
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

// HasPriorityExternalAgent returns true if any external agent in the session
// has a priority status (waiting or permission).
func HasPriorityExternalAgent(s Info) bool {
	if len(s.Agents) == 0 {
		return false
	}
	for _, agent := range s.Agents {
		if agent.Status == StatusWaiting || agent.Status == StatusPermission {
			return true
		}
	}
	return false
}

func statusRank(status string) int {
	for i, candidate := range StatusPriority {
		if status == candidate {
			return i
		}
	}

	return len(StatusPriority)
}

// CompositeStatus returns the highest-priority status across Claude Code and external agents.
// Source is empty when Claude Code provides the winning status.
func CompositeStatus(s Info) (status string, source string) {
	if len(s.Agents) == 0 {
		return s.Status, ""
	}

	bestStatus := s.Status
	bestSource := ""
	bestRank := statusRank(bestStatus)

	for agentType, agent := range s.Agents {
		rank := statusRank(agent.Status)
		if rank < bestRank {
			bestStatus = agent.Status
			bestSource = agentType
			bestRank = rank
			continue
		}

		if rank == bestRank && bestSource != "" && agentType < bestSource {
			bestStatus = agent.Status
			bestSource = agentType
		}
	}

	return bestStatus, bestSource
}

func sessionSortTier(s Info) int {
	compositeStatus, _ := CompositeStatus(s)
	compositeRank := statusRank(compositeStatus)

	if HasPriorityTeammate(s) || compositeRank <= statusRank(StatusWaiting) {
		return sortTierPriority
	}
	if compositeRank == statusRank(StatusWorking) {
		return sortTierActive
	}
	return sortTierDefault
}

// SortSessions sorts sessions with priority statuses (waiting, permission) first,
// then by timestamp descending (most recent first).
func SortSessions(sessions []Info) {
	sort.Slice(sessions, func(i, j int) bool {
		iTier := sessionSortTier(sessions[i])
		jTier := sessionSortTier(sessions[j])

		if iTier != jTier {
			return iTier < jTier
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
