package rules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
)

// EnqueueForAggregation adds a payload to the aggregation buffer for later digest delivery.
func (e *Engine) EnqueueForAggregation(ctx context.Context, ruleID, eventType, channel string, payload map[string]any) error {
	if e.store == nil {
		return nil
	}

	key := cache.NewKey("notify", "agg:buffer", ruleID+":"+eventType+":"+channel)

	data, err := sonic.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal aggregate payload: %w", err)
	}

	if _, err := e.store.Push(ctx, key, string(data)); err != nil {
		return fmt.Errorf("failed to push to aggregate list: %w", err)
	}

	return nil
}

// SetAggregateTimer sets a timer key for the aggregation window.
// Returns true if this is the first element (i.e., timer was created).
func (e *Engine) SetAggregateTimer(ctx context.Context, ruleID, eventType, channel string, window time.Duration) (bool, error) {
	if e.store == nil {
		return false, nil
	}

	key := cache.NewKey("notify", "agg:timer", ruleID+":"+eventType+":"+channel)

	ok, err := e.store.SetNX(ctx, key, "1", cache.TTL(window))
	if err != nil {
		return false, fmt.Errorf("failed to set aggregate timer: %w", err)
	}

	return ok, nil
}

// FlushAggregation retrieves all buffered payloads for a given rule/channel and clears the buffer.
func (e *Engine) FlushAggregation(ctx context.Context, ruleID, eventType, channel string) ([]map[string]any, error) {
	if e.store == nil {
		return nil, nil
	}

	key := cache.NewKey("notify", "agg:buffer", ruleID+":"+eventType+":"+channel)

	items, err := e.store.Range(ctx, key, 0, -1)
	if err != nil {
		return nil, fmt.Errorf("failed to read aggregate list: %w", err)
	}

	if len(items) == 0 {
		return nil, nil
	}

	var payloads []map[string]any
	for _, item := range items {
		var payload map[string]any
		if err := sonic.Unmarshal([]byte(item), &payload); err != nil {
			flog.Warn("[notify-rules] failed to unmarshal aggregate payload: %v", err)
			continue
		}
		payloads = append(payloads, payload)
	}

	if err := e.store.Clear(ctx, key); err != nil {
		flog.Warn("[notify-rules] failed to delete aggregate list: %v", err)
	}

	return payloads, nil
}

// ScanExpiredAggregates finds aggregate timer keys that have expired and returns their rule/channel info.
func (e *Engine) ScanExpiredAggregates(ctx context.Context) ([]AggregateKey, error) {
	if e.store == nil {
		return nil, nil
	}

	var keys []AggregateKey

	results, err := e.store.ScanKeys(ctx, "notify:agg:timer:*", 100)
	if err != nil {
		return nil, fmt.Errorf("failed to scan aggregate timers: %w", err)
	}

	for _, key := range results {
		exists, err := e.store.ExistsRaw(ctx, key)
		if err != nil {
			continue
		}
		if !exists {
			if aggKey, ok := parseAggregateKey(key); ok {
				keys = append(keys, aggKey)
			}
		}
	}

	return keys, nil
}

// AggregateKey holds the parsed components of an aggregate timer key.
type AggregateKey struct {
	RuleID    string
	EventType string
	Channel   string
}

func parseAggregateKey(key string) (AggregateKey, bool) {
	// key format: notify:agg:timer:{ruleID}:{eventType}:{channel}
	parts := strings.SplitN(key, ":", 6)
	if len(parts) < 6 {
		return AggregateKey{}, false
	}
	return AggregateKey{
		RuleID:    parts[3],
		EventType: parts[4],
		Channel:   parts[5],
	}, true
}
