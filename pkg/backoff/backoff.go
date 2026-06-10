// Package backoff provides a unified retry strategy with configurable backoff,
// jitter, adaptive behavior, and error filtering.
package backoff

import (
	"context"
	"errors"
	"math/rand/v2"
	"slices"
	"sync"
	"time"
)

// Config defines the retry strategy for all consumers (pipeline, workflow, event).
type Config struct {
	// MaxAttempts is the total number of execution attempts. 0 or negative means no retry (runs once).
	MaxAttempts int
	// InitialInterval is the delay before the first retry. Zero defaults to 1s.
	InitialInterval time.Duration
	// MaxInterval caps the delay between retries. Zero defaults to 30s.
	MaxInterval time.Duration
	// Multiplier controls delay growth per retry. 1.0 = linear, 2.0 = exponential. Zero defaults to 2.0.
	Multiplier float64
	// Jitter enables full jitter: actual sleep is randomly chosen from [0, delay).
	Jitter bool
	// Adaptive enables success-based adaptive backoff: delay halves after success, doubles after failure.
	Adaptive bool
	// RetryOn lists error codes that trigger retry. Empty means retry all errors.
	RetryOn []string
	// IsRetryable is an optional custom predicate. When set, it overrides RetryOn.
	IsRetryable func(err error) bool
	// MaxElapsedTime caps total retry wall-clock time. 0 means no limit.
	MaxElapsedTime time.Duration
	// OnRetry is called before each retry attempt (not on the first attempt). May be nil.
	OnRetry func(attempt int, delay time.Duration, err error)

	// adaptiveState persists adaptive delay across Do calls. Lazily initialized.
	adaptiveState *adaptiveState
}

// Do executes fn, retrying on error according to cfg.
// Returns the total number of attempts taken and the last error (nil on success).
// When Adaptive is enabled, delay state persists across calls: the delay halves
// after a successful call and doubles after each failed retry.
func Do(ctx context.Context, cfg Config, fn func(ctx context.Context) error) (attempt int, err error) {
	cfg.normalize()
	start := time.Now()
	delay := cfg.loadAdaptiveDelay()

	for attempt = 1; attempt <= cfg.MaxAttempts; attempt++ {
		err = fn(ctx)
		if err == nil {
			cfg.saveAdaptiveDelay(max(cfg.InitialInterval, delay/2))
			return attempt, nil
		}

		if !shouldRetry(err, &cfg) || attempt >= cfg.MaxAttempts {
			cfg.saveAdaptiveDelay(delay)
			return attempt, err
		}

		sleepDuration := cfg.sleepDuration(delay)

		if cfg.MaxElapsedTime > 0 && time.Since(start) >= cfg.MaxElapsedTime {
			return attempt, err
		}

		if cfg.OnRetry != nil {
			cfg.OnRetry(attempt, sleepDuration, err)
		}

		if ctxErr := sleepWithContext(ctx, sleepDuration); ctxErr != nil {
			return attempt, ctxErr
		}

		delay = cfg.nextDelay(delay)
	}

	cfg.saveAdaptiveDelay(delay)
	return attempt, err
}

// loadAdaptiveDelay returns the persisted delay from a prior Do call.
// Falls back to InitialInterval when adaptive mode is off or no state exists.
func (c *Config) loadAdaptiveDelay() time.Duration {
	if !c.Adaptive || c.adaptiveState == nil {
		return c.InitialInterval
	}
	c.adaptiveState.mu.Lock()
	defer c.adaptiveState.mu.Unlock()
	if c.adaptiveState.lastDelay > 0 {
		return c.adaptiveState.lastDelay
	}
	return c.InitialInterval
}

// saveAdaptiveDelay persists the given delay for the next Do call when adaptive mode is on.
func (c *Config) saveAdaptiveDelay(d time.Duration) {
	if !c.Adaptive {
		return
	}
	state := c.getAdaptive()
	state.mu.Lock()
	state.lastDelay = d
	state.mu.Unlock()
}

// sleepWithContext blocks for d or until ctx is cancelled, whichever comes first.
// Returns ctx.Err() on cancellation, nil after a full sleep.
func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		// Drain the timer channel if Stop returns false.
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *Config) normalize() {
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = 1
	}
	if c.Multiplier <= 0 {
		c.Multiplier = 2.0
	}
	if c.InitialInterval <= 0 {
		c.InitialInterval = time.Second
	}
	if c.MaxInterval <= 0 {
		c.MaxInterval = 30 * time.Second
	}
}

func (c *Config) sleepDuration(delay time.Duration) time.Duration {
	if c.Jitter {
		return jitterDuration(delay)
	}
	return delay
}

// nextDelay computes the delay for the next retry attempt.
// In adaptive mode the delay doubles; otherwise it is scaled by Multiplier.
// Both modes cap the result at MaxInterval.
func (c *Config) nextDelay(current time.Duration) time.Duration {
	if c.Adaptive {
		return min(c.MaxInterval, current*2)
	}
	return min(c.MaxInterval, time.Duration(float64(current)*c.Multiplier))
}

// adaptiveState holds the persisted delay level for adaptive backoff across Do calls.
type adaptiveState struct {
	mu        sync.Mutex
	lastDelay time.Duration
}

// getAdaptive lazily initializes and returns the adaptive state.
func (c *Config) getAdaptive() *adaptiveState {
	if c.adaptiveState == nil {
		c.adaptiveState = &adaptiveState{}
	}
	return c.adaptiveState
}

// jitterDuration returns a random duration in [0, d).
func jitterDuration(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(int64(d))) // #nosec G404 -- jitter does not require cryptographic randomness
}

// retryableError is an optional interface that errors can implement to guide retry decisions.
type retryableError interface {
	RetryableCode() string
	IsRetryableError() bool
}

// shouldRetry determines whether an error warrants a retry attempt.
func shouldRetry(err error, cfg *Config) bool {
	if cfg.IsRetryable != nil {
		return cfg.IsRetryable(err)
	}
	if len(cfg.RetryOn) == 0 {
		return true
	}
	var re retryableError
	if errors.As(err, &re) {
		if re.IsRetryableError() {
			return true
		}
		if slices.Contains(cfg.RetryOn, re.RetryableCode()) {
			return true
		}
	}
	return false
}
