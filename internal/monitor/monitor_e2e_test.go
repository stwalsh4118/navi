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
	m.Start(ctx, nil, nil)

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
	m.Start(ctx, nil, nil)
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
	m.Start(ctx, map[string]string{"s1": session.StatusWorking}, nil)
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
	m2.Start(ctx2, m.States(), nil)
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
	m.Start(ctx, map[string]string{"remote-s1": session.StatusWorking}, nil)

	requireEventually(t, func() bool {
		return m.States()["remote-s1"] == session.StatusPermission
	}, 500*time.Millisecond)
}

func TestAttachMonitorNotifiesOnAgentStatusChange(t *testing.T) {
	dir := t.TempDir()
	if err := writeMonitorStatus(dir, session.Info{
		TmuxSession: "s1",
		Status:      session.StatusWorking,
		Agents: map[string]session.ExternalAgent{
			"opencode": {Status: session.StatusWorking},
		},
	}); err != nil {
		t.Fatalf("writeMonitorStatus failed: %v", err)
	}

	m := New(nil, dir, testPollInterval)
	notifications := make(chan [2]string, 2)
	m.notifyFn = func(sessionName, status string) {
		notifications <- [2]string{sessionName, status}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx, nil, nil)

	requireEventually(t, func() bool {
		return m.AgentStates()["s1"]["opencode"] == session.StatusWorking
	}, 300*time.Millisecond)

	if err := writeMonitorStatus(dir, session.Info{
		TmuxSession: "s1",
		Status:      session.StatusWorking,
		Agents: map[string]session.ExternalAgent{
			"opencode": {Status: session.StatusPermission},
		},
	}); err != nil {
		t.Fatalf("writeMonitorStatus update failed: %v", err)
	}

	select {
	case got := <-notifications:
		if got != ([2]string{"s1:opencode", session.StatusPermission}) {
			t.Fatalf("notification = %#v, want %#v", got, [2]string{"s1:opencode", session.StatusPermission})
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected agent notification")
	}
}

func TestAttachMonitorAgentStateHandoff(t *testing.T) {
	dir := t.TempDir()
	if err := writeMonitorStatus(dir, session.Info{
		TmuxSession: "s1",
		Status:      session.StatusWorking,
		Agents: map[string]session.ExternalAgent{
			"opencode": {Status: session.StatusPermission},
		},
	}); err != nil {
		t.Fatalf("writeMonitorStatus failed: %v", err)
	}

	m := New(nil, dir, testPollInterval)
	firstRunNotifyCount := 0
	m.notifyFn = func(_, _ string) { firstRunNotifyCount++ }

	ctx, cancel := context.WithCancel(context.Background())
	m.Start(ctx, map[string]string{"s1": session.StatusWorking}, map[string]map[string]string{
		"s1": {"opencode": session.StatusWorking},
	})

	requireEventually(t, func() bool {
		return m.AgentStates()["s1"]["opencode"] == session.StatusPermission
	}, 300*time.Millisecond)
	cancel()

	if firstRunNotifyCount != 1 {
		t.Fatalf("notifications on first run = %d, want 1", firstRunNotifyCount)
	}

	m2 := New(nil, dir, testPollInterval)
	secondRunNotifyCount := 0
	m2.notifyFn = func(_, _ string) { secondRunNotifyCount++ }
	ctx2, cancel2 := context.WithCancel(context.Background())
	m2.Start(ctx2, m.States(), m.AgentStates())
	time.Sleep(50 * time.Millisecond)
	cancel2()

	if secondRunNotifyCount != 0 {
		t.Fatalf("notifications after handoff = %d, want 0", secondRunNotifyCount)
	}
}

func TestAttachMonitorAgentAndSessionIndependent(t *testing.T) {
	dir := t.TempDir()
	if err := writeMonitorStatus(dir, session.Info{
		TmuxSession: "s1",
		Status:      session.StatusPermission,
		Agents: map[string]session.ExternalAgent{
			"opencode": {Status: session.StatusIdle},
		},
	}); err != nil {
		t.Fatalf("writeMonitorStatus failed: %v", err)
	}

	m := New(nil, dir, testPollInterval)
	notifications := make(chan [2]string, 2)
	m.notifyFn = func(sessionName, status string) {
		notifications <- [2]string{sessionName, status}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx, map[string]string{"s1": session.StatusWorking}, map[string]map[string]string{
		"s1": {"opencode": session.StatusWorking},
	})

	var got [][2]string
	requireEventually(t, func() bool {
		for {
			select {
			case n := <-notifications:
				got = append(got, n)
			default:
				return len(got) >= 2
			}
		}
	}, 500*time.Millisecond)

	foundSession := false
	foundAgent := false
	for _, n := range got {
		if n == ([2]string{"s1", session.StatusPermission}) {
			foundSession = true
		}
		if n == ([2]string{"s1:opencode", session.StatusIdle}) {
			foundAgent = true
		}
	}
	if !foundSession || !foundAgent {
		t.Fatalf("expected both session and agent notifications, got %#v", got)
	}
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
