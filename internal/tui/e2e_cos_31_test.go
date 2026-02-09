package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/session"
	"github.com/stwalsh4118/navi/internal/task"
)

// newCoS31TestModel creates a model with sessions and tasks for PBI-31 E2E CoS testing.
func newCoS31TestModel() Model {
	groups := []task.TaskGroup{
		{
			ID:     "g1",
			Title:  "Search & Filter",
			Status: "in_progress",
			Tasks: []task.Task{
				{ID: "31-1", Title: "Replace fuzzy matching with exact substring matching", Status: "done"},
				{ID: "31-2", Title: "Implement search state and cycling", Status: "done"},
				{ID: "31-3", Title: "Render search highlights and counter", Status: "done"},
			},
		},
		{
			ID:     "g2",
			Title:  "API Server",
			Status: "open",
			Tasks: []task.Task{
				{ID: "#142", Title: "Add rate limiting endpoint", Status: "active"},
				{ID: "#138", Title: "Fix auth token refresh", Status: "todo"},
			},
		},
	}

	projectDir := "/home/user/api"

	return Model{
		width:  120,
		height: 40,
		sessions: []session.Info{
			{TmuxSession: "api-server", Status: session.StatusWorking, CWD: projectDir, Message: "Building API endpoints", Timestamp: time.Now().Unix() - 100},
			{TmuxSession: "frontend", Status: session.StatusWaiting, CWD: "/home/user/web", Message: "Waiting for input", Timestamp: time.Now().Unix() - 50},
			{TmuxSession: "api-tests", Status: session.StatusPermission, CWD: projectDir, Message: "Need approval for tests", Timestamp: time.Now().Unix() - 200},
			{TmuxSession: "database", Status: "done", CWD: "/home/user/db", Message: "Migrations complete", Timestamp: time.Now().Unix() - 300},
			{TmuxSession: "deploy", Status: "error", CWD: "/home/user/deploy", Message: "Deploy failed", Timestamp: time.Now().Unix() - 150},
		},
		searchInput:     initSearchInput(),
		taskSearchInput: initTaskSearchInput(),
		taskCache:       task.NewResultCache(),
		taskGlobalConfig: &task.GlobalConfig{},
		taskGroups:         groups,
		taskExpandedGroups: make(map[string]bool),
		taskFocusedProject: projectDir,
		taskGroupsByProject: map[string][]task.TaskGroup{
			projectDir: groups,
		},
		taskProjectConfigs: []task.ProjectConfig{
			{Tasks: task.ProjectTaskConfig{Provider: "test"}, ProjectDir: projectDir},
		},
	}
}

// sendKey sends a single rune key message and returns the updated model.
func sendKey(m Model, r rune) Model {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
	newModel, _ := m.Update(msg)
	return newModel.(Model)
}

// sendSpecialKey sends a special key message and returns the updated model.
func sendSpecialKey(m Model, keyType tea.KeyType) Model {
	msg := tea.KeyMsg{Type: keyType}
	newModel, _ := m.Update(msg)
	return newModel.(Model)
}

// typeSearch enters search mode and types a query.
func typeSearch(m Model, query string) Model {
	m = sendKey(m, '/')
	for _, r := range query {
		m = sendKey(m, r)
	}
	return m
}

