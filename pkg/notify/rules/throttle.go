package rules

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
)

// CheckThrottle checks whether a notification is within the rate limit for the given key.
// Returns true if the notification is allowed, false if it should be dropped.
func (e *Engine) CheckThrottle(ctx context.Context, ruleID, eventType, channel string, window time.Duration, limit int) (bool, error) {
	if e.store == nil {
		return true, nil
	}

	key := cache.NewKey("notify", "throttle", ruleID+":"+eventType+":"+channel)

	newCount, err := e.store.IncrWithTTL(ctx, key, cache.TTL(window))
	if err != nil {
		flog.Warn("[notify-rules] throttle incr error: %v", err)
		return true, nil
	}

	return newCount <= int64(limit), nil
}

// ClearThrottle removes the throttle counter for a given key, resetting the rate limiter.
func (e *Engine) ClearThrottle(ctx context.Context, ruleID, eventType, channel string) {
	if e.store == nil {
		return
	}
	key := cache.NewKey("notify", "throttle", ruleID+":"+eventType+":"+channel)
	if err := e.store.Del(ctx, key); err != nil {
		flog.Warn("[notify-rules] throttle clear error: %v", err)
	}
}

// parseHour parses an hour string into an integer.
func parseHour(s string) int {
	var h int
	_, _ = fmt.Sscanf(s, "%d", &h)
	return h
}

// currentHour returns the current hour (0-23) for testing.
// Override in tests via a variable.
var currentHour = func() int {
	return time.Now().Hour()
}
