package main

import (
	"encoding/json"
	"testing"
)

func TestTokenMetricsJSON(t *testing.T) {
	tm := TokenMetrics{
		Input:  45000,
		Output: 12000,
		Total:  57000,
	}

	data, err := json.Marshal(tm)
	if err != nil {
		t.Fatalf("failed to marshal TokenMetrics: %v", err)
	}

	expected := `{"input":45000,"output":12000,"total":57000}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}

	var unmarshaled TokenMetrics
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal TokenMetrics: %v", err)
	}

	if unmarshaled.Input != tm.Input || unmarshaled.Output != tm.Output || unmarshaled.Total != tm.Total {
		t.Errorf("unmarshaled values don't match: got %+v, want %+v", unmarshaled, tm)
	}
}

func TestTimeMetricsJSON(t *testing.T) {
	tm := TimeMetrics{
		Started:        1738620000,
		TotalSeconds:   7200,
		WorkingSeconds: 3600,
		WaitingSeconds: 1800,
	}

	data, err := json.Marshal(tm)
	if err != nil {
		t.Fatalf("failed to marshal TimeMetrics: %v", err)
	}

	expected := `{"started":1738620000,"total_seconds":7200,"working_seconds":3600,"waiting_seconds":1800}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}

	var unmarshaled TimeMetrics
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal TimeMetrics: %v", err)
	}

	if unmarshaled != tm {
		t.Errorf("unmarshaled values don't match: got %+v, want %+v", unmarshaled, tm)
	}
}

func TestToolMetricsJSON(t *testing.T) {
	tm := ToolMetrics{
		Recent: []string{"Read", "Edit", "Bash"},
		Counts: map[string]int{
			"Read":  45,
			"Edit":  12,
			"Bash":  8,
			"Write": 3,
		},
	}

	data, err := json.Marshal(tm)
	if err != nil {
		t.Fatalf("failed to marshal ToolMetrics: %v", err)
	}

	var unmarshaled ToolMetrics
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal ToolMetrics: %v", err)
	}

	if len(unmarshaled.Recent) != len(tm.Recent) {
		t.Errorf("recent tools count mismatch: got %d, want %d", len(unmarshaled.Recent), len(tm.Recent))
	}

	for i, tool := range tm.Recent {
		if unmarshaled.Recent[i] != tool {
			t.Errorf("recent tool %d mismatch: got %s, want %s", i, unmarshaled.Recent[i], tool)
		}
	}

	for tool, count := range tm.Counts {
		if unmarshaled.Counts[tool] != count {
			t.Errorf("tool count for %s mismatch: got %d, want %d", tool, unmarshaled.Counts[tool], count)
		}
	}
}

func TestMetricsJSONWithAllFields(t *testing.T) {
	m := Metrics{
		Tokens: &TokenMetrics{
			Input:  45000,
			Output: 12000,
			Total:  57000,
		},
		Time: &TimeMetrics{
			Started:        1738620000,
			TotalSeconds:   7200,
			WorkingSeconds: 3600,
			WaitingSeconds: 1800,
		},
		Tools: &ToolMetrics{
			Recent: []string{"Read", "Edit", "Bash"},
			Counts: map[string]int{"Read": 45, "Edit": 12},
		},
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("failed to marshal Metrics: %v", err)
	}

	var unmarshaled Metrics
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal Metrics: %v", err)
	}

	if unmarshaled.Tokens == nil {
		t.Error("expected Tokens to be non-nil")
	}
	if unmarshaled.Time == nil {
		t.Error("expected Time to be non-nil")
	}
	if unmarshaled.Tools == nil {
		t.Error("expected Tools to be non-nil")
	}

	if unmarshaled.Tokens.Total != m.Tokens.Total {
		t.Errorf("token total mismatch: got %d, want %d", unmarshaled.Tokens.Total, m.Tokens.Total)
	}
}

