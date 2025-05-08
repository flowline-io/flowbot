package rdb

import (
	"context"
	"fmt"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	json "github.com/json-iterator/go"
)

func Unique(ctx context.Context, id string, latest []any) ([]types.KV, error) {
	result := make([]types.KV, 0)
	uniqueKey := fmt.Sprintf("unique:%s", id)

	for _, item := range latest {
		val, err := kvHash(item)
		if err != nil {
			return nil, fmt.Errorf("failed to hash kv: %w", err)
		}
		if len(val) == 0 {
			continue
		}
		b, err := Client.SAdd(ctx, uniqueKey, val).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to set unique key: %w", err)
		}
		if b == 1 {
			kv, ok := item.(map[string]any)
			if !ok {
				continue
			}
			result = append(result, kv)
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

func UniqueString(ctx context.Context, id string, latest string) (bool, error) {
	uniqueKey := fmt.Sprintf("unique:%s", id)
	b, err := Client.SAdd(ctx, uniqueKey, latest).Result()
	if err != nil {
		return false, fmt.Errorf("failed to set unique key: %w", err)
	}
	if b == 1 {
		return true, nil
	}

	return false, nil
}
