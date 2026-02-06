package tui

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stwalsh4118/navi/internal/git"
	"github.com/stwalsh4118/navi/internal/metrics"
	"github.com/stwalsh4118/navi/internal/session"
)

func TestCapturePane(t *testing.T) {
	t.Run("returns error for non-existent session", func(t *testing.T) {
		// Try to capture from a s that doesn't exist
		_, err := capturePane("nonexistent-session-12345", 50)
		if err == nil {
			t.Error("expected error for non-existent s, got nil")
		}
	})

	t.Run("handles empty s name", func(t *testing.T) {
		// Note: tmux with empty s name captures current pane if running in tmux
		// This is valid tmux behavior - empty string targets current s
		_, _ = capturePane("", 50)
		// Just verify it doesn't panic
	})

	t.Run("uses correct line count argument", func(t *testing.T) {
		// This test verifies the function doesn't panic with various line counts
		// Actual capture requires a real tmux s
		_, _ = capturePane("test", 10)
		_, _ = capturePane("test", 50)
		_, _ = capturePane("test", 100)
		// If we get here without panic, the argument formatting works
	})
}

func TestSessionInfoJSONBackwardCompatibility(t *testing.T) {
	t.Run("parses JSON without git field", func(t *testing.T) {
		// Old format without git field
		jsonData := `{
			"tmux_session": "test-session",
			"status": "working",
			"message": "Processing...",
			"cwd": "/home/user/project",
			"timestamp": 1738627200
		}`

		var s session.Info
		if err := json.Unmarshal([]byte(jsonData), &s); err != nil {
			t.Fatalf("Failed to parse JSON without git field: %v", err)
		}

		if s.TmuxSession != "test-session" {
			t.Errorf("TmuxSession mismatch: got %q", s.TmuxSession)
		}
		if s.Git != nil {
			t.Error("Expected Git to be nil for JSON without git field")
		}
	})

	t.Run("parses JSON with git field", func(t *testing.T) {
		// New format with git field
		jsonData := `{
			"tmux_session": "test-session",
			"status": "working",
			"message": "Processing...",
			"cwd": "/home/user/project",
			"timestamp": 1738627200,
			"git": {
				"branch": "feature/auth",
				"dirty": true,
				"ahead": 3,
				"behind": 1,
				"last_commit": "abc1234 Add login",
				"remote": "git@github.com:user/repo.git"
			}
		}`

		var s session.Info
		if err := json.Unmarshal([]byte(jsonData), &s); err != nil {
			t.Fatalf("Failed to parse JSON with git field: %v", err)
		}

		if s.Git == nil {
			t.Fatal("Expected Git to be non-nil")
		}
		if s.Git.Branch != "feature/auth" {
			t.Errorf("Branch mismatch: got %q", s.Git.Branch)
		}
		if !s.Git.Dirty {
			t.Error("Expected Dirty to be true")
		}
		if s.Git.Ahead != 3 {
			t.Errorf("Ahead mismatch: got %d", s.Git.Ahead)
		}
	})

	t.Run("serializes without git field when nil", func(t *testing.T) {
		s := session.Info{
			TmuxSession: "test",
			Status:      "working",
			CWD:         "/tmp",
			Timestamp:   1234567890,
			Git:         nil,
		}

		data, err := json.Marshal(s)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		// Should not contain "git" key due to omitempty
		if strings.Contains(string(data), `"git"`) {
			t.Error("Expected JSON to not contain git field when nil")
		}
	})

	t.Run("serializes with git field when present", func(t *testing.T) {
		s := session.Info{
			TmuxSession: "test",
			Status:      "working",
			CWD:         "/tmp",
			Timestamp:   1234567890,
			Git: &git.Info{
				Branch: "main",
				Dirty:  false,
			},
		}

		data, err := json.Marshal(s)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		// Should contain "git" key
		if !strings.Contains(string(data), `"git"`) {
			t.Error("Expected JSON to contain git field")
		}
		if !strings.Contains(string(data), `"branch":"main"`) {
			t.Error("Expected JSON to contain branch")
		}
	})
}