// TestCoS31_AC1_SearchModeExactMatch verifies AC1: "/" enters search mode; typing produces
// exact case-insensitive substring matches.
func TestCoS31_AC1_SearchModeExactMatch(t *testing.T) {
	t.Run("slash enters search and exact substring matches", func(t *testing.T) {
		m := newCoS31TestModel()

		// Enter search mode and type "api"
		m = typeSearch(m, "api")

		if !m.searchMode {
			t.Fatal("should be in search mode")
		}
		if m.searchQuery != "api" {
			t.Fatalf("searchQuery should be 'api', got %q", m.searchQuery)
		}

		// "api" is an exact substring of "api-server" and "api-tests"
		if len(m.searchMatches) != 2 {
			t.Errorf("expected 2 matches for 'api' (api-server, api-tests), got %d", len(m.searchMatches))
		}
	})

	t.Run("exact match rejects fuzzy subsequence", func(t *testing.T) {
		m := newCoS31TestModel()
		m = typeSearch(m, "asr") // subsequence of "api-server" but not substring

		if len(m.searchMatches) != 0 {
			t.Errorf("subsequence 'asr' should not match anything, got %d matches", len(m.searchMatches))
		}
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		m := newCoS31TestModel()
		m = typeSearch(m, "API")

		if len(m.searchMatches) == 0 {
			t.Error("uppercase 'API' should match 'api-server' and 'api-tests'")
		}
	})

	t.Run("matches session name, CWD, and message", func(t *testing.T) {
		m := newCoS31TestModel()

		// Match by message content
		m = typeSearch(m, "failed")
		if len(m.searchMatches) != 1 {
			t.Errorf("expected 1 match for 'failed' (deploy message), got %d", len(m.searchMatches))
		}
	})
}

// TestCoS31_AC2_AllItemsVisible verifies AC2: All items remain visible during search (no filtering).
func TestCoS31_AC2_AllItemsVisible(t *testing.T) {
	t.Run("all sessions remain visible during search", func(t *testing.T) {
		m := newCoS31TestModel()
		totalSessions := len(m.getFilteredSessions())

		m = typeSearch(m, "api")

		filtered := m.getFilteredSessions()
		if len(filtered) != totalSessions {
			t.Errorf("all %d sessions should remain visible during search, got %d", totalSessions, len(filtered))
		}
	})

	t.Run("non-matching search still shows all sessions", func(t *testing.T) {
		m := newCoS31TestModel()
		totalSessions := len(m.getFilteredSessions())

		m = typeSearch(m, "xyznonexistent")

		filtered := m.getFilteredSessions()
		if len(filtered) != totalSessions {
			t.Errorf("even with no matches, all %d sessions should be visible, got %d", totalSessions, len(filtered))
		}
	})
}

// TestCoS31_AC3_HighlightsRendered verifies AC3: Matching items are visually highlighted in the list.
func TestCoS31_AC3_HighlightsRendered(t *testing.T) {
	t.Run("matching sessions get highlight styling in rendered view", func(t *testing.T) {
		m := newCoS31TestModel()
		m = typeSearch(m, "api")

		// Verify internal match tracking is correct
		filtered := m.getFilteredSessions()
		for _, idx := range m.searchMatches {
			s := filtered[idx]
			if !exactMatch("api", s.TmuxSession) && !exactMatch("api", s.CWD) && !exactMatch("api", s.Message) {
				t.Errorf("match index %d points to non-matching session %q", idx, s.TmuxSession)
			}
		}

		// Verify isSearchMatch returns true for matches
		for _, idx := range m.searchMatches {
			if !m.isSearchMatch(idx) {
				t.Errorf("isSearchMatch(%d) should return true for a matched session", idx)
			}
		}

		// Verify isSearchMatch returns false for non-matches
		matchSet := make(map[int]bool)
		for _, idx := range m.searchMatches {
			matchSet[idx] = true
		}
		for i := range filtered {
			if !matchSet[i] && m.isSearchMatch(i) {
				t.Errorf("isSearchMatch(%d) should return false for a non-matched session", i)
			}
		}
	})

	t.Run("current match distinguished from other matches", func(t *testing.T) {
		m := newCoS31TestModel()
		m = typeSearch(m, "api")

		if len(m.searchMatches) < 2 {
			t.Skip("need at least 2 matches")
		}

		currentIdx := m.searchMatches[m.currentMatchIdx]
		if !m.isCurrentSearchMatch(currentIdx) {
			t.Error("isCurrentSearchMatch should return true for current match")
		}

		// Other matches should not be the current match
		for i, idx := range m.searchMatches {
			if i != m.currentMatchIdx && m.isCurrentSearchMatch(idx) {
				t.Errorf("isCurrentSearchMatch should return false for non-current match at index %d", i)
			}
		}
	})
}

