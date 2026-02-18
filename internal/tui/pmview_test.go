package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/stwalsh4118/navi/internal/pm"
	"github.com/stwalsh4118/navi/internal/session"
	"github.com/stwalsh4118/navi/internal/task"
)

func TestPMViewToggleAndFooter(t *testing.T) {
	m := Model{
		width:              120,
		height:             40,
		searchInput:        initSearchInput(),
		taskSearchInput:    initTaskSearchInput(),
		previewVisible:     true,
		taskPanelVisible:   true,
		searchQuery:        "abc",
		pmExpandedProjects: make(map[string]bool),
	}

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}})
	updated := updatedModel.(Model)

	if !updated.pmViewVisible {
		t.Fatalf("expected PM view to be visible after P toggle")
	}
	if updated.previewVisible {
		t.Fatalf("expected preview to be hidden when PM view opens")
	}
	if updated.taskPanelVisible {
		t.Fatalf("expected task panel to be hidden when PM view opens")
	}
	if updated.searchQuery != "" {
		t.Fatalf("expected search query to be cleared when PM view opens")
	}

	footer := updated.renderFooter()
	for _, expected := range []string{"P close", "Tab focus", "select"} {
		if !strings.Contains(footer, expected) {
			t.Fatalf("expected PM footer to contain %q, got: %s", expected, footer)
		}
	}
}

func TestPMViewImmediateRefreshOnOpen(t *testing.T) {
	t.Run("open PM view starts PM run when engine is ready", func(t *testing.T) {
		m := Model{
			width:           120,
			height:          40,
			searchInput:     initSearchInput(),
			taskSearchInput: initTaskSearchInput(),
			pmEngine:        pm.NewEngine(),
			pmTaskResults:   make(map[string]*task.ProviderResult),
		}

		updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}})
		updated := updatedModel.(Model)

		if !updated.pmViewVisible {
			t.Fatalf("expected PM view to be visible after opening")
		}
		if !updated.pmRunInFlight {
			t.Fatalf("expected pmRunInFlight to be true after opening PM view")
		}
		if cmd == nil {
			t.Fatalf("expected immediate PM refresh command when opening PM view")
		}
	})

	t.Run("open PM view skips run when already in flight", func(t *testing.T) {
		m := Model{
			width:           120,
			height:          40,
			searchInput:     initSearchInput(),
			taskSearchInput: initTaskSearchInput(),
			pmEngine:        pm.NewEngine(),
			pmRunInFlight:   true,
		}

		updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}})
		updated := updatedModel.(Model)

		if !updated.pmViewVisible {
			t.Fatalf("expected PM view to be visible after opening")
		}
		if !updated.pmRunInFlight {
			t.Fatalf("expected pmRunInFlight to remain true")
		}
		if cmd != nil {
			t.Fatalf("expected no command when PM run is already in flight")
		}
	})

	t.Run("closing PM view with P returns nil command", func(t *testing.T) {
		m := Model{pmViewVisible: true, pmZoneFocus: pmZoneProjects}

		updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}})
		updated := updatedModel.(Model)

		if updated.pmViewVisible {
			t.Fatalf("expected PM view to close when pressing P while PM view is open")
		}
		if cmd != nil {
			t.Fatalf("expected nil command when closing PM view")
		}
	})
}

func TestPMViewHeaderAndPlaceholder(t *testing.T) {
	m := Model{width: 120, height: 40, pmViewVisible: true}

	header := m.renderHeader()
	if !strings.Contains(header, pmHeaderTitle) {
		t.Fatalf("expected PM header title in header")
	}

	view := m.renderPMView(120, 20)
	if !strings.Contains(view, "No PM briefing yet") {
		t.Fatalf("expected PM briefing placeholder")
	}
	if !strings.Contains(view, "No projects detected") {
		t.Fatalf("expected empty projects placeholder")
	}
	if !strings.Contains(view, "No events yet") {
		t.Fatalf("expected empty events placeholder")
	}
}

