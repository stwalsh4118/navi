package monitor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stwalsh4118/navi/internal/session"
)

func TestAttachMonitorNotifiesOnStatusChange(t *testing.T) {
	dir := t.TempDir()
	if err := writeMonitorStatus(dir, session.Info{TmuxSession: "s1", Status: session.StatusWorking}); err != nil {
		t.Fatalf("writeMonitorStatus failed: %v", err)
	}

	m := New(nil, dir, testPollInterval)
	notified := make(chan string, 1)
	m.notifyFn = func(sessionName, status string) {
		notified <- sessionName + ":" + status
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx, nil)

	requireEventually(t, func() bool {
		return m.States()["s1"] == session.StatusWorking
	}, 300*time.Millisecond)

	if err := writeMonitorStatus(dir, session.Info{TmuxSession: "s1", Status: session.StatusWaiting}); err != nil {
		t.Fatalf("writeMonitorStatus update failed: %v", err)
	}

	select {
	case got := <-notified:
		if got != "s1:"+session.StatusWaiting {
			t.Fatalf("notification = %q, want %q", got, "s1:"+session.StatusWaiting)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected notification")
	}
}

func TestAttachMonitorStopsCleanly(t *testing.T) {
	dir := t.TempDir()
	if err := writeMonitorStatus(dir, session.Info{TmuxSession: "s1", Status: session.StatusWorking}); err != nil {
		t.Fatalf("writeMonitorStatus failed: %v", err)
	}

	m := New(nil, dir, testPollInterval)
	ctx, cancel := context.WithCancel(context.Background())
	m.Start(ctx, nil)
	requireEventually(t, func() bool {
		return m.States()["s1"] == session.StatusWorking
	}, 300*time.Millisecond)

	cancel()
	if err := writeMonitorStatus(dir, session.Info{TmuxSession: "s1", Status: session.StatusPermission}); err != nil {
		t.Fatalf("writeMonitorStatus update failed: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	if got := m.States()["s1"]; got != session.StatusWorking {
		t.Fatalf("state after cancel = %q, want %q", got, session.StatusWorking)
	}
}

func TestAttachMonitorStateHandoffNoDuplicateNotification(t *testing.T) {
	dir := t.TempDir()
	if err := writeMonitorStatus(dir, session.Info{TmuxSession: "s1", Status: session.StatusPermission}); err != nil {
		t.Fatalf("writeMonitorStatus failed: %v", err)
	}

	m := New(nil, dir, testPollInterval)
	notifyCount := 0
	m.notifyFn = func(_, _ string) { notifyCount++ }

	ctx, cancel := context.WithCancel(context.Background())
	m.Start(ctx, map[string]string{"s1": session.StatusWorking})
	requireEventually(t, func() bool {
		return m.States()["s1"] == session.StatusPermission
	}, 300*time.Millisecond)
	cancel()

	if notifyCount != 1 {
		t.Fatalf("monitor notifications = %d, want 1", notifyCount)
	}

	// Restart with handed-off states and unchanged status: should not notify again.
	m2 := New(nil, dir, testPollInterval)
	notifyCount2 := 0
	m2.notifyFn = func(_, _ string) { notifyCount2++ }
	ctx2, cancel2 := context.WithCancel(context.Background())
	m2.Start(ctx2, m.States())
	time.Sleep(50 * time.Millisecond)
	cancel2()

	if notifyCount2 != 0 {
		t.Fatalf("notifications after handoff = %d, want 0", notifyCount2)
	}
}

func TestAttachMonitorWithRemoteSessions(t *testing.T) {
	dir := t.TempDir()
	if err := writeMonitorStatus(dir, session.Info{TmuxSession: "remote-s1", Status: session.StatusPermission, Remote: "devbox"}); err != nil {
		t.Fatalf("writeMonitorStatus failed: %v", err)
	}

	m := New(nil, dir, testPollInterval)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx, map[string]string{"remote-s1": session.StatusWorking})

	requireEventually(t, func() bool {
		return m.States()["remote-s1"] == session.StatusPermission
	}, 500*time.Millisecond)
}

func writeMonitorStatus(dir string, info session.Info) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, info.TmuxSession+".json"), data, 0644)
}
