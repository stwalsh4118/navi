package metrics

import "fmt"

// Constants
const (
	// RecentToolsMax is the maximum number of recent tools to track
	RecentToolsMax = 10

	// TokenThresholdWarning is the token count threshold for warning display
	TokenThresholdWarning = 100000

	// TokenThresholdCritical is the token count threshold for critical display
	TokenThresholdCritical = 500000
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

// ResourceMetrics tracks resource usage for a session.
type ResourceMetrics struct {
	RSSBytes int64 `json:"rss_bytes"`
}

// Metrics aggregates all session metrics data.
type Metrics struct {
	Tokens   *TokenMetrics    `json:"tokens,omitempty"`
	Time     *TimeMetrics     `json:"time,omitempty"`
	Tools    *ToolMetrics     `json:"tools,omitempty"`
	Resource *ResourceMetrics `json:"resource,omitempty"`
}

// FormatTokenCount returns an abbreviated token count string.
// Examples: "0", "500", "1.2k", "45k", "1.5M"
func FormatTokenCount(tokens int64) string {
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

// FormatDuration returns an abbreviated duration string.
// Examples: "0s", "45s", "5m", "1h 23m", "2h"
func FormatDuration(seconds int64) string {
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

// FormatToolCount returns a count of total tool uses.
func FormatToolCount(tools *ToolMetrics) int {
	if tools == nil || tools.Counts == nil {
		return 0
	}
	total := 0
	for _, count := range tools.Counts {
		total += count
	}
	return total
}

// Byte unit thresholds for FormatBytes.
const (
	bytesPerKiB = 1024
	bytesPerMiB = 1024 * 1024
	bytesPerGiB = 1024 * 1024 * 1024
)

// FormatBytes returns a human-readable byte size string.
// Examples: "0", "512K", "256M", "1.2G"
func FormatBytes(bytes int64) string {
	if bytes < bytesPerKiB {
		return "0"
	}
	if bytes < bytesPerMiB {
		return fmt.Sprintf("%dK", bytes/bytesPerKiB)
	}
	if bytes < bytesPerGiB {
		return fmt.Sprintf("%dM", bytes/bytesPerMiB)
	}
	return fmt.Sprintf("%.1fG", float64(bytes)/bytesPerGiB)
}
