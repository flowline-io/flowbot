package rules

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/redis/go-redis/v9"
)

// CheckThrottle checks whether a notification is within the rate limit for the given key.
// Returns true if the notification is allowed, false if it should be dropped.
// Uses an atomic INCR-first approach to avoid TOCTOU races.
func (e *Engine) CheckThrottle(ctx context.Context, ruleID, eventType, channel string, window time.Duration, limit int) (bool, error) {
	if e.redis == nil {
		return true, nil // no Redis, allow all
	}

	key := fmt.Sprintf("notify:throttle:%s:%s:%s", ruleID, eventType, channel)

	// atomic INCR: get count and increment in one operation
	newCount, err := e.redis.Incr(ctx, key).Result()
	if err != nil {
		flog.Warn("[notify-rules] throttle incr error: %v", err)
		return true, nil // on error, allow (fail-open)
	}

	// set expiry on first increment (count == 1 means key was newly created)
	if newCount == 1 {
		if err := e.redis.Expire(ctx, key, window).Err(); err != nil {
			flog.Warn("[notify-rules] throttle expire error: %v", err)
		}
	}

	return newCount <= int64(limit), nil
}

// ClearThrottle removes the throttle counter for a given key, resetting the rate limiter.
func (e *Engine) ClearThrottle(ctx context.Context, ruleID, eventType, channel string) {
	if e.redis == nil {
		return
	}
	key := fmt.Sprintf("notify:throttle:%s:%s:%s", ruleID, eventType, channel)
	if err := e.redis.Del(ctx, key).Err(); err != nil && !errors.Is(err, redis.Nil) {
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
