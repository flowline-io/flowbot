package dev

import (
	"context"
	"fmt"
	"regexp"

	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/utils"
	json "github.com/json-iterator/go"
)

func unique(ctx context.Context, id string, latest []any) ([]types.KV, error) {
	uniqueKey := fmt.Sprintf("unique:%s", id)

	oldArr, err := cache.DB.SMembers(ctx, uniqueKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get unique key: %w", err)
	}

	diff, err := kvDiff(latest, oldArr)
	if err != nil {
		return nil, fmt.Errorf("failed to diff kv: %w", err)
	}

	for _, item := range diff {
		val, err := kvHash(item)
		if err != nil {
			return nil, fmt.Errorf("failed to hash kv: %w", err)
		}
		if len(val) == 0 {
			continue
		}
		err = cache.DB.SAdd(ctx, uniqueKey, val).Err()
		if err != nil {
			return nil, fmt.Errorf("failed to set unique key: %w", err)
		}
	}

	return diff, nil
}

func kvHash(item types.KV) (string, error) {
	b, err := json.Marshal(item)
	if err != nil {
		return "", fmt.Errorf("failed to marshal kv: %w", err)
	}
	return utils.SHA1(utils.BytesToString(b)), nil
}

func kvDiff(latest []any, old []string) ([]types.KV, error) {
	result := make([]types.KV, 0, len(latest))
	for _, item := range latest {
		kv, ok := item.(map[string]any)
		if !ok {
			continue
		}
		val, err := kvHash(kv)
		if err != nil {
			return nil, fmt.Errorf("failed to hash kv: %w", err)
		}
		if len(val) == 0 {
			continue
		}

		if !utils.InStringSlice(old, val) {
			result = append(result, kv)
		}
	}
	return result, nil
}

func kvGrep(pattern string, input types.KV) (types.KV, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	var data = make(map[string][]types.KV)

	for k, v := range input {
		list, ok := v.([]any)
		if !ok {
			continue
		}

		for _, item := range list {
			kv, ok := item.(map[string]any)
			if !ok {
				continue
			}

			for _, value := range kv {
				valueStr, ok := value.(string)
				if !ok {
					continue
				}

				if re.MatchString(valueStr) {
					data[k] = append(data[k], kv)
				}
			}
		}
	}

	result := make(types.KV)
	for k := range data {
		result[k] = data[k]
	}

	return result, nil
}