func TestPMBriefingLoadingIndicator(t *testing.T) {
	t.Run("shows refreshing when pm run in flight", func(t *testing.T) {
		m := Model{pmRunInFlight: true}
		briefing := m.renderPMBriefing(120, 6)
		if !strings.Contains(briefing, pmRefreshingLabel) {
			t.Fatalf("expected refreshing indicator when pmRunInFlight is true")
		}
	})

	t.Run("single-line briefing shows refreshing when in flight", func(t *testing.T) {
		m := Model{pmRunInFlight: true}
		briefing := m.renderPMBriefing(120, 1)
		if !strings.Contains(briefing, pmRefreshingLabel) {
			t.Fatalf("expected refreshing indicator in single-line briefing")
		}
	})

	t.Run("does not show refreshing when pm run is idle", func(t *testing.T) {
		m := Model{pmRunInFlight: false}
		briefing := m.renderPMBriefing(120, 6)
		if strings.Contains(briefing, pmRefreshingLabel) {
			t.Fatalf("did not expect refreshing indicator when pmRunInFlight is false")
		}
	})

	t.Run("shows refreshing and error when both present", func(t *testing.T) {
		m := Model{pmRunInFlight: true, pmLastError: "boom"}
		briefing := m.renderPMBriefing(120, 6)
		if !strings.Contains(briefing, pmRefreshingLabel) {
			t.Fatalf("expected refreshing indicator when pm run is in flight")
		}
		if !strings.Contains(briefing, "PM refresh error: boom") {
			t.Fatalf("expected PM error text when pmLastError is set")
		}
	})

	t.Run("pm view rendering includes refreshing indicator", func(t *testing.T) {
		m := Model{pmRunInFlight: true}
		view := m.renderPMView(120, 20)
		if !strings.Contains(view, pmRefreshingLabel) {
			t.Fatalf("expected PM view to include refreshing indicator")
		}
	})
}

func TestPMViewManualRefreshKey(t *testing.T) {
	t.Run("r triggers PM refresh and invalidates task cache", func(t *testing.T) {
		cache := task.NewResultCache()
		cache.Set("/tmp/proj", &task.ProviderResult{}, nil)
		cache.Set("/tmp/proj-2", &task.ProviderResult{}, nil)

		m := Model{
			pmViewVisible:    true,
			pmZoneFocus:      pmZoneEvents,
			pmEngine:         pm.NewEngine(),
			pmTaskResults:    make(map[string]*task.ProviderResult),
			taskCache:        cache,
			taskGlobalConfig: &task.GlobalConfig{},
			taskProjectConfigs: []task.ProjectConfig{
				{ProjectDir: "/tmp/proj", Tasks: task.ProjectTaskConfig{Provider: "test"}},
			},
		}

		updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
		updated := updatedModel.(Model)

		if !updated.pmRunInFlight {
			t.Fatalf("expected pmRunInFlight to be true after manual refresh")
		}
		if cmd == nil {
			t.Fatalf("expected refresh command from r key in PM view")
		}

		if _, ok := cache.Get("/tmp/proj", 5*time.Minute); ok {
			t.Fatalf("expected task cache entry /tmp/proj to be invalidated before dispatch")
		}
		if _, ok := cache.Get("/tmp/proj-2", 5*time.Minute); ok {
			t.Fatalf("expected task cache entry /tmp/proj-2 to be invalidated before dispatch")
		}
	})

	t.Run("r is ignored while PM run is already in flight", func(t *testing.T) {
		cache := task.NewResultCache()
		cache.Set("/tmp/proj", &task.ProviderResult{}, nil)

		m := Model{
			pmViewVisible: true,
			pmRunInFlight: true,
			pmEngine:      pm.NewEngine(),
			taskCache:     cache,
		}

		updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
		updated := updatedModel.(Model)

		if !updated.pmRunInFlight {
			t.Fatalf("expected pmRunInFlight to remain true")
		}
		if cmd != nil {
			t.Fatalf("expected nil command when pmRunInFlight is already true")
		}
		if _, ok := cache.Get("/tmp/proj", 5*time.Minute); !ok {
			t.Fatalf("expected cache to remain untouched when refresh is ignored")
		}
	})

	t.Run("r is ignored when PM engine is unavailable", func(t *testing.T) {
		cache := task.NewResultCache()
		cache.Set("/tmp/proj", &task.ProviderResult{}, nil)

		m := Model{
			pmViewVisible: true,
			taskCache:     cache,
		}

		updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
		updated := updatedModel.(Model)

		if updated.pmRunInFlight {
			t.Fatalf("expected pmRunInFlight to remain false when PM engine is nil")
		}
		if cmd != nil {
			t.Fatalf("expected nil command when PM engine is unavailable")
		}
		if _, ok := cache.Get("/tmp/proj", 5*time.Minute); !ok {
			t.Fatalf("expected cache to remain untouched when PM engine is unavailable")
		}
	})
}

