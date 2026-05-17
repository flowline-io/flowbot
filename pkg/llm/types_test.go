package llm_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/llm"
)

func TestRoleConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		role  string
		value string
	}{
		{name: "system role", role: "system", value: llm.SystemRole},
		{name: "user role", role: "user", value: llm.UserRole},
		{name: "assistant role", role: "model", value: llm.AssistantRole},
		{name: "tool role", role: "tool", value: llm.ToolRole},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.role, tt.value)
		})
	}
}

func TestMessage_Fields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  llm.Message
	}{
		{
			name: "full message with content",
			msg:  llm.Message{Role: llm.UserRole, Content: "hello", Name: "test"},
		},
		{
			name: "empty content message",
			msg:  llm.Message{Role: llm.AssistantRole, Content: ""},
		},
		{
			name: "message without name",
			msg:  llm.Message{Role: llm.SystemRole, Content: "system prompt"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, tt.msg.Role)
		})
	}
}

func TestParamsOneOf_ToJSONSchema(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		params *llm.ParamsOneOf
		check  func(t *testing.T, result map[string]any, err error)
	}{
		{
			name: "with properties and required",
			params: &llm.ParamsOneOf{
				OneOf: []llm.Schema{{
					Type:        "object",
					Description: "test",
					Properties: map[string]llm.Schema{
						"key": {Type: "string", Description: "a key"},
					},
					Required: []string{"key"},
				}},
			},
			check: func(t *testing.T, result map[string]any, err error) {
				require.NoError(t, err)
				assert.Equal(t, "object", result["type"])
				props, ok := result["properties"].(map[string]any)
				assert.True(t, ok)
				assert.Contains(t, props, "key")
				required, ok := result["required"].([]string)
				assert.True(t, ok)
				assert.Contains(t, required, "key")
			},
		},
		{
			name:   "nil receiver returns default",
			params: nil,
			check: func(t *testing.T, result map[string]any, err error) {
				require.NoError(t, err)
				assert.Equal(t, "object", result["type"])
			},
		},
		{
			name: "empty oneOf returns default",
			params: &llm.ParamsOneOf{
				OneOf: []llm.Schema{},
			},
			check: func(t *testing.T, result map[string]any, err error) {
				require.NoError(t, err)
				assert.Equal(t, "object", result["type"])
			},
		},
		{
			name: "multiple schemas merge properties and required",
			params: &llm.ParamsOneOf{
				OneOf: []llm.Schema{
					{
						Properties: map[string]llm.Schema{
							"a": {Type: "string"},
						},
						Required: []string{"a"},
					},
					{
						Properties: map[string]llm.Schema{
							"b": {Type: "number"},
						},
						Required: []string{"b"},
					},
				},
			},
			check: func(t *testing.T, result map[string]any, err error) {
				require.NoError(t, err)
				props, ok := result["properties"].(map[string]any)
				assert.True(t, ok)
				assert.Contains(t, props, "a")
				assert.Contains(t, props, "b")
				required, ok := result["required"].([]string)
				assert.True(t, ok)
				assert.Contains(t, required, "a")
				assert.Contains(t, required, "b")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := tt.params.ToJSONSchema()
			tt.check(t, result, err)
		})
	}
}