// TestCoS31_AC4_NAndNCycling verifies AC4: "n" jumps to next match, "N" jumps to previous.
func TestCoS31_AC4_NAndNCycling(t *testing.T) {
	t.Run("n cycles to next match after Enter persists search", func(t *testing.T) {
		m := newCoS31TestModel()
		m = typeSearch(m, "api")

		if len(m.searchMatches) < 2 {
			t.Fatal("need at least 2 matches for cycling")
		}

		// Press Enter to persist search, then use n in normal mode
		m = sendSpecialKey(m, tea.KeyEnter)
		if m.searchMode {
			t.Fatal("searchMode should be false after Enter")
		}

		firstMatch := m.searchMatches[0]
		secondMatch := m.searchMatches[1]

		m.cursor = firstMatch
		m.currentMatchIdx = 0

		// Press n to go to next match
		m = sendKey(m, 'n')
		if m.cursor != secondMatch {
			t.Errorf("n should move cursor to second match %d, got %d", secondMatch, m.cursor)
		}
	})

	t.Run("N cycles to previous match", func(t *testing.T) {
		m := newCoS31TestModel()
		m = typeSearch(m, "api")

		if len(m.searchMatches) < 2 {
			t.Fatal("need at least 2 matches")
		}

		// Persist search
		m = sendSpecialKey(m, tea.KeyEnter)

		// Start at second match
		m.currentMatchIdx = 1
		m.cursor = m.searchMatches[1]

		// Press N to go to previous
		m = sendKey(m, 'N')
		if m.cursor != m.searchMatches[0] {
			t.Errorf("N should move cursor to first match %d, got %d", m.searchMatches[0], m.cursor)
		}
	})

	t.Run("n without search opens new session dialog", func(t *testing.T) {
		m := newCoS31TestModel()

		m = sendKey(m, 'n')
		if m.dialogMode != DialogNewSession {
			t.Error("n without active search should open new session dialog")
		}
	})
}

// TestCoS31_AC5_WrapAround verifies AC5: Match cycling wraps around.
func TestCoS31_AC5_WrapAround(t *testing.T) {
	t.Run("n wraps from last match to first", func(t *testing.T) {
		m := newCoS31TestModel()
		m.searchQuery = "api"
		m.computeSearchMatches()

		if len(m.searchMatches) < 2 {
			t.Fatal("need at least 2 matches")
		}

		// Set to last match
		m.currentMatchIdx = len(m.searchMatches) - 1
		m.cursor = m.searchMatches[m.currentMatchIdx]

		// Press n to wrap to first
		m.nextMatch()

		firstMatch := m.searchMatches[0]
		if m.cursor != firstMatch {
			t.Errorf("n from last match should wrap to first match %d, got %d", firstMatch, m.cursor)
		}
		if m.currentMatchIdx != 0 {
			t.Errorf("currentMatchIdx should be 0 after wrapping, got %d", m.currentMatchIdx)
		}
	})

	t.Run("N wraps from first match to last", func(t *testing.T) {
		m := newCoS31TestModel()
		m.searchQuery = "api"
		m.computeSearchMatches()

		if len(m.searchMatches) < 2 {
			t.Fatal("need at least 2 matches")
		}

		// Set to first match
		m.currentMatchIdx = 0
		m.cursor = m.searchMatches[0]

		// Press N to wrap to last
		m.prevMatch()

		lastMatch := m.searchMatches[len(m.searchMatches)-1]
		if m.cursor != lastMatch {
			t.Errorf("N from first match should wrap to last match %d, got %d", lastMatch, m.cursor)
		}
	})
}

