package karakeep

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/types"
)

func idAndTags(params map[string]any) (string, []string, error) {
	id, err := capability.RequiredString(params, "id")
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

func listInvokeResult(operation string, result *capability.ListResult[capability.Bookmark]) *capability.InvokeResult {
	if result == nil {
		result = &capability.ListResult[capability.Bookmark]{Items: []*capability.Bookmark{}, Page: &capability.PageInfo{}}
	}
	return &capability.InvokeResult{Operation: operation, Data: result.Items, Page: result.Page}
}
