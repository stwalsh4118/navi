package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/pm"
	"github.com/stwalsh4118/navi/internal/task"
)

func TestPMRefreshFlow_OpenRunOutputCycle(t *testing.T) {
	now := time.Now().UTC()
	m := Model{
		searchInput:        initSearchInput(),
		taskSearchInput:    initTaskSearchInput(),
		pmEngine:           pm.NewEngine(),
		pmTaskResults:      make(map[string]*task.ProviderResult),
		pmExpandedProjects: make(map[string]bool),
		pmOutput: &pm.PMOutput{
			Snapshots: []pm.ProjectSnapshot{{
				ProjectName:   "proj-existing",
				ProjectDir:    "/tmp/proj-existing",
				CurrentPBIID:  "54",
				SessionStatus: "working",
				LastActivity:  now,
			}},
			Events: []pm.Event{{
				Type:        pm.EventCommit,
				ProjectName: "proj-existing",
				Timestamp:   now,
				Payload:     map[string]string{"new_head_sha": "abcdef1"},
			}},
		},
	}

	openedModel, openCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}})
	opened := openedModel.(Model)

	if openCmd == nil {
		t.Fatalf("expected immediate PM run command when opening PM view")
	}
	if !opened.pmRunInFlight {
		t.Fatalf("expected pmRunInFlight to be true after opening PM view")
	}

	loadingView := opened.renderPMView(120, 20)
	if !strings.Contains(loadingView, pmRefreshingLabel) {
		t.Fatalf("expected PM loading indicator while PM run is in flight")
	}
	if !strings.Contains(loadingView, "proj-existing") {
		t.Fatalf("expected existing PM project data to remain visible during refresh")
	}

	newOutput := &pm.PMOutput{
		Snapshots: []pm.ProjectSnapshot{{
			ProjectName:     "proj-existing",
			ProjectDir:      "/tmp/proj-existing",
			CurrentPBIID:    "55",
			CurrentPBITitle: "PM Refresh UX",
			SessionStatus:   "working",
			LastActivity:    now.Add(30 * time.Second),
		}},
	}

	updatedModel, _ := opened.Update(pmOutputMsg{output: newOutput, err: nil})
	updated := updatedModel.(Model)

	if updated.pmRunInFlight {
		t.Fatalf("expected pmRunInFlight to be false after pmOutputMsg")
	}
	if updated.pmOutput == nil || len(updated.pmOutput.Snapshots) != 1 {
		t.Fatalf("expected PM output snapshot to be updated")
	}
	if updated.pmOutput.Snapshots[0].CurrentPBIID != "55" {
		t.Fatalf("expected current PBI to update to 55, got %q", updated.pmOutput.Snapshots[0].CurrentPBIID)
	}

	finalView := updated.renderPMView(120, 20)
	if strings.Contains(finalView, pmRefreshingLabel) {
		t.Fatalf("expected loading indicator to clear after pmOutputMsg")
	}
	if !strings.Contains(finalView, "55: PM Refresh UX") {
		t.Fatalf("expected updated PBI snapshot content in PM view")
	}
}

func TestPMRefreshFlow_ManualRefreshCycle(t *testing.T) {
	cache := task.NewResultCache()
	cache.Set("/tmp/proj", &task.ProviderResult{}, nil)

	m := Model{
		pmViewVisible:      true,
		pmEngine:           pm.NewEngine(),
		pmTaskResults:      make(map[string]*task.ProviderResult),
		taskCache:          cache,
		taskGlobalConfig:   &task.GlobalConfig{},
		taskProjectConfigs: []task.ProjectConfig{{ProjectDir: "/tmp/proj", Tasks: task.ProjectTaskConfig{Provider: "test"}}},
	}

	refreshModel, refreshCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	refreshing := refreshModel.(Model)

	if refreshCmd == nil {
		t.Fatalf("expected command when pressing r in PM view")
	}
	if !refreshing.pmRunInFlight {
		t.Fatalf("expected pmRunInFlight true during manual refresh")
	}
	if _, ok := cache.Get("/tmp/proj", 5*time.Minute); ok {
		t.Fatalf("expected cache to be invalidated on manual PM refresh")
	}

	doneModel, _ := refreshing.Update(pmOutputMsg{output: &pm.PMOutput{}, err: nil})
	done := doneModel.(Model)
	if done.pmRunInFlight {
		t.Fatalf("expected pmRunInFlight false after PM output arrives")
	}
}