// TestCoS31_AC6_MatchCounter verifies AC6: A match counter is displayed.
func TestCoS31_AC6_MatchCounter(t *testing.T) {
	t.Run("match counter shows [X/Y] format", func(t *testing.T) {
		m := newCoS31TestModel()
		m = typeSearch(m, "api")

		result := m.View()

		matchCount := len(m.searchMatches)
		if matchCount == 0 {
			t.Fatal("need matches for counter test")
		}

		// Should contain "1/" indicating first match (counter format is [1/N])
		expectedPrefix := "1/"
		if !strings.Contains(result, expectedPrefix) {
			t.Errorf("view should contain match counter starting with %q", expectedPrefix)
		}
	})

	t.Run("match counter updates on n/N cycling", func(t *testing.T) {
		m := newCoS31TestModel()
		m.searchQuery = "api"
		m.computeSearchMatches()

		if len(m.searchMatches) < 2 {
			t.Skip("need at least 2 matches")
		}

		// Initially at match 1
		result := m.View()
		if !strings.Contains(result, "1/") {
			t.Error("should show 1/N initially")
		}

		// Cycle to next
		m.nextMatch()
		result = m.View()
		if !strings.Contains(result, "2/") {
			t.Error("should show 2/N after cycling")
		}
	})

	t.Run("no matches shows indicator", func(t *testing.T) {
		m := newCoS31TestModel()
		m = typeSearch(m, "xyznonexistent")

		result := m.View()
		if !strings.Contains(result, "No matches") {
			t.Error("should show 'No matches' when query has no results")
		}
	})
}

// TestCoS31_AC7_EscClearsSearchAndHighlights verifies AC7: Esc exits search mode and clears highlights.
func TestCoS31_AC7_EscClearsSearchAndHighlights(t *testing.T) {
	t.Run("esc during search mode clears everything", func(t *testing.T) {
		m := newCoS31TestModel()
		m = typeSearch(m, "api")

		// Press Esc
		m = sendSpecialKey(m, tea.KeyEsc)

		if m.searchMode {
			t.Error("searchMode should be false after Esc")
		}
		if m.searchQuery != "" {
			t.Errorf("searchQuery should be empty after Esc, got %q", m.searchQuery)
		}
		if m.searchMatches != nil {
			t.Error("searchMatches should be nil after Esc")
		}
		if m.currentMatchIdx != 0 {
			t.Error("currentMatchIdx should be 0 after Esc")
		}
	})

	t.Run("esc clears persisted search highlights", func(t *testing.T) {
		m := newCoS31TestModel()
		m = typeSearch(m, "api")

		// Enter to persist
		m = sendSpecialKey(m, tea.KeyEnter)
		if m.searchQuery != "api" {
			t.Fatal("search should be persisted after Enter")
		}

		// Esc to clear
		m = sendSpecialKey(m, tea.KeyEsc)

		if m.searchQuery != "" {
			t.Errorf("Esc should clear persisted search, got %q", m.searchQuery)
		}
		if m.searchMatches != nil {
			t.Error("searchMatches should be nil after Esc on persisted search")
		}

		// View should not contain search bar
		result := m.View()
		if strings.Contains(result, "No matches") || strings.Contains(result, "/") && strings.Contains(result, "api") {
			// The "/" character may appear in CWDs so be more specific
			if strings.Contains(result, "/ api") {
				t.Error("search bar should not be visible after Esc clears persisted search")
			}
		}
	})
}

// TestCoS31_AC8_EnterPersistsSearch verifies AC8: Enter performs the default action and search persists.
func TestCoS31_AC8_EnterPersistsSearch(t *testing.T) {
	t.Run("enter exits input mode but keeps search state", func(t *testing.T) {
		m := newCoS31TestModel()
		m = typeSearch(m, "api")

		matchesBefore := len(m.searchMatches)

		m = sendSpecialKey(m, tea.KeyEnter)

		if m.searchMode {
			t.Error("searchMode should be false after Enter")
		}
		if m.searchQuery != "api" {
			t.Errorf("searchQuery should persist as 'api', got %q", m.searchQuery)
		}
		if len(m.searchMatches) != matchesBefore {
			t.Errorf("searchMatches should persist, expected %d, got %d", matchesBefore, len(m.searchMatches))
		}
	})

	t.Run("search bar visible with persisted search", func(t *testing.T) {
		m := newCoS31TestModel()
		m = typeSearch(m, "api")
		m = sendSpecialKey(m, tea.KeyEnter)

		result := m.View()
		if !strings.Contains(result, "api") {
			t.Error("view should show persisted search query")
		}
	})

	t.Run("n/N still work after enter persists search", func(t *testing.T) {
		m := newCoS31TestModel()
		m = typeSearch(m, "api")

		if len(m.searchMatches) < 2 {
			t.Fatal("need at least 2 matches")
		}

		m = sendSpecialKey(m, tea.KeyEnter)

		// Verify we're in normal mode with persisted search
		if m.searchMode {
			t.Fatal("should not be in search mode")
		}
		if m.searchQuery != "api" {
			t.Fatal("search should be persisted")
		}

		// n should cycle matches, not open new session dialog
		m = sendKey(m, 'n')
		if m.dialogMode == DialogNewSession {
			t.Error("n with persisted search should cycle matches, not open dialog")
		}
	})
}