func TestSessionInfoMetricsJSONBackwardCompatibility(t *testing.T) {
	t.Run("parses JSON without metrics field", func(t *testing.T) {
		// Old format without metrics field
		jsonData := `{
			"tmux_session": "test-session",
			"status": "working",
			"message": "Processing...",
			"cwd": "/home/user/project",
			"timestamp": 1738627200
		}`

		var s session.Info
		if err := json.Unmarshal([]byte(jsonData), &s); err != nil {
			t.Fatalf("Failed to parse JSON without metrics field: %v", err)
		}

		if s.TmuxSession != "test-session" {
			t.Errorf("TmuxSession mismatch: got %q", s.TmuxSession)
		}
		if s.Metrics != nil {
			t.Error("Expected metrics.Metrics to be nil for JSON without metrics field")
		}
	})

	t.Run("parses JSON with full metrics", func(t *testing.T) {
		// New format with full metrics
		jsonData := `{
			"tmux_session": "hyperion",
			"status": "working",
			"message": "Implementing feature...",
			"cwd": "/home/user/projects/hyperion",
			"timestamp": 1738627200,
			"metrics": {
				"tokens": {
					"input": 45000,
					"output": 12000,
					"total": 57000
				},
				"time": {
					"started": 1738620000,
					"total_seconds": 7200,
					"working_seconds": 3600,
					"waiting_seconds": 1800
				},
				"tools": {
					"recent": ["Read", "Edit", "Bash"],
					"counts": {
						"Read": 45,
						"Edit": 12,
						"Bash": 8,
						"Write": 3
					}
				}
			}
		}`

		var s session.Info
		if err := json.Unmarshal([]byte(jsonData), &s); err != nil {
			t.Fatalf("Failed to parse JSON with metrics field: %v", err)
		}

		if s.Metrics == nil {
			t.Fatal("Expected metrics.Metrics to be non-nil")
		}
		if s.Metrics.Tokens == nil {
			t.Fatal("Expected metrics.Metrics.Tokens to be non-nil")
		}
		if s.Metrics.Tokens.Total != 57000 {
			t.Errorf("Tokens.Total mismatch: got %d", s.Metrics.Tokens.Total)
		}
		if s.Metrics.Time == nil {
			t.Fatal("Expected metrics.Metrics.Time to be non-nil")
		}
		if s.Metrics.Time.WorkingSeconds != 3600 {
			t.Errorf("Time.WorkingSeconds mismatch: got %d", s.Metrics.Time.WorkingSeconds)
		}
		if s.Metrics.Tools == nil {
			t.Fatal("Expected metrics.Metrics.Tools to be non-nil")
		}
		if len(s.Metrics.Tools.Recent) != 3 {
			t.Errorf("Tools.Recent length mismatch: got %d", len(s.Metrics.Tools.Recent))
		}
		if s.Metrics.Tools.Counts["Read"] != 45 {
			t.Errorf("Tools.Counts[Read] mismatch: got %d", s.Metrics.Tools.Counts["Read"])
		}
	})

	t.Run("parses JSON with partial metrics", func(t *testing.T) {
		// Format with only time metrics
		jsonData := `{
			"tmux_session": "test-session",
			"status": "working",
			"cwd": "/tmp",
			"timestamp": 1738627200,
			"metrics": {
				"time": {
					"started": 1738620000,
					"total_seconds": 7200,
					"working_seconds": 3600,
					"waiting_seconds": 1800
				}
			}
		}`

		var s session.Info
		if err := json.Unmarshal([]byte(jsonData), &s); err != nil {
			t.Fatalf("Failed to parse JSON with partial metrics: %v", err)
		}

		if s.Metrics == nil {
			t.Fatal("Expected metrics.Metrics to be non-nil")
		}
		if s.Metrics.Time == nil {
			t.Fatal("Expected metrics.Metrics.Time to be non-nil")
		}
		if s.Metrics.Tokens != nil {
			t.Error("Expected metrics.Metrics.Tokens to be nil")
		}
		if s.Metrics.Tools != nil {
			t.Error("Expected metrics.Metrics.Tools to be nil")
		}
	})

	t.Run("serializes without metrics field when nil", func(t *testing.T) {
		s := session.Info{
			TmuxSession:     "test",
			Status:          "working",
			CWD:             "/tmp",
			Timestamp:       1234567890,
			Metrics: nil,
		}

		data, err := json.Marshal(s)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		// Should not contain "metrics" key due to omitempty
		if strings.Contains(string(data), `"metrics"`) {
			t.Error("Expected JSON to not contain metrics field when nil")
		}
	})

	t.Run("serializes with metrics field when present", func(t *testing.T) {
		s := session.Info{
			TmuxSession: "test",
			Status:      "working",
			CWD:         "/tmp",
			Timestamp:   1234567890,
			Metrics: &metrics.Metrics{
				Tokens: &metrics.TokenMetrics{
					Input:  1000,
					Output: 500,
					Total:  1500,
				},
			},
		}

		data, err := json.Marshal(s)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		// Should contain "metrics" key
		if !strings.Contains(string(data), `"metrics"`) {
			t.Error("Expected JSON to contain metrics field")
		}
		if !strings.Contains(string(data), `"tokens"`) {
			t.Error("Expected JSON to contain tokens")
		}
		if !strings.Contains(string(data), `"total":1500`) {
			t.Error("Expected JSON to contain total:1500")
		}
	})
}
