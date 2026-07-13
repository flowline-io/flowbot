package agent

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/types"
)

func optionalStringListParam(params map[string]any, key string) ([]string, error) {
	value, ok := params[key]
	if !ok || value == nil {
		return nil, nil
	}
	switch v := value.(type) {
	case []string:
		if len(v) == 0 {
			return nil, nil
		}
		return append([]string(nil), v...), nil
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			} else if item != nil {
				result = append(result, fmt.Sprintf("%v", item))
			}
		}
		if len(result) == 0 {
			return nil, nil
		}
		return result, nil
	default:
		return nil, types.Errorf(types.ErrInvalidArgument, "%s must be an array", key)
	}
}
