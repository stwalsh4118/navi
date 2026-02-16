package monitor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stwalsh4118/navi/internal/session"
)

const testPollInterval = 10 * time.Millisecond

func TestNew(t *testing.T) {
	m := New(nil, "/tmp/status", testPollInterval)
	if m == nil {
		t.Fatal("New() returned nil")
	}
	if m.statusDir != "/tmp/status" {
		t.Fatalf("statusDir = %q, want %q", m.statusDir, "/tmp/status")
	}
	if m.interval != testPollInterval {
		t.Fatalf("interval = %v, want %v", m.interval, testPollInterval)
	}
}

func TestStartTracksStatesAndNotifies(t *testing.T) {
	dir := t.TempDir()
	if err := writeStatus(dir, session.Info{TmuxSession: "s1", Status: session.StatusWorking}); err != nil {
		t.Fatalf("writeStatus setup failed: %v", err)
	}

	m := New(nil, dir, testPollInterval)
	var mu sync.Mutex
	notifications := make([]string, 0)
	m.notifyFn = func(sessionName, newStatus string) {
		mu.Lock()
		defer mu.Unlock()
		notifications = append(notifications, sessionName+":"+newStatus)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx, nil, nil)

	requireEventually(t, func() bool {
		states := m.States()
		return states["s1"] == session.StatusWorking
	}, 300*time.Millisecond)

	if err := writeStatus(dir, session.Info{TmuxSession: "s1", Status: session.StatusWaiting}); err != nil {
		t.Fatalf("writeStatus update failed: %v", err)
	}

	requireEventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		for _, n := range notifications {
			if n == "s1:"+session.StatusWaiting {
				return true
			}
		}
		return false
	}, 500*time.Millisecond)
}

func TestStartWithInitialStatesDoesNotSkipTransitions(t *testing.T) {
	dir := t.TempDir()
	if err := writeStatus(dir, session.Info{TmuxSession: "s1", Status: session.StatusWaiting}); err != nil {
		t.Fatalf("writeStatus failed: %v", err)
	}

	m := New(nil, dir, testPollInterval)
	called := make(chan string, 1)
	m.notifyFn = func(sessionName, newStatus string) {
		called <- sessionName + ":" + newStatus
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx, map[string]string{"s1": session.StatusWorking}, nil)

	select {
	case got := <-called:
		if got != "s1:"+session.StatusWaiting {
			t.Fatalf("notification = %q, want %q", got, "s1:"+session.StatusWaiting)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected transition notification")
	}
}

func TestStartCancelStopsPolling(t *testing.T) {
	dir := t.TempDir()
	if err := writeStatus(dir, session.Info{TmuxSession: "s1", Status: session.StatusWorking}); err != nil {
		t.Fatalf("writeStatus setup failed: %v", err)
	}

	m := New(nil, dir, testPollInterval)
	ctx, cancel := context.WithCancel(context.Background())
	m.Start(ctx, nil, nil)

	requireEventually(t, func() bool {
		states := m.States()
		return states["s1"] == session.StatusWorking
	}, 300*time.Millisecond)

	cancel()

	if err := writeStatus(dir, session.Info{TmuxSession: "s1", Status: session.StatusPermission}); err != nil {
		t.Fatalf("writeStatus update failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	states := m.States()
	if states["s1"] != session.StatusWorking {
		t.Fatalf("state after cancel = %q, want %q", states["s1"], session.StatusWorking)
	}
}

func TestStatesIsThreadSafeUnderConcurrentAccess(t *testing.T) {
	dir := t.TempDir()
	if err := writeStatus(dir, session.Info{TmuxSession: "s1", Status: session.StatusWorking}); err != nil {
		t.Fatalf("writeStatus setup failed: %v", err)
	}

	m := New(nil, dir, testPollInterval)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx, nil, nil)

	requireEventually(t, func() bool {
		states := m.States()
		return states["s1"] == session.StatusWorking
	}, 300*time.Millisecond)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = m.States()
			}
		}()
	}
	wg.Wait()
}

func TestNilNotifierDoesNotPanicAndTracksState(t *testing.T) {
	dir := t.TempDir()
	if err := writeStatus(dir, session.Info{TmuxSession: "s1", Status: session.StatusWorking}); err != nil {
		t.Fatalf("writeStatus setup failed: %v", err)
	}

	m := New(nil, dir, testPollInterval)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx, map[string]string{"s1": session.StatusWorking}, nil)

	if err := writeStatus(dir, session.Info{TmuxSession: "s1", Status: session.StatusPermission}); err != nil {
		t.Fatalf("writeStatus update failed: %v", err)
	}

	requireEventually(t, func() bool {
		states := m.States()
		return states["s1"] == session.StatusPermission
	}, 500*time.Millisecond)
}

func writeStatus(dir string, info session.Info) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, info.TmuxSession+".json")
	return os.WriteFile(path, data, 0644)
}

func requireEventually(t *testing.T, condition func() bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}
