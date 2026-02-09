package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/session"
)

// newSearchFilterTestModel creates a model with varied sessions for testing search/filter/sort.
func newSearchFilterTestModel() Model {
	return Model{
		width:  120,
		height: 24,
		sessions: []session.Info{
			{TmuxSession: "api-server", Status: session.StatusWorking, CWD: "/home/user/api", Message: "Building API endpoints", Timestamp: time.Now().Unix() - 100},
			{TmuxSession: "frontend", Status: session.StatusWaiting, CWD: "/home/user/web", Message: "Waiting for input", Timestamp: time.Now().Unix() - 50},
			{TmuxSession: "database", Status: session.StatusPermission, CWD: "/home/user/db", Message: "Need approval", Timestamp: time.Now().Unix() - 200},
			{TmuxSession: "tests", Status: "done", CWD: "/home/user/api", Message: "All tests passed", Timestamp: time.Now().Unix() - 300},
			{TmuxSession: "deploy", Status: "error", CWD: "/home/user/deploy", Message: "Deploy failed", Timestamp: time.Now().Unix() - 150},
		},
		searchInput: initSearchInput(),
	}
}

// TestE2E_AC1_SearchModeOpens tests AC1: `/` opens search mode with exact matching.
func TestE2E_AC1_SearchModeOpens(t *testing.T) {
	t.Run("slash key enters search mode", func(t *testing.T) {
		m := newSearchFilterTestModel()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !updated.searchMode {
			t.Error("search mode should be active after pressing /")
		}
	})

	t.Run("search bar visible in view when search mode active", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.searchMode = true
		m.searchInput.Focus()

		result := m.View()
		if !strings.Contains(result, "/") {
			t.Error("view should contain search bar indicator when in search mode")
		}
	})
}

// TestE2E_AC2_SearchExactMatch tests AC2: Search uses exact substring matching, all items remain visible.
func TestE2E_AC2_SearchExactMatch(t *testing.T) {
	t.Run("typing in search sets query and computes matches", func(t *testing.T) {
		m := newSearchFilterTestModel()

		// Enter search mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
		newModel, _ := m.Update(msg)
		m = newModel.(Model)

		// Type "api" to search
		for _, r := range "api" {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
			newModel, _ = m.Update(msg)
			m = newModel.(Model)
		}

		if m.searchQuery != "api" {
			t.Errorf("searchQuery should be 'api', got %q", m.searchQuery)
		}

		// All sessions should remain visible (search does not filter)
		filtered := m.getFilteredSessions()
		if len(filtered) != 5 {
			t.Errorf("all 5 sessions should remain visible during search, got %d", len(filtered))
		}

		// But match indices should identify the matching sessions
		if len(m.searchMatches) == 0 {
			t.Error("searchMatches should contain at least one match for 'api'")
		}
	})

	t.Run("search matches CWD via findMatches", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.searchQuery = "deploy"
		m.computeSearchMatches()

		// "deploy" matches session name "deploy" and CWD "/home/user/deploy"
		if len(m.searchMatches) != 1 {
			t.Errorf("expected 1 match for 'deploy', got %d", len(m.searchMatches))
		}
	})

	t.Run("search matches message content via findMatches", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.searchQuery = "failed"
		m.computeSearchMatches()

		if len(m.searchMatches) == 0 {
			t.Error("search for 'failed' should match deploy session via message")
		}
		// Verify the match index points to the deploy session
		filtered := m.getFilteredSessions()
		found := false
		for _, idx := range m.searchMatches {
			if filtered[idx].TmuxSession == "deploy" {
				found = true
			}
		}
		if !found {
			t.Error("match indices should include deploy session for query 'failed'")
		}
	})

	t.Run("search is case insensitive", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.searchQuery = "API"
		m.computeSearchMatches()

		if len(m.searchMatches) == 0 {
			t.Error("case-insensitive search for 'API' should match sessions")
		}
	})

	t.Run("exact match rejects subsequence", func(t *testing.T) {
		// "msr" should NOT match "my-server" (exact match, not fuzzy)
		if exactMatch("msr", "my-server") {
			t.Error("exact match should not match subsequences like 'msr' in 'my-server'")
		}
	})

	t.Run("empty query matches nothing", func(t *testing.T) {
		if exactMatch("", "anything") {
			t.Error("empty query should match nothing")
		}
	})
}

