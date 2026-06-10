package rdb

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
)

// Bloom filter configuration for deduplication.
const (
	bloomErrorRate = 0.001
	bloomCapacity  = 1_000_000
	bloomKeyTTL    = 30 * 24 * time.Hour
	bloomKeyPrefix = "cache:dedup:%s"
)

// BloomUnique filters a slice of items through a Redis bloom filter,
// returning only the items that are newly added (i.e. unique).
// The bloom filter is identified by id and expires after bloomKeyTTL.
func BloomUnique(ctx context.Context, id string, latest []any) ([]any, error) {
	result := make([]any, 0)
	uniqueKey := fmt.Sprintf(bloomKeyPrefix, id)
	Client.BFReserve(ctx, uniqueKey, bloomErrorRate, bloomCapacity)
	if err := Client.Expire(ctx, uniqueKey, bloomKeyTTL).Err(); err != nil {
		flog.Warn("failed to set bloom filter TTL for %s: %v", uniqueKey, err)
	}

	for i, item := range latest {
		val, err := kvHash(item)
		if err != nil {
			return nil, fmt.Errorf("failed to hash kv: %w", err)
		}
		if val == "" {
			continue
		}
		b, err := Client.BFAdd(ctx, uniqueKey, val).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to set unique key: %w", err)
		}
		if b {
			result = append(result, latest[i])
			flog.Info("[unique] key: %s added: %s", id, val)
		}
	}

	return result, nil
}

func kvHash(item any) (string, error) {
	b, err := sonic.ConfigStd.Marshal(item)
	if err != nil {
		return "", fmt.Errorf("failed to marshal kv: %w", err)
	}
	return utils.SHA256(utils.BytesToString(b)), nil
}

// BloomUniqueString checks whether a single string is unique using a Redis bloom
// filter identified by id. Returns true if the string was newly added (unique),
// false if it was already seen. The bloom filter expires after bloomKeyTTL.
func BloomUniqueString(ctx context.Context, id, latest string) (bool, error) {
	uniqueKey := fmt.Sprintf(bloomKeyPrefix, id)
	Client.BFReserve(ctx, uniqueKey, bloomErrorRate, bloomCapacity)
	if err := Client.Expire(ctx, uniqueKey, bloomKeyTTL).Err(); err != nil {
		flog.Warn("failed to set bloom filter TTL for %s: %v", uniqueKey, err)
	}
	b, err := Client.BFAdd(ctx, uniqueKey, latest).Result()
	if err != nil {
		return false, fmt.Errorf("failed to set unique key: %w", err)
	}
	if b {
		return true, nil
	}

	return false, nil
}
