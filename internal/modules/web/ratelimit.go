package web

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
)

// rateLimitStore is the subset of cache operations needed by the login rate limiter.
// It combines integer counter operations from IntCache with key existence and deletion
// from StringCache, both of which are satisfied by cache.RedisStore.
type rateLimitStore interface {
	cache.IntCache
	Exists(ctx context.Context, key cache.Key) (bool, error)
	Del(ctx context.Context, key cache.Key) error
}

// loginRateLimiter implements IP-based brute force protection for the login endpoint.
// It provides progressive delays after a threshold and full lockout after a limit.
type loginRateLimiter struct {
	store        rateLimitStore
	maxAttempts  int64
	lockoutLimit int64
	windowTTL    cache.TTL
	lockoutTTL   cache.TTL
	maxDelay     time.Duration
}

// newLoginRateLimiter creates a login rate limiter with the given thresholds.
// Default maxDelay is 5 seconds.
func newLoginRateLimiter(store rateLimitStore, maxAttempts, lockoutLimit int64, windowTTL, lockoutTTL cache.TTL) *loginRateLimiter {
	return &loginRateLimiter{
		store:        store,
		maxAttempts:  maxAttempts,
		lockoutLimit: lockoutLimit,
		windowTTL:    windowTTL,
		lockoutTTL:   lockoutTTL,
		maxDelay:     5 * time.Second,
	}
}

// Allow checks if a login attempt from the given IP should proceed.
// It returns a progressive delay duration and whether the IP is locked out.
// If locked is true, the request should be rejected.
func (l *loginRateLimiter) Allow(ctx context.Context, ip string) (delay time.Duration, locked bool) {
	exists, err := l.store.Exists(ctx, lockKey(ip))
	if err != nil {
		flog.Debug("login rate limiter allow exists error: %v", err)
	}
	if exists {
		return 0, true
	}

	count, err := l.store.GetInt64(ctx, attemptKey(ip))
	if err != nil {
		flog.Debug("login rate limiter allow getint64 error: %v", err)
	}
	if count >= l.maxAttempts {
		seconds := int64(1 << uint(min(count, 63)))
		delay = time.Duration(min(seconds, int64(l.maxDelay.Seconds()))) * time.Second
	}
	return delay, false
}

// RecordFailure records a failed login attempt.
// Returns whether the IP is now locked and the retry-after duration.
func (l *loginRateLimiter) RecordFailure(ctx context.Context, ip string) (locked bool, retryAfter time.Duration) {
	count, err := l.store.IncrWithTTL(ctx, attemptKey(ip), l.windowTTL)
	if err != nil {
		flog.Debug("login rate limiter record incr error: %v", err)
		return false, 0
	}
	if count >= l.lockoutLimit {
		if err := l.store.Del(ctx, attemptKey(ip)); err != nil {
			flog.Debug("login rate limiter record del error: %v", err)
		}
		if err := l.store.SetInt64(ctx, lockKey(ip), 1, l.lockoutTTL); err != nil {
			flog.Debug("login rate limiter record setint64 error: %v", err)
		}
		return true, l.lockoutTTL.Duration()
	}
	return false, 0
}

// RecordSuccess clears all rate limit state for the IP after a successful login.
func (l *loginRateLimiter) RecordSuccess(ctx context.Context, ip string) {
	if err := l.store.Del(ctx, attemptKey(ip)); err != nil {
		flog.Debug("login rate limiter success clear attempts error: %v", err)
	}
	if err := l.store.Del(ctx, lockKey(ip)); err != nil {
		flog.Debug("login rate limiter success clear lock error: %v", err)
	}
}

func attemptKey(ip string) cache.Key {
	return cache.NewKey("login", "attempt", ip)
}

func lockKey(ip string) cache.Key {
	return cache.NewKey("login", "locked", ip)
}
