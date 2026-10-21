package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/capability/confluence"
	"github.com/flowline-io/flowbot/pkg/capability/core"
	"github.com/flowline-io/flowbot/pkg/capability/devops"
	"github.com/flowline-io/flowbot/pkg/capability/email"
	"github.com/flowline-io/flowbot/pkg/capability/fireflyiii"
	"github.com/flowline-io/flowbot/pkg/capability/functions"
	"github.com/flowline-io/flowbot/pkg/capability/gitea"
	"github.com/flowline-io/flowbot/pkg/capability/github"
	"github.com/flowline-io/flowbot/pkg/capability/kanboard"
	"github.com/flowline-io/flowbot/pkg/capability/karakeep"
	"github.com/flowline-io/flowbot/pkg/capability/memos"
	"github.com/flowline-io/flowbot/pkg/capability/miniflux"
	"github.com/flowline-io/flowbot/pkg/capability/nocodb"
	"github.com/flowline-io/flowbot/pkg/capability/transmission"
	"github.com/flowline-io/flowbot/pkg/capability/trello"
	"github.com/flowline-io/flowbot/pkg/capability/trilium"
	"github.com/flowline-io/flowbot/pkg/hub"
	pkgmcp "github.com/flowline-io/flowbot/pkg/mcp"
)

func TestAdvertisedToolDescriptions(t *testing.T) {
	t.Parallel()
	specs := []capability.Spec{
		confluence.CatalogSpec(),
		core.CatalogSpec(),
		devops.CatalogSpec(),
		email.CatalogSpec(),
		fireflyiii.CatalogSpec(),
		functions.CatalogSpec(),
		gitea.CatalogSpec(),
		github.CatalogSpec(),
		kanboard.CatalogSpec(),
		karakeep.CatalogSpec(),
		memos.CatalogSpec(),
		miniflux.CatalogSpec(),
		nocodb.CatalogSpec(),
		transmission.CatalogSpec(),
		trello.CatalogSpec(),
		trilium.CatalogSpec(),
	}
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
