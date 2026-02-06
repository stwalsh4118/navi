package session

import (
	"testing"

	"github.com/stwalsh4118/navi/internal/metrics"
)

func TestSortSessions(t *testing.T) {
	sessions := []Info{
		{TmuxSession: "working", Status: StatusWorking, Timestamp: 100},
		{TmuxSession: "waiting", Status: StatusWaiting, Timestamp: 50},
		{TmuxSession: "permission", Status: StatusPermission, Timestamp: 75},
		{TmuxSession: "old-working", Status: StatusWorking, Timestamp: 25},
	}

	SortSessions(sessions)

	// Priority sessions first (waiting, permission)
	if sessions[0].Status != StatusWaiting && sessions[0].Status != StatusPermission {
		t.Errorf("Expected priority session first, got %q", sessions[0].Status)
	}

	// Non-priority sorted by timestamp descending
	lastNonPriority := sessions[len(sessions)-1]
	if lastNonPriority.Timestamp != 25 {
		t.Errorf("Expected oldest non-priority last, got timestamp %d", lastNonPriority.Timestamp)
	}
}

func TestAggregateMetrics(t *testing.T) {
	t.Run("empty sessions returns nil", func(t *testing.T) {
		result := AggregateMetrics([]Info{})
		if result != nil {
			t.Errorf("AggregateMetrics([]) should return nil, got %+v", result)
		}
	})

	t.Run("sessions without metrics returns nil", func(t *testing.T) {
		sessions := []Info{
			{TmuxSession: "test1", Metrics: nil},
			{TmuxSession: "test2", Metrics: nil},
		}
		result := AggregateMetrics(sessions)
		if result != nil {
			t.Errorf("AggregateMetrics with no metrics should return nil, got %+v", result)
		}
	})

	t.Run("aggregates token metrics correctly", func(t *testing.T) {
		sessions := []Info{
			{
				TmuxSession: "test1",
				Metrics: &metrics.Metrics{
					Tokens: &metrics.TokenMetrics{Input: 10000, Output: 5000, Total: 15000},
				},
			},
			{
				TmuxSession: "test2",
				Metrics: &metrics.Metrics{
					Tokens: &metrics.TokenMetrics{Input: 20000, Output: 8000, Total: 28000},
				},
			},
		}
		result := AggregateMetrics(sessions)
		if result == nil {
			t.Fatal("AggregateMetrics should not return nil")
		}
		if result.Tokens.Input != 30000 {
			t.Errorf("Input = %d, want 30000", result.Tokens.Input)
		}
		if result.Tokens.Output != 13000 {
			t.Errorf("Output = %d, want 13000", result.Tokens.Output)
		}
		if result.Tokens.Total != 43000 {
			t.Errorf("Total = %d, want 43000", result.Tokens.Total)
		}
	})

	t.Run("aggregates time metrics correctly", func(t *testing.T) {
		sessions := []Info{
			{
				TmuxSession: "test1",
				Metrics: &metrics.Metrics{
					Time: &metrics.TimeMetrics{TotalSeconds: 3600, WorkingSeconds: 2400, WaitingSeconds: 1200},
				},
			},
			{
				TmuxSession: "test2",
				Metrics: &metrics.Metrics{
					Time: &metrics.TimeMetrics{TotalSeconds: 1800, WorkingSeconds: 1200, WaitingSeconds: 600},
				},
			},
		}
		result := AggregateMetrics(sessions)
		if result == nil {
			t.Fatal("AggregateMetrics should not return nil")
		}
		if result.Time.TotalSeconds != 5400 {
			t.Errorf("TotalSeconds = %d, want 5400", result.Time.TotalSeconds)
		}
	})

	t.Run("aggregates tool counts correctly", func(t *testing.T) {
		sessions := []Info{
			{
				TmuxSession: "test1",
				Metrics: &metrics.Metrics{
					Tools: &metrics.ToolMetrics{Counts: map[string]int{"Read": 10, "Edit": 5}},
				},
			},
			{
				TmuxSession: "test2",
				Metrics: &metrics.Metrics{
					Tools: &metrics.ToolMetrics{Counts: map[string]int{"Read": 8, "Bash": 3}},
				},
			},
		}
		result := AggregateMetrics(sessions)
		if result == nil {
			t.Fatal("AggregateMetrics should not return nil")
		}
		if result.Tools.Counts["Read"] != 18 {
			t.Errorf("Read count = %d, want 18", result.Tools.Counts["Read"])
		}
	})
}