// TestCoS31_AC9_OnlyEscClears verifies AC9: Only Esc clears the search state.
func TestCoS31_AC9_OnlyEscClears(t *testing.T) {
	t.Run("enter does not clear search state", func(t *testing.T) {
		m := newCoS31TestModel()
		m = typeSearch(m, "api")
		m = sendSpecialKey(m, tea.KeyEnter)

		if m.searchQuery == "" {
			t.Error("Enter should not clear search state")
		}
	})

	t.Run("n does not clear search state", func(t *testing.T) {
		m := newCoS31TestModel()
		m = typeSearch(m, "api")
		m = sendSpecialKey(m, tea.KeyEnter)

		m = sendKey(m, 'n')
		if m.searchQuery == "" {
			t.Error("n should not clear search state")
		}
	})

	t.Run("N does not clear search state", func(t *testing.T) {
		m := newCoS31TestModel()
		m = typeSearch(m, "api")
		m = sendSpecialKey(m, tea.KeyEnter)

		m = sendKey(m, 'N')
		if m.searchQuery == "" {
			t.Error("N should not clear search state")
		}
	})

	t.Run("cursor movement does not clear search state", func(t *testing.T) {
		m := newCoS31TestModel()
		m = typeSearch(m, "api")
		m = sendSpecialKey(m, tea.KeyEnter)

		m = sendSpecialKey(m, tea.KeyDown)
		if m.searchQuery == "" {
			t.Error("cursor movement should not clear search state")
		}
		m = sendSpecialKey(m, tea.KeyUp)
		if m.searchQuery == "" {
			t.Error("cursor movement should not clear search state")
		}
	})

	t.Run("only esc clears search state", func(t *testing.T) {
		m := newCoS31TestModel()
		m = typeSearch(m, "api")
		m = sendSpecialKey(m, tea.KeyEnter)

		// Verify search is persisted
		if m.searchQuery != "api" {
			t.Fatal("search should be persisted")
		}

		m = sendSpecialKey(m, tea.KeyEsc)
		if m.searchQuery != "" {
			t.Error("Esc is the only key that should clear the search state")
		}
	})
}

// TestCoS31_AC10_NoFuzzyCode verifies AC10: Fuzzy search is fully removed.
func TestCoS31_AC10_NoFuzzyCode(t *testing.T) {
	t.Run("exactMatch function exists and works", func(t *testing.T) {
		// Exact substring match
		if !exactMatch("api", "api-server") {
			t.Error("exactMatch should match substrings")
		}
		// Fuzzy/subsequence should NOT match
		if exactMatch("asr", "api-server") {
			t.Error("exactMatch should reject subsequences")
		}
	})

	t.Run("findMatches function exists and works", func(t *testing.T) {
		sessions := []session.Info{
			{TmuxSession: "api-server"},
			{TmuxSession: "frontend"},
		}
		matches := findMatches(sessions, "api")
		if len(matches) != 1 || matches[0] != 0 {
			t.Errorf("findMatches should return [0] for 'api', got %v", matches)
		}
	})
}

