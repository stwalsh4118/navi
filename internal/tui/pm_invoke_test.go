package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/pm"
	"github.com/stwalsh4118/navi/internal/task"
)

func TestPMCheckTrigger(t *testing.T) {
	t.Run("task_completed triggers invocation", func(t *testing.T) {
		events := []pm.Event{{Type: pm.EventTaskCompleted}}
		trigger := pmCheckTrigger(events)
		if trigger != pm.TriggerTaskCompleted {
			t.Fatalf("expected task_completed trigger, got %q", trigger)
		}
	})

	t.Run("commit triggers invocation", func(t *testing.T) {
		events := []pm.Event{{Type: pm.EventCommit}}
		trigger := pmCheckTrigger(events)
		if trigger != pm.TriggerCommit {
			t.Fatalf("expected commit trigger, got %q", trigger)
		}
	})

	t.Run("session_status_change does not trigger", func(t *testing.T) {
		events := []pm.Event{{Type: pm.EventSessionStatusChange}}
		trigger := pmCheckTrigger(events)
		if trigger != "" {
			t.Fatalf("expected no trigger, got %q", trigger)
		}
	})

	t.Run("empty events no trigger", func(t *testing.T) {
		trigger := pmCheckTrigger(nil)
		if trigger != "" {
			t.Fatalf("expected no trigger for nil events, got %q", trigger)
		}
	})

	t.Run("first matching event wins", func(t *testing.T) {
		events := []pm.Event{
			{Type: pm.EventSessionStatusChange},
			{Type: pm.EventCommit},
			{Type: pm.EventTaskCompleted},
		}
		trigger := pmCheckTrigger(events)
		if trigger != pm.TriggerCommit {
			t.Fatalf("expected commit trigger (first match), got %q", trigger)
		}
	})
}

func TestPMOutputMsgTriggersInvocation(t *testing.T) {
	t.Run("task_completed event in output triggers invoke", func(t *testing.T) {
		m := Model{
			pmEngine:  pm.NewEngine(),
			pmInvoker: &pm.Invoker{}, // non-nil stub
		}

		output := &pm.PMOutput{
			Events: []pm.Event{{Type: pm.EventTaskCompleted, Timestamp: time.Now()}},
		}
		updatedModel, cmd := m.Update(pmOutputMsg{output: output, err: nil})
		updated := updatedModel.(Model)

		if !updated.pmInvokeInFlight {
			t.Fatal("expected pmInvokeInFlight=true after trigger event")
		}
		if cmd == nil {
			t.Fatal("expected pmInvokeCmd to be returned")
		}
	})

	t.Run("non-trigger events do not invoke", func(t *testing.T) {
		m := Model{
			pmEngine:  pm.NewEngine(),
			pmInvoker: &pm.Invoker{},
		}

		output := &pm.PMOutput{
			Events: []pm.Event{{Type: pm.EventSessionStatusChange}},
		}
		updatedModel, cmd := m.Update(pmOutputMsg{output: output, err: nil})
		updated := updatedModel.(Model)

		if updated.pmInvokeInFlight {
			t.Fatal("expected pmInvokeInFlight=false for non-trigger events")
		}
		if cmd != nil {
			t.Fatal("expected no command for non-trigger events")
		}
	})

	t.Run("in-flight guard prevents second invocation", func(t *testing.T) {
		m := Model{
			pmEngine:         pm.NewEngine(),
			pmInvoker:        &pm.Invoker{},
			pmInvokeInFlight: true, // already running
		}

		output := &pm.PMOutput{
			Events: []pm.Event{{Type: pm.EventTaskCompleted}},
		}
		updatedModel, cmd := m.Update(pmOutputMsg{output: output, err: nil})
		updated := updatedModel.(Model)

		if cmd != nil {
			t.Fatal("expected no command when invocation already in flight")
		}
		if !updated.pmInvokeInFlight {
			t.Fatal("expected in-flight flag to remain true")
		}
	})

	t.Run("nil invoker skips trigger", func(t *testing.T) {
		m := Model{
			pmEngine:  pm.NewEngine(),
			pmInvoker: nil,
		}

		output := &pm.PMOutput{
			Events: []pm.Event{{Type: pm.EventTaskCompleted}},
		}
		updatedModel, cmd := m.Update(pmOutputMsg{output: output, err: nil})
		updated := updatedModel.(Model)

		if updated.pmInvokeInFlight {
			t.Fatal("expected no invocation with nil invoker")
		}
		if cmd != nil {
			t.Fatal("expected no command with nil invoker")
		}
	})
}

