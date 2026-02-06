package task

import (
	"sync"
	"time"
)

// CachedResult holds a cached provider result with its fetch timestamp.
type CachedResult struct {
	Result    *ProviderResult
	FetchedAt time.Time
	Error     error // cached errors too, so we don't spam failing providers
}

// ResultCache stores provider results per project path with a TTL.
type ResultCache struct {
	mu      sync.RWMutex
	entries map[string]*CachedResult
}

// NewResultCache creates a new empty ResultCache.
func NewResultCache() *ResultCache {
	return &ResultCache{
		entries: make(map[string]*CachedResult),
	}
}

// Get returns the cached result for a project path if it exists and is within the TTL.
// Returns nil, false if not found or expired.
func (c *ResultCache) Get(projectPath string, ttl time.Duration) (*CachedResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[projectPath]
	if !ok {
		return nil, false
	}

	if time.Since(entry.FetchedAt) > ttl {
		return nil, false
	}

	return entry, true
}

// Set stores a result in the cache.
func (c *ResultCache) Set(projectPath string, result *ProviderResult, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[projectPath] = &CachedResult{
		Result:    result,
		FetchedAt: time.Now(),
		Error:     err,
	}
}

// Invalidate removes the cached entry for a project path (for manual refresh).
func (c *ResultCache) Invalidate(projectPath string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, projectPath)
}

// InvalidateAll removes all cached entries.
func (c *ResultCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CachedResult)
}
