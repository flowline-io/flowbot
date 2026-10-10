package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaFromFields(t *testing.T) {
	t.Parallel()
	schema := SchemaFromFields([]Field{
		{Name: "id", Type: "int64", Required: true, Description: "Entry ID"},
		{Name: "tags", Type: "[]string", Required: false},
	})
	assert.Equal(t, "object", schema["type"])
	props, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	id, ok := props["id"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "number", id["type"])
	assert.Equal(t, "Entry ID", id["description"])
	tags, ok := props["tags"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "array", tags["type"])
	assert.Equal(t, map[string]any{"type": "string"}, tags["items"])
	assert.Equal(t, []string{"id"}, schema["required"])

	empty := SchemaFromFields(nil)
	assert.Equal(t, "object", empty["type"])
	assert.Empty(t, empty["required"])
}
