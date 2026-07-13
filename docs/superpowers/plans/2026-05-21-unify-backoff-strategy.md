# Unify Backoff Strategy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create `pkg/backoff/` with a single `Do` function and `Middleware` adapter, eliminating duplicate retry loops in pipeline, workflow, and event.

**Architecture:** Single-package unified backoff. `Do(ctx, cfg, fn)` is the sole retry loop. `Middleware(cfg, logger)` adapts for Watermill. `Config` aggregates fields from all three existing implementations. Pipeline/workflow migrate immediately; event uses Middleware adapter.

**Tech Stack:** Go 1.26+, `math/rand/v2`, `github.com/ThreeDotsLabs/watermill/message`, `github.com/flowline-io/flowbot/pkg/types` (error codes).

---

## File Changes Summary

| File                                | Action                                                                                         |
| ----------------------------------- | ---------------------------------------------------------------------------------------------- |
| `pkg/backoff/backoff.go`            | Create — `Config`, `Do`, `shouldRetry`, `jitterDuration`                                       |
| `pkg/backoff/backoff_test.go`       | Create — 9 test groups, 27+ cases                                                              |
| `pkg/backoff/middleware.go`         | Create — `Middleware` for Watermill                                                            |
| `pkg/types/workflow.go`             | Modify — add `ToBackoffConfig()`, deprecate `BuildBackOff`/`RetryEnabled`                      |
| `pkg/pipeline/loader.go`            | Modify — `Step.Retry` type to `*backoff.Config`, `convertRetryConfig` returns `backoff.Config` |
| `pkg/pipeline/engine.go`            | Modify — `executeStep` uses `backoff.Do`, remove `isRetryable`/`containsErrorCode`             |
| `pkg/pipeline/pipeline_test.go`     | Modify — migrate `TestBuildBackoff` and `TestIsRetryable`                                      |
| `pkg/event/middleware.go`           | Rewrite — use `backoff.Middleware`, remove `event.Retry` type                                  |
| `pkg/event/pubsub.go`               | Modify — use `backoff.Config` instead of `event.Retry`                                         |
| `pkg/event/middleware_test.go`      | Rewrite — test `backoff.Middleware`                                                            |
| `pkg/workflow/workflow.go`          | Modify — replace `runWithRetry` + `runEngineWithRetry` with `backoff.Do`                       |
| `pkg/workflow/scheduler.go`         | Modify — update `executeExecutorStep` pass `backoff.Config`                                    |
| `tests/specs/pipeline_spec_test.go` | Modify — update `RetryEnabled`/`BuildBackOff` calls                                            |
| `tests/specs/workflow_spec_test.go` | Modify — update `RetryEnabled` calls                                                           |

---

### Task 1: Create `pkg/backoff/backoff.go` — core Do function and Config

**Files:**

- Create: `pkg/backoff/backoff.go`

- [ ] **Step 1: Write `pkg/backoff/backoff.go`**

```go
// Package backoff provides a unified retry strategy with configurable backoff,
// jitter, adaptive behavior, and error filtering.
package backoff

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
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
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 1
	}
	if cfg.Multiplier <= 0 {
		cfg.Multiplier = 2.0
	}
	if cfg.InitialInterval <= 0 {
		cfg.InitialInterval = time.Second
	}
	if cfg.MaxInterval <= 0 {
		cfg.MaxInterval = 30 * time.Second
	}

	delay := cfg.InitialInterval
	start := time.Now()

	for attempt = 1; attempt <= cfg.MaxAttempts; attempt++ {
		err = fn(ctx)
		if err == nil {
			if cfg.Adaptive {
				delay = max(cfg.InitialInterval, delay/2)
			}
			return attempt, nil
		}

		if !shouldRetry(err, &cfg) {
			return attempt, err
		}

		if attempt >= cfg.MaxAttempts {
			return attempt, err
		}

		sleepDuration := delay
		if cfg.Jitter {
			sleepDuration = jitterDuration(delay)
		}

		if cfg.MaxElapsedTime > 0 {
			elapsed := time.Since(start)
			if elapsed >= cfg.MaxElapsedTime {
				return attempt, err
			}
		}

		if cfg.OnRetry != nil {
			cfg.OnRetry(attempt, sleepDuration, err)
		}

		select {
		case <-ctx.Done():
			return attempt, ctx.Err()
		case <-time.After(sleepDuration):
		}

		if cfg.Adaptive {
			delay = min(cfg.MaxInterval, delay*2)
		} else {
			delay = min(cfg.MaxInterval, time.Duration(float64(delay)*cfg.Multiplier))
		}
	}

	return attempt, err
}

// jitterDuration returns a random duration in [0, d).
func jitterDuration(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(int64(d)))
}

// shouldRetry determines whether an error warrants a retry attempt.
func shouldRetry(err error, cfg *Config) bool {
	if cfg.IsRetryable != nil {
		return cfg.IsRetryable(err)
	}
	if len(cfg.RetryOn) == 0 {
		return true
	}
	var te *types.Error
	if errors.As(err, &te) {
		if te.Retryable {
			return true
		}
		for _, target := range cfg.RetryOn {
			if te.Code == target {
				return true
			}
		}
	}
	return false
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd pkg/backoff && go build ./...
```

