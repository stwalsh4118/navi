package main

import "fmt"

// Metrics constants
const (
	// metricsRecentToolsMax is the maximum number of recent tools to track
	metricsRecentToolsMax = 10

	// metricsTokenThresholdWarning is the token count threshold for warning display
	metricsTokenThresholdWarning = 100000

	// metricsTokenThresholdCritical is the token count threshold for critical display
	metricsTokenThresholdCritical = 500000
)

// TokenMetrics tracks token usage for a session.
type TokenMetrics struct {
	Input  int64 `json:"input"`
	Output int64 `json:"output"`
	Total  int64 `json:"total"`
}

// TimeMetrics tracks time spent in a session.
type TimeMetrics struct {
	Started        int64 `json:"started"`
	TotalSeconds   int64 `json:"total_seconds"`
	WorkingSeconds int64 `json:"working_seconds"`
	WaitingSeconds int64 `json:"waiting_seconds"`
}

// ToolMetrics tracks tool usage activity in a session.
type ToolMetrics struct {
	Recent []string       `json:"recent"`
	Counts map[string]int `json:"counts"`
}

// Metrics aggregates all session metrics data.
type Metrics struct {
	Tokens *TokenMetrics `json:"tokens,omitempty"`
	Time   *TimeMetrics  `json:"time,omitempty"`
	Tools  *ToolMetrics  `json:"tools,omitempty"`
}

// formatTokenCount returns an abbreviated token count string.
// Examples: "0", "500", "1.2k", "45k", "1.5M"
func formatTokenCount(tokens int64) string {
	if tokens < 1000 {
		return fmt.Sprintf("%d", tokens)
	}
	if tokens < 10000 {
		// Show one decimal for 1k-9.9k
		return fmt.Sprintf("%.1fk", float64(tokens)/1000)
	}
	if tokens < 1000000 {
		// Show whole number for 10k-999k
		return fmt.Sprintf("%dk", tokens/1000)
	}
	// Show one decimal for millions
	return fmt.Sprintf("%.1fM", float64(tokens)/1000000)
}

// formatDuration returns an abbreviated duration string.
// Examples: "0s", "45s", "5m", "1h 23m", "2h"
func formatDuration(seconds int64) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm", seconds/60)
	}
	hours := seconds / 3600
	mins := (seconds % 3600) / 60
	if mins > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dh", hours)
}

// formatToolCount returns a count of total tool uses.
func formatToolCount(tools *ToolMetrics) int {
	if tools == nil || tools.Counts == nil {
		return 0
	}
	total := 0
	for _, count := range tools.Counts {
		total += count
	}
	return total
}

// AggregateMetrics calculates combined metrics across all sessions.
// Returns nil if no sessions have metrics data.
func AggregateMetrics(sessions []SessionInfo) *Metrics {
	if len(sessions) == 0 {
		return nil
	}

	aggregate := &Metrics{
		Tokens: &TokenMetrics{},
		Time:   &TimeMetrics{},
		Tools:  &ToolMetrics{Counts: make(map[string]int)},
	}

	hasData := false

	for _, session := range sessions {
		if session.Metrics == nil {
			continue
		}

		// Aggregate token metrics
		if session.Metrics.Tokens != nil {
			hasData = true
			aggregate.Tokens.Input += session.Metrics.Tokens.Input
			aggregate.Tokens.Output += session.Metrics.Tokens.Output
			aggregate.Tokens.Total += session.Metrics.Tokens.Total
		}

		// Aggregate time metrics
		if session.Metrics.Time != nil {
			hasData = true
			aggregate.Time.TotalSeconds += session.Metrics.Time.TotalSeconds
			aggregate.Time.WorkingSeconds += session.Metrics.Time.WorkingSeconds
			aggregate.Time.WaitingSeconds += session.Metrics.Time.WaitingSeconds
		}

		// Aggregate tool metrics
		if session.Metrics.Tools != nil && session.Metrics.Tools.Counts != nil {
			hasData = true
			for tool, count := range session.Metrics.Tools.Counts {
				aggregate.Tools.Counts[tool] += count
			}
		}
	}

	if !hasData {
		return nil
	}

	return aggregate
}