// TestE2E_AC3_NumberKeyFilters tests AC3: Number keys (0-5) toggle status filters.
func TestE2E_AC3_NumberKeyFilters(t *testing.T) {
	t.Run("key 1 filters to waiting sessions", func(t *testing.T) {
		m := newSearchFilterTestModel()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.statusFilter != session.StatusWaiting {
			t.Errorf("statusFilter should be 'waiting', got %q", updated.statusFilter)
		}

		filtered := updated.getFilteredSessions()
		for _, s := range filtered {
			if s.Status != session.StatusWaiting {
				t.Errorf("all filtered sessions should be 'waiting', got %q", s.Status)
			}
		}
	})

	t.Run("key 2 filters to permission sessions", func(t *testing.T) {
		m := newSearchFilterTestModel()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.statusFilter != session.StatusPermission {
			t.Errorf("statusFilter should be 'permission', got %q", updated.statusFilter)
		}
	})

	t.Run("key 3 filters to working sessions", func(t *testing.T) {
		m := newSearchFilterTestModel()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.statusFilter != session.StatusWorking {
			t.Errorf("statusFilter should be 'working', got %q", updated.statusFilter)
		}
	})

	t.Run("key 4 filters to done sessions", func(t *testing.T) {
		m := newSearchFilterTestModel()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.statusFilter != "done" {
			t.Errorf("statusFilter should be 'done', got %q", updated.statusFilter)
		}
	})

	t.Run("key 5 filters to error sessions", func(t *testing.T) {
		m := newSearchFilterTestModel()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.statusFilter != "error" {
			t.Errorf("statusFilter should be 'error', got %q", updated.statusFilter)
		}
	})

	t.Run("key 0 clears filter", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.statusFilter = session.StatusWaiting

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.statusFilter != "" {
			t.Errorf("statusFilter should be empty after key 0, got %q", updated.statusFilter)
		}
	})

	t.Run("pressing same key toggles filter off", func(t *testing.T) {
		m := newSearchFilterTestModel()

		// Press 1 to filter
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}
		newModel, _ := m.Update(msg)
		m = newModel.(Model)

		if m.statusFilter != session.StatusWaiting {
			t.Fatal("statusFilter should be 'waiting' after first press")
		}

		// Press 1 again to toggle off
		newModel, _ = m.Update(msg)
		m = newModel.(Model)

		if m.statusFilter != "" {
			t.Errorf("statusFilter should be empty after toggle, got %q", m.statusFilter)
		}
	})

	t.Run("number keys ignored during search mode", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.searchMode = true
		m.searchInput.Focus()

		// Press '1' while in search mode - should go to search input, not filter
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.statusFilter != "" {
			t.Error("number keys should not activate filter during search mode")
		}
	})

	t.Run("number keys ignored during dialog", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.dialogMode = DialogKillConfirm

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.statusFilter != "" {
			t.Error("number keys should not activate filter during dialog")
		}
	})
}

// TestE2E_AC4_OfflineToggle tests AC4: `o` toggles offline session visibility.
func TestE2E_AC4_OfflineToggle(t *testing.T) {
	t.Run("o key hides done sessions", func(t *testing.T) {
		m := newSearchFilterTestModel()

		// Verify done session is initially visible
		allCount := len(m.getFilteredSessions())

		// Press 'o' to hide offline
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !updated.hideOffline {
			t.Error("hideOffline should be true after pressing o")
		}

		filteredCount := len(updated.getFilteredSessions())
		if filteredCount >= allCount {
			t.Error("hiding offline should reduce the session count")
		}

		// Verify no done sessions in filtered list
		for _, s := range updated.getFilteredSessions() {
			if s.Status == "done" {
				t.Error("done sessions should be hidden when offline is toggled")
			}
		}
	})

	t.Run("o key toggles back to show all", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.hideOffline = true

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.hideOffline {
			t.Error("hideOffline should be false after second toggle")
		}
	})
}

