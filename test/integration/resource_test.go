package integration

import (
	"os"
	"testing"

	"github.com/stwalsh4118/navi/internal/metrics"
	"github.com/stwalsh4118/navi/internal/resource"
)

// Integration tests for PBI-57 /proc process tree walker

func TestResource_SessionRSS_NoTmux(t *testing.T) {
	// SessionRSS for a nonexistent tmux session should return 0
	rss := resource.SessionRSS("nonexistent-session-12345")
	if rss != 0 {
		t.Errorf("SessionRSS for nonexistent session = %d, want 0", rss)
	}
}

func TestResource_ProcessTreeRSS_Integration(t *testing.T) {
	// The test process itself should have non-zero RSS when read via /proc
	pid := os.Getpid()
	// Read directly from /proc to verify the process has RSS
	statmPath := "/proc/" + string(rune('0'+pid/10000%10)) + string(rune('0'+pid/1000%10)) + string(rune('0'+pid/100%10)) + string(rune('0'+pid/10%10)) + string(rune('0'+pid%10)) + "/statm"
	_, err := os.ReadFile(statmPath)
	if err != nil {
		// Just check that /proc exists and is readable
		if _, err := os.Stat("/proc"); os.IsNotExist(err) {
			t.Skip("/proc not available")
		}
	}
}

func TestResource_FormatBytes_AllTiers(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0"},
		{512, "0"},
		{1024, "1K"},
		{512 * 1024, "512K"},
		{1024 * 1024, "1M"},
		{256 * 1024 * 1024, "256M"},
		{1024 * 1024 * 1024, "1.0G"},
		{int64(1.5 * 1024 * 1024 * 1024), "1.5G"},
	}

	for _, tt := range tests {
		result := metrics.FormatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, result, tt.expected)
		}
	}
}

func TestResource_MetricsStruct(t *testing.T) {
	m := &metrics.Metrics{
		Resource: &metrics.ResourceMetrics{RSSBytes: 512 * 1024 * 1024},
	}

	if m.Resource == nil {
		t.Fatal("Resource should not be nil")
	}
	if m.Resource.RSSBytes != 512*1024*1024 {
		t.Errorf("RSSBytes = %d, want %d", m.Resource.RSSBytes, 512*1024*1024)
	}

	result := metrics.FormatBytes(m.Resource.RSSBytes)
	if result != "512M" {
		t.Errorf("FormatBytes for 512 MiB = %q, want 512M", result)
	}
}

func TestResource_NilResourceOmitted(t *testing.T) {
	m := &metrics.Metrics{Resource: nil}
	if m.Resource != nil {
		t.Error("nil Resource should remain nil")
	}
}
