package dev

import (
	"context"
	"fmt"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	json "github.com/json-iterator/go"
	"regexp"
)

func unique(ctx context.Context, id string, latest []any) ([]types.KV, error) {
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
		b, err := cache.DB.SAdd(ctx, uniqueKey, val).Result()
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
	b, err := json.Marshal(item)
	if err != nil {
		return "", fmt.Errorf("failed to marshal kv: %w", err)
	}
	return utils.SHA1(utils.BytesToString(b)), nil
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
