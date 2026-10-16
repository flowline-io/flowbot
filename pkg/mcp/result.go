package mcp

import (
	"errors"
	"reflect"

	"github.com/bytedance/sonic"
)

// ServerInstructions is the MCP initialize instructions string.
const ServerInstructions = "flowbot operates a self-hosted homelab. Read before write: call .list or .get first. Tools with destructiveHint delete or remove data. On error, parse structuredContent.error; the text body is the same JSON. Retry only when retryable is true, honoring retry_after_ms when present. Use hint and field to correct arguments. Every result includes trace.request_id. Capability pages use page.next_cursor when that field is present."

// OpenWorld reports whether tools in group may affect systems outside this process.
// hub, pipeline, workflow, and example stay closed.
func OpenWorld(group string) bool {
	switch group {
	case "hub", "pipeline", "workflow", "example":
		return false
	default:
		return true
	}
}

// Title is the human-readable tool title: "{group}: {description}".
func Title(group, description string) string {
	if description == "" {
		return group
	}
	return group + ": " + description
}

// AttachTrace adds trace beside a successful tool result.
// wrapItems is true only for pipeline.list, workflow.list, function.list, and hub.apps.
// Those slices become {"items", "trace"}. Objects gain a trace field.
// A nil result becomes {"trace": trace}.
func AttachTrace(result any, trace map[string]any, wrapItems bool) (any, error) {
	if trace == nil {
		trace = map[string]any{}
	}
	if result == nil {
		return map[string]any{"trace": trace}, nil
	}
	rv := reflect.ValueOf(result)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return map[string]any{"trace": trace}, nil
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		if !wrapItems {
			return nil, errors.New("mcp result list is not an items response")
		}
		return map[string]any{"items": result, "trace": trace}, nil
	case reflect.Map, reflect.Struct:
		raw, err := sonic.Marshal(result)
		if err != nil {
			return nil, err
		}
		obj := map[string]any{}
		if err := sonic.Unmarshal(raw, &obj); err != nil {
			return nil, err
		}
		obj["trace"] = trace
		return obj, nil
	default:
		return nil, errors.New("mcp result must be an object")
	}
}

// ErrorEnvelope builds the failed structuredContent object.
func ErrorEnvelope(errObj, trace map[string]any) map[string]any {
	if trace == nil {
		trace = map[string]any{}
	}
	return map[string]any{
		"error": errObj,
		"trace": trace,
	}
}
