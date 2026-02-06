package task

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewResultCache(t *testing.T) {
	cache := NewResultCache()
	if cache == nil {
		t.Fatal("expected non-nil cache")
	}
}

func TestResultCache_SetAndGet(t *testing.T) {
	cache := NewResultCache()
	result := &ProviderResult{
		Tasks: []Task{
			{ID: "1", Title: "Test", Status: "open"},
		},
	}

	cache.Set("/project/a", result, nil)

	got, ok := cache.Get("/project/a", 5*time.Minute)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.Result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(got.Result.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(got.Result.Tasks))
	}
	if got.Result.Tasks[0].ID != "1" {
		t.Errorf("expected task ID '1', got %q", got.Result.Tasks[0].ID)
	}
	if got.Error != nil {
		t.Errorf("expected nil error, got %v", got.Error)
	}
}

func TestResultCache_GetMiss(t *testing.T) {
	cache := NewResultCache()

	_, ok := cache.Get("/project/nonexistent", 5*time.Minute)
	if ok {
		t.Error("expected cache miss for nonexistent key")
	}
}

func TestResultCache_GetExpired(t *testing.T) {
	cache := NewResultCache()
	result := &ProviderResult{
		Tasks: []Task{
			{ID: "1", Title: "Test", Status: "open"},
		},
	}

	cache.Set("/project/a", result, nil)

	// Use a very short TTL so the entry is already expired.
	_, ok := cache.Get("/project/a", 0)
	if ok {
		t.Error("expected cache miss for expired entry")
	}
}

func TestResultCache_SetWithError(t *testing.T) {
	cache := NewResultCache()
	testErr := errors.New("provider failed")

	cache.Set("/project/a", nil, testErr)

	got, ok := cache.Get("/project/a", 5*time.Minute)
	if !ok {
		t.Fatal("expected cache hit even for error entries")
	}
	if got.Result != nil {
		t.Error("expected nil result for error entry")
	}
	if got.Error == nil {
		t.Fatal("expected non-nil error")
	}
	if got.Error.Error() != "provider failed" {
		t.Errorf("expected 'provider failed', got %q", got.Error.Error())
	}
}

func TestResultCache_Invalidate(t *testing.T) {
	cache := NewResultCache()
	result := &ProviderResult{
		Tasks: []Task{
			{ID: "1", Title: "Test", Status: "open"},
		},
	}

	cache.Set("/project/a", result, nil)
	cache.Invalidate("/project/a")

	_, ok := cache.Get("/project/a", 5*time.Minute)
	if ok {
		t.Error("expected cache miss after invalidation")
	}
}

func TestResultCache_Invalidate_NonExistent(t *testing.T) {
	cache := NewResultCache()
	// Should not panic.
	cache.Invalidate("/project/nonexistent")
}

func TestResultCache_InvalidateAll(t *testing.T) {
	cache := NewResultCache()
	result := &ProviderResult{
		Tasks: []Task{
			{ID: "1", Title: "Test", Status: "open"},
		},
	}

	cache.Set("/project/a", result, nil)
	cache.Set("/project/b", result, nil)
	cache.InvalidateAll()

	_, okA := cache.Get("/project/a", 5*time.Minute)
	_, okB := cache.Get("/project/b", 5*time.Minute)
	if okA || okB {
		t.Error("expected all entries invalidated")
	}
}

func TestResultCache_MultipleProjects(t *testing.T) {
	cache := NewResultCache()
	resultA := &ProviderResult{
		Tasks: []Task{{ID: "a1", Title: "Task A", Status: "open"}},
	}
	resultB := &ProviderResult{
		Tasks: []Task{{ID: "b1", Title: "Task B", Status: "done"}},
	}

	cache.Set("/project/a", resultA, nil)
	cache.Set("/project/b", resultB, nil)

	gotA, okA := cache.Get("/project/a", 5*time.Minute)
	gotB, okB := cache.Get("/project/b", 5*time.Minute)
	if !okA || !okB {
		t.Fatal("expected both cache hits")
	}
	if gotA.Result.Tasks[0].ID != "a1" {
		t.Errorf("expected task ID 'a1', got %q", gotA.Result.Tasks[0].ID)
	}
	if gotB.Result.Tasks[0].ID != "b1" {
		t.Errorf("expected task ID 'b1', got %q", gotB.Result.Tasks[0].ID)
	}
}

func TestResultCache_ConcurrentAccess(t *testing.T) {
	cache := NewResultCache()
	result := &ProviderResult{
		Tasks: []Task{{ID: "1", Title: "Test", Status: "open"}},
	}

	const numGoroutines = 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3)

	// Concurrent writes.
	for i := range numGoroutines {
		go func(i int) {
			defer wg.Done()
			cache.Set("/project/concurrent", result, nil)
			_ = i
		}(i)
	}

	// Concurrent reads.
	for range numGoroutines {
		go func() {
			defer wg.Done()
			cache.Get("/project/concurrent", 5*time.Minute)
		}()
	}

	// Concurrent invalidations.
	for range numGoroutines {
		go func() {
			defer wg.Done()
			cache.Invalidate("/project/concurrent")
		}()
	}

	wg.Wait()
}
