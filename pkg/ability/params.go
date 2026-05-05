package ability

import (
	"fmt"
	"strconv"

	"github.com/flowline-io/flowbot/pkg/types"
)

func PageRequestFromParams(params map[string]any) PageRequest {
	limit, _ := IntParam(params, "limit")
	cursor, _ := StringParam(params, "cursor")
	sortBy, _ := StringParam(params, "sort_by")
	sortOrder, _ := StringParam(params, "sort_order")
	return PageRequest{
		Limit:     limit,
		Cursor:    cursor,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}
}

func RequiredString(params map[string]any, key string) (string, error) {
	value, ok := StringParam(params, key)
	if !ok || value == "" {
		return "", types.Errorf(types.ErrInvalidArgument, "%s is required", key)
	}
	return value, nil
}

func StringParam(params map[string]any, key string) (string, bool) {
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

func IntParam(params map[string]any, key string) (int, bool) {
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

func RequiredInt(params map[string]any, key string) (int, error) {
	value, ok := IntParam(params, key)
	if !ok {
		return 0, types.Errorf(types.ErrInvalidArgument, "%s is required", key)
	}
	return value, nil
}

func Int64Param(params map[string]any, key string) (int64, bool) {
	value, ok := params[key]
	if !ok || value == nil {
		return 0, false
	}
	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case float64:
		return int64(v), true
	}
	return 0, false
}

func RequiredInt64(params map[string]any, key string) (int64, error) {
	value, ok := Int64Param(params, key)
	if !ok {
		return 0, types.Errorf(types.ErrInvalidArgument, "%s is required", key)
	}
	return value, nil
}

func BoolParam(params map[string]any, key string) (bool, bool) {
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
