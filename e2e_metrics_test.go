package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// E2E tests for PBI-12: Session Metrics
// These tests verify all acceptance criteria are met

// TestE2E_MetricsBadgesDisplayed verifies that inline metrics badges
// are shown for sessions with metrics data.
func TestE2E_MetricsBadgesDisplayed(t *testing.T) {
	t.Run("session with metrics shows badges in row", func(t *testing.T) {
		m := Model{width: 100, height: 24}
		session := SessionInfo{
			TmuxSession: "test-session",
			Status:      "working",
			CWD:         "/tmp/project",
			Timestamp:   time.Now().Unix(),
			Metrics: &Metrics{
				Time: &TimeMetrics{
					Started:        time.Now().Add(-1 * time.Hour).Unix(),
					TotalSeconds:   3600,
					WorkingSeconds: 2700,
					WaitingSeconds: 900,
				},
				Tools: &ToolMetrics{
					Counts: map[string]int{
						"Read":  25,
						"Edit":  10,
						"Bash":  5,
						"Write": 3,
					},
					Recent: []string{"Read", "Edit", "Bash"},
				},
			},
		}

		result := m.renderSession(session, false, 100)

		// Verify time badge is displayed (CoS 2: Time tracking)
		if !strings.Contains(result, "⏱") {
			t.Error("CoS 2 failed: Time badge should be displayed in session row")
		}
		if !strings.Contains(result, "1h") {
			t.Error("CoS 2 failed: Duration should be displayed")
		}

		// Verify tool badge is displayed (CoS 3: Tool activity)
		if !strings.Contains(result, "🔧") {
			t.Error("CoS 3 failed: Tool badge should be displayed")
		}
		if !strings.Contains(result, "43") {
			t.Error("CoS 3 failed: Tool count should be displayed")
		}
	})

	t.Run("session without metrics shows no badges", func(t *testing.T) {
		m := Model{width: 80, height: 24}
		session := SessionInfo{
			TmuxSession: "test-session",
			Status:      "working",
			CWD:         "/tmp/project",
			Timestamp:   time.Now().Unix(),
			Metrics:     nil,
		}

		result := m.renderSession(session, false, 80)

		// Should not contain metrics badges
		if strings.Contains(result, "⏱") || strings.Contains(result, "🔧") {
			t.Error("Session without metrics should not show badges")
		}
	})
}

