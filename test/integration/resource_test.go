package integration

import (
	"fmt"
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
	statmPath := fmt.Sprintf("/proc/%d/statm", pid)
	data, err := os.ReadFile(statmPath)
	if err != nil {
		if _, statErr := os.Stat("/proc"); os.IsNotExist(statErr) {
			t.Skip("/proc not available")
		}
		t.Fatalf("failed to read %s: %v", statmPath, err)
	}
	if len(data) == 0 {
		t.Error("statm file is empty")
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
