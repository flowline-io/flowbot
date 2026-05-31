package web

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/cache"
)

// mockRateLimitStore is an in-memory test double implementing rateLimitStore.
type mockRateLimitStore struct {
	mu    sync.RWMutex
	ints  map[string]int64
	strs  map[string]bool
}

func newMockRateLimitStore() *mockRateLimitStore {
	return &mockRateLimitStore{
		ints: make(map[string]int64),
		strs: make(map[string]bool),
	}
}

func (m *mockRateLimitStore) GetInt64(_ context.Context, key cache.Key) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ints[key.String()], nil
}

func (m *mockRateLimitStore) SetInt64(_ context.Context, key cache.Key, value int64, _ cache.TTL) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ints[key.String()] = value
	m.strs[key.String()] = true // make visible to Exists
	return nil
}

func (m *mockRateLimitStore) Incr(_ context.Context, key cache.Key) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ints[key.String()]++
	return m.ints[key.String()], nil
}

func (m *mockRateLimitStore) IncrWithTTL(_ context.Context, key cache.Key, _ cache.TTL) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ints[key.String()]++
	return m.ints[key.String()], nil
}

func (m *mockRateLimitStore) Exists(_ context.Context, key cache.Key) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.strs[key.String()], nil
}

func (m *mockRateLimitStore) Del(_ context.Context, key cache.Key) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.ints, key.String())
	delete(m.strs, key.String())
	return nil
}

func (m *mockRateLimitStore) setLock(ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.strs[lockKeyStr(ip)] = true
}

func (m *mockRateLimitStore) setAttempts(ip string, n int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ints[attemptKeyStr(ip)] = n
}

func attemptKeyStr(ip string) string {
	return attemptKey(ip).String()
}

func lockKeyStr(ip string) string {
	return lockKey(ip).String()
}

func TestLoginRateLimiterAllow(t *testing.T) {
	tests := []struct {
		name       string
		ip         string
		setup      func(*mockRateLimitStore)
		wantDelay  bool
		wantLocked bool
	}{
		{
			name:       "first attempt is allowed with no delay",
			ip:         "10.0.0.1",
			wantDelay:  false,
			wantLocked: false,
		},
		{
			name: "locked IP returns locked",
			ip:   "10.0.0.2",
			setup: func(m *mockRateLimitStore) {
				m.setLock("10.0.0.2")
			},
			wantDelay:  false,
			wantLocked: true,
		},
		{
			name: "below threshold returns no delay",
			ip:   "10.0.0.3",
			setup: func(m *mockRateLimitStore) {
				m.setAttempts("10.0.0.3", 3)
			},
			wantDelay:  false,
			wantLocked: false,
		},
		{
			name: "at threshold returns progressive delay",
			ip:   "10.0.0.4",
			setup: func(m *mockRateLimitStore) {
				m.setAttempts("10.0.0.4", 5)
			},
			wantDelay:  true,
			wantLocked: false,
		},
		{
			name: "above threshold returns progressive delay",
			ip:   "10.0.0.5",
			setup: func(m *mockRateLimitStore) {
				m.setAttempts("10.0.0.5", 8)
			},
			wantDelay:  true,
			wantLocked: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockRateLimitStore()
			if tt.setup != nil {
				tt.setup(store)
			}
			l := newLoginRateLimiter(store, 5, 10, cache.TTLMedium, cache.TTLMedium)
			delay, locked := l.Allow(context.Background(), tt.ip)

			if tt.wantLocked != locked {
				t.Errorf("Allow() locked = %v, want %v", locked, tt.wantLocked)
			}
			if tt.wantDelay != (delay > 0) {
				t.Errorf("Allow() delay = %v, wantDelay = %v", delay, tt.wantDelay)
			}
		})
	}
}

func TestLoginRateLimiterAllowDelayCalculation(t *testing.T) {
	tests := []struct {
		name      string
		attempts  int64
		wantDelay time.Duration
	}{
		{
			name:      "no delay below threshold",
			attempts:  4,
			wantDelay: 0,
		},
		{
			name:      "delay at threshold",
			attempts:  5,
			wantDelay: 5 * time.Second, // 2^5=32s capped at 5s
		},
		{
			name:      "delay capped at max",
			attempts:  10,
			wantDelay: 5 * time.Second, // 2^10=1024s capped at 5s
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockRateLimitStore()
			store.setAttempts("10.0.0.1", tt.attempts)
			l := newLoginRateLimiter(store, 5, 10, cache.TTLMedium, cache.TTLMedium)
			delay, locked := l.Allow(context.Background(), "10.0.0.1")

			if locked {
				t.Fatal("unexpected lockout")
			}
			if delay != tt.wantDelay {
				t.Errorf("Allow() delay = %v, want %v", delay, tt.wantDelay)
			}
		})
	}
}

func TestLoginRateLimiterRecordFailure(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(*mockRateLimitStore)
		ip             string
		wantLocked     bool
		wantRetryAfter bool
	}{
		{
			name:           "first failure does not lock",
			ip:             "10.0.0.1",
			wantLocked:     false,
			wantRetryAfter: false,
		},
		{
			name: "above threshold but below lockout does not lock",
			ip:   "10.0.0.2",
			setup: func(m *mockRateLimitStore) {
				m.setAttempts("10.0.0.2", 8)
			},
			wantLocked:     false,
			wantRetryAfter: false,
		},
		{
			name: "reaching lockout limit sets lock",
			ip:   "10.0.0.3",
			setup: func(m *mockRateLimitStore) {
				m.setAttempts("10.0.0.3", 9)
			},
			wantLocked:     true,
			wantRetryAfter: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockRateLimitStore()
			if tt.setup != nil {
				tt.setup(store)
			}
			l := newLoginRateLimiter(store, 5, 10, cache.TTLMedium, cache.TTLMedium)
			locked, retryAfter := l.RecordFailure(context.Background(), tt.ip)

			if tt.wantLocked != locked {
				t.Errorf("RecordFailure() locked = %v, want %v", locked, tt.wantLocked)
			}
			if tt.wantRetryAfter != (retryAfter > 0) {
				t.Errorf("RecordFailure() retryAfter = %v, wantRetryAfter = %v", retryAfter, tt.wantRetryAfter)
			}
		})
	}
}

func TestLoginRateLimiterRecordSuccess(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*mockRateLimitStore)
		ip    string
	}{
		{
			name: "clears attempt counter",
			ip:   "10.0.0.1",
			setup: func(m *mockRateLimitStore) {
				m.setAttempts("10.0.0.1", 5)
			},
		},
		{
			name: "clears lock state",
			ip:   "10.0.0.2",
			setup: func(m *mockRateLimitStore) {
				m.setLock("10.0.0.2")
			},
		},
		{
			name: "clears both attempt and lock",
			ip:   "10.0.0.3",
			setup: func(m *mockRateLimitStore) {
				m.setAttempts("10.0.0.3", 9)
				m.setLock("10.0.0.3")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockRateLimitStore()
			if tt.setup != nil {
				tt.setup(store)
			}
			l := newLoginRateLimiter(store, 5, 10, cache.TTLMedium, cache.TTLMedium)
			l.RecordSuccess(context.Background(), tt.ip)

			attemptVal, _ := store.GetInt64(context.Background(), attemptKey(tt.ip))
			if attemptVal != 0 {
				t.Errorf("RecordSuccess() did not clear attempts, got %d", attemptVal)
			}
			exists, _ := store.Exists(context.Background(), lockKey(tt.ip))
			if exists {
				t.Error("RecordSuccess() did not clear lock")
			}
		})
	}
}
