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
)

func TestScanExpiredAggregates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(*testing.T, *Engine, *miniredis.Miniredis)
		wantLen  int
		wantRule string
	}{
		{
			name: "expired timer key is returned",
			setup: func(t *testing.T, engine *Engine, mr *miniredis.Miniredis) {
				t.Helper()
				ctx := context.Background()
				require.NoError(t, engine.EnqueueForAggregation(ctx, "rule-exp", "event.a", "slack", map[string]any{"n": 1}))
				_, err := engine.SetAggregateTimer(ctx, "rule-exp", "event.a", "slack", 50*time.Millisecond)
				require.NoError(t, err)
				mr.FastForward(100 * time.Millisecond)
			},
			wantLen:  1,
			wantRule: "rule-exp",
		},
		{
			name: "active timer is not returned",
			setup: func(t *testing.T, engine *Engine, _ *miniredis.Miniredis) {
				t.Helper()
				ctx := context.Background()
				require.NoError(t, engine.EnqueueForAggregation(ctx, "rule-live", "event.b", "slack", map[string]any{"n": 1}))
				_, err := engine.SetAggregateTimer(ctx, "rule-live", "event.b", "slack", time.Minute)
				require.NoError(t, err)
			},
			wantLen: 0,
		},
		{
			name:    "nil store returns empty",
			setup:   func(_ *testing.T, _ *Engine, _ *miniredis.Miniredis) {},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var engine *Engine
			var mr *miniredis.Miniredis
			if tt.name == "nil store returns empty" {
				engine = New(nil)
			} else {
				mr = miniredis.RunT(t)
				client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
				engine = New(cache.NewRedisStore(client))
			}
			tt.setup(t, engine, mr)

			keys, err := engine.ScanExpiredAggregates(context.Background())
			require.NoError(t, err)
			assert.Len(t, keys, tt.wantLen)
			if tt.wantRule != "" {
				assert.Equal(t, tt.wantRule, keys[0].RuleID)
			}
		})
	}
}

func TestWorkerScanAndFlushExpiredTimer(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	engine := New(cache.NewRedisStore(client))
	ctx := context.Background()

	require.NoError(t, engine.EnqueueForAggregation(ctx, "worker-rule", "event.z", "ntfy", map[string]any{"n": 1}))
	_, err := engine.SetAggregateTimer(ctx, "worker-rule", "event.z", "ntfy", 20*time.Millisecond)
	require.NoError(t, err)
	mr.FastForward(30 * time.Millisecond)

	var flushed int
	w := NewWorker(engine, time.Hour, func(_ context.Context, ruleID, eventType, channel string, items []map[string]any) {
		flushed++
		assert.Equal(t, "worker-rule", ruleID)
		assert.Equal(t, "event.z", eventType)
		assert.Equal(t, "ntfy", channel)
		assert.Len(t, items, 1)
	})
	w.scanAndFlush(ctx)
	assert.Equal(t, 1, flushed)
}