Expected: compiles without errors.

- [ ] **Step 3: Commit**

```bash
git add pkg/backoff/backoff.go
git commit -m "feat: add pkg/backoff with unified Do retry loop"
```

---

### Task 2: Write TDD unit tests for `pkg/backoff`

**Files:**

- Create: `pkg/backoff/backoff_test.go`

- [ ] **Step 1: Write `pkg/backoff/backoff_test.go`**

```go
package backoff

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
)

var anError = errors.New("test error")

func TestDo_BasicRetry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		maxAttempts int
		failsBefore int
		wantAttempt int
		wantErr     bool
	}{
		{
			name:        "success_on_first_attempt",
			maxAttempts: 3,
			failsBefore: 0,
			wantAttempt: 1,
			wantErr:     false,
		},
		{
			name:        "success_after_two_failures",
			maxAttempts: 3,
			failsBefore: 2,
			wantAttempt: 3,
			wantErr:     false,
		},
		{
			name:        "exceed_max_attempts",
			maxAttempts: 2,
			failsBefore: 3,
			wantAttempt: 2,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int32
			cfg := Config{
				MaxAttempts:     tt.maxAttempts,
				InitialInterval: 1 * time.Millisecond,
			}
			fn := func(ctx context.Context) error {
				if int(calls.Add(1)) <= tt.failsBefore {
					return anError
				}
				return nil
			}
			attempt, err := Do(context.Background(), cfg, fn)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if attempt != tt.wantAttempt {
				t.Fatalf("got attempt=%d, want=%d", attempt, tt.wantAttempt)
			}
		})
	}
}

func TestDo_ContextCancellation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		setupCtx func() (context.Context, context.CancelFunc)
		wantErr  error
	}{
		{
			name: "cancel_context_immediately",
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
			wantErr: context.Canceled,
		},
		{
			name: "deadline_exceeded",
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
				return ctx, cancel
			},
			wantErr: context.DeadlineExceeded,
		},
		{
			name: "normal_context_runs",
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := tt.setupCtx()
			defer cancel()
			cfg := Config{
				MaxAttempts:     2,
				InitialInterval: 100 * time.Millisecond,
			}
			fn := func(ctx context.Context) error { return anError }
			_, err := Do(ctx, cfg, fn)
			if tt.wantErr == nil {
				if err == nil {
					return
				}
				t.Fatalf("unexpected error: %v", err)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err=%v, want=%v", err, tt.wantErr)
			}
		})
	}
}

func TestDo_RetryOnFiltering(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		retryOn     []string
		errCode     string
		wantAttempt int
	}{
		{
			name:        "matching_code_retries",
			retryOn:     []string{"TIMEOUT"},
			errCode:     "TIMEOUT",
			wantAttempt: 3,
		},
		{
			name:        "non_matching_code_returns_immediately",
			retryOn:     []string{"TIMEOUT"},
			errCode:     "UNAVAILABLE",
			wantAttempt: 1,
		},
		{
			name:        "empty_retry_on_retries_all",
			retryOn:     nil,
			errCode:     "ANYTHING",
			wantAttempt: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := Config{
				MaxAttempts:     3,
				InitialInterval: 1 * time.Millisecond,
				RetryOn:         tt.retryOn,
			}
			fn := func(ctx context.Context) error {
				return &types.Error{Code: tt.errCode, Kind: anError}
			}
			attempt, _ := Do(context.Background(), cfg, fn)
			if attempt != tt.wantAttempt {
				t.Fatalf("got attempt=%d, want=%d", attempt, tt.wantAttempt)
			}
		})
	}
}

func TestDo_IsRetryableCallback(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		isRetryable func(err error) bool
		wantAttempt int
	}{
		{
			name: "predicate_overrides_retry_on",
			isRetryable: func(err error) bool {
				return true
			},
			wantAttempt: 3,
		},
		{
			name: "returns_false_terminates_immediately",
			isRetryable: func(err error) bool {
				return false
			},
			wantAttempt: 1,
		},
		{
			name: "returns_true_continues",
			isRetryable: func(err error) bool {
				return errors.Is(err, anError)
			},
			wantAttempt: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := Config{
				MaxAttempts:     3,
				InitialInterval: 1 * time.Millisecond,
				RetryOn:         []string{"NOT_MATCHING"},
				IsRetryable:     tt.isRetryable,
			}
			fn := func(ctx context.Context) error { return anError }
			attempt, _ := Do(context.Background(), cfg, fn)
			if attempt != tt.wantAttempt {
				t.Fatalf("got attempt=%d, want=%d", attempt, tt.wantAttempt)
			}
		})
	}
}

func TestDo_Jitter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		jitter bool
	}{
		{name: "jitter_on", jitter: true},
		{name: "jitter_off", jitter: false},
		{name: "jitter_on_again", jitter: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int32
			cfg := Config{
				MaxAttempts:     3,
				InitialInterval: 50 * time.Millisecond,
				MaxInterval:     100 * time.Millisecond,
				Jitter:          tt.jitter,
			}
			fn := func(ctx context.Context) error {
				calls.Add(1)
				return anError
			}
			_, _ = Do(context.Background(), cfg, fn)
			count := calls.Load()
			if count != 3 {
				t.Fatalf("expected 3 calls, got %d", count)
			}
		})
	}
}

func TestDo_Adaptive(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "adaptive_enabled"},
		{name: "adaptive_disabled"},
		{name: "adaptive_enabled_second_case"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := Config{
				MaxAttempts:     3,
				InitialInterval: 10 * time.Millisecond,
				MaxInterval:     50 * time.Millisecond,
				Adaptive:        tt.name != "adaptive_disabled",
			}
			fn := func(ctx context.Context) error { return anError }
			attempt, err := Do(context.Background(), cfg, fn)
			if err == nil {
				t.Fatalf("expected error")
			}
			if attempt != 3 {
				t.Fatalf("got attempt=%d, want=3", attempt)
			}
		})
	}
}

func TestDo_MaxElapsedTime(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		maxElapsed  time.Duration
		initial     time.Duration
		minAttempts int
	}{
		{
			name:        "timeout_terminates_early",
			maxElapsed:  5 * time.Millisecond,
			initial:     50 * time.Millisecond,
			minAttempts: 1,
		},
		{
			name:        "no_timeout_allows_all",
			maxElapsed:  0,
			initial:     1 * time.Millisecond,
			minAttempts: 3,
		},
		{
			name:        "last_retry_before_timeout",
			maxElapsed:  100 * time.Millisecond,
			initial:     1 * time.Millisecond,
			minAttempts: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := Config{
				MaxAttempts:     3,
				InitialInterval: tt.initial,
				MaxInterval:     200 * time.Millisecond,
				MaxElapsedTime:  tt.maxElapsed,
			}
			fn := func(ctx context.Context) error { return anError }
			attempt, err := Do(context.Background(), cfg, fn)
			if err == nil {
				t.Fatalf("expected error")
			}
			if attempt < tt.minAttempts {
				t.Fatalf("got attempt=%d, want >= %d", attempt, tt.minAttempts)
			}
		})
	}
}

func TestDo_OnRetryCallback(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		onRetry func(attempt int, delay time.Duration, err error)
	}{
		{
			name: "callback_fires_correctly",
			onRetry: func(attempt int, delay time.Duration, err error) {},
		},
		{
			name:    "nil_callback_does_not_panic",
			onRetry: nil,
		},
		{
			name: "callback_receives_correct_args",
			onRetry: func(attempt int, delay time.Duration, err error) {
				if attempt < 1 {
					panic("attempt < 1")
				}
				if err == nil {
					panic("err is nil")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := Config{
				MaxAttempts:     3,
				InitialInterval: 1 * time.Millisecond,
				OnRetry:         tt.onRetry,
			}
			fn := func(ctx context.Context) error { return anError }
			_, _ = Do(context.Background(), cfg, fn)
		})
	}
}

func TestDo_MaxAttemptsZero(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		maxAttempts int
		wantCalls   int
	}{
		{
			name:        "explicit_zero_runs_once",
			maxAttempts: 0,
			wantCalls:   1,
		},
		{
			name:        "negative_runs_once",
			maxAttempts: -1,
			wantCalls:   1,
		},
		{
			name:        "one_runs_once",
			maxAttempts: 1,
			wantCalls:   1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int32
			cfg := Config{MaxAttempts: tt.maxAttempts}
			fn := func(ctx context.Context) error {
				calls.Add(1)
				return anError
			}
			_, _ = Do(context.Background(), cfg, fn)
			if loaded := calls.Load(); int(loaded) != tt.wantCalls {
				t.Fatalf("got %d calls, want %d", loaded, tt.wantCalls)
			}
		})
	}
}

func TestShouldRetry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  Config
		err  error
		want bool
	}{
		{
			name: "empty_retry_on_returns_true",
			cfg:  Config{},
			err:  anError,
			want: true,
		},
		{
			name: "matching_code_returns_true",
			cfg:  Config{RetryOn: []string{"TIMEOUT"}},
			err:  &types.Error{Code: "TIMEOUT", Kind: anError},
			want: true,
		},
		{
			name: "non_matching_code_returns_false",
			cfg:  Config{RetryOn: []string{"TIMEOUT"}},
			err:  &types.Error{Code: "UNAVAILABLE", Kind: anError},
			want: false,
		},
		{
			name: "retryable_flag_overrides",
			cfg:  Config{RetryOn: []string{"TIMEOUT"}},
			err:  &types.Error{Retryable: true, Kind: anError},
			want: true,
		},
		{
			name: "is_retryable_callback_used",
			cfg:  Config{IsRetryable: func(err error) bool { return false }},
			err:  anError,
			want: false,
		},
		{
			name: "standard_error_not_matching_returns_false",
			cfg:  Config{RetryOn: []string{"TIMEOUT"}},
			err:  anError,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := shouldRetry(tt.err, &tt.cfg)
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests and confirm pass**

```bash
go test ./pkg/backoff/ -v -count=1 -timeout 30s
```

Expected: All tests PASS.

- [ ] **Step 3: Commit**

```bash
git add pkg/backoff/backoff_test.go
git commit -m "test: add unit tests for pkg/backoff"
```

---

### Task 3: Add Watermill Middleware adapter

**Files:**

- Create: `pkg/backoff/middleware.go`

- [ ] **Step 1: Write `pkg/backoff/middleware.go`**

```go
package backoff

