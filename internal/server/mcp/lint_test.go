package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
	pkgmcp "github.com/flowline-io/flowbot/pkg/mcp"
)

func TestAdvertisedToolDescriptions(t *testing.T) {
	t.Parallel()
	specs := advertisedCatalogSpecs()
	descs := make([]hub.Descriptor, 0, len(specs))
	for _, spec := range specs {
		descs = append(descs, capability.DescriptorFromSpec(spec))
	}
	tools := BuildCatalog(descs, nil, nil)
	require.NotEmpty(t, tools)
	for _, tool := range tools {
		assert.GreaterOrEqual(t, len(tool.Description), 60, tool.Name)
		assert.Contains(t, tool.Description, "Example:", tool.Name)
		for _, p := range tool.Input {
			assert.NotEmpty(t, p.Description, tool.Name+" "+p.Name)
			if len(p.Enum) == 0 {
				continue
			}
			schema := schemaFromParams([]hub.ParamDef{p})
			props, ok := schema["properties"].(map[string]any)
			require.True(t, ok, tool.Name+" "+p.Name)
			prop, ok := props[p.Name].(map[string]any)
			require.True(t, ok, tool.Name+" "+p.Name)
			assert.Equal(t, p.Enum, prop["enum"], tool.Name+" "+p.Name)
		}
	}
	assert.NotContains(t, toolNames(tools), "core.run_terminal")
}

func toolNames(tools []ToolSpec) []string {
	out := make([]string, 0, len(tools))
	for _, tool := range tools {
		if pkgmcp.Denied(tool.Group, tool.Operation) {
			out = append(out, tool.Name)
		}
		out = append(out, tool.Name)
	}
	return out
}