// TestE2E_AC5_SortCycling tests AC5: `s` cycles through sort modes.
func TestE2E_AC5_SortCycling(t *testing.T) {
	t.Run("s key cycles through all sort modes", func(t *testing.T) {
		m := newSearchFilterTestModel()

		expectedModes := []SortMode{SortName, SortAge, SortStatus, SortDirectory, SortPriority}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}

		for _, expected := range expectedModes {
			newModel, _ := m.Update(msg)
			m = newModel.(Model)

			if m.sortMode != expected {
				t.Errorf("expected sort mode %d, got %d", expected, m.sortMode)
			}
		}
	})

	t.Run("SortName sorts alphabetically", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.sortMode = SortName

		filtered := m.getFilteredSessions()
		for i := 1; i < len(filtered); i++ {
			if strings.ToLower(filtered[i-1].TmuxSession) > strings.ToLower(filtered[i].TmuxSession) {
				t.Errorf("sessions not sorted by name: %q > %q", filtered[i-1].TmuxSession, filtered[i].TmuxSession)
			}
		}
	})

	t.Run("SortAge sorts by most recent first", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.sortMode = SortAge

		filtered := m.getFilteredSessions()
		for i := 1; i < len(filtered); i++ {
			if filtered[i-1].Timestamp < filtered[i].Timestamp {
				t.Error("sessions not sorted by age (most recent first)")
			}
		}
	})

	t.Run("SortDirectory groups by CWD", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.sortMode = SortDirectory

		filtered := m.getFilteredSessions()
		for i := 1; i < len(filtered); i++ {
			if strings.ToLower(filtered[i-1].CWD) > strings.ToLower(filtered[i].CWD) {
				t.Errorf("sessions not sorted by directory: %q > %q", filtered[i-1].CWD, filtered[i].CWD)
			}
		}
	})
}

// TestE2E_AC6_FooterState tests AC6: Current filter/sort state shown in footer.
func TestE2E_AC6_FooterState(t *testing.T) {
	t.Run("footer shows active status filter", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.statusFilter = session.StatusWaiting

		result := m.View()
		if !strings.Contains(result, "Filter: waiting") {
			t.Error("footer should show 'Filter: waiting' when status filter is active")
		}
	})

	t.Run("footer shows sort mode when not default", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.sortMode = SortName

		result := m.View()
		if !strings.Contains(result, "Sort: name") {
			t.Error("footer should show 'Sort: name' when sort mode is not default")
		}
	})

	t.Run("footer shows offline hidden state", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.hideOffline = true

		result := m.View()
		if !strings.Contains(result, "Offline: hidden") {
			t.Error("footer should show 'Offline: hidden' when offline sessions are hidden")
		}
	})

	t.Run("footer does not show sort mode when default", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.sortMode = SortPriority

		result := m.View()
		if strings.Contains(result, "Sort:") {
			t.Error("footer should not show sort mode when it's the default (priority)")
		}
	})
}

// TestE2E_AC7_FilteredCount tests AC7: Filtered count displayed (e.g., "3/8 shown").
func TestE2E_AC7_FilteredCount(t *testing.T) {
	t.Run("count shown when filter reduces list", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.statusFilter = session.StatusWorking

		result := m.View()
		// Should show something like "1/5 shown"
		if !strings.Contains(result, "/5 shown") {
			t.Errorf("footer should show filtered count, got view:\n%s", result)
		}
	})

	t.Run("count not shown when no filter active", func(t *testing.T) {
		m := newSearchFilterTestModel()

		result := m.View()
		if strings.Contains(result, "shown") {
			t.Error("footer should not show count when no filter is active")
		}
	})
}

