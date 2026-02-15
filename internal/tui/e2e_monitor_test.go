package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/audio"
	"github.com/stwalsh4118/navi/internal/session"
)

func TestAttachStartsMonitor(t *testing.T) {
	tmpDir := t.TempDir()
	origDir := session.StatusDir
	session.StatusDir = tmpDir
	t.Cleanup(func() { session.StatusDir = origDir })

	m := Model{
		width:    80,
		height:   24,
		sessions: []session.Info{{TmuxSession: "s1", Status: session.StatusWorking}},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(Model)
	if m2.attachMonitor == nil {
		t.Fatal("attachMonitor should be started")
	}
	m2.stopAttachMonitor()
}

func TestDetachStopsMonitorAndHandsOffState(t *testing.T) {
	tmpDir := t.TempDir()
	origDir := session.StatusDir
	session.StatusDir = tmpDir
	t.Cleanup(func() { session.StatusDir = origDir })

	if err := writeTUIStatus(tmpDir, session.Info{TmuxSession: "s1", Status: session.StatusPermission}); err != nil {
		t.Fatalf("writeTUIStatus failed: %v", err)
	}

	m := Model{
		width:             80,
		height:            24,
		sessions:          []session.Info{{TmuxSession: "s1", Status: session.StatusWorking}},
		lastSessionStates: map[string]string{"s1": session.StatusWorking},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(Model)

	waitForTUICondition(t, func() bool {
		if m2.attachMonitor == nil {
			return false
		}
		return m2.attachMonitor.States()["s1"] == session.StatusPermission
	}, 2*time.Second)

	updated, _ = m2.Update(attachDoneMsg{})
	m3 := updated.(Model)
	if m3.attachMonitor != nil {
		t.Fatal("attachMonitor should be nil after detach")
	}
	if got := m3.lastSessionStates["s1"]; got != session.StatusPermission {
		t.Fatalf("lastSessionStates handoff = %q, want %q", got, session.StatusPermission)
	}
}

func TestNoDuplicateNotificationsAcrossAttachCycle(t *testing.T) {
	tmpDir := t.TempDir()
	origDir := session.StatusDir
	session.StatusDir = tmpDir
	t.Cleanup(func() { session.StatusDir = origDir })

	if err := writeTUIStatus(tmpDir, session.Info{TmuxSession: "s1", Status: session.StatusWorking}); err != nil {
		t.Fatalf("writeTUIStatus failed: %v", err)
	}

	notifyCount := 0
	m := Model{
		width:             80,
		height:            24,
		sessions:          []session.Info{{TmuxSession: "s1", Status: session.StatusWorking}},
		audioNotifier:     &audio.Notifier{},
		lastSessionStates: map[string]string{"s1": session.StatusWorking},
		audioNotifyFn: func(string, string) {
			notifyCount++
		},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(Model)

	if err := writeTUIStatus(tmpDir, session.Info{TmuxSession: "s1", Status: session.StatusPermission}); err != nil {
		t.Fatalf("writeTUIStatus update failed: %v", err)
	}

	waitForTUICondition(t, func() bool {
		if m2.attachMonitor == nil {
			return false
		}
		return m2.attachMonitor.States()["s1"] == session.StatusPermission
	}, 2*time.Second)

	updated, _ = m2.Update(attachDoneMsg{})
	m3 := updated.(Model)

	updated, _ = m3.Update(sessionsMsg{{TmuxSession: "s1", Status: session.StatusPermission}})
	_ = updated

	if notifyCount != 0 {
		t.Fatalf("notifyCount after handoff = %d, want 0", notifyCount)
	}
}

func writeTUIStatus(dir string, info session.Info) error {
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, info.TmuxSession+".json"), data, 0644)
}

func waitForTUICondition(t *testing.T, condition func() bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}
