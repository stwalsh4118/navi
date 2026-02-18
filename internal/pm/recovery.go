package pm

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/stwalsh4118/navi/internal/debug"
)

var sleepFn = time.Sleep

// InvokeWithRecovery calls InvokeWithRecoveryStream with no stream channel.
func (i *Invoker) InvokeWithRecovery(inbox *InboxPayload) (*PMBriefing, bool, error) {
	return i.InvokeWithRecoveryStream(inbox, nil)
}

// InvokeWithRecoveryStream invokes the PM agent with streaming support,
// rate-limit retry/backoff, and cache fallback.
func (i *Invoker) InvokeWithRecoveryStream(inbox *InboxPayload, stream chan<- StreamEvent) (*PMBriefing, bool, error) {
	debug.Log("pm: InvokeWithRecovery starting (streaming=%t)", stream != nil)
	result, err := i.invokeWithRateLimitRetries(inbox, stream)
	if err == nil {
		debug.Log("pm: invoke succeeded, caching output")
		_ = CacheOutput(result.Output)
		return result.Output, false, nil
	}

	debug.Log("pm: invoke failed: %v, falling back to cached output", err)
	cached, cacheErr := LoadCachedOutput()
	if cacheErr != nil {
		debug.Log("pm: cache load failed: %v", cacheErr)
		return nil, false, fmt.Errorf("invoke failed: %w; load cache: %v", err, cacheErr)
	}
	if cached != nil && cached.Briefing != nil {
		debug.Log("pm: returning stale cached briefing from %s", cached.CachedAt)
		return cached.Briefing, true, nil
	}

	debug.Log("pm: no cached output available, returning error")
	return nil, false, err
}

func (i *Invoker) invokeWithRateLimitRetries(inbox *InboxPayload, stream chan<- StreamEvent) (*InvokeResult, error) {
	result, err := i.InvokeStream(inbox, stream)
	if err == nil {
		return result, nil
	}

	backoff := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}
	if !isRateLimitError(err) {
		return nil, err
	}

	lastErr := err
	for _, delay := range backoff {
		sleepFn(delay)
		result, err = i.InvokeStream(inbox, stream)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !isRateLimitError(err) {
			break
		}
	}

	return nil, lastErr
}

func isRateLimitError(err error) bool {
	var invokeErr *InvokeError
	if errors.As(err, &invokeErr) {
		stderr := strings.ToLower(invokeErr.Stderr)
		if strings.Contains(stderr, "rate limit") || strings.Contains(stderr, "too many requests") || strings.Contains(stderr, "429") {
			return true
		}
	}

	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "rate limit") || strings.Contains(lower, "too many requests") || strings.Contains(lower, "429")
}