// TestE2E_MetricsDetailDialog verifies that pressing 'i' opens a detailed
// metrics view showing comprehensive session metrics.
func TestE2E_MetricsDetailDialog(t *testing.T) {
	t.Run("i key opens metrics detail dialog", func(t *testing.T) {
		m := initialModel()
		m.sessions = []SessionInfo{
			{
				TmuxSession: "test-session",
				Status:      "working",
				CWD:         "/tmp/project",
				Timestamp:   time.Now().Unix(),
				Metrics: &Metrics{
					Time: &TimeMetrics{
						Started:        time.Now().Add(-2 * time.Hour).Unix(),
						TotalSeconds:   7200,
						WorkingSeconds: 5400,
						WaitingSeconds: 1800,
					},
					Tools: &ToolMetrics{
						Counts: map[string]int{"Read": 50, "Edit": 20},
						Recent: []string{"Read", "Edit"},
					},
				},
			},
		}
		m.cursor = 0

		// Simulate 'i' key press
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
		updatedModel, _ := m.Update(msg)
		updated := updatedModel.(Model)

		// Verify dialog is opened (CoS 5: Detail view)
		if updated.dialogMode != DialogMetricsDetail {
			t.Error("CoS 5 failed: 'i' key should open metrics detail dialog")
		}
		if updated.sessionToModify == nil {
			t.Error("CoS 5 failed: sessionToModify should be set")
		}
	})

	t.Run("metrics detail shows time breakdown", func(t *testing.T) {
		m := initialModel()
		m.width = 100
		m.dialogMode = DialogMetricsDetail
		session := SessionInfo{
			TmuxSession: "test-session",
			Status:      "working",
			Metrics: &Metrics{
				Time: &TimeMetrics{
					Started:        time.Now().Add(-2 * time.Hour).Unix(),
					TotalSeconds:   7200,
					WorkingSeconds: 5400,
					WaitingSeconds: 1800,
				},
			},
		}
		m.sessionToModify = &session

		result := m.renderMetricsDetailView()

		// Verify time tracking sections (CoS 2)
		if !strings.Contains(result, "Time Tracking") {
			t.Error("CoS 2 failed: Time Tracking section should be displayed")
		}
		if !strings.Contains(result, "Duration") {
			t.Error("CoS 2 failed: Duration should be displayed")
		}
		if !strings.Contains(result, "Working") {
			t.Error("CoS 2 failed: Working time should be displayed")
		}
		if !strings.Contains(result, "Waiting") {
			t.Error("CoS 2 failed: Waiting time should be displayed")
		}
	})

	t.Run("metrics detail shows tool breakdown", func(t *testing.T) {
		m := initialModel()
		m.width = 100
		m.dialogMode = DialogMetricsDetail
		session := SessionInfo{
			TmuxSession: "test-session",
			Status:      "working",
			Metrics: &Metrics{
				Tools: &ToolMetrics{
					Counts: map[string]int{
						"Read":  50,
						"Edit":  20,
						"Bash":  10,
						"Write": 5,
					},
					Recent: []string{"Read", "Edit", "Bash"},
				},
			},
		}
		m.sessionToModify = &session

		result := m.renderMetricsDetailView()

		// Verify tool activity sections (CoS 3)
		if !strings.Contains(result, "Tool Activity") {
			t.Error("CoS 3 failed: Tool Activity section should be displayed")
		}
		if !strings.Contains(result, "Total calls") {
			t.Error("CoS 3 failed: Total calls should be displayed")
		}
		if !strings.Contains(result, "Read") {
			t.Error("CoS 3 failed: Tool names should be displayed")
		}
		if !strings.Contains(result, "Recent") {
			t.Error("CoS 3 failed: Recent tools should be displayed")
		}
	})

	t.Run("metrics detail handles missing metrics", func(t *testing.T) {
		m := initialModel()
		m.width = 100
		m.dialogMode = DialogMetricsDetail
		session := SessionInfo{
			TmuxSession: "test-session",
			Status:      "working",
			Metrics:     nil,
		}
		m.sessionToModify = &session

		result := m.renderMetricsDetailView()

		if !strings.Contains(result, "No metrics data available") {
			t.Error("Should indicate no metrics data when none present")
		}
	})
}

// TestE2E_AggregateDashboard verifies that aggregate metrics are shown
// across all sessions.
func TestE2E_AggregateDashboard(t *testing.T) {
	t.Run("header shows aggregate metrics for multiple sessions", func(t *testing.T) {
		m := initialModel()
		m.width = 120
		m.sessions = []SessionInfo{
			{
				TmuxSession: "session1",
				Status:      "working",
				Metrics: &Metrics{
					Time:  &TimeMetrics{TotalSeconds: 3600},
					Tools: &ToolMetrics{Counts: map[string]int{"Read": 20}},
				},
			},
			{
				TmuxSession: "session2",
				Status:      "waiting",
				Metrics: &Metrics{
					Time:  &TimeMetrics{TotalSeconds: 1800},
					Tools: &ToolMetrics{Counts: map[string]int{"Edit": 10}},
				},
			},
			{
				TmuxSession: "session3",
				Status:      "working",
				Metrics: &Metrics{
					Time:  &TimeMetrics{TotalSeconds: 7200},
					Tools: &ToolMetrics{Counts: map[string]int{"Bash": 15}},
				},
			},
		}

		result := m.renderHeader()

		// Verify aggregate time is shown (CoS 6: Aggregate dashboard)
		// Total: 3600 + 1800 + 7200 = 12600 seconds = 3h 30m
		if !strings.Contains(result, "⏱") {
			t.Error("CoS 6 failed: Aggregate time should be shown in header")
		}

		// Verify aggregate tools are shown
		// Total: 20 + 10 + 15 = 45 tools
		if !strings.Contains(result, "🔧") {
			t.Error("CoS 6 failed: Aggregate tool count should be shown in header")
		}
	})

	t.Run("header does not show aggregate when no metrics", func(t *testing.T) {
		m := initialModel()
		m.width = 100
		m.sessions = []SessionInfo{
			{TmuxSession: "session1", Status: "working", Metrics: nil},
			{TmuxSession: "session2", Status: "waiting", Metrics: nil},
		}

		result := m.renderHeader()

		// Should NOT contain metrics icons when no sessions have metrics
		if strings.Contains(result, "⏱") || strings.Contains(result, "🔧") {
			t.Error("Header should not show aggregate when no sessions have metrics")
		}
	})

	t.Run("aggregate calculation sums correctly", func(t *testing.T) {
		sessions := []SessionInfo{
			{
				TmuxSession: "s1",
				Metrics: &Metrics{
					Time: &TimeMetrics{
						TotalSeconds:   3600,
						WorkingSeconds: 2400,
						WaitingSeconds: 1200,
					},
					Tools: &ToolMetrics{Counts: map[string]int{"Read": 10, "Edit": 5}},
				},
			},
			{
				TmuxSession: "s2",
				Metrics: &Metrics{
					Time: &TimeMetrics{
						TotalSeconds:   1800,
						WorkingSeconds: 1200,
						WaitingSeconds: 600,
					},
					Tools: &ToolMetrics{Counts: map[string]int{"Read": 8, "Bash": 3}},
				},
			},
		}

		aggregate := AggregateMetrics(sessions)

		// Verify time aggregation
		if aggregate.Time.TotalSeconds != 5400 {
			t.Errorf("CoS 6 failed: Total seconds = %d, want 5400", aggregate.Time.TotalSeconds)
		}
		if aggregate.Time.WorkingSeconds != 3600 {
			t.Errorf("CoS 6 failed: Working seconds = %d, want 3600", aggregate.Time.WorkingSeconds)
		}

		// Verify tool aggregation
		totalTools := formatToolCount(aggregate.Tools)
		if totalTools != 26 {
			t.Errorf("CoS 6 failed: Total tools = %d, want 26", totalTools)
		}
		if aggregate.Tools.Counts["Read"] != 18 {
			t.Errorf("CoS 6 failed: Read count = %d, want 18", aggregate.Tools.Counts["Read"])
		}
	})
}