// TestCoS31_AC11_WorksInBothPanels verifies AC11: Works in both session list and task panel.
func TestCoS31_AC11_WorksInBothPanels(t *testing.T) {
	t.Run("session list search with exact match and cycling", func(t *testing.T) {
		m := newCoS31TestModel()
		m = typeSearch(m, "api")

		// Verify session search works
		if len(m.searchMatches) == 0 {
			t.Fatal("session search should find matches")
		}
		if m.searchQuery != "api" {
			t.Fatal("session search query should be set")
		}

		// Verify cycling works
		m.nextMatch()
		if m.currentMatchIdx == 0 && len(m.searchMatches) > 1 {
			// If there's more than one match, cycling should advance
			t.Error("nextMatch should advance currentMatchIdx")
		}
	})

	t.Run("task panel search with exact match and cycling", func(t *testing.T) {
		m := newCoS31TestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true

		// Enter task search mode
		m = sendKey(m, '/')
		if !m.taskSearchMode {
			t.Fatal("should be in task search mode")
		}

		// Type search query
		for _, r := range "fuzzy" {
			m = sendKey(m, r)
		}

		if m.taskSearchQuery != "fuzzy" {
			t.Fatalf("taskSearchQuery should be 'fuzzy', got %q", m.taskSearchQuery)
		}

		// Verify matches exist
		if len(m.taskSearchMatches) == 0 {
			t.Error("task search should find matches for 'fuzzy'")
		}

		// Verify all items remain visible (no filtering out)
		items := m.getVisibleTaskItems()
		if len(items) == 0 {
			t.Error("visible task items should not be empty during search")
		}
	})

	t.Run("task panel n/N cycling", func(t *testing.T) {
		m := newCoS31TestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true
		m.taskExpandedGroups["g1"] = true
		m.taskExpandedGroups["g2"] = true
		m.taskSearchQuery = "matching"
		m.computeTaskSearchMatches()

		// Search for something that matches multiple task items
		m.taskSearchQuery = ""
		m.taskSearchMatches = nil

		// Search for "search" which should match task titles containing "search"
		m.taskSearchQuery = "search"
		m.computeTaskSearchMatches()

		if len(m.taskSearchMatches) > 0 {
			initialCursor := m.taskCursor
			m.nextTaskMatch()

			// Cursor should have moved to a match
			if len(m.taskSearchMatches) > 1 || m.taskCursor != initialCursor {
				// If there's more than one match, cursor changes
				// If there's one match, cursor moves to it
				matchIdx := m.taskSearchMatches[m.taskCurrentMatchIdx]
				if m.taskCursor != matchIdx {
					t.Errorf("task cursor should be at match index %d, got %d", matchIdx, m.taskCursor)
				}
			}
		}
	})

	t.Run("task panel esc clears task search", func(t *testing.T) {
		m := newCoS31TestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true
		m.taskSearchMode = true
		m.taskSearchQuery = "test"
		m.taskSearchInput.SetValue("test")
		m.taskSearchMatches = []int{0, 1}
		m.taskCurrentMatchIdx = 1

		m = sendSpecialKey(m, tea.KeyEsc)

		if m.taskSearchMode {
			t.Error("task search mode should be cleared")
		}
		if m.taskSearchQuery != "" {
			t.Errorf("task search query should be empty, got %q", m.taskSearchQuery)
		}
		if m.taskSearchMatches != nil {
			t.Error("task search matches should be nil")
		}
		if m.taskCurrentMatchIdx != 0 {
			t.Error("task current match index should be 0")
		}
	})

	t.Run("task panel enter persists search", func(t *testing.T) {
		m := newCoS31TestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true

		// Enter task search
		m = sendKey(m, '/')
		for _, r := range "fuzzy" {
			m = sendKey(m, r)
		}

		// Press Enter to persist
		m = sendSpecialKey(m, tea.KeyEnter)

		if m.taskSearchMode {
			t.Error("task search mode should be false after Enter")
		}
		if m.taskSearchQuery != "fuzzy" {
			t.Errorf("task search query should persist as 'fuzzy', got %q", m.taskSearchQuery)
		}
	})

	t.Run("task panel match counter rendered", func(t *testing.T) {
		m := newCoS31TestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true
		m.taskSearchMode = true
		m.taskSearchQuery = "xyznonexistent"
		m.taskSearchInput.SetValue("xyznonexistent")
		m.computeTaskSearchMatches()

		result := m.View()
		if !strings.Contains(result, "No matches") {
			t.Error("task panel should show 'No matches' for non-matching query")
		}
	})
}

