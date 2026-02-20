package metrics

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
	if RecentToolsMax <= 0 {
		t.Errorf("RecentToolsMax should be positive, got %d", RecentToolsMax)
	}

	if TokenThresholdWarning <= 0 {
		t.Errorf("TokenThresholdWarning should be positive, got %d", TokenThresholdWarning)
	}

	if TokenThresholdCritical <= TokenThresholdWarning {
		t.Errorf("TokenThresholdCritical (%d) should be greater than TokenThresholdWarning (%d)",
			TokenThresholdCritical, TokenThresholdWarning)
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
		result := FormatTokenCount(tt.tokens)
		if result != tt.expected {
			t.Errorf("FormatTokenCount(%d) = %q, want %q", tt.tokens, result, tt.expected)
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
		result := FormatDuration(tt.seconds)
		if result != tt.expected {
			t.Errorf("FormatDuration(%d) = %q, want %q", tt.seconds, result, tt.expected)
		}
	}
}

func TestFormatToolCount(t *testing.T) {
	t.Run("nil tools returns 0", func(t *testing.T) {
		result := FormatToolCount(nil)
		if result != 0 {
			t.Errorf("FormatToolCount(nil) = %d, want 0", result)
		}
	})

	t.Run("empty counts returns 0", func(t *testing.T) {
		result := FormatToolCount(&ToolMetrics{Counts: map[string]int{}})
		if result != 0 {
			t.Errorf("FormatToolCount(empty) = %d, want 0", result)
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
		result := FormatToolCount(tools)
		expected := 45 + 12 + 8 + 3
		if result != expected {
			t.Errorf("FormatToolCount = %d, want %d", result, expected)
		}
	})
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0"},
		{512, "0"},
		{1023, "0"},
		{1024, "1K"},
		{512 * 1024, "512K"},
		{1024*1024 - 1, "1023K"},
		{1024 * 1024, "1M"},
		{256 * 1024 * 1024, "256M"},
		{999 * 1024 * 1024, "999M"},
		{1024 * 1024 * 1024, "1.0G"},
		{int64(1.5 * 1024 * 1024 * 1024), "1.5G"},
		{10 * 1024 * 1024 * 1024, "10.0G"},
	}

	for _, tt := range tests {
		result := FormatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, result, tt.expected)
		}
	}
}

func TestResourceMetricsJSON(t *testing.T) {
	rm := ResourceMetrics{RSSBytes: 268435456}
	data, err := json.Marshal(rm)
	if err != nil {
		t.Fatalf("failed to marshal ResourceMetrics: %v", err)
	}

	expected := `{"rss_bytes":268435456}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}

	var unmarshaled ResourceMetrics
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal ResourceMetrics: %v", err)
	}
	if unmarshaled.RSSBytes != rm.RSSBytes {
		t.Errorf("RSSBytes mismatch: got %d, want %d", unmarshaled.RSSBytes, rm.RSSBytes)
	}
}

func TestMetricsResourceOmitEmpty(t *testing.T) {
	m := Metrics{Resource: nil}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	if string(data) != `{}` {
		t.Errorf("expected {}, got %s", string(data))
	}

	m = Metrics{Resource: &ResourceMetrics{RSSBytes: 1024}}
	data, err = json.Marshal(m)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	var unmarshaled Metrics
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if unmarshaled.Resource == nil {
		t.Error("expected Resource to be non-nil")
	}
	if unmarshaled.Resource.RSSBytes != 1024 {
		t.Errorf("RSSBytes mismatch: got %d, want 1024", unmarshaled.Resource.RSSBytes)
	}
}
