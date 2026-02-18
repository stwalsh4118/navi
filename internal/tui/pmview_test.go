package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/pm"
	"github.com/stwalsh4118/navi/internal/session"
	"github.com/stwalsh4118/navi/internal/task"
)

func TestPMViewToggleAndFooter(t *testing.T) {
	m := Model{
		width:              120,
		height:             40,
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

	b, p, e := pmZoneHeights(40)
	if b == 0 || p == 0 || e == 0 {
		t.Fatalf("expected all zones for normal height")
	}
	b, _, _ = pmZoneHeights(14)
	if b != 0 {
		t.Fatalf("expected briefing hidden for very short terminal")
	}

	narrow := updated.renderPMView(70, 20)
	if !strings.Contains(narrow, "Terminal too narrow") {
		t.Fatalf("expected narrow terminal warning")
	}
}
