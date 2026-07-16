package rules

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/config"
)

func newTestRedisStore(t *testing.T) *cache.RedisStore {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return cache.NewRedisStore(client)
}

func TestCheckThrottle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		limit     int
		calls     int
		wantAllow []bool
	}{
		{
			name:      "first call within limit is allowed",
			limit:     2,
			calls:     1,
			wantAllow: []bool{true},
		},
		{
			name:      "third call exceeds limit of two",
			limit:     2,
			calls:     3,
			wantAllow: []bool{true, true, false},
		},
		{
			name:      "single limit allows one then blocks",
			limit:     1,
			calls:     2,
			wantAllow: []bool{true, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := newTestRedisStore(t)
			engine := New(store)
			ctx := context.Background()

			var got []bool
			for range tt.calls {
				allowed, err := engine.CheckThrottle(ctx, "rule1", "event.a", "slack", time.Minute, tt.limit)
				require.NoError(t, err)
				got = append(got, allowed)
			}
			assert.Equal(t, tt.wantAllow, got)
		})
	}
}

func TestCheckThrottle_NilStore(t *testing.T) {
	t.Parallel()
	engine := New(nil)
	allowed, err := engine.CheckThrottle(context.Background(), "r", "e", "c", time.Minute, 1)
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestClearThrottle(t *testing.T) {
	t.Parallel()
	store := newTestRedisStore(t)
	engine := New(store)
	ctx := context.Background()

	allowed, err := engine.CheckThrottle(ctx, "rule1", "event.a", "slack", time.Minute, 1)
	require.NoError(t, err)
	require.True(t, allowed)

	allowed, err = engine.CheckThrottle(ctx, "rule1", "event.a", "slack", time.Minute, 1)
	require.NoError(t, err)
	require.False(t, allowed)

	engine.ClearThrottle(ctx, "rule1", "event.a", "slack")

	allowed, err = engine.CheckThrottle(ctx, "rule1", "event.a", "slack", time.Minute, 1)
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestEnqueueAndFlushAggregation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		payloads []map[string]any
		wantLen  int
	}{
		{
			name: "single payload round-trips through buffer",
			payloads: []map[string]any{
				{"summary": "host down", "host": "nas"},
			},
			wantLen: 1,
		},
		{
			name: "multiple payloads preserve order",
			payloads: []map[string]any{
				{"n": 1},
				{"n": 2},
				{"n": 3},
			},
			wantLen: 3,
		},
		{
			name:     "empty buffer returns nil",
			payloads: nil,
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := newTestRedisStore(t)
			engine := New(store)
			ctx := context.Background()

			for _, payload := range tt.payloads {
				err := engine.EnqueueForAggregation(ctx, "agg1", "infra.down", "slack", payload)
				require.NoError(t, err)
			}

			got, err := engine.FlushAggregation(ctx, "agg1", "infra.down", "slack")
			require.NoError(t, err)
			if tt.wantLen == 0 {
				assert.Nil(t, got)
				return
			}
			require.Len(t, got, tt.wantLen)
			assert.Equal(t, tt.payloads[0]["summary"], got[0]["summary"])
		})
	}
}

func TestSetAggregateTimer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		calls     int
		wantFirst []bool
	}{
		{
			name:      "first call creates timer",
			calls:     1,
			wantFirst: []bool{true},
		},
		{
			name:      "second call does not recreate timer",
			calls:     2,
			wantFirst: []bool{true, false},
		},
		{
			name:      "three calls only first is new timer",
			calls:     3,
			wantFirst: []bool{true, false, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := newTestRedisStore(t)
			engine := New(store)
			ctx := context.Background()

			var got []bool
			for range tt.calls {
				first, err := engine.SetAggregateTimer(ctx, "agg1", "event.a", "slack", time.Minute)
				require.NoError(t, err)
				got = append(got, first)
			}
			assert.Equal(t, tt.wantFirst, got)
		})
	}
}

func TestParseAggregateKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		key    string
		want   AggregateKey
		wantOK bool
	}{
		{
			name:   "valid timer key parses components",
			key:    "notify:agg:timer:rule1:infra.down:slack",
			want:   AggregateKey{RuleID: "rule1", EventType: "infra.down", Channel: "slack"},
			wantOK: true,
		},
		{
			name:   "channel with colon in event type",
			key:    "notify:agg:timer:r:e:extra:ntfy",
			want:   AggregateKey{RuleID: "r", EventType: "e", Channel: "extra:ntfy"},
			wantOK: true,
		},
		{
			name:   "too few segments is rejected",
			key:    "notify:agg:timer:only",
			wantOK: false,
		},
		{
			name:   "empty key is rejected",
			key:    "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := parseAggregateKey(tt.key)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestScanExpiredAggregates_NilStore(t *testing.T) {
	t.Parallel()
	engine := New(nil)
	keys, err := engine.ScanExpiredAggregates(context.Background())
	require.NoError(t, err)
	assert.Nil(t, keys)
}

func TestReload(t *testing.T) {
	t.Parallel()
	engine := New(nil)
	err := engine.Reload(context.Background(), func(_ context.Context) ([]config.NotifyRule, error) {
		return []config.NotifyRule{
			{ID: "r1", Action: config.NotifyRuleActionDrop, Match: config.NotifyRuleMatch{Event: "*", Channel: "*"}, Priority: 1},
		}, nil
	})
	require.NoError(t, err)

	result := engine.Evaluate(context.Background(), "any", "slack")
	require.NotNil(t, result)
	assert.Equal(t, "r1", result.RuleID)
}

func TestGetEngine_AfterInit(t *testing.T) {
	prevRules := config.App.Notify.Rules
	config.App.Notify.Rules = []config.NotifyRule{
		{ID: "init_test", Action: config.NotifyRuleActionDrop, Match: config.NotifyRuleMatch{Event: "x", Channel: "*"}, Priority: 1},
	}
	t.Cleanup(func() { config.App.Notify.Rules = prevRules })

	require.NoError(t, Init(newTestRedisStore(t)))
	engine := GetEngine()
	require.NotNil(t, engine)
	result := engine.Evaluate(context.Background(), "x", "slack")
	require.NotNil(t, result)
	assert.Equal(t, "init_test", result.RuleID)
}

func TestWorker_ScanAndFlush(t *testing.T) {
	store := newTestRedisStore(t)
	engine := New(store)
	ctx := context.Background()

	require.NoError(t, engine.EnqueueForAggregation(ctx, "w1", "event.a", "slack", map[string]any{"n": 1}))
	require.NoError(t, engine.EnqueueForAggregation(ctx, "w1", "event.a", "slack", map[string]any{"n": 2}))
	_, err := engine.SetAggregateTimer(ctx, "w1", "event.a", "slack", time.Minute)
	require.NoError(t, err)

	var flushed int
	var flushedItems []map[string]any
	w := NewWorker(engine, time.Hour, func(_ context.Context, ruleID, eventType, channel string, items []map[string]any) {
		flushed++
		flushedItems = items
		assert.Equal(t, "w1", ruleID)
		assert.Equal(t, "event.a", eventType)
		assert.Equal(t, "slack", channel)
	})

	// Active timer means the aggregation window is still open.
	w.scanAndFlush(ctx)
	assert.Equal(t, 0, flushed)

	// Manual flush path via FlushAggregation + onFlush callback.
	items, err := engine.FlushAggregation(ctx, "w1", "event.a", "slack")
	require.NoError(t, err)
	require.Len(t, items, 2)
	if len(items) > 0 {
		w.onFlush(ctx, "w1", "event.a", "slack", items)
	}
	assert.Equal(t, 1, flushed)
	assert.Len(t, flushedItems, 2)
}

func TestWorker_RunStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	w := NewWorker(New(nil), 5*time.Millisecond, nil)

	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not stop after context cancel")
	}
}