// TestE2E_AC8_EscClears tests AC8: `Esc` clears search and filters.
func TestE2E_AC8_EscClears(t *testing.T) {
	t.Run("esc clears search mode", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.searchMode = true
		m.searchQuery = "api"
		m.searchInput.SetValue("api")
		m.searchInput.Focus()

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.searchMode {
			t.Error("search mode should be cleared after Esc")
		}
		if updated.searchQuery != "" {
			t.Errorf("search query should be empty after Esc, got %q", updated.searchQuery)
		}
		if updated.searchMatches != nil {
			t.Error("searchMatches should be nil after Esc")
		}
	})

	t.Run("esc clears persisted search state", func(t *testing.T) {
		m := newSearchFilterTestModel()
		// Simulate persisted search (searchMode=false but searchQuery set)
		m.searchQuery = "api"
		m.searchMatches = []int{0, 3}
		m.currentMatchIdx = 1

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.searchQuery != "" {
			t.Errorf("search query should be empty after Esc, got %q", updated.searchQuery)
		}
		if updated.searchMatches != nil {
			t.Error("searchMatches should be nil after Esc")
		}
		if updated.currentMatchIdx != 0 {
			t.Error("currentMatchIdx should be 0 after Esc")
		}
	})

	t.Run("esc clears status filter when no search active", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.statusFilter = session.StatusWaiting

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.statusFilter != "" {
			t.Errorf("status filter should be cleared after Esc, got %q", updated.statusFilter)
		}
	})

	t.Run("esc clears offline filter", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.hideOffline = true

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.hideOffline {
			t.Error("hideOffline should be cleared after Esc")
		}
	})
}

// TestE2E_AC9_CursorPreservation tests AC9: Cursor position preserved when possible during filtering.
func TestE2E_AC9_CursorPreservation(t *testing.T) {
	t.Run("cursor preserved when session stays in filtered list", func(t *testing.T) {
		m := newSearchFilterTestModel()
		// Select api-server (working status) at cursor 0
		m.cursor = 0

		// Verify what session is selected
		filtered := m.getFilteredSessions()
		selectedName := filtered[m.cursor].TmuxSession

		// Apply filter that keeps this session
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}} // Filter to working
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		// The working session should still be at cursor position
		newFiltered := updated.getFilteredSessions()
		if len(newFiltered) > 0 && updated.cursor < len(newFiltered) {
			if newFiltered[updated.cursor].TmuxSession != selectedName {
				t.Errorf("cursor should preserve selection on %q, but got %q at cursor %d",
					selectedName, newFiltered[updated.cursor].TmuxSession, updated.cursor)
			}
		}
	})

	t.Run("cursor resets when session filtered out", func(t *testing.T) {
		m := newSearchFilterTestModel()
		// Select deploy (error status)
		for i, s := range m.getFilteredSessions() {
			if s.TmuxSession == "deploy" {
				m.cursor = i
				break
			}
		}

		// Apply filter to waiting (should exclude deploy)
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}} // Filter to waiting
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		// Cursor should be valid
		newFiltered := updated.getFilteredSessions()
		if updated.cursor >= len(newFiltered) && len(newFiltered) > 0 {
			t.Error("cursor should be within bounds of filtered list")
		}
	})

	t.Run("cursor preserved across offline toggle", func(t *testing.T) {
		m := newSearchFilterTestModel()

		// Select api-server (working, should survive offline toggle)
		for i, s := range m.getFilteredSessions() {
			if s.TmuxSession == "api-server" {
				m.cursor = i
				break
			}
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}} // Toggle offline
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		// api-server should still be selected
		newFiltered := updated.getFilteredSessions()
		if updated.cursor < len(newFiltered) {
			if newFiltered[updated.cursor].TmuxSession != "api-server" {
				t.Errorf("cursor should stay on api-server after offline toggle, got %q",
					newFiltered[updated.cursor].TmuxSession)
			}
		}
	})
}