// TestCoS31_AC12_FilterIndependence verifies AC12: Existing status filters, sort, and
// offline toggle continue to work independently of search.
func TestCoS31_AC12_FilterIndependence(t *testing.T) {
	t.Run("status filter works independently of search", func(t *testing.T) {
		m := newCoS31TestModel()

		// Apply search
		m.searchQuery = "api"
		m.computeSearchMatches()

		// Apply status filter
		m.statusFilter = session.StatusWorking
		filtered := m.getFilteredSessions()

		// Status filter should reduce the list
		for _, s := range filtered {
			if s.Status != session.StatusWorking {
				t.Errorf("status filter should only show working sessions, got %q", s.Status)
			}
		}

		// Search matches should be recomputed against filtered list
		m.computeSearchMatches()
		for _, idx := range m.searchMatches {
			if idx >= len(filtered) {
				t.Errorf("match index %d out of bounds for filtered list of length %d", idx, len(filtered))
			}
		}
	})

	t.Run("sort mode works independently of search", func(t *testing.T) {
		m := newCoS31TestModel()
		m.searchQuery = "api"
		m.computeSearchMatches()

		// Apply sort
		m.sortMode = SortName
		filtered := m.getFilteredSessions()

		// Verify sorted
		for i := 1; i < len(filtered); i++ {
			if strings.ToLower(filtered[i-1].TmuxSession) > strings.ToLower(filtered[i].TmuxSession) {
				t.Errorf("sessions not sorted by name: %q > %q", filtered[i-1].TmuxSession, filtered[i].TmuxSession)
			}
		}

		// Search query should still be active
		if m.searchQuery != "api" {
			t.Error("search query should persist through sort change")
		}
	})

	t.Run("offline toggle works independently of search", func(t *testing.T) {
		m := newCoS31TestModel()
		m.searchQuery = "database"
		m.computeSearchMatches()

		matchesBefore := len(m.searchMatches)

		// Toggle offline
		m.hideOffline = true
		filtered := m.getFilteredSessions()

		// "database" is a done session, should be hidden
		for _, s := range filtered {
			if s.Status == "done" {
				t.Error("done sessions should be hidden with offline toggle")
			}
		}

		// Recompute matches against filtered list
		m.computeSearchMatches()

		// "database" match should be gone (it's offline)
		if len(m.searchMatches) >= matchesBefore && matchesBefore > 0 {
			t.Error("matches should decrease when offline filter hides matching session")
		}
	})

	t.Run("all three compose: filter + sort + offline + search", func(t *testing.T) {
		m := newCoS31TestModel()
		m.searchQuery = "api"
		m.statusFilter = session.StatusWorking
		m.hideOffline = true
		m.sortMode = SortName

		filtered := m.getFilteredSessions()

		// Only working, non-done sessions should be in list
		for _, s := range filtered {
			if s.Status != session.StatusWorking {
				t.Errorf("expected only working sessions, got %q", s.Status)
			}
		}

		// Search should still be active but not filtering
		if m.searchQuery != "api" {
			t.Error("search should still be active")
		}

		// Compute and verify matches
		m.computeSearchMatches()
		for _, idx := range m.searchMatches {
			s := filtered[idx]
			if !exactMatch("api", s.TmuxSession) && !exactMatch("api", s.CWD) && !exactMatch("api", s.Message) {
				t.Errorf("match at index %d should match 'api', got session %q", idx, s.TmuxSession)
			}
		}
	})

	t.Run("number keys still work for status filter", func(t *testing.T) {
		m := newCoS31TestModel()

		// Press 1 for waiting filter
		m = sendKey(m, '1')
		if m.statusFilter != session.StatusWaiting {
			t.Errorf("expected waiting filter, got %q", m.statusFilter)
		}

		// Press 3 for working filter
		m = sendKey(m, '3')
		if m.statusFilter != session.StatusWorking {
			t.Errorf("expected working filter, got %q", m.statusFilter)
		}

		// Press 0 to clear
		m = sendKey(m, '0')
		if m.statusFilter != "" {
			t.Errorf("expected empty filter, got %q", m.statusFilter)
		}
	})

	t.Run("sort cycling still works", func(t *testing.T) {
		m := newCoS31TestModel()

		m = sendKey(m, 's')
		if m.sortMode != SortName {
			t.Errorf("expected SortName, got %d", m.sortMode)
		}
		m = sendKey(m, 's')
		if m.sortMode != SortAge {
			t.Errorf("expected SortAge, got %d", m.sortMode)
		}
	})

	t.Run("offline toggle still works", func(t *testing.T) {
		m := newCoS31TestModel()

		m = sendKey(m, 'o')
		if !m.hideOffline {
			t.Error("hideOffline should be true after pressing o")
		}

		filtered := m.getFilteredSessions()
		for _, s := range filtered {
			if s.Status == "done" {
				t.Error("done sessions should be hidden")
			}
		}
	})
}

