package pm

import (
	"os"
	"testing"
	"time"
)

func TestInvokeWithRecoveryFallsBackToCacheOnNonZeroExit(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	restore := swapCommandContext(t)
	defer restore()

	t.Setenv("INVOKER_HELPER_MODE", "exit-failure")

	invoker, err := NewInvoker()
	if err != nil {
		t.Fatalf("NewInvoker failed: %v", err)
	}

	if err := CacheOutput(&PMBriefing{Summary: "cached-nonzero"}); err != nil {
		t.Fatalf("CacheOutput failed: %v", err)
	}

	inbox, err := BuildInbox(TriggerOnDemand, nil, nil)
	if err != nil {
		t.Fatalf("BuildInbox failed: %v", err)
	}

	briefing, stale, err := invoker.InvokeWithRecovery(inbox)
	if err != nil {
		t.Fatalf("InvokeWithRecovery failed: %v", err)
	}
	if !stale {
		t.Fatal("expected stale=true when using cache fallback")
	}
	if briefing == nil || briefing.Summary != "cached-nonzero" {
		t.Fatalf("unexpected briefing from fallback: %#v", briefing)
	}
}

func TestInvokeWithRecoveryFallsBackToCacheOnParseFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	restore := swapCommandContext(t)
	defer restore()

	t.Setenv("INVOKER_HELPER_MODE", "parse-failure")

	invoker, err := NewInvoker()
	if err != nil {
		t.Fatalf("NewInvoker failed: %v", err)
	}

	if err := CacheOutput(&PMBriefing{Summary: "cached-parse"}); err != nil {
		t.Fatalf("CacheOutput failed: %v", err)
	}

	inbox, err := BuildInbox(TriggerOnDemand, nil, nil)
	if err != nil {
		t.Fatalf("BuildInbox failed: %v", err)
	}

	briefing, stale, err := invoker.InvokeWithRecovery(inbox)
	if err != nil {
		t.Fatalf("InvokeWithRecovery failed: %v", err)
	}
	if !stale {
		t.Fatal("expected stale=true when parse fails and cache is used")
	}
	if briefing == nil || briefing.Summary != "cached-parse" {
		t.Fatalf("unexpected briefing from parse-failure fallback: %#v", briefing)
	}
}

func TestInvokeWithRecoveryRetriesRateLimitAndSucceeds(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	restoreCmd := swapCommandContext(t)
	defer restoreCmd()

	originalSleep := sleepFn
	defer func() {
		sleepFn = originalSleep
	}()

	sleepCalls := 0
	sleepFn = func(time.Duration) {
		sleepCalls++
	}

	counterPath := t.TempDir() + "/counter.txt"
	t.Setenv("INVOKER_HELPER_MODE", "rate-limit-then-success")
	t.Setenv("INVOKER_HELPER_COUNTER_PATH", counterPath)

	invoker, err := NewInvoker()
	if err != nil {
		t.Fatalf("NewInvoker failed: %v", err)
	}

	inbox, err := BuildInbox(TriggerCommit, nil, nil)
	if err != nil {
		t.Fatalf("BuildInbox failed: %v", err)
	}

	briefing, stale, err := invoker.InvokeWithRecovery(inbox)
	if err != nil {
		t.Fatalf("InvokeWithRecovery failed: %v", err)
	}
	if stale {
		t.Fatal("expected stale=false after successful retries")
	}
	if briefing == nil || briefing.Summary != "rate-recovered" {
		t.Fatalf("unexpected briefing after rate-limit retry: %#v", briefing)
	}
	if sleepCalls != 2 {
		t.Fatalf("expected 2 backoff sleeps before success, got %d", sleepCalls)
	}
}

func TestInvokeWithRecoveryRateLimitExhaustedUsesCache(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	restoreCmd := swapCommandContext(t)
	defer restoreCmd()

	originalSleep := sleepFn
	defer func() {
		sleepFn = originalSleep
	}()
	sleepFn = func(time.Duration) {}

	t.Setenv("INVOKER_HELPER_MODE", "always-rate-limit")

	invoker, err := NewInvoker()
	if err != nil {
		t.Fatalf("NewInvoker failed: %v", err)
	}
	if err := CacheOutput(&PMBriefing{Summary: "cached-rate-limit"}); err != nil {
		t.Fatalf("CacheOutput failed: %v", err)
	}

	inbox, err := BuildInbox(TriggerCommit, nil, nil)
	if err != nil {
		t.Fatalf("BuildInbox failed: %v", err)
	}

	briefing, stale, err := invoker.InvokeWithRecovery(inbox)
	if err != nil {
		t.Fatalf("InvokeWithRecovery failed: %v", err)
	}
	if !stale {
		t.Fatal("expected stale=true when rate limit retries are exhausted")
	}
	if briefing == nil || briefing.Summary != "cached-rate-limit" {
		t.Fatalf("unexpected cached briefing: %#v", briefing)
	}
}

func TestInvokeWithRecoveryReturnsErrorWhenNoCache(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	restore := swapCommandContext(t)
	defer restore()

	t.Setenv("INVOKER_HELPER_MODE", "exit-failure")

	invoker, err := NewInvoker()
	if err != nil {
		t.Fatalf("NewInvoker failed: %v", err)
	}

	inbox, err := BuildInbox(TriggerOnDemand, nil, nil)
	if err != nil {
		t.Fatalf("BuildInbox failed: %v", err)
	}

	briefing, stale, err := invoker.InvokeWithRecovery(inbox)
	if err == nil {
		t.Fatal("expected error when invocation fails and no cache exists")
	}
	if stale {
		t.Fatal("expected stale=false when returning hard error")
	}
	if briefing != nil {
		t.Fatalf("expected nil briefing on hard error, got %#v", briefing)
	}
}

func TestInvokeWithRecoveryIgnoresCacheWriteFailureAfterSuccess(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	restore := swapCommandContext(t)
	defer restore()

	t.Setenv("INVOKER_HELPER_MODE", "fresh-success")

	invoker, err := NewInvoker()
	if err != nil {
		t.Fatalf("NewInvoker failed: %v", err)
	}

	pmPath := resolveStoragePath(pmDir)
	if err := os.Chmod(pmPath, 0555); err != nil {
		t.Fatalf("chmod pm dir failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(pmPath, 0755)
	})

	inbox, err := BuildInbox(TriggerOnDemand, nil, nil)
	if err != nil {
		t.Fatalf("BuildInbox failed: %v", err)
	}

	briefing, stale, err := invoker.InvokeWithRecovery(inbox)
	if err != nil {
		t.Fatalf("InvokeWithRecovery should still succeed when cache write fails: %v", err)
	}
	if stale {
		t.Fatal("expected stale=false for fresh output")
	}
	if briefing == nil || briefing.Summary != "fresh-run" {
		t.Fatalf("unexpected briefing: %#v", briefing)
	}
}