// TestE2E_FilterComposition tests that multiple filters compose correctly.
func TestE2E_FilterComposition(t *testing.T) {
	t.Run("status filter reduces list independently of search", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.statusFilter = session.StatusWorking
		m.searchQuery = "api"

		filtered := m.getFilteredSessions()
		// Status filter reduces to 1 working session; search does not filter further
		if len(filtered) != 1 {
			t.Errorf("expected 1 session with working status, got %d", len(filtered))
		}
		if len(filtered) > 0 && filtered[0].TmuxSession != "api-server" {
			t.Errorf("expected api-server, got %q", filtered[0].TmuxSession)
		}

		// But search matches should be computed against the filtered list
		m.computeSearchMatches()
		if len(m.searchMatches) != 1 {
			t.Errorf("expected 1 search match in working sessions for 'api', got %d", len(m.searchMatches))
		}
	})

	t.Run("offline filter and status filter compose", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.hideOffline = true
		m.statusFilter = "" // No status filter, but hide done

		filtered := m.getFilteredSessions()
		for _, s := range filtered {
			if s.Status == "done" {
				t.Error("done sessions should be hidden when offline toggle is active")
			}
		}
	})

	t.Run("all filters compose together", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.statusFilter = session.StatusWorking
		m.hideOffline = true
		m.searchQuery = "server"
		m.sortMode = SortName

		filtered := m.getFilteredSessions()
		// Status filter reduces to 1 working session; search doesn't filter
		if len(filtered) != 1 {
			t.Errorf("expected 1 session with all filters, got %d", len(filtered))
		}
	})
}

// TestE2E_SearchModeKeyRouting tests that keys are properly routed during search mode.
func TestE2E_SearchModeKeyRouting(t *testing.T) {
	t.Run("navigation keys work during search", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.searchMode = true
		m.searchInput.Focus()
		m.cursor = 0

		// Press down
		msg := tea.KeyMsg{Type: tea.KeyDown}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.cursor != 1 {
			t.Errorf("down key should move cursor during search, got %d", updated.cursor)
		}
	})

	t.Run("enter key persists search and attaches", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.searchMode = true
		m.searchQuery = "api"
		m.searchInput.SetValue("api")
		m.searchInput.Focus()
		m.cursor = 0

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, cmd := m.Update(msg)
		updated := newModel.(Model)

		// Search mode should be exited but query persists
		if updated.searchMode {
			t.Error("searchMode should be false after Enter (input mode exits)")
		}
		if updated.searchQuery != "api" {
			t.Errorf("searchQuery should persist after Enter, got %q", updated.searchQuery)
		}
		if cmd == nil {
			t.Error("enter should return a command (attach) during search mode")
		}
	})
}

