package tool

import (
	"fmt"
	"strings"
)

// ValidateArgs checks required fields and top-level JSON Schema types against args.
// It does not implement a full JSON Schema engine.
func ValidateArgs(schema map[string]any, args map[string]any) error {
	if schema == nil {
		return nil
	}
	if args == nil {
		args = map[string]any{}
	}

	if err := validateRequired(schema, args); err != nil {
		return err
	}
	return validatePropertyTypes(schema, args)
}

func validateRequired(schema map[string]any, args map[string]any) error {
	requiredRaw, ok := schema["required"].([]any)
	if !ok {
		return nil
	}
	for _, item := range requiredRaw {
		name, ok := item.(string)
		if !ok || name == "" {
			continue
		}
		value, exists := args[name]
		if !exists || isEmptyArg(value) {
			return fmt.Errorf("%s", FormatToolError(
				"invalid_args",
				fmt.Sprintf("missing required argument %q", name),
				"provide all required parameters from the tool schema",
			))
		}
	}
	return nil
}

func validatePropertyTypes(schema map[string]any, args map[string]any) error {
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		return nil
	}
	for name, rawProp := range properties {
		value, exists := args[name]
		if !exists || value == nil {
			continue
		}
		prop, ok := rawProp.(map[string]any)
		if !ok {
			continue
		}
		expected, ok := prop["type"].(string)
		if !ok || expected == "" {
			continue
		}
		if !matchesJSONType(expected, value) {
			return fmt.Errorf("%s", FormatToolError(
				"invalid_args",
				fmt.Sprintf("argument %q must be type %s", name, expected),
				"fix the argument type and retry",
			))
		}
	}
	return nil
}

func isEmptyArg(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(v) == ""
	default:
		return false
	}
}

func matchesJSONType(expected string, value any) bool {
	switch expected {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		switch value.(type) {
		case float64, float32, int, int32, int64, uint, uint32, uint64:
			return true
		default:
			return false
		}
	case "integer":
		return isIntegerValue(value)
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		switch value.(type) {
		case []any, []string:
			return true
		default:
			return false
		}
	default:
		return true
	}
}

func isIntegerValue(value any) bool {
	switch v := value.(type) {
	case int, int32, int64, uint, uint32, uint64:
		return true
	case float64:
		return v == float64(int64(v))
	default:
		return false
	}
}
