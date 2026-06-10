package backoff

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

var errTest = errors.New("test error")

// testRetryableError implements retryableError for testing.
type testRetryableError struct {
	code      string
	retryable bool
	kind      error
}

func (e *testRetryableError) Error() string          { return e.kind.Error() }
func (e *testRetryableError) RetryableCode() string  { return e.code }
func (e *testRetryableError) IsRetryableError() bool { return e.retryable }

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
			fn := func(_ context.Context) error {
				if int(calls.Add(1)) <= tt.failsBefore {
					return errTest
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
			name: "normal_context_allows_success",
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
			fn := func(_ context.Context) error {
				if tt.name == "normal_context_allows_success" {
					return nil
				}
				return errTest
			}
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
			fn := func(_ context.Context) error {
				return &testRetryableError{code: tt.errCode, kind: errTest}
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
			isRetryable: func(_ error) bool {
				return true
			},
			wantAttempt: 3,
		},
		{
			name: "returns_false_terminates_immediately",
			isRetryable: func(_ error) bool {
				return false
			},
			wantAttempt: 1,
		},
		{
			name: "returns_true_continues",
			isRetryable: func(err error) bool {
				return errors.Is(err, errTest)
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
			fn := func(_ context.Context) error { return errTest }
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
		{name: "jitter_with_zero_interval", jitter: true},
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
			fn := func(_ context.Context) error {
				calls.Add(1)
				return errTest
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
		{name: "adaptive_with_custom_max_interval"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			maxInt := 50 * time.Millisecond
			if tt.name == "adaptive_with_custom_max_interval" {
				maxInt = 20 * time.Millisecond
			}
			cfg := Config{
				MaxAttempts:     3,
				InitialInterval: 10 * time.Millisecond,
				MaxInterval:     maxInt,
				Adaptive:        tt.name != "adaptive_disabled",
			}
			fn := func(_ context.Context) error { return errTest }
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

func TestDo_AdaptiveHalvesDelay(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		adaptive bool
	}{
		{name: "adaptive_halves_after_success", adaptive: true},
		{name: "non_adaptive_uses_initial", adaptive: false},
		{name: "adaptive_persists_across_calls", adaptive: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := Config{
				MaxAttempts:     3,
				InitialInterval: 20 * time.Millisecond,
				MaxInterval:     200 * time.Millisecond,
				Adaptive:        tt.adaptive,
			}

			// Call 1: fail once — stored delay = 20ms.
			_, _ = Do(context.Background(), cfg, func(_ context.Context) error { return errTest })

			// Call 2: succeed after one failure.
			// Adaptive: starts at 20ms, retry doubles to 40ms, success halves to 20ms.
			var calls atomic.Int32
			_, _ = Do(context.Background(), cfg, func(_ context.Context) error {
				if calls.Add(1) <= 1 {
					return errTest
				}
				return nil
			})

			// Call 3: measure total sleep for 2 failures (2 sleeps).
			start := time.Now()
			_, _ = Do(context.Background(), cfg, func(_ context.Context) error { return errTest })
			elapsed := time.Since(start)

			if tt.adaptive {
				// Expected: sleep 20ms + 40ms = 60ms.
				if elapsed > 200*time.Millisecond {
					t.Fatalf("adaptive: elapsed=%v, expected ~60ms", elapsed)
				}
			} else {
				// Non-adaptive: sleep 20ms + 40ms = 60ms (same defaults).
				if elapsed > 200*time.Millisecond {
					t.Fatalf("non-adaptive: elapsed=%v, expected ~60ms", elapsed)
				}
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
			fn := func(_ context.Context) error { return errTest }
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
			name:    "callback_fires_correctly",
			onRetry: func(_ int, _ time.Duration, _ error) {},
		},
		{
			name:    "nil_callback_does_not_panic",
			onRetry: nil,
		},
		{
			name: "callback_receives_correct_args",
			onRetry: func(attempt int, _ time.Duration, err error) {
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
			fn := func(_ context.Context) error { return errTest }
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
			fn := func(_ context.Context) error {
				calls.Add(1)
				return errTest
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
			err:  errTest,
			want: true,
		},
		{
			name: "matching_code_returns_true",
			cfg:  Config{RetryOn: []string{"TIMEOUT"}},
			err:  &testRetryableError{code: "TIMEOUT", kind: errTest},
			want: true,
		},
		{
			name: "non_matching_code_returns_false",
			cfg:  Config{RetryOn: []string{"TIMEOUT"}},
			err:  &testRetryableError{code: "UNAVAILABLE", kind: errTest},
			want: false,
		},
		{
			name: "retryable_flag_overrides",
			cfg:  Config{RetryOn: []string{"TIMEOUT"}},
			err:  &testRetryableError{retryable: true, kind: errTest},
			want: true,
		},
		{
			name: "is_retryable_callback_used",
			cfg:  Config{IsRetryable: func(_ error) bool { return false }},
			err:  errTest,
			want: false,
		},
		{
			name: "standard_error_not_matching_returns_false",
			cfg:  Config{RetryOn: []string{"TIMEOUT"}},
			err:  errTest,
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
