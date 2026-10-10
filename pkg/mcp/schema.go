package mcp

import "strings"

// Field is one JSON-object property used to build an MCP input schema.
type Field struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

// SchemaFromFields builds a JSON Schema object for MCP / agent tool parameters.
func SchemaFromFields(fields []Field) map[string]any {
	properties := map[string]any{}
	required := make([]string, 0)
	for _, p := range fields {
		if p.Name == "" {
			continue
		}
		prop := map[string]any{
			"type": jsonSchemaType(p.Type),
		}
		if p.Description != "" {
			prop["description"] = p.Description
		}
		if items, ok := jsonSchemaItems(p.Type); ok {
			prop["items"] = items
		}
		properties[p.Name] = prop
		if p.Required {
			required = append(required, p.Name)
		}
	}
	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func jsonSchemaType(t string) string {
	t = strings.TrimSpace(t)
	switch t {
	case "int", "int32", "int64", "float", "float64", "number":
		return "number"
	case "bool", "boolean":
		return "boolean"
	case "object", "map[string]any", "map[string]interface{}":
		return "object"
	case "array":
		return "array"
	default:
		if strings.HasPrefix(t, "[]") {
			return "array"
		}
		if strings.HasPrefix(t, "map[") {
			return "object"
		}
		return "string"
	}
}

func jsonSchemaItems(t string) (map[string]any, bool) {
	t = strings.TrimSpace(t)
	if t == "array" || !strings.HasPrefix(t, "[]") {
		return nil, false
	}
	return map[string]any{"type": jsonSchemaType(strings.TrimPrefix(t, "[]"))}, true
}