// TestExactMatch tests the exact matching function directly.
func TestExactMatch(t *testing.T) {
	t.Run("exact substring matches", func(t *testing.T) {
		if !exactMatch("api", "my-api-server") {
			t.Error("'api' should match 'my-api-server'")
		}
	})

	t.Run("subsequence does NOT match", func(t *testing.T) {
		if exactMatch("msr", "my-server") {
			t.Error("'msr' should NOT match 'my-server' (not a substring)")
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		if !exactMatch("API", "my-api-server") {
			t.Error("'API' should match 'my-api-server' case-insensitively")
		}
	})

	t.Run("no match returns false", func(t *testing.T) {
		if exactMatch("xyz", "my-api-server") {
			t.Error("'xyz' should not match 'my-api-server'")
		}
	})

	t.Run("empty query matches nothing", func(t *testing.T) {
		if exactMatch("", "anything") {
			t.Error("empty query should match nothing")
		}
	})

	t.Run("partial word matches", func(t *testing.T) {
		if !exactMatch("serv", "api-server") {
			t.Error("'serv' should match 'api-server' as substring")
		}
	})
}

// TestFindMatches tests the findMatches function directly.
func TestFindMatches(t *testing.T) {
	sessions := []session.Info{
		{TmuxSession: "api-server", CWD: "/home/user/api", Message: "Building API"},
		{TmuxSession: "frontend", CWD: "/home/user/web", Message: "Waiting"},
		{TmuxSession: "database", CWD: "/home/user/db", Message: "Need approval"},
		{TmuxSession: "tests", CWD: "/home/user/api", Message: "All tests passed"},
	}

	t.Run("matches by name", func(t *testing.T) {
		matches := findMatches(sessions, "api")
		// "api-server" name, and "tests" CWD contains "api"
		if len(matches) < 1 {
			t.Error("should find at least 1 match for 'api'")
		}
	})

	t.Run("matches by CWD", func(t *testing.T) {
		matches := findMatches(sessions, "/home/user/db")
		if len(matches) != 1 || matches[0] != 2 {
			t.Errorf("should find database at index 2, got %v", matches)
		}
	})

	t.Run("matches by message", func(t *testing.T) {
		matches := findMatches(sessions, "passed")
		if len(matches) != 1 || matches[0] != 3 {
			t.Errorf("should find tests at index 3, got %v", matches)
		}
	})

	t.Run("empty query returns nil", func(t *testing.T) {
		matches := findMatches(sessions, "")
		if matches != nil {
			t.Errorf("empty query should return nil, got %v", matches)
		}
	})

	t.Run("no matches returns nil", func(t *testing.T) {
		matches := findMatches(sessions, "nonexistent")
		if matches != nil {
			t.Errorf("non-matching query should return nil, got %v", matches)
		}
	})
}

// TestMatchCycling tests n/N match cycling with wrap-around.
func TestMatchCycling(t *testing.T) {
	t.Run("n cycles through matches forward with wrap", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.searchQuery = "api"
		m.computeSearchMatches()

		if len(m.searchMatches) < 2 {
			t.Skip("need at least 2 matches for cycling test")
		}

		// Record initial state
		firstMatchIdx := m.searchMatches[0]
		secondMatchIdx := m.searchMatches[1]

		// First n should go to second match
		m.nextMatch()
		if m.cursor != secondMatchIdx {
			t.Errorf("first n should move to match index %d, got %d", secondMatchIdx, m.cursor)
		}

		// Keep pressing n until we wrap back to first match
		for i := 0; i < len(m.searchMatches); i++ {
			m.nextMatch()
		}
		if m.cursor != secondMatchIdx {
			t.Errorf("after wrapping, cursor should be at %d, got %d", secondMatchIdx, m.cursor)
		}

		// Verify we can get back to first
		for m.cursor != firstMatchIdx {
			m.nextMatch()
		}
		if m.cursor != firstMatchIdx {
			t.Errorf("should be able to cycle back to first match %d", firstMatchIdx)
		}
	})

	t.Run("N cycles through matches backward with wrap", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.searchQuery = "api"
		m.computeSearchMatches()

		if len(m.searchMatches) < 2 {
			t.Skip("need at least 2 matches for cycling test")
		}

		lastMatchIdx := m.searchMatches[len(m.searchMatches)-1]

		// First N should wrap to last match
		m.prevMatch()
		if m.cursor != lastMatchIdx {
			t.Errorf("first N should wrap to last match index %d, got %d", lastMatchIdx, m.cursor)
		}
	})

	t.Run("n does nothing with no matches", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.searchQuery = "nonexistent"
		m.computeSearchMatches()
		m.cursor = 2

		m.nextMatch()
		if m.cursor != 2 {
			t.Errorf("n should not move cursor when no matches, cursor was %d", m.cursor)
		}
	})

	t.Run("n opens new session when no search active", func(t *testing.T) {
		m := newSearchFilterTestModel()
		// No search query - n should open new session dialog
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogNewSession {
			t.Error("n without search should open new session dialog")
		}
	})

	t.Run("n jumps to next match when search persisted", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.searchQuery = "api"
		m.computeSearchMatches()

		if len(m.searchMatches) == 0 {
			t.Fatal("need matches for this test")
		}

		// Simulate persisted search (not in searchMode, but query set)
		m.searchMode = false
		firstMatch := m.searchMatches[0]
		m.cursor = firstMatch

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		// Should NOT open new session dialog
		if updated.dialogMode == DialogNewSession {
			t.Error("n with persisted search should cycle matches, not open new session")
		}
	})
}