func TestPMProjectsSortAndSelectionJump(t *testing.T) {
	now := time.Now().UTC()
	m := Model{
		width:              120,
		height:             40,
		pmViewVisible:      true,
		pmZoneFocus:        pmZoneProjects,
		pmExpandedProjects: make(map[string]bool),
		sessions: []session.Info{
			{TmuxSession: "a", CWD: "/tmp/working/sub", Timestamp: now.Unix()},
			{TmuxSession: "b", CWD: "/tmp/idle", Timestamp: now.Unix()},
		},
		pmOutput: &pm.PMOutput{Snapshots: []pm.ProjectSnapshot{
			{ProjectName: "idle", ProjectDir: "/tmp/idle", SessionStatus: "idle", LastActivity: now.Add(-10 * time.Minute)},
			{ProjectName: "permission", ProjectDir: "/tmp/permission", SessionStatus: "permission", LastActivity: now.Add(-30 * time.Minute)},
			{ProjectName: "working", ProjectDir: "/tmp/working", SessionStatus: "working", LastActivity: now.Add(-2 * time.Minute)},
		}},
	}

	projects := m.sortedPMSnapshots()
	if projects[0].ProjectName != "permission" {
		t.Fatalf("expected permission project first, got %s", projects[0].ProjectName)
	}

	m.pmProjectCursor = 1
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedModel.(Model)
	if updated.pmViewVisible {
		t.Fatalf("expected PM view to close after selecting project")
	}
	if updated.pmProjectFilterDir != "/tmp/working" {
		t.Fatalf("expected project filter to be set, got %q", updated.pmProjectFilterDir)
	}

	filtered := updated.getFilteredSessions()
	if len(filtered) != 1 || filtered[0].TmuxSession != "a" {
		t.Fatalf("expected filtered sessions to include only selected project sessions")
	}
}

