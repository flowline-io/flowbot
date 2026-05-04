package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
)

// EnqueueForAggregation adds a payload to the aggregation buffer for later digest delivery.
func (e *Engine) EnqueueForAggregation(ctx context.Context, ruleID, eventType, channel string, payload map[string]any) error {
	if e.redis == nil {
		return nil
	}

	dataKey := fmt.Sprintf("notify:agg:%s:%s:%s", ruleID, eventType, channel)

	// serialize payload
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal aggregate payload: %w", err)
	}

	// append to list
	if err := e.redis.RPush(ctx, dataKey, data).Err(); err != nil {
		return fmt.Errorf("failed to push to aggregate list: %w", err)
	}

	return nil
}

// SetAggregateTimer sets a timer key for the aggregation window.
// Returns true if this is the first element (i.e., timer was created).
func (e *Engine) SetAggregateTimer(ctx context.Context, ruleID, eventType, channel string, window time.Duration) (bool, error) {
	if e.redis == nil {
		return false, nil
	}

	timerKey := fmt.Sprintf("notify:agg:timer:%s:%s:%s", ruleID, eventType, channel)

	// SET NX: only set if not already exists
	ok, err := e.redis.SetNX(ctx, timerKey, "1", window).Result()
	if err != nil {
		return false, fmt.Errorf("failed to set aggregate timer: %w", err)
	}

	return ok, nil
}

// FlushAggregation retrieves all buffered payloads for a given rule/channel and clears the buffer.
func (e *Engine) FlushAggregation(ctx context.Context, ruleID, eventType, channel string) ([]map[string]any, error) {
	if e.redis == nil {
		return nil, nil
	}

	dataKey := fmt.Sprintf("notify:agg:%s:%s:%s", ruleID, eventType, channel)

	// get all items
	items, err := e.redis.LRange(ctx, dataKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to read aggregate list: %w", err)
	}

	if len(items) == 0 {
		return nil, nil
	}

	// parse items before deleting, to avoid data loss on parse failure
	var payloads []map[string]any
	for _, item := range items {
		var payload map[string]any
		if err := json.Unmarshal([]byte(item), &payload); err != nil {
			flog.Warn("[notify-rules] failed to unmarshal aggregate payload: %v", err)
			continue
		}
		payloads = append(payloads, payload)
	}

	// delete the list after successful parse
	if err := e.redis.Del(ctx, dataKey).Err(); err != nil {
		flog.Warn("[notify-rules] failed to delete aggregate list: %v", err)
	}

	return payloads, nil
}

// ScanExpiredAggregates finds aggregate timer keys that have expired and returns their rule/channel info.
func (e *Engine) ScanExpiredAggregates(ctx context.Context) ([]AggregateKey, error) {
	if e.redis == nil {
		return nil, nil
	}

	var keys []AggregateKey
	var cursor uint64
	for {
		result, nextCursor, err := e.redis.Scan(ctx, cursor, "notify:agg:timer:*", 100).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to scan aggregate timers: %w", err)
		}

		for _, key := range result {
			// check if timer exists (if not, it expired and was auto-deleted)
			exists, err := e.redis.Exists(ctx, key).Result()
			if err != nil {
				continue
			}
			if exists == 0 {
				// timer expired, parse the key to extract rule, event, channel
				if aggKey, ok := parseAggregateKey(key); ok {
					keys = append(keys, aggKey)
				}
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
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
