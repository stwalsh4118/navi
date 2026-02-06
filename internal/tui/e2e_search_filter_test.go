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

// TestE2E_AC1_SearchModeOpens tests AC1: `/` opens search mode with fuzzy matching.
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

// TestE2E_AC2_SearchFiltersRealTime tests AC2: Search filters session list in real-time.
func TestE2E_AC2_SearchFiltersRealTime(t *testing.T) {
	t.Run("typing in search filters sessions", func(t *testing.T) {
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

		filtered := m.getFilteredSessions()
		// "api-server" matches name, "tests" matches CWD (/home/user/api)
		if len(filtered) == 0 {
			t.Error("search for 'api' should return at least one result")
		}

		// Verify api-server is in results
		found := false
		for _, s := range filtered {
			if s.TmuxSession == "api-server" {
				found = true
				break
			}
		}
		if !found {
			t.Error("api-server should be in search results for 'api'")
		}
	})

	t.Run("search matches CWD", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.searchMode = true
		m.searchQuery = "deploy"

		filtered := m.getFilteredSessions()
		if len(filtered) != 1 || filtered[0].TmuxSession != "deploy" {
			t.Errorf("search for 'deploy' should match deploy session via name/CWD, got %d results", len(filtered))
		}
	})

	t.Run("search matches message content", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.searchMode = true
		m.searchQuery = "failed"

		filtered := m.getFilteredSessions()
		found := false
		for _, s := range filtered {
			if s.TmuxSession == "deploy" {
				found = true
			}
		}
		if !found {
			t.Error("search for 'failed' should match deploy session via message")
		}
	})

	t.Run("search is case insensitive", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.searchMode = true
		m.searchQuery = "API"

		filtered := m.getFilteredSessions()
		if len(filtered) == 0 {
			t.Error("case-insensitive search for 'API' should match 'api-server'")
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
	})

	t.Run("esc clears status filter when not in search mode", func(t *testing.T) {
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
	t.Run("status filter and search compose", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.statusFilter = session.StatusWorking
		m.searchQuery = "api"

		filtered := m.getFilteredSessions()
		// Only api-server is working AND matches "api"
		if len(filtered) != 1 {
			t.Errorf("expected 1 session matching working+api, got %d", len(filtered))
		}
		if len(filtered) > 0 && filtered[0].TmuxSession != "api-server" {
			t.Errorf("expected api-server, got %q", filtered[0].TmuxSession)
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
		// Only api-server matches: working status + "server" in name + not done
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

	t.Run("enter key works during search", func(t *testing.T) {
		m := newSearchFilterTestModel()
		m.searchMode = true
		m.searchInput.Focus()
		m.cursor = 0

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		_, cmd := m.Update(msg)

		if cmd == nil {
			t.Error("enter should return a command during search mode")
		}
	})
}

// TestFuzzyMatch tests the fuzzy matching algorithm directly.
func TestFuzzyMatch(t *testing.T) {
	t.Run("exact substring matches", func(t *testing.T) {
		ok, score := fuzzyMatch("api", "my-api-server")
		if !ok || score == 0 {
			t.Error("'api' should match 'my-api-server'")
		}
	})

	t.Run("subsequence matches", func(t *testing.T) {
		ok, _ := fuzzyMatch("msr", "my-server")
		// m...s...r subsequence match
		if !ok {
			t.Error("'msr' should match 'my-server' as subsequence")
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		ok, _ := fuzzyMatch("API", "my-api-server")
		if !ok {
			t.Error("'API' should match 'my-api-server' case-insensitively")
		}
	})

	t.Run("no match returns false", func(t *testing.T) {
		ok, _ := fuzzyMatch("xyz", "my-api-server")
		if ok {
			t.Error("'xyz' should not match 'my-api-server'")
		}
	})

	t.Run("empty query matches everything", func(t *testing.T) {
		ok, _ := fuzzyMatch("", "anything")
		if !ok {
			t.Error("empty query should match everything")
		}
	})

	t.Run("start-of-word bonus gives higher score", func(t *testing.T) {
		_, score1 := fuzzyMatch("a", "api-server")   // Start of word
		_, score2 := fuzzyMatch("p", "api-server")   // Middle of word
		_, score3 := fuzzyMatch("s", "api-server")   // Start of second word (after -)

		if score1 <= score2 {
			t.Error("start-of-string match should score higher than mid-word match")
		}
		if score3 <= score2 {
			t.Error("start-of-word match should score higher than mid-word match")
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
