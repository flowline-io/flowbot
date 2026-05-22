// Package backoff provides a unified retry strategy with configurable backoff,
// jitter, adaptive behavior, and error filtering.
package backoff

import (
	"context"
	"errors"
	"math/rand/v2"
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
}

// Do executes fn, retrying on error according to cfg.
// Returns the total number of attempts taken and the last error (nil on success).
func Do(ctx context.Context, cfg Config, fn func(ctx context.Context) error) (attempt int, err error) {
	cfg.normalize()
	delay := cfg.InitialInterval
	start := time.Now()

	for attempt = 1; attempt <= cfg.MaxAttempts; attempt++ {
		err = fn(ctx)
		if err == nil {
			return attempt, nil
		}

		if !shouldRetry(err, &cfg) || attempt >= cfg.MaxAttempts {
			return attempt, err
		}

		sleepDuration := cfg.sleepDuration(delay)

		if cfg.MaxElapsedTime > 0 && time.Since(start) >= cfg.MaxElapsedTime {
			return attempt, err
		}

		if cfg.OnRetry != nil {
			cfg.OnRetry(attempt, sleepDuration, err)
		}

		select {
		case <-ctx.Done():
			return attempt, ctx.Err()
		case <-time.After(sleepDuration):
		}

		delay = cfg.nextDelay(delay)
	}

	return attempt, err
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

func (c *Config) nextDelay(current time.Duration) time.Duration {
	if c.Adaptive {
		return min(c.MaxInterval, current*2)
	}
	return min(c.MaxInterval, time.Duration(float64(current)*c.Multiplier))
}

// jitterDuration returns a random duration in [0, d).
func jitterDuration(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(int64(d)))
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
		for _, target := range cfg.RetryOn {
			if re.RetryableCode() == target {
				return true
			}
		}
	}
	return false
}