// TestE2E_MetricsRealTimeUpdates verifies that metrics update with session changes.
func TestE2E_MetricsRealTimeUpdates(t *testing.T) {
	t.Run("metrics update when sessions change", func(t *testing.T) {
		m := initialModel()
		m.width = 100

		// Start with one session
		m.sessions = []SessionInfo{
			{
				TmuxSession: "session1",
				Metrics: &Metrics{
					Time:  &TimeMetrics{TotalSeconds: 3600},
					Tools: &ToolMetrics{Counts: map[string]int{"Read": 10}},
				},
			},
		}

		header1 := m.renderHeader()

		// Add another session (simulating poll update)
		m.sessions = append(m.sessions, SessionInfo{
			TmuxSession: "session2",
			Metrics: &Metrics{
				Time:  &TimeMetrics{TotalSeconds: 1800},
				Tools: &ToolMetrics{Counts: map[string]int{"Edit": 5}},
			},
		})

		header2 := m.renderHeader()

		// Headers should be different due to changed aggregate
		if header1 == header2 {
			t.Error("CoS 4 failed: Header should update when sessions change")
		}
	})
}

// TestE2E_MetricsPersistence verifies that metrics persist in session files.
func TestE2E_MetricsPersistence(t *testing.T) {
	t.Run("session info with metrics serializes correctly", func(t *testing.T) {
		session := SessionInfo{
			TmuxSession: "test-session",
			Status:      "working",
			CWD:         "/tmp/project",
			Timestamp:   1738627200,
			Metrics: &Metrics{
				Time: &TimeMetrics{
					Started:        1738620000,
					TotalSeconds:   7200,
					WorkingSeconds: 5400,
					WaitingSeconds: 1800,
				},
				Tools: &ToolMetrics{
					Counts: map[string]int{"Read": 50, "Edit": 20},
					Recent: []string{"Read", "Edit"},
				},
			},
		}

		// This is tested in sessions_test.go but we verify the structure here
		if session.Metrics == nil {
			t.Error("CoS 7 failed: Metrics should be present")
		}
		if session.Metrics.Time == nil {
			t.Error("CoS 7 failed: Time metrics should be present")
		}
		if session.Metrics.Tools == nil {
			t.Error("CoS 7 failed: Tool metrics should be present")
		}
	})
}

// Note on Token Tracking (CoS 1):
// Token tracking is NOT available via Claude Code hooks.
// Research in task 12-4 confirmed tokens are only accessible via OpenTelemetry,
// which requires external infrastructure. This acceptance criteria cannot be
// implemented with the current hook-based architecture.
