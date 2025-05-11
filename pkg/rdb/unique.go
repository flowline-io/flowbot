package rdb

import (
	"context"
	"fmt"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
	json "github.com/json-iterator/go"
)

func BloomUnique(ctx context.Context, id string, latest []any) ([]any, error) {
	result := make([]any, 0)
	uniqueKey := fmt.Sprintf("bloom:unique:%s", id)
	Client.BFReserve(ctx, uniqueKey, 0.001, 1000000)

	for i, item := range latest {
		val, err := kvHash(item)
		if err != nil {
			return nil, fmt.Errorf("failed to hash kv: %w", err)
		}
		if len(val) == 0 {
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
	b, err := json.ConfigCompatibleWithStandardLibrary.Marshal(item)
	if err != nil {
		return "", fmt.Errorf("failed to marshal kv: %w", err)
	}
	return utils.SHA1(utils.BytesToString(b)), nil
}

func BloomUniqueString(ctx context.Context, id string, latest string) (bool, error) {
	uniqueKey := fmt.Sprintf("bloom:unique:%s", id)
	Client.BFReserve(ctx, uniqueKey, 0.001, 1000000)
	b, err := Client.BFAdd(ctx, uniqueKey, latest).Result()
	if err != nil {
		return false, fmt.Errorf("failed to set unique key: %w", err)
	}
	if b {
		return true, nil
	}

	return false, nil
}