func TestMetricsJSONOmitEmpty(t *testing.T) {
	// Test that nil sub-structs are omitted from JSON
	m := Metrics{
		Tokens: nil,
		Time:   nil,
		Tools:  nil,
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("failed to marshal empty Metrics: %v", err)
	}

	expected := `{}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}

	// Test partial: only tokens
	m = Metrics{
		Tokens: &TokenMetrics{Input: 100, Output: 50, Total: 150},
	}

	data, err = json.Marshal(m)
	if err != nil {
		t.Fatalf("failed to marshal partial Metrics: %v", err)
	}

	var unmarshaled Metrics
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal partial Metrics: %v", err)
	}

	if unmarshaled.Tokens == nil {
		t.Error("expected Tokens to be non-nil")
	}
	if unmarshaled.Time != nil {
		t.Error("expected Time to be nil")
	}
	if unmarshaled.Tools != nil {
		t.Error("expected Tools to be nil")
	}
}

func TestMetricsConstants(t *testing.T) {
	// Verify constants are defined and have reasonable values
	if metricsRecentToolsMax <= 0 {
		t.Errorf("metricsRecentToolsMax should be positive, got %d", metricsRecentToolsMax)
	}

	if metricsTokenThresholdWarning <= 0 {
		t.Errorf("metricsTokenThresholdWarning should be positive, got %d", metricsTokenThresholdWarning)
	}

	if metricsTokenThresholdCritical <= metricsTokenThresholdWarning {
		t.Errorf("metricsTokenThresholdCritical (%d) should be greater than metricsTokenThresholdWarning (%d)",
			metricsTokenThresholdCritical, metricsTokenThresholdWarning)
	}
}

func TestFormatTokenCount(t *testing.T) {
	tests := []struct {
		tokens   int64
		expected string
	}{
		{0, "0"},
		{100, "100"},
		{999, "999"},
		{1000, "1.0k"},
		{1500, "1.5k"},
		{9999, "10.0k"},
		{10000, "10k"},
		{45000, "45k"},
		{100000, "100k"},
		{999999, "999k"},
		{1000000, "1.0M"},
		{1500000, "1.5M"},
		{10000000, "10.0M"},
	}

	for _, tt := range tests {
		result := formatTokenCount(tt.tokens)
		if result != tt.expected {
			t.Errorf("formatTokenCount(%d) = %q, want %q", tt.tokens, result, tt.expected)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds  int64
		expected string
	}{
		{0, "0s"},
		{30, "30s"},
		{59, "59s"},
		{60, "1m"},
		{90, "1m"},
		{300, "5m"},
		{3599, "59m"},
		{3600, "1h"},
		{4980, "1h 23m"},
		{7200, "2h"},
		{7260, "2h 1m"},
		{86400, "24h"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.seconds)
		if result != tt.expected {
			t.Errorf("formatDuration(%d) = %q, want %q", tt.seconds, result, tt.expected)
		}
	}
}

func TestFormatToolCount(t *testing.T) {
	t.Run("nil tools returns 0", func(t *testing.T) {
		result := formatToolCount(nil)
		if result != 0 {
			t.Errorf("formatToolCount(nil) = %d, want 0", result)
		}
	})

	t.Run("empty counts returns 0", func(t *testing.T) {
		result := formatToolCount(&ToolMetrics{Counts: map[string]int{}})
		if result != 0 {
			t.Errorf("formatToolCount(empty) = %d, want 0", result)
		}
	})

	t.Run("counts are summed", func(t *testing.T) {
		tools := &ToolMetrics{
			Counts: map[string]int{
				"Read":  45,
				"Edit":  12,
				"Bash":  8,
				"Write": 3,
			},
		}
		result := formatToolCount(tools)
		expected := 45 + 12 + 8 + 3
		if result != expected {
			t.Errorf("formatToolCount = %d, want %d", result, expected)
		}
	})
}

func TestAggregateMetrics(t *testing.T) {
	t.Run("empty sessions returns nil", func(t *testing.T) {
		result := AggregateMetrics([]SessionInfo{})
		if result != nil {
			t.Errorf("AggregateMetrics([]) should return nil, got %+v", result)
		}
	})

	t.Run("sessions without metrics returns nil", func(t *testing.T) {
		sessions := []SessionInfo{
			{TmuxSession: "test1", Metrics: nil},
			{TmuxSession: "test2", Metrics: nil},
		}
		result := AggregateMetrics(sessions)
		if result != nil {
			t.Errorf("AggregateMetrics with no metrics should return nil, got %+v", result)
		}
	})

	t.Run("aggregates time metrics correctly", func(t *testing.T) {
		sessions := []SessionInfo{
			{
				TmuxSession: "test1",
				Metrics: &Metrics{
					Time: &TimeMetrics{
						TotalSeconds:   3600,
						WorkingSeconds: 2400,
						WaitingSeconds: 1200,
					},
				},
			},
			{
				TmuxSession: "test2",
				Metrics: &Metrics{
					Time: &TimeMetrics{
						TotalSeconds:   1800,
						WorkingSeconds: 1200,
						WaitingSeconds: 600,
					},
				},
			},
		}
		result := AggregateMetrics(sessions)
		if result == nil {
			t.Fatal("AggregateMetrics should not return nil")
		}
		if result.Time == nil {
			t.Fatal("Aggregate Time should not be nil")
		}
		if result.Time.TotalSeconds != 5400 {
			t.Errorf("TotalSeconds = %d, want 5400", result.Time.TotalSeconds)
		}
		if result.Time.WorkingSeconds != 3600 {
			t.Errorf("WorkingSeconds = %d, want 3600", result.Time.WorkingSeconds)
		}
		if result.Time.WaitingSeconds != 1800 {
			t.Errorf("WaitingSeconds = %d, want 1800", result.Time.WaitingSeconds)
		}
	})

	t.Run("aggregates tool counts correctly", func(t *testing.T) {
		sessions := []SessionInfo{
			{
				TmuxSession: "test1",
				Metrics: &Metrics{
					Tools: &ToolMetrics{
						Counts: map[string]int{
							"Read": 10,
							"Edit": 5,
						},
					},
				},
			},
			{
				TmuxSession: "test2",
				Metrics: &Metrics{
					Tools: &ToolMetrics{
						Counts: map[string]int{
							"Read": 8,
							"Bash": 3,
						},
					},
				},
			},
		}
		result := AggregateMetrics(sessions)
		if result == nil {
			t.Fatal("AggregateMetrics should not return nil")
		}
		if result.Tools == nil {
			t.Fatal("Aggregate Tools should not be nil")
		}
		if result.Tools.Counts["Read"] != 18 {
			t.Errorf("Read count = %d, want 18", result.Tools.Counts["Read"])
		}
		if result.Tools.Counts["Edit"] != 5 {
			t.Errorf("Edit count = %d, want 5", result.Tools.Counts["Edit"])
		}
		if result.Tools.Counts["Bash"] != 3 {
			t.Errorf("Bash count = %d, want 3", result.Tools.Counts["Bash"])
		}
	})

	t.Run("handles mixed sessions with and without metrics", func(t *testing.T) {
		sessions := []SessionInfo{
			{TmuxSession: "test1", Metrics: nil},
			{
				TmuxSession: "test2",
				Metrics: &Metrics{
					Time: &TimeMetrics{
						TotalSeconds:   3600,
						WorkingSeconds: 2400,
						WaitingSeconds: 1200,
					},
					Tools: &ToolMetrics{
						Counts: map[string]int{"Read": 5},
					},
				},
			},
			{TmuxSession: "test3", Metrics: nil},
		}
		result := AggregateMetrics(sessions)
		if result == nil {
			t.Fatal("AggregateMetrics should not return nil")
		}
		if result.Time.TotalSeconds != 3600 {
			t.Errorf("TotalSeconds = %d, want 3600", result.Time.TotalSeconds)
		}
		if result.Tools.Counts["Read"] != 5 {
			t.Errorf("Read count = %d, want 5", result.Tools.Counts["Read"])
		}
	})

	t.Run("aggregates token metrics correctly", func(t *testing.T) {
		sessions := []SessionInfo{
			{
				TmuxSession: "test1",
				Metrics: &Metrics{
					Tokens: &TokenMetrics{
						Input:  10000,
						Output: 5000,
						Total:  15000,
					},
				},
			},
			{
				TmuxSession: "test2",
				Metrics: &Metrics{
					Tokens: &TokenMetrics{
						Input:  20000,
						Output: 8000,
						Total:  28000,
					},
				},
			},
		}
		result := AggregateMetrics(sessions)
		if result == nil {
			t.Fatal("AggregateMetrics should not return nil")
		}
		if result.Tokens == nil {
			t.Fatal("Aggregate Tokens should not be nil")
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
}
