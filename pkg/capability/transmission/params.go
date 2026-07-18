package transmission

import (
	"fmt"
	"strconv"

	"github.com/flowline-io/flowbot/pkg/types"
)

// requiredInt64Slice extracts a required non-empty []int64 from invoke params.
func requiredInt64Slice(params map[string]any, key string) ([]int64, error) {
	value, ok := params[key]
	if !ok || value == nil {
		return nil, types.Errorf(types.ErrInvalidArgument, "%s is required", key)
	}
	ids, err := toInt64Slice(value)
	if err != nil {
		return nil, types.WrapError(types.ErrInvalidArgument, fmt.Sprintf("invalid %s", key), err)
	}
	if len(ids) == 0 {
		return nil, types.Errorf(types.ErrInvalidArgument, "%s is required", key)
	}
	return ids, nil
}

func toInt64Slice(value any) ([]int64, error) {
	switch v := value.(type) {
	case []int64:
		return append([]int64(nil), v...), nil
	case []int:
		out := make([]int64, len(v))
		for i, n := range v {
			out[i] = int64(n)
		}
		return out, nil
	case []any:
		out := make([]int64, 0, len(v))
		for _, item := range v {
			n, err := toInt64(item)
			if err != nil {
				return nil, err
			}
			out = append(out, n)
		}
		return out, nil
	default:
		n, err := toInt64(v)
		if err != nil {
			return nil, err
		}
		return []int64{n}, nil
	}
}

func toInt64(value any) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, err
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unsupported id type %T", value)
	}
}