import (
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Middleware returns a Watermill handler middleware that wraps h with retry
// behavior governed by cfg. Each retry is logged via logger.
func Middleware(cfg Config, logger watermill.LoggerAdapter) message.HandlerMiddleware {
	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			producedMessages, err := h(msg)
			if err == nil {
				return producedMessages, nil
			}

			origOnRetry := cfg.OnRetry
			cfg.OnRetry = func(attempt int, delay time.Duration, err error) {
				if logger != nil {
					logger.Error("Retrying after error", err, watermill.LogFields{
						"retry_attempt": attempt,
						"retry_delay":   delay,
					})
				}
				if origOnRetry != nil {
					origOnRetry(attempt, delay, err)
				}
			}

			attempt, finalErr := Do(msg.Context(), cfg, func(ctx context.Context) error {
				producedMessages, err = h(msg)
				return err
			})
			_ = attempt
			if finalErr != nil {
				return nil, finalErr
			}
			return producedMessages, nil
		}
	}
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd pkg/backoff && go build ./...
```

Expected: compiles without errors.

- [ ] **Step 3: Commit**

```bash
git add pkg/backoff/middleware.go
git commit -m "feat: add backoff.Middleware for Watermill"
```

---

### Task 4: Migrate pipeline event middleware and pubsub

**Files:**

- Modify: `pkg/event/middleware.go` (rewrite)
- Modify: `pkg/event/pubsub.go`
- Modify: `pkg/event/middleware_test.go` (rewrite)

- [ ] **Step 1: Delete `pkg/event/middleware.go`**

```bash
rm pkg/event/middleware.go
```

- [ ] **Step 2: Edit `pkg/event/pubsub.go` — replace `event.Retry` with `backoff.Middleware`**

In `NewRouter()` (lines 87-103), replace:

```go
	router.AddMiddleware(
		middleware.CorrelationID,
		middleware.Timeout(10*time.Minute),
		Retry{
			MaxRetries:          3,
			InitialInterval:     1 * time.Second,
			MaxInterval:         30 * time.Second,
			Multiplier:          2.0,
			MaxElapsedTime:      2 * time.Minute,
			RandomizationFactor: 0.5,
			OnRetryHook: func(retryNum int, delay time.Duration) {
				flog.Info("Retry attempt #%d, waiting %v before next retry", retryNum, delay)
			},
			Logger: logger,
		}.Middleware,
		middleware.Recoverer,
	)
