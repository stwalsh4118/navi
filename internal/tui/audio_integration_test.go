package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/audio"
	"github.com/stwalsh4118/navi/internal/session"
)

func TestDetectStatusChangesFirstPollSkipsNotifications(t *testing.T) {
	var calls [][2]string
	m := Model{
		audioNotifier:     &audio.Notifier{},
		lastSessionStates: make(map[string]string),
		audioNotifyFn: func(sessionName, status string) {
			calls = append(calls, [2]string{sessionName, status})
		},
	}

	m.detectStatusChanges([]session.Info{{TmuxSession: "s1", Status: "working"}})

	if len(calls) != 0 {
		t.Fatalf("expected no notifications on first poll, got %d", len(calls))
	}
	if got := m.lastSessionStates["s1"]; got != "working" {
		t.Fatalf("expected state map to be initialized, got %q", got)
	}
}

func TestDetectStatusChangesScenarios(t *testing.T) {
	var calls [][2]string
	m := Model{
		audioNotifier: &audio.Notifier{},
		lastSessionStates: map[string]string{
			"s1": "working",
			"s2": "done",
		},
		audioNotifyFn: func(sessionName, status string) {
			calls = append(calls, [2]string{sessionName, status})
		},
	}

	m.detectStatusChanges([]session.Info{
		{TmuxSession: "s1", Status: "permission"}, // changed
		{TmuxSession: "s3", Status: "working"},    // new session, no notify
	})

	if len(calls) != 1 {
		t.Fatalf("expected one notification, got %d", len(calls))
	}
	if calls[0] != ([2]string{"s1", "permission"}) {
		t.Fatalf("unexpected notification payload: %#v", calls[0])
	}
	if _, ok := m.lastSessionStates["s2"]; ok {
		t.Fatalf("expected removed session state to be cleaned")
	}
}

func TestSessionsMsgTriggersStatusChangeNotification(t *testing.T) {
	var calls [][2]string
	m := Model{
		audioNotifier:     &audio.Notifier{},
		lastSessionStates: map[string]string{"local": "working"},
		audioNotifyFn: func(sessionName, status string) {
			calls = append(calls, [2]string{sessionName, status})
		},
	}

	updated, _ := m.Update(sessionsMsg{{TmuxSession: "local", Status: "permission"}})
	newModel := updated.(Model)

	if len(calls) != 1 {
		t.Fatalf("expected one notification from sessionsMsg, got %d", len(calls))
	}
	if calls[0] != ([2]string{"local", "permission"}) {
		t.Fatalf("unexpected notification payload: %#v", calls[0])
	}
	if got := newModel.lastSessionStates["local"]; got != "permission" {
		t.Fatalf("expected updated state tracking, got %q", got)
	}
}

func TestRemoteSessionsMsgTriggersStatusChangeNotification(t *testing.T) {
	var calls [][2]string
	m := Model{
		audioNotifier: &audio.Notifier{},
		sessions: []session.Info{
			{TmuxSession: "local", Status: "working"},
			{TmuxSession: "remote-1", Status: "working", Remote: "devbox"},
		},
		lastSessionStates: map[string]string{
			"local":    "working",
			"remote-1": "working",
		},
		audioNotifyFn: func(sessionName, status string) {
			calls = append(calls, [2]string{sessionName, status})
		},
	}

	updated, _ := m.Update(remoteSessionsMsg{sessions: []session.Info{{TmuxSession: "remote-1", Status: "error", Remote: "devbox"}}})
	newModel := updated.(Model)

	if len(calls) != 1 {
		t.Fatalf("expected one notification from remoteSessionsMsg, got %d", len(calls))
	}
	if calls[0] != ([2]string{"remote-1", "error"}) {
		t.Fatalf("unexpected notification payload: %#v", calls[0])
	}
	if got := newModel.lastSessionStates["remote-1"]; got != "error" {
		t.Fatalf("expected updated remote state, got %q", got)
	}
}

func TestNoChangeDoesNotNotify(t *testing.T) {
	var calls [][2]string
	m := Model{
		audioNotifier: &audio.Notifier{},
		lastSessionStates: map[string]string{
			"s1": "working",
		},
		audioNotifyFn: func(sessionName, status string) {
			calls = append(calls, [2]string{sessionName, status})
		},
	}

	m.detectStatusChanges([]session.Info{{TmuxSession: "s1", Status: "working"}})
	if len(calls) != 0 {
		t.Fatalf("expected no notifications when status does not change")
	}
}

func TestNotifyHookIsOptional(t *testing.T) {
	m := Model{
		audioNotifier: &audio.Notifier{},
	}
	m.notifyStatusChange("x", "done")
}

var _ tea.Model = Model{}