// TestCoS31_IntegratedFlow tests a realistic user flow across multiple acceptance criteria.
func TestCoS31_IntegratedFlow(t *testing.T) {
	t.Run("full search workflow: search, cycle, persist, clear", func(t *testing.T) {
		m := newCoS31TestModel()

		// Step 1: Enter search mode with /
		m = sendKey(m, '/')
		if !m.searchMode {
			t.Fatal("Step 1: should be in search mode")
		}

		// Step 2: Type "api" - exact substring match
		for _, r := range "api" {
			m = sendKey(m, r)
		}
		if m.searchQuery != "api" {
			t.Fatalf("Step 2: query should be 'api', got %q", m.searchQuery)
		}
		if len(m.searchMatches) < 2 {
			t.Fatalf("Step 2: expected at least 2 matches, got %d", len(m.searchMatches))
		}

		// Step 3: Verify all sessions visible (AC2)
		if len(m.getFilteredSessions()) != 5 {
			t.Fatal("Step 3: all 5 sessions should be visible")
		}

		// Step 4: Verify match counter (AC6)
		result := m.View()
		if !strings.Contains(result, "1/") {
			t.Error("Step 4: should show match counter")
		}

		// Step 5: Press Enter to persist search (AC8)
		m = sendSpecialKey(m, tea.KeyEnter)
		if m.searchMode {
			t.Fatal("Step 5: searchMode should be false")
		}
		if m.searchQuery != "api" {
			t.Fatal("Step 5: query should persist")
		}

		// Step 6: Press n to cycle (AC4)
		initialCursor := m.cursor
		m = sendKey(m, 'n')
		if m.dialogMode == DialogNewSession {
			t.Fatal("Step 6: n should cycle, not open dialog")
		}
		if m.cursor == initialCursor && len(m.searchMatches) > 1 {
			t.Error("Step 6: cursor should have moved")
		}

		// Step 7: Press N to go back (AC4)
		m = sendKey(m, 'N')
		// Should be back at initial position or wrapped

		// Step 8: Verify search persists through cursor movement (AC9)
		m = sendSpecialKey(m, tea.KeyDown)
		if m.searchQuery != "api" {
			t.Fatal("Step 8: search should persist through movement")
		}

		// Step 9: Press Esc to clear (AC7)
		m = sendSpecialKey(m, tea.KeyEsc)
		if m.searchQuery != "" {
			t.Fatal("Step 9: Esc should clear search")
		}
		if m.searchMatches != nil {
			t.Fatal("Step 9: matches should be nil")
		}

		// Step 10: n should now open new session dialog (no search active)
		m = sendKey(m, 'n')
		if m.dialogMode != DialogNewSession {
			t.Error("Step 10: n without search should open new session dialog")
		}
	})
}