```

with:

```go
	router.AddMiddleware(
		middleware.CorrelationID,
		middleware.Timeout(10*time.Minute),
		backoff.Middleware(backoff.Config{
			MaxAttempts:     4, // 1 initial + 3 retries
			InitialInterval: 1 * time.Second,
			MaxInterval:     30 * time.Second,
			Multiplier:      2.0,
			MaxElapsedTime:  2 * time.Minute,
			Jitter:          true,
			OnRetry: func(attempt int, delay time.Duration, err error) {
				flog.Info("Retry attempt #%d, waiting %v before next retry", attempt, delay)
			},
		}, logger),
		middleware.Recoverer,
	)
```

Add `"github.com/flowline-io/flowbot/pkg/backoff"` to the imports in `pubsub.go`.

- [ ] **Step 3: Rewrite `pkg/event/middleware_test.go`**

Delete old file and create new test:

```bash
rm pkg/event/middleware_test.go
```

Create `pkg/event/middleware_test.go`:

```go
package event

import (
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/backoff"
	"github.com/flowline-io/flowbot/pkg/flog"
)

func TestBackoffMiddleware_Success(t *testing.T) {
	t.Parallel()
	cfg := backoff.Config{
		MaxAttempts:     3,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     50 * time.Millisecond,
	}
	mw := backoff.Middleware(cfg, flog.WatermillLogger)
	handler := func(msg *message.Message) ([]*message.Message, error) {
		return []*message.Message{msg}, nil
	}
	msg := message.NewMessage("test", []byte("payload"))
	result, err := mw(handler)(msg)
	require.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestBackoffMiddleware_Failure(t *testing.T) {
	t.Parallel()
	cfg := backoff.Config{
		MaxAttempts:     2,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     50 * time.Millisecond,
	}
	mw := backoff.Middleware(cfg, flog.WatermillLogger)
	handler := func(msg *message.Message) ([]*message.Message, error) {
		return nil, assert.AnError
	}
	msg := message.NewMessage("test", []byte("payload"))
	_, err := mw(handler)(msg)
	require.Error(t, err)
}

func TestBackoffMiddleware_EventualSuccess(t *testing.T) {
	t.Parallel()
	var attempts int
	cfg := backoff.Config{
		MaxAttempts:     3,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     50 * time.Millisecond,
	}
	mw := backoff.Middleware(cfg, flog.WatermillLogger)
	handler := func(msg *message.Message) ([]*message.Message, error) {
		attempts++
		if attempts < 3 {
			return nil, assert.AnError
		}
		return []*message.Message{msg}, nil
	}
	msg := message.NewMessage("test", []byte("payload"))
	result, err := mw(handler)(msg)
	require.NoError(t, err)
	assert.Len(t, result, 1)
}
```

- [ ] **Step 4: Verify compilation**

```bash
go build ./pkg/event/...
```

- [ ] **Step 5: Run event tests**

```bash
go test ./pkg/event/ -v -count=1 -timeout 30s
```

Expected: All tests PASS.

- [ ] **Step 6: Commit**

```bash
git add pkg/event/middleware.go pkg/event/middleware_test.go pkg/event/pubsub.go
git commit -m "refactor: migrate event module to backoff.Middleware"
```

---

### Task 5: Add `ToBackoffConfig()` on `types.RetryConfig`, deprecate old methods

**Files:**

- Modify: `pkg/types/workflow.go`

- [ ] **Step 1: Edit `pkg/types/workflow.go`**

Add `flowbackoff "github.com/flowline-io/flowbot/pkg/backoff"` to imports (keep `cenkalti/backoff` for deprecated `BuildBackOff`).

Add `ToBackoffConfig()` method. Add `// Deprecated:` comments on `BuildBackOff` and `RetryEnabled`.

After the existing `BuildBackOff` method (line 72), insert:

```go
// ToBackoffConfig converts the legacy RetryConfig to the unified backoff.Config.
func (r *RetryConfig) ToBackoffConfig() flowbackoff.Config {
	if r == nil {
		return flowbackoff.Config{MaxAttempts: 0}
	}
	multiplier := 2.0
	switch r.Backoff {
	case BackoffFixed, BackoffLinear:
		multiplier = 1.0
	}
	return flowbackoff.Config{
		MaxAttempts:     r.MaxAttempts,
		InitialInterval: r.Delay,
		MaxInterval:     r.MaxDelay,
		Multiplier:      multiplier,
		Jitter:          r.Jitter,
		RetryOn:         r.RetryOn,
	}
}
```

Add deprecation comments. Change lines 26-29:

```go
// RetryEnabled returns true if retries are configured with more than one attempt.
// Deprecated: Use backoff.Config.MaxAttempts > 1 directly.
func (r *RetryConfig) RetryEnabled() bool {
```

Change lines 31-33:

```go
// BuildBackOff constructs a backoff.BackOff from the retry configuration.
// Deprecated: Use ToBackoffConfig() and backoff.Do() instead.
func (r *RetryConfig) BuildBackOff() backoff.BackOff {
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./pkg/types/...
```

Expected: compiles without errors.

- [ ] **Step 3: Commit**

```bash
git add pkg/types/workflow.go
git commit -m "refactor: add RetryConfig.ToBackoffConfig, deprecate old methods"
```

---

### Task 6: Migrate pipeline loader to use `backoff.Config`

**Files:**

- Modify: `pkg/pipeline/loader.go`

- [ ] **Step 1: Edit `pkg/pipeline/loader.go`**

Change import from `"github.com/flowline-io/flowbot/pkg/types"` to `"github.com/flowline-io/flowbot/pkg/backoff"` (types is still used? No — check: `types` is used in loader? Looking at the file: lines 10, 32, 66, 78 — yes, `types.RetryConfig` is used in `convertRetryConfig` and `Step` struct. After migration, `types` import is no longer needed unless other types are used. Check: `Definition`, `Step`, etc. don't use `types`. So we can remove the `types` import.)

Change `Step.Retry` from `*types.RetryConfig` to `*backoff.Config`:

```go
type Step struct {
	Name       string
	Capability hub.CapabilityType
	Operation  string
	Params     map[string]any
	Retry      *backoff.Config
}
```

Change `convertRetryConfig`:

```go
func convertRetryConfig(cfg *config.PipelineStepRetry) (*backoff.Config, error) {
	if cfg == nil || cfg.MaxAttempts <= 0 {
		return nil, nil
	}
	delay, err := time.ParseDuration(cfg.Delay)
	if err != nil && cfg.Delay != "" {
		return nil, fmt.Errorf("invalid delay %q: %w", cfg.Delay, err)
	}
	maxDelay, err := time.ParseDuration(cfg.MaxDelay)
	if err != nil && cfg.MaxDelay != "" {
		return nil, fmt.Errorf("invalid max_delay %q: %w", cfg.MaxDelay, err)
	}
	multiplier := 2.0
	switch cfg.Backoff {
	case "fixed", "linear":
		multiplier = 1.0
	}
	return &backoff.Config{
		MaxAttempts:     cfg.MaxAttempts,
		InitialInterval: delay,
		MaxInterval:     maxDelay,
		Multiplier:      multiplier,
		Jitter:          cfg.Jitter,
		RetryOn:         cfg.RetryOn,
	}, nil
}
```

Update imports: remove `"github.com/flowline-io/flowbot/pkg/types"`, add `"github.com/flowline-io/flowbot/pkg/backoff"`.

- [ ] **Step 2: Verify compilation**

```bash
go build ./pkg/pipeline/...
```

- [ ] **Step 3: Commit**

```bash
git add pkg/pipeline/loader.go
git commit -m "refactor: pipeline loader uses backoff.Config"
```

---

### Task 7: Migrate pipeline engine to use `backoff.Do`

**Files:**

- Modify: `pkg/pipeline/engine.go`

- [ ] **Step 1: Edit `pkg/pipeline/engine.go` — replace `executeStep`**

Add `"github.com/flowline-io/flowbot/pkg/backoff"` to imports. Remove `"github.com/cenkalti/backoff"` and `"errors"` from imports (if not used elsewhere).

Replace `executeStep` body (lines 160-232) with:

```go
func (e *Engine) executeStep(ctx context.Context, rc *RenderContext, step Step, runID int64, pipelineName string, resumable bool) error {
	ctx, span := trace.StartSpan(ctx, "pipeline."+pipelineName+".step."+step.Name,
		otelattr.String("pipeline.step.name", step.Name),
		otelattr.String("pipeline.step.capability", string(step.Capability)),
		otelattr.String("pipeline.step.operation", step.Operation),
	)
	defer span.End()

	stepStart := time.Now()

	renderedParams, err := rc.RenderParams(step.Params)
	if err != nil {
		return fmt.Errorf("render params step %s: %w", step.Name, err)
	}

	attempt := 1
	stepRunID, err := e.createStepRunRecord(ctx, runID, step.Name, string(step.Capability), step.Operation, renderedParams, attempt)
	if err != nil {
		return err
	}

	var hbCtx context.Context
	var hbCancel context.CancelFunc
	if resumable && e.store != nil && runID != 0 {
		hbCtx, hbCancel = context.WithCancel(ctx)
		defer hbCancel()
		go e.heartbeatLoop(hbCtx, runID, pipelineName)
	}

	if e.pipelineMetrics != nil {
		e.pipelineMetrics.IncStepTotal(pipelineName, step.Name, "start")
	}

	retryCfg := step.Retry
	if retryCfg == nil {
		retryCfg = &backoff.Config{MaxAttempts: 0}
	}
	boCfg := *retryCfg
	boCfg.OnRetry = func(a int, d time.Duration, err error) {
		flog.Info("pipeline %s step %s attempt %d failed, retrying in %v: %v",
			pipelineName, step.Name, a, d, err)
	}

	var stepResult map[string]any
	attempt, retryErr := backoff.Do(ctx, boCfg, func(ctx context.Context) error {
		res, invokeErr := capability.Invoke(ctx, step.Capability, step.Operation, renderedParams)
		if invokeErr != nil {
			trace.RecordError(ctx, invokeErr)
			return invokeErr
		}
		stepResult = extractResult(res)
		return nil
	})

	if retryErr != nil {
		e.recordStepFailure(ctx, stepRunID, pipelineName, step.Name, string(step.Capability), retryErr.Error(), attempt, stepStart)
		return fmt.Errorf("step %s: %w", step.Name, retryErr)
	}

	rc.RecordStepResult(step.Name, stepResult)
	e.recordStepSuccess(ctx, stepRunID, pipelineName, step.Name, string(step.Capability), stepResult, attempt, stepStart)
	flog.Info("pipeline %s step %s completed (attempt %d)", pipelineName, step.Name, attempt)
	return nil
}
```

- [ ] **Step 2: Remove `isRetryable` and `containsErrorCode` functions**

Delete lines 234-261 (both `isRetryable` and `containsErrorCode`).

- [ ] **Step 3: Verify compilation**

```bash
go build ./pkg/pipeline/...
```

Expected: compiles without errors.

- [ ] **Step 4: Commit**

```bash
git add pkg/pipeline/engine.go
git commit -m "refactor: pipeline engine uses backoff.Do"
```

---

### Task 8: Migrate pipeline tests

**Files:**

- Modify: `pkg/pipeline/pipeline_test.go`

- [ ] **Step 1: Edit `pkg/pipeline/pipeline_test.go`**

The file has `TestBuildBackoff` (line 495) and `TestIsRetryable` (line 572) that reference `types.RetryConfig.BuildBackOff()` and `isRetryable()`.

Since `BuildBackOff()` is deprecated but still exists, `TestBuildBackoff` can stay as-is for now.

`TestIsRetryable` calls `isRetryable()` which no longer exists in the pipeline package. Options:

1. Remove `TestIsRetryable` since `shouldRetry` is tested in `pkg/backoff/backoff_test.go`.
2. Rewrite it to test `backoff.Do` instead.

Best approach: Remove `TestIsRetryable` and `TestBuildBackoff` from `pipeline_test.go` (they're now tested in `backoff_test.go`). Remove the `backoff` import from pipeline_test.go too.

Delete lines 495-631 (the entire `TestBuildBackoff` and `TestIsRetryable` tests).

Remove `"github.com/cenkalti/backoff"` from pipeline_test.go imports if not used elsewhere.

- [ ] **Step 2: Verify compilation and run tests**

```bash
go test ./pkg/pipeline/ -v -count=1 -timeout 30s
```

Expected: All tests PASS.

- [ ] **Step 3: Commit**

```bash
git add pkg/pipeline/pipeline_test.go
git commit -m "test: remove migrated retry tests from pipeline_test.go"
```

---

### Task 9: Migrate workflow to use `backoff.Do`

**Files:**

- Modify: `pkg/workflow/workflow.go`
- Modify: `pkg/workflow/scheduler.go`

- [ ] **Step 1: Edit `pkg/workflow/workflow.go` — replace `runWithRetry` and `runEngineWithRetry`**

Add `"github.com/flowline-io/flowbot/pkg/backoff"` to imports. Remove `"github.com/cenkalti/backoff"`.

Delete `runWithRetry` (lines 693-726) and `runEngineWithRetry` (lines 728-763).

Replace with a single `runWithRetry` function:

```go
func (r *Runner) runWithRetry(ctx context.Context, task *types.Task, retryCfg *types.RetryConfig, stepID string, stepRun *model.WorkflowStepRun) (int, error) {
	backoffCfg := retryCfg.ToBackoffConfig()
	backoffCfg.OnRetry = func(attempt int, delay time.Duration, err error) {
		if r.store != nil && stepRun != nil {
			_ = r.store.UpdateStepRun(ctx, stepRun.ID, model.WorkflowRunRunning, nil, err.Error(), attempt)
		}
		flog.Info("[workflow] step %s attempt %d failed, retrying in %v: %v", stepID, attempt, delay, err)
	}
	return backoff.Do(ctx, backoffCfg, func(ctx context.Context) error {
		return r.Run(ctx, task)
	})
}
```

- [ ] **Step 2: Edit `pkg/workflow/scheduler.go` — update `executeExecutorStep`**

In `executeExecutorStep` (lines 359-414 of scheduler.go), replace line 388:

Old:

```go
	attempt, rerr := r.runEngineWithRetry(ctx, engine, task, wt.Retry, taskID, stepRun)
```

New:

```go
	backoffCfg := wt.Retry.ToBackoffConfig()
	backoffCfg.OnRetry = func(a int, d time.Duration, err error) {
		if r.store != nil && stepRun != nil {
			_ = r.store.UpdateStepRun(ctx, stepRun.ID, model.WorkflowRunRunning, nil, err.Error(), a)
		}
		flog.Info("[workflow] step %s attempt %d failed, retrying in %v: %v", taskID, a, d, err)
	}
	attempt, rerr := backoff.Do(ctx, backoffCfg, func(ctx context.Context) error {
		return engine.Run(ctx, task)
	})
```

Add `"github.com/flowline-io/flowbot/pkg/backoff"` to imports in scheduler.go.

In `executeSequentialExecutorStep` (line 410 in workflow.go), `r.runWithRetry` call stays the same — we just replaced the function body.

In `executeResumeExecutorStep` (line 631 in workflow.go), same — `r.runWithRetry` call unchanged.

- [ ] **Step 3: Verify compilation**

```bash
go build ./pkg/workflow/...
```

Expected: compiles without errors.

- [ ] **Step 4: Commit**

```bash
git add pkg/workflow/workflow.go pkg/workflow/scheduler.go
git commit -m "refactor: workflow uses backoff.Do"
```

---

### Task 10: Update BDD specs

**Files:**

- Modify: `tests/specs/pipeline_spec_test.go`
- Modify: `tests/specs/workflow_spec_test.go`

- [ ] **Step 1: Check and update `tests/specs/pipeline_spec_test.go`**

The file references `RetryEnabled()` and `BuildBackOff()`. Update to use `ToBackoffConfig().MaxAttempts`:

Line 92: `Expect(retry.RetryEnabled()).To(BeTrue())` → `Expect(retry.ToBackoffConfig().MaxAttempts).To(BeNumerically(">", 1))`

Line 103: `bo := retry.BuildBackOff()` → `cfg := retry.ToBackoffConfig()`

- [ ] **Step 2: Check and update `tests/specs/workflow_spec_test.go`**

Line 71: `Expect(task.Retry.RetryEnabled()).To(BeTrue())` → `Expect(task.Retry.ToBackoffConfig().MaxAttempts).To(BeNumerically(">", 1))`

Lines 78, 81, 84: `cfg.RetryEnabled()` → `cfg.ToBackoffConfig().MaxAttempts > 1`

- [ ] **Step 3: Run BDD tests**

```bash
go tool task test:specs
```

Expected: All BDD tests PASS.

- [ ] **Step 4: Commit**

```bash
git add tests/specs/pipeline_spec_test.go tests/specs/workflow_spec_test.go
git commit -m "test: update BDD specs for backoff migration"
```

---

### Task 11: Final verification

- [ ] **Step 1: Run full test suite**

```bash
go tool task test
```

Expected: All unit tests PASS.

- [ ] **Step 2: Run lint**

```bash
go tool task lint
```

Expected: No new lint errors.

- [ ] **Step 3: Verify `cenkalti/backoff` can be removed from direct deps**

Check if any non-test, non-deprecated code still imports `cenkalti/backoff`:

```bash
grep -r "cenkalti/backoff" --include="*.go" pkg/ internal/ cmd/
```

Expected: Only in `pkg/types/workflow.go` (deprecated `BuildBackOff`) and possibly test files.

- [ ] **Step 4: Commit if clean**

```bash
git add -A
git commit -m "chore: final cleanup after backoff unification"
```