// TestSearchPersistence tests that search persists after Enter and clears only on Esc.
func TestSearchPersistence(t *testing.T) {
	t.Run("enter exits input mode but preserves search", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.searchMode = true
		m.searchQuery = "api"
		m.searchInput.SetValue("api")
		m.searchInput.Focus()
		m.computeSearchMatches()

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.searchMode {
			t.Error("searchMode should be false after Enter")
		}
		if updated.searchQuery != "api" {
			t.Error("searchQuery should persist after Enter")
		}
		if updated.searchMatches == nil {
			t.Error("searchMatches should persist after Enter")
		}
	})

	t.Run("search bar visible when search persisted", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.searchQuery = "api"
		m.computeSearchMatches()
		// Not in searchMode but query is set

		result := m.View()
		if !strings.Contains(result, "api") {
			t.Error("view should show persisted search query")
		}
	})

	t.Run("match counter visible in persisted search", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.searchQuery = "api"
		m.computeSearchMatches()

		result := m.View()
		if len(m.searchMatches) > 0 {
			// Should contain a match counter like [1/2]
			if !strings.Contains(result, "/") {
				t.Error("view should show match counter in persisted search")
			}
		}
	})
}

// TestMatchCounter tests the match counter display.
func TestMatchCounter(t *testing.T) {
	t.Run("no matches shows indicator", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.searchMode = true
		m.searchQuery = "nonexistent"
		m.searchInput.SetValue("nonexistent")
		m.searchInput.Focus()
		m.computeSearchMatches()

		result := m.View()
		if !strings.Contains(result, "No matches") {
			t.Error("should show 'No matches' when query has no results")
		}
	})
}

// TestSortModes tests sort mode functions directly.
func TestSortModes(t *testing.T) {
	t.Run("SortModeLabel returns correct labels", func(t *testing.T) {
		expectations := map[SortMode]string{
			SortPriority:  "priority",
			SortName:      "name",
			SortAge:       "age",
			SortStatus:    "status",
			SortDirectory: "directory",
		}
		for mode, expected := range expectations {
			if SortModeLabel(mode) != expected {
				t.Errorf("SortModeLabel(%d) should be %q, got %q", mode, expected, SortModeLabel(mode))
			}
		}
	})

	t.Run("sortSessions does not mutate original", func(t *testing.T) {
		sessions := []session.Info{
			{TmuxSession: "b"},
			{TmuxSession: "a"},
		}

		sorted := sortSessions(sessions, SortName)

		if sessions[0].TmuxSession != "b" {
			t.Error("original slice should not be mutated")
		}
		if sorted[0].TmuxSession != "a" {
			t.Error("sorted slice should have 'a' first")
		}
	})

	t.Run("SortPriority returns original order", func(t *testing.T) {
		sessions := []session.Info{
			{TmuxSession: "first"},
			{TmuxSession: "second"},
		}

		sorted := sortSessions(sessions, SortPriority)

		if sorted[0].TmuxSession != "first" {
			t.Error("SortPriority should preserve original order")
		}
	})
}

// TestFilterByStatus tests the status filter function directly.
func TestFilterByStatus(t *testing.T) {
	sessions := []session.Info{
		{TmuxSession: "a", Status: session.StatusWaiting},
		{TmuxSession: "b", Status: session.StatusWorking},
		{TmuxSession: "c", Status: session.StatusWaiting},
		{TmuxSession: "d", Status: "done"},
	}

	t.Run("filter by waiting", func(t *testing.T) {
		result := filterByStatus(sessions, session.StatusWaiting)
		if len(result) != 2 {
			t.Errorf("expected 2 waiting sessions, got %d", len(result))
		}
	})

	t.Run("filter by non-existent status returns empty", func(t *testing.T) {
		result := filterByStatus(sessions, "nonexistent")
		if len(result) != 0 {
			t.Errorf("expected 0 sessions for non-existent status, got %d", len(result))
		}
	})
}

// TestFilterOffline tests the offline filter function directly.
func TestFilterOffline(t *testing.T) {
	sessions := []session.Info{
		{TmuxSession: "a", Status: session.StatusWorking},
		{TmuxSession: "b", Status: "done"},
		{TmuxSession: "c", Status: session.StatusWaiting},
	}

	result := filterOffline(sessions)
	if len(result) != 2 {
		t.Errorf("expected 2 non-done sessions, got %d", len(result))
	}
	for _, s := range result {
		if s.Status == "done" {
			t.Error("filterOffline should remove done sessions")
		}
	}
}
