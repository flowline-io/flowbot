package mcp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateInputSchema_Schema(t *testing.T) {
	t.Parallel()
	valid := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tags":    map[string]any{"type": "array", "description": "Tags", "items": map[string]any{"type": "string"}},
			"goals":   map[string]any{"type": "array", "description": "Goals"},
			"payload": map[string]any{"type": "object", "description": "Payload"},
		},
	}
	tests := []struct {
		name    string
		schema  map[string]any
		wantErr bool
	}{
		{name: "valid object with items and bare object", schema: valid},
		{name: "nil schema", schema: nil, wantErr: true},
		{name: "forbidden keyword oneOf", schema: map[string]any{"type": "object", "oneOf": []any{}}, wantErr: true},
		{
			name: "property missing description",
			schema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"name": map[string]any{"type": "string"}},
			},
			wantErr: true,
		},
		{
			name: "type is an array",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": []any{"string", "null"}, "description": "Name"},
				},
			},
			wantErr: true,
		},
		{
			name: "empty enum",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string", "description": "Name", "enum": []string{}},
				},
			},
			wantErr: true,
		},
		{
			name: "ref inside items",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"tags": map[string]any{
						"type":        "array",
						"description": "Tags",
						"items":       map[string]any{"type": "string", "$ref": "#/defs/tag"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "required field missing from properties",
			schema: map[string]any{
				"type":       "object",
				"required":   []string{"name"},
				"properties": map[string]any{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateInputSchema(tt.schema)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
