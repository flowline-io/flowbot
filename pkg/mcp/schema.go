package mcp

import (
	"bytes"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

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

// OutputSchema returns the MCP outputSchema for a tool, or nil when v1.0 does not publish one.
// Listed tools are pipeline, workflow, and function list/get/run/get_run, hub apps and health,
// and capability operations whose name contains "list".
func OutputSchema(group, op string) map[string]any {
	switch group {
	case "pipeline", "workflow", "function":
		switch op {
		case "list":
			return oneOfOutput(itemsSuccessSchema())
		case "get", "run", "get_run":
			return oneOfOutput(objectSuccessSchema())
		default:
			return nil
		}
	case "hub":
		switch op {
		case "apps":
			return oneOfOutput(itemsSuccessSchema())
		case "health":
			return oneOfOutput(objectSuccessSchema())
		default:
			return nil
		}
	default:
		if strings.Contains(op, "list") {
			return oneOfOutput(invokeSuccessSchema())
		}
		return nil
	}
}

// ValidateOutput checks instance against a tool outputSchema.
func ValidateOutput(schema, instance any) error {
	rawSchema, err := sonic.Marshal(schema)
	if err != nil {
		return err
	}
	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(rawSchema))
	if err != nil {
		return err
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource("output.schema.json", schemaDoc); err != nil {
		return err
	}
	sch, err := c.Compile("output.schema.json")
	if err != nil {
		return err
	}
	rawInst, err := sonic.Marshal(instance)
	if err != nil {
		return err
	}
	var doc any
	if err := sonic.Unmarshal(rawInst, &doc); err != nil {
		return err
	}
	return sch.Validate(doc)
}

func oneOfOutput(success map[string]any) map[string]any {
	return map[string]any{
		"oneOf": []any{success, errorEnvelopeSchema()},
	}
}

func itemsSuccessSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"required":             []string{"items", "trace"},
		"additionalProperties": false,
		"properties": map[string]any{
			"items": map[string]any{"type": "array"},
			"trace": traceSchema(),
		},
		"not": notToolErrorObject(),
	}
}

func objectSuccessSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"required":             []string{"trace"},
		"additionalProperties": true,
		"properties": map[string]any{
			"trace": traceSchema(),
		},
		"not": notToolErrorObject(),
	}
}

func invokeSuccessSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"required":             []string{"trace"},
		"additionalProperties": true,
		"properties": map[string]any{
			"capability": map[string]any{"type": "string"},
			"operation":  map[string]any{"type": "string"},
			"data":       map[string]any{},
			"page":       pageSchema(),
			"text":       map[string]any{"type": "string"},
			"meta":       map[string]any{"type": "object"},
			"events":     map[string]any{"type": "array"},
			"_resource":  map[string]any{"type": "object"},
			"trace":      traceSchema(),
		},
		"not": notToolErrorObject(),
	}
}

// notToolErrorObject rejects a tool-error envelope on the success branch.
// A domain string field named error still matches success.
func notToolErrorObject() map[string]any {
	return map[string]any{
		"required": []string{"error"},
		"properties": map[string]any{
			"error": map[string]any{
				"type":     "object",
				"required": []string{"code", "message", "retryable", "hint"},
			},
		},
	}
}

func pageSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"limit":       map[string]any{"type": "integer"},
			"has_more":    map[string]any{"type": "boolean"},
			"next_cursor": map[string]any{"type": "string"},
			"prev_cursor": map[string]any{"type": "string"},
			"total":       map[string]any{"type": "integer"},
		},
		"additionalProperties": false,
	}
}

func traceSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"required":             []string{"request_id"},
		"additionalProperties": false,
		"properties": map[string]any{
			"request_id":     map[string]any{"type": "string", "minLength": 1},
			"trace_id":       map[string]any{"type": "string", "pattern": "^[0-9a-f]{32}$"},
			"correlation_id": map[string]any{"type": "string", "minLength": 1},
		},
	}
}

func errorEnvelopeSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"required":             []string{"error", "trace"},
		"additionalProperties": false,
		"properties": map[string]any{
			"error": errorObjectSchema(),
			"trace": traceSchema(),
		},
	}
}

func errorObjectSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"required":             []string{"code", "message", "retryable", "hint"},
		"additionalProperties": false,
		"properties": map[string]any{
			"code":      map[string]any{"type": "string", "minLength": 1},
			"message":   map[string]any{"type": "string", "minLength": 1},
			"retryable": map[string]any{"type": "boolean"},
			"hint":      map[string]any{"type": "string"},
			"retry_after_ms": map[string]any{
				"type":    "integer",
				"minimum": 1,
			},
			"field":                 map[string]any{"type": "string", "minLength": 1},
			"job_id":                map[string]any{"type": "string", "minLength": 1},
			"confirmation_required": map[string]any{"type": "object"},
		},
	}
}

func jsonSchemaItems(t string) (map[string]any, bool) {
	t = strings.TrimSpace(t)
	if t == "array" || !strings.HasPrefix(t, "[]") {
		return nil, false
	}
	return map[string]any{"type": jsonSchemaType(strings.TrimPrefix(t, "[]"))}, true
}
