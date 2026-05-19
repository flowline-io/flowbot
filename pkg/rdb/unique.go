// Deprecated: Use cache.RedisStore for new cache operations. These bloom helpers
// will be removed in Phase 3 cleanup.
package rdb

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
)

func BloomUnique(ctx context.Context, id string, latest []any) ([]any, error) {
	result := make([]any, 0)
	uniqueKey := fmt.Sprintf("cache:dedup:%s", id)
	Client.BFReserve(ctx, uniqueKey, 0.001, 1000000)
	Client.Expire(ctx, uniqueKey, 30*24*time.Hour)

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
	return utils.SHA1(utils.BytesToString(b)), nil
}

func BloomUniqueString(ctx context.Context, id string, latest string) (bool, error) {
	uniqueKey := fmt.Sprintf("cache:dedup:%s", id)
	Client.BFReserve(ctx, uniqueKey, 0.001, 1000000)
	Client.Expire(ctx, uniqueKey, 30*24*time.Hour)
	b, err := Client.BFAdd(ctx, uniqueKey, latest).Result()
	if err != nil {
		return false, fmt.Errorf("failed to set unique key: %w", err)
	}
	if b {
		return true, nil
	}

	return false, nil
}
