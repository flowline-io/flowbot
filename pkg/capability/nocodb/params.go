package nocodb

import (
	"github.com/flowline-io/flowbot/pkg/types"
)

func requiredFields(params map[string]any, key string) (map[string]any, error) {
	v, ok := params[key]
	if !ok || v == nil {
		return nil, types.Errorf(types.ErrInvalidArgument, "%s is required", key)
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, types.Errorf(types.ErrInvalidArgument, "%s must be an object", key)
	}
	if len(m) == 0 {
		return nil, types.Errorf(types.ErrInvalidArgument, "%s is required", key)
	}
	return m, nil
}
