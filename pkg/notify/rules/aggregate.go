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

const (
	aggregateBufferMax   = 1000
	aggregateBufferGrace = 5 * time.Minute
)

func aggregateDueKey() cache.Key {
	return cache.NewKey("notify", "agg", "due")
}

func aggregateMember(ruleID, eventType, channel string) string {
	return ruleID + ":" + eventType + ":" + channel
}

// EnqueueForAggregation adds a payload to the aggregation buffer for later digest delivery.
// The buffer is capped at aggregateBufferMax entries (newest retained).
func (e *Engine) EnqueueForAggregation(ctx context.Context, ruleID, eventType, channel string, payload map[string]any) error {
	if e.store == nil {
		return nil
	}

	key := cache.NewKey("notify", "agg:buffer", ruleID+":"+eventType+":"+channel)

	data, err := sonic.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal aggregate payload: %w", err)
	}

	n, err := e.store.Push(ctx, key, string(data))
	if err != nil {
		return fmt.Errorf("failed to push to aggregate list: %w", err)
	}
	if n > aggregateBufferMax {
		if err := e.store.Trim(ctx, key, -aggregateBufferMax, -1); err != nil {
			return fmt.Errorf("failed to trim aggregate list: %w", err)
		}
	}

	return nil
}

// SetAggregateTimer sets a timer key for the aggregation window.
// Returns true if this is the first element (i.e., timer was created).
// On first create it also indexes the window in a due ZSET for O(log N) expiry scans.
func (e *Engine) SetAggregateTimer(ctx context.Context, ruleID, eventType, channel string, window time.Duration) (bool, error) {
	if e.store == nil {
		return false, nil
	}

	key := cache.NewKey("notify", "agg:timer", ruleID+":"+eventType+":"+channel)

	ok, err := e.store.SetNX(ctx, key, "1", cache.TTL(window))
	if err != nil {
		return false, fmt.Errorf("failed to set aggregate timer: %w", err)
	}
	if !ok {
		return false, nil
	}

	member := aggregateMember(ruleID, eventType, channel)
	dueAt := float64(time.Now().Add(window).Unix())
	if err := e.store.ZAdd(ctx, aggregateDueKey(), dueAt, member); err != nil {
		if delErr := e.store.Del(ctx, key); delErr != nil {
			flog.Warn("[notify-rules] failed to roll back aggregate timer after due index error: %v", delErr)
		}
		return false, fmt.Errorf("failed to index aggregate due: %w", err)
	}

	bufKey := cache.NewKey("notify", "agg:buffer", member)
	if err := e.store.Expire(ctx, bufKey, cache.TTL(window+aggregateBufferGrace)); err != nil {
		flog.Warn("[notify-rules] failed to expire aggregate buffer: %v", err)
	}

	return true, nil
}

// FlushAggregation retrieves all buffered payloads for a given rule/channel and clears the buffer.
func (e *Engine) FlushAggregation(ctx context.Context, ruleID, eventType, channel string) ([]map[string]any, error) {
	if e.store == nil {
		return nil, nil
	}

	member := aggregateMember(ruleID, eventType, channel)
	key := cache.NewKey("notify", "agg:buffer", member)

	items, err := e.store.Range(ctx, key, 0, -1)
	if err != nil {
		return nil, fmt.Errorf("failed to read aggregate list: %w", err)
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
	if _, err := e.store.ZRem(ctx, aggregateDueKey(), member); err != nil {
		flog.Warn("[notify-rules] failed to remove aggregate due member: %v", err)
	}

	return payloads, nil
}

// ScanExpiredAggregates finds aggregation windows whose due score has passed.
func (e *Engine) ScanExpiredAggregates(ctx context.Context) ([]AggregateKey, error) {
	if e.store == nil {
		return nil, nil
	}

	now := float64(time.Now().Unix())
	members, err := e.store.ZRangeByScore(ctx, aggregateDueKey(), 0, now)
	if err != nil {
		return nil, fmt.Errorf("failed to scan aggregate due set: %w", err)
	}

	keys := make([]AggregateKey, 0, len(members))
	for _, member := range members {
		aggKey, ok := parseAggregateMember(member)
		if !ok {
			continue
		}
		keys = append(keys, aggKey)
	}

	return keys, nil
}

// AggregateKey holds the parsed components of an aggregate timer key.
type AggregateKey struct {
	RuleID    string
	EventType string
	Channel   string
}

func parseAggregateMember(member string) (AggregateKey, bool) {
	parts := strings.SplitN(member, ":", 3)
	if len(parts) < 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return AggregateKey{}, false
	}
	return AggregateKey{
		RuleID:    parts[0],
		EventType: parts[1],
		Channel:   parts[2],
	}, true
}
