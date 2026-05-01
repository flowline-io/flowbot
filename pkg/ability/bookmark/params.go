package bookmark

import (
	"fmt"
	"strconv"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/types"
)

func pageRequestFromParams(params map[string]any) ability.PageRequest {
	limit, _ := intParam(params, "limit")
	cursor, _ := stringParam(params, "cursor")
	sortBy, _ := stringParam(params, "sort_by")
	sortOrder, _ := stringParam(params, "sort_order")
	return ability.PageRequest{
		Limit:     limit,
		Cursor:    cursor,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}
}

func requiredString(params map[string]any, key string) (string, error) {
	value, ok := stringParam(params, key)
	if !ok || value == "" {
		return "", types.Errorf(types.ErrInvalidArgument, "%s is required", key)
	}
	return value, nil
}

func stringParam(params map[string]any, key string) (string, bool) {
	value, ok := params[key]
	if !ok || value == nil {
		return "", false
	}
	switch v := value.(type) {
	case string:
		return v, true
	case fmt.Stringer:
		return v.String(), true
	default:
		return fmt.Sprintf("%v", value), true
	}
}

func intParam(params map[string]any, key string) (int, bool) {
	value, ok := params[key]
	if !ok || value == nil {
		return 0, false
	}
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func boolParam(params map[string]any, key string) (bool, bool) {
	value, ok := params[key]
	if !ok || value == nil {
		return false, false
	}
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			return false, false
		}
		return parsed, true
	default:
		return false, false
	}
}

func idAndTags(params map[string]any) (string, []string, error) {
	id, err := requiredString(params, "id")
	if err != nil {
		return "", nil, err
	}
	tags, err := tagsParam(params)
	if err != nil {
		return "", nil, err
	}
	return id, tags, nil
}

func tagsParam(params map[string]any) ([]string, error) {
	value, ok := params["tags"]
	if !ok || value == nil {
		return nil, types.Errorf(types.ErrInvalidArgument, "tags are required")
	}
	switch v := value.(type) {
	case []string:
		if len(v) == 0 {
			return nil, types.Errorf(types.ErrInvalidArgument, "tags are required")
		}
		return v, nil
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			result = append(result, fmt.Sprintf("%v", item))
		}
		if len(result) == 0 {
			return nil, types.Errorf(types.ErrInvalidArgument, "tags are required")
		}
		return result, nil
	default:
		return nil, types.Errorf(types.ErrInvalidArgument, "tags must be an array")
	}
}