func TestPMInvokeMsgUpdatesModel(t *testing.T) {
	t.Run("successful invocation stores briefing", func(t *testing.T) {
		m := Model{pmInvokeInFlight: true}
		briefing := &pm.PMBriefing{Summary: "test-summary"}

		updatedModel, cmd := m.Update(pmInvokeMsg{briefing: briefing, isStale: false})
		updated := updatedModel.(Model)

		if updated.pmInvokeInFlight {
			t.Fatal("expected pmInvokeInFlight=false after pmInvokeMsg")
		}
		if updated.pmBriefing == nil || updated.pmBriefing.Summary != "test-summary" {
			t.Fatalf("expected briefing to be stored, got %v", updated.pmBriefing)
		}
		if updated.pmBriefingStale {
			t.Fatal("expected stale=false for fresh briefing")
		}
		if cmd != nil {
			t.Fatal("expected no follow-up command")
		}
	})

	t.Run("stale invocation sets stale flag", func(t *testing.T) {
		m := Model{pmInvokeInFlight: true}
		briefing := &pm.PMBriefing{Summary: "cached"}

		updatedModel, _ := m.Update(pmInvokeMsg{briefing: briefing, isStale: true})
		updated := updatedModel.(Model)

		if !updated.pmBriefingStale {
			t.Fatal("expected pmBriefingStale=true for stale briefing")
		}
		if updated.pmBriefing == nil || updated.pmBriefing.Summary != "cached" {
			t.Fatalf("expected cached briefing, got %v", updated.pmBriefing)
		}
	})

	t.Run("failed invocation stores error", func(t *testing.T) {
		m := Model{pmInvokeInFlight: true}

		updatedModel, _ := m.Update(pmInvokeMsg{err: errors.New("invoke failed")})
		updated := updatedModel.(Model)

		if updated.pmInvokeInFlight {
			t.Fatal("expected pmInvokeInFlight=false after error")
		}
		if updated.pmLastError == "" {
			t.Fatal("expected pmLastError to be set")
		}
	})
}

func TestPMOnDemandInvoke(t *testing.T) {
	t.Run("i key triggers on-demand invocation", func(t *testing.T) {
		m := Model{
			pmViewVisible:      true,
			pmEngine:           pm.NewEngine(),
			pmInvoker:          &pm.Invoker{},
			pmTaskResults:      make(map[string]*task.ProviderResult),
			pmExpandedProjects: make(map[string]bool),
			pmOutput: &pm.PMOutput{
				Snapshots: []pm.ProjectSnapshot{{ProjectName: "proj"}},
				Events:    []pm.Event{{Type: pm.EventSessionStatusChange}},
			},
		}

		updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
		updated := updatedModel.(Model)

		if !updated.pmInvokeInFlight {
			t.Fatal("expected pmInvokeInFlight=true after i key")
		}
		if cmd == nil {
			t.Fatal("expected invocation command from i key")
		}
	})

	t.Run("i key skipped when already in flight", func(t *testing.T) {
		m := Model{
			pmViewVisible:    true,
			pmInvoker:        &pm.Invoker{},
			pmInvokeInFlight: true,
		}

		updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
		updated := updatedModel.(Model)

		if cmd != nil {
			t.Fatal("expected no command when already in flight")
		}
		if !updated.pmInvokeInFlight {
			t.Fatal("expected in-flight to remain true")
		}
	})

	t.Run("i key skipped when no invoker", func(t *testing.T) {
		m := Model{
			pmViewVisible: true,
			pmInvoker:     nil,
		}

		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
		if cmd != nil {
			t.Fatal("expected no command when invoker is nil")
		}
	})
}

func TestPMBriefingRendering(t *testing.T) {
	t.Run("renders briefing summary", func(t *testing.T) {
		m := Model{
			pmBriefing: &pm.PMBriefing{
				Summary: "All projects healthy",
			},
		}
		view := m.renderPMBriefing(120, 10)
		if !strings.Contains(view, "PM Briefing") {
			t.Fatal("expected PM Briefing header")
		}
		if !strings.Contains(view, "All projects healthy") {
			t.Fatal("expected summary text")
		}
	})

	t.Run("renders stale indicator", func(t *testing.T) {
		m := Model{
			pmBriefing:      &pm.PMBriefing{Summary: "old data"},
			pmBriefingStale: true,
		}
		view := m.renderPMBriefing(120, 10)
		if !strings.Contains(view, "stale") {
			t.Fatal("expected stale indicator")
		}
	})

	t.Run("renders attention items", func(t *testing.T) {
		m := Model{
			pmBriefing: &pm.PMBriefing{
				Summary: "Has issues",
				AttentionItems: []pm.AttentionItem{
					{Priority: "high", Title: "Critical bug in auth"},
					{Priority: "medium", Title: "Test coverage low"},
				},
			},
		}
		view := m.renderPMBriefing(120, 15)
		if !strings.Contains(view, "Critical bug in auth") {
			t.Fatal("expected high-priority attention item")
		}
		if !strings.Contains(view, "Test coverage low") {
			t.Fatal("expected medium-priority attention item")
		}
	})

	t.Run("shows invoke loading state", func(t *testing.T) {
		m := Model{pmInvokeInFlight: true}
		view := m.renderPMBriefing(120, 10)
		if !strings.Contains(view, "Invoking PM") {
			t.Fatal("expected invoke loading indicator")
		}
	})

	t.Run("no briefing shows placeholder with invoke hint", func(t *testing.T) {
		m := Model{}
		view := m.renderPMBriefing(120, 10)
		if !strings.Contains(view, "press i to invoke") {
			t.Fatal("expected invoke hint in placeholder")
		}
	})
}