func TestPMEventsOrderAndScroll(t *testing.T) {
	now := time.Now().UTC()
	events := []pm.Event{
		{Type: pm.EventCommit, Timestamp: now.Add(-5 * time.Minute), ProjectName: "proj", Payload: map[string]string{"commits": "abc123 first commit"}},
		{Type: pm.EventTaskCompleted, Timestamp: now.Add(-1 * time.Minute), ProjectName: "proj", Payload: map[string]string{"old_done": "1", "new_done": "2"}},
		{Type: pm.EventSessionStatusChange, Timestamp: now.Add(-3 * time.Minute), ProjectName: "proj", Payload: map[string]string{"old_status": "working", "new_status": "waiting"}},
	}
	for i := 0; i < 8; i++ {
		events = append(events, pm.Event{Type: pm.EventCommit, Timestamp: now.Add(time.Duration(-10-i) * time.Minute), ProjectName: "proj", Payload: map[string]string{"new_head_sha": "abcdef1"}})
	}
	m := Model{
		width:               120,
		height:              20,
		pmViewVisible:       true,
		pmZoneFocus:         pmZoneEvents,
		pmEventScrollOffset: 0,
		pmOutput:            &pm.PMOutput{Events: events},
	}

	rendered := m.renderPMEvents(120, 8)
	firstIdx := strings.Index(rendered, string(pm.EventTaskCompleted))
	secondIdx := strings.Index(rendered, string(pm.EventSessionStatusChange))
	thirdIdx := strings.Index(rendered, string(pm.EventCommit))
	if !(firstIdx >= 0 && secondIdx > firstIdx && thirdIdx > secondIdx) {
		t.Fatalf("expected reverse chronological event order")
	}

	updatedModel, _ := m.updatePMView(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated := updatedModel.(Model)
	if updated.pmEventScrollOffset <= 0 {
		t.Fatalf("expected event scroll offset to increase on j")
	}
}

func TestPMTabFocusAndEscClose(t *testing.T) {
	m := Model{pmViewVisible: true, pmZoneFocus: pmZoneProjects}

	updatedModel, _ := m.updatePMView(tea.KeyMsg{Type: tea.KeyTab})
	updated := updatedModel.(Model)
	if updated.pmZoneFocus != pmZoneEvents {
		t.Fatalf("expected tab to switch PM focus to events")
	}

	closedModel, _ := updated.updatePMView(tea.KeyMsg{Type: tea.KeyEsc})
	closed := closedModel.(Model)
	if closed.pmViewVisible {
		t.Fatalf("expected esc to close PM view")
	}
}

func TestPMProjectExpansionAndResponsiveLayout(t *testing.T) {
	m := Model{
		width:              120,
		height:             40,
		pmViewVisible:      true,
		pmZoneFocus:        pmZoneProjects,
		pmProjectCursor:    0,
		pmExpandedProjects: make(map[string]bool),
		pmTaskResults: map[string]*task.ProviderResult{
			"/tmp/proj": {
				Tasks: []task.Task{
					{ID: "47-1", Title: "Build PM view", Status: "InProgress"},
				},
			},
		},
		pmOutput: &pm.PMOutput{Snapshots: []pm.ProjectSnapshot{
			{ProjectName: "proj", ProjectDir: "/tmp/proj", Branch: "feature/pbi-47", Dirty: true, HeadSHA: "abcdef1234", CommitsAhead: 2},
		}},
	}

	updatedModel, _ := m.updatePMView(tea.KeyMsg{Type: tea.KeySpace})
	updated := updatedModel.(Model)
	if !updated.pmExpandedProjects["/tmp/proj"] {
		t.Fatalf("expected space to expand selected project")
	}

	projects := updated.renderPMProjects(120, 10)
	if !strings.Contains(projects, "branch:") || !strings.Contains(projects, "tasks:") {
		t.Fatalf("expected expanded project details in render")
	}

	b, p, e := pmZoneHeights(40, 10)
	if b == 0 || p == 0 || e == 0 {
		t.Fatalf("expected all zones for normal height")
	}
	b, _, _ = pmZoneHeights(14, 10)
	if b != 0 {
		t.Fatalf("expected briefing hidden for very short terminal")
	}

	narrow := updated.renderPMView(70, 20)
	if !strings.Contains(narrow, "Terminal too narrow") {
		t.Fatalf("expected narrow terminal warning")
	}
}

func TestZeroClearsProjectFilter(t *testing.T) {
	now := time.Now().Unix()
	m := Model{
		width:              120,
		height:             40,
		pmProjectFilterDir: "/tmp/proj-a",
		sessions: []session.Info{
			{TmuxSession: "a", CWD: "/tmp/proj-a", Status: "working", Timestamp: now},
			{TmuxSession: "b", CWD: "/tmp/proj-b", Status: "done", Timestamp: now},
		},
		cursor: 0,
	}

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	updated := updatedModel.(Model)

	if updated.pmProjectFilterDir != "" {
		t.Fatalf("expected project filter to be cleared by 0 key")
	}
}

func TestPMTruncateANSIRespectsDisplayWidth(t *testing.T) {
	styled := greenStyle.Render("abcdef") + " " + boldStyle.Render("ghij")
	truncated := pmTruncateANSI(styled, 5)
	if lipgloss.Width(truncated) > 5 {
		t.Fatalf("expected ANSI truncate width <= 5, got %d", lipgloss.Width(truncated))
	}
}

func TestPMProjectsShowsBelowIndicatorWhenExpansionConsumesSlots(t *testing.T) {
	now := time.Now().UTC()
	m := Model{
		width:              120,
		height:             40,
		pmViewVisible:      true,
		pmZoneFocus:        pmZoneProjects,
		pmProjectCursor:    0,
		pmExpandedProjects: map[string]bool{"/tmp/proj-a": true},
		pmTaskResults: map[string]*task.ProviderResult{
			"/tmp/proj-a": {
				Tasks: []task.Task{
					{ID: "47-1", Title: "Task one", Status: "InProgress"},
					{ID: "47-2", Title: "Task two", Status: "Review"},
				},
			},
		},
		pmOutput: &pm.PMOutput{Snapshots: []pm.ProjectSnapshot{
			{ProjectName: "A", ProjectDir: "/tmp/proj-a", Branch: "feature/a", HeadSHA: "abcdef1", LastActivity: now},
			{ProjectName: "B", ProjectDir: "/tmp/proj-b", Branch: "feature/b", HeadSHA: "abcdef2", LastActivity: now.Add(-1 * time.Minute)},
			{ProjectName: "C", ProjectDir: "/tmp/proj-c", Branch: "feature/c", HeadSHA: "abcdef3", LastActivity: now.Add(-2 * time.Minute)},
		}},
	}

	projects := m.renderPMProjects(120, 6)
	if !strings.Contains(projects, "below") {
		t.Fatalf("expected below indicator when expansion hides project rows")
	}
}
